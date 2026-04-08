package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"zentora-service/internal/config"
	"zentora-service/internal/db"
	authHandler "zentora-service/internal/handlers/auth"
	cartHandler "zentora-service/internal/handlers/cart"
	catalogH "zentora-service/internal/handlers/catalog"
	discoveryH "zentora-service/internal/handlers/discovery"
	notifyH "zentora-service/internal/handlers/notification"
	orderHandler "zentora-service/internal/handlers/order"
	userH "zentora-service/internal/handlers/user"
	wsHandler "zentora-service/internal/handlers/websocket"
	"zentora-service/internal/middleware"
	"zentora-service/internal/pkg/jwt"
	"zentora-service/internal/pkg/session"
	"zentora-service/internal/repository/postgres"
	authUsecase "zentora-service/internal/service/auth"
	cartsvc "zentora-service/internal/service/cart"
	catalogUsecase "zentora-service/internal/service/catalog"
	discoveryUsecase "zentora-service/internal/service/discovery"
	"zentora-service/internal/service/email"
	notifyUsecase "zentora-service/internal/service/notification"
	orderusecase "zentora-service/internal/service/order"
	userUsecase "zentora-service/internal/service/user"
	wishlistsvc "zentora-service/internal/service/wishlist"
	workerUsecase "zentora-service/internal/service/worker"
	"zentora-service/internal/websocket"
	wsHandlers "zentora-service/internal/websocket/handler"

	wishlistHandler "zentora-service/internal/handlers/wishlist"

	"github.com/gin-gonic/gin"
	"github.com/imagekit-developer/imagekit-go/v2"
	"github.com/imagekit-developer/imagekit-go/v2/option"
	"go.uber.org/zap"
)

type Server struct {
	cfg         config.AppConfig
	engine      *gin.Engine
	logger      *zap.Logger
	authService *authUsecase.AuthService
}

func NewServer() *Server {
	cfg := config.Load()
	engine := gin.Default()
	logger, _ := zap.NewProduction()
	return &Server{cfg: cfg, engine: engine, logger: logger}
}

func (s *Server) Start() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.ConnectDB()
	if err != nil {
		return fmt.Errorf("postgres connect: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("postgres ping: %w", err)
	}

	redisCfg := db.RedisConfig{
		ClusterMode: false,
		Addresses:   []string{s.cfg.RedisAddr},
		Password:    s.cfg.RedisPass,
		DB:          0,
		PoolSize:    10,
	}

	redisClient, err := db.NewRedisClient(redisCfg)
	if err != nil {
		return fmt.Errorf("redis connect: %w", err)
	}

	defer func() {
		_ = s.logger.Sync()
	}()

	jwtManager, err := jwt.LoadAndBuild(s.cfg.JWT)
	if err != nil {
		return fmt.Errorf("jwt load: %w", err)
	}

	var ikClient *imagekit.Client
	if key := s.cfg.ImageKitPrivateKey; key != "" {
		ikClient = &imagekit.Client{}
		*ikClient = imagekit.NewClient(option.WithPrivateKey(key))
	}

	emailSender := email.NewEmailSender(
		s.cfg.SMTPHost,
		s.cfg.SMTPPort,
		s.cfg.SMTPUser,
		s.cfg.SMTPPass,
		s.cfg.SMTPFromName,
		s.cfg.BaseURL,
		s.cfg.LogoURL,
		s.cfg.SMTPSecure,
	)

	authRepo := postgres.NewAuthRepository(pool)
	notifyRepo := postgres.NewNotificationRepository(pool)
	discoveryRepo := postgres.NewDiscoveryRepository(pool)
	productEventsRepo := postgres.NewProductEventsRepository(pool)
	wishlistRepo := postgres.NewWishlistRepository(pool, productEventsRepo)

	categoryRepo := postgres.NewCategoryRepository(pool)
	brandRepo := postgres.NewBrandRepository(pool)
	tagRepo := postgres.NewTagRepository(pool)
	attributeRepo := postgres.NewAttributeRepository(pool)
	variantRepo := postgres.NewVariantRepository(pool)

	userAddressRepo := postgres.NewUserAddressRepository(pool)
	inventoryRepo := postgres.NewInventoryRepository(pool)
	discountRepo := postgres.NewDiscountRepository(pool)
	searchRepo := postgres.NewProductSearchRepository()

	productRepo := postgres.NewProductRepository(
		pool,
		attributeRepo,
		brandRepo,
		categoryRepo,
		discountRepo,
		inventoryRepo,
		tagRepo,
		variantRepo,
		searchRepo,
		productEventsRepo,
	)

	cartRepo := postgres.NewCartRepository(pool, productRepo, variantRepo)
	orderRepo := postgres.NewOrderRepository(pool)

	sessionManager := session.NewManager(redisClient, authRepo)
	rateLimiter := session.NewRateLimiter(redisClient)

	hub := websocket.NewHub(jwtManager.Verifier, sessionManager)
	hub.RegisterHandler(wsHandlers.NewNotificationHandler(notifyRepo))
	go hub.Run(ctx)

	authService := authUsecase.NewAuthService(
		authRepo,
		jwtManager,
		sessionManager,
		rateLimiter,
		emailSender,
		hub,
		redisClient,
		s.logger,
	)
	s.authService = authService

	notifService := notifyUsecase.NewNotificationService(notifyRepo, hub)

	catalogService := catalogUsecase.NewCatalogService(
		categoryRepo,
		brandRepo,
		tagRepo,
		productRepo,
		attributeRepo,
		variantRepo,
		inventoryRepo,
		discountRepo,
		ikClient,
	)

	userService := userUsecase.NewUserService(userAddressRepo)
	discoveryService := discoveryUsecase.NewDiscoveryService(discoveryRepo, categoryRepo, redisClient)
	metricsJobService := workerUsecase.NewMetricsJobService(
		discoveryRepo,
		s.cfg.WorkerMetricsInterval,
		zap.NewStdLog(s.logger),
	)

	go func() {
		if err := metricsJobService.Start(ctx); err != nil && ctx.Err() == nil {
			log.Printf("metrics worker stopped: %v", err)
		}
	}()

	cartService := cartsvc.NewService(cartRepo, redisClient)
	wishlistService := wishlistsvc.NewService(wishlistRepo, redisClient)
	orderService := orderusecase.NewService(
		pool,
		orderRepo,
		cartRepo,
		productRepo,
		variantRepo,
		inventoryRepo,
		userAddressRepo,
		discountRepo,
	)

	_ = s.initializeSuperAdmin()

	authHandlerInst := authHandler.NewAuthHandler(authService, s.logger)
	notifHandler := notifyH.NewNotificationHandler(notifService)
	wsHandlerInst := wsHandler.NewWebSocketHandler(hub, s.logger)
	catalogHandlerInst := catalogH.NewCatalogHandler(catalogService, s.logger)
	discoveryHandlerInst := discoveryH.NewHandler(discoveryService, metricsJobService)
	userHandlerInst := userH.NewUserHandler(userService)
	cartHandlerInst := cartHandler.NewHandler(cartService)
	wishListHandlerInst := wishlistHandler.NewHandler(wishlistService)
	orderHandlerInst := orderHandler.NewHandler(orderService)

	authMiddleware := middleware.NewAuthMiddleware(authService)

	s.engine.Use(
		middleware.RecoveryMiddleware(s.logger),
		middleware.LoggingMiddleware(s.logger),
		middleware.CORSMiddleware(),
	)

	handlers := &Handlers{
		AuthHandler:      authHandlerInst,
		NotifHandler:     notifHandler,
		WSHandler:        wsHandlerInst,
		CatalogHandler:   catalogHandlerInst,
		DiscoveryHandler: discoveryHandlerInst,
		UserHandler:      userHandlerInst,
		AuthMiddleware:   authMiddleware,
		CartHandler:      cartHandlerInst,
		OrderHandler:     orderHandlerInst,
		WishlistHandler:  wishListHandlerInst,
	}
	SetupRouter(s.engine, s.logger, handlers)

	srv := &http.Server{
		Addr:    s.cfg.HTTPAddr,
		Handler: s.engine,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		return nil
	case err := <-errCh:
		return err
	}
}

func (s *Server) initializeSuperAdmin() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	email := os.Getenv("SUPER_ADMIN_EMAIL")
	password := os.Getenv("SUPER_ADMIN_PASSWORD")
	fullName := os.Getenv("SUPER_ADMIN_NAME")

	if email == "" {
		email = "admin@zentorashop.co.ke"
	}
	if password == "" {
		password = "HappyOwl58&"
	}
	if fullName == "" {
		fullName = "Zentora Administrator"
	}

	if len(password) < 8 {
		return fmt.Errorf("super admin password must be at least 8 characters")
	}

	if err := s.authService.EnsureSuperAdminExists(ctx, email, password, fullName); err != nil {
		return fmt.Errorf("ensure super admin: %w", err)
	}

	return nil
}