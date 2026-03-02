// internal/app/server.go
package app

import (
	"context"
	"fmt"
	"log"

	"zentora-service/internal/config"
	"zentora-service/internal/db"
	authHandler "zentora-service/internal/handlers/auth"
	catalogH "zentora-service/internal/handlers/catalog"
	notifyH "zentora-service/internal/handlers/notification"
	userH "zentora-service/internal/handlers/user"
	wsHandler "zentora-service/internal/handlers/websocket"
	"zentora-service/internal/middleware"
	"zentora-service/internal/pkg/jwt"
	"zentora-service/internal/pkg/session"
	"zentora-service/internal/repository/postgres"
	authUsecase "zentora-service/internal/service/auth"
	catalogUsecase "zentora-service/internal/service/catalog"
	"zentora-service/internal/service/email"
	notifyUsecase "zentora-service/internal/service/notification"
	userUsecase "zentora-service/internal/service/user"
	"zentora-service/internal/websocket"
	wsHandlers "zentora-service/internal/websocket/handler"

	"github.com/gin-gonic/gin"
	"github.com/imagekit-developer/imagekit-go/v2"
	"github.com/imagekit-developer/imagekit-go/v2/option"
	"go.uber.org/zap"
)

type Server struct {
	cfg    config.AppConfig
	engine *gin.Engine
}

func NewServer() *Server {
	cfg := config.Load()
	engine := gin.Default()
	return &Server{cfg: cfg, engine: engine}
}

func (s *Server) Start() error {
	ctx := context.Background()

	// ----- PostgreSQL -----
	pool, err := db.ConnectDB()
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	// ----- Redis -----
	redisCfg := db.RedisConfig{
		ClusterMode: false,
		Addresses:   []string{s.cfg.RedisAddr},
		Password:    s.cfg.RedisPass,
		DB:          0,
		PoolSize:    10,
	}

	redisClient, err := db.NewRedisClient(redisCfg)
	if err != nil {
		log.Fatalf("[REDIS] ❌ Failed to connect to Redis: %v", err)
	}
	log.Println("[REDIS] ✅ Connected successfully")

	// ----- Logger -----
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// ----- JWT Manager -----
	jwtManager, err := jwt.LoadAndBuild(s.cfg.JWT)
	if err != nil {
		return fmt.Errorf("failed to load JWT manager: %w", err)
	}

	// ----- Session Manager & Rate Limiter -----
	sessionManager := session.NewManager(redisClient, nil) // Will set authRepo later
	rateLimiter := session.NewRateLimiter(redisClient)

	var ikClient *imagekit.Client
	if key := s.cfg.ImageKitPrivateKey; key != "" {
		ikClient = &imagekit.Client{}
		*ikClient = imagekit.NewClient(
			option.WithPrivateKey(key),
		)
	}

	// ----- Email -----
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

	// ----- Repositories -----
	authRepo := postgres.NewAuthRepository(pool)
	notifyRepo := postgres.NewNotificationRepository(pool)

	// Catalog and user repositories
	categoryRepo := postgres.NewCategoryRepository(pool)
	brandRepo := postgres.NewBrandRepository(pool)
	tagRepo := postgres.NewTagRepository(pool)
	attributeRepo := postgres.NewAttributeRepository(pool)
	variantRepo := postgres.NewVariantRepository(pool)
	userAddressRepo := postgres.NewUserAddressRepository(pool)
	inventoryRepo := postgres.NewInventoryRepository(pool)
	discountRepo := postgres.NewDiscountRepository(pool)
	productRepo := postgres.NewProductRepository(
		pool,
		attributeRepo,
		brandRepo,
		categoryRepo,
		discountRepo,
		inventoryRepo,
		tagRepo,
		variantRepo,
	)

	// Update session manager with auth repo
	sessionManager = session.NewManager(redisClient, authRepo)

	// ----- WebSocket Hub -----
	hub := websocket.NewHub(jwtManager.Verifier, sessionManager)

	// Register WebSocket handlers
	notificationWSHandler := wsHandlers.NewNotificationHandler(notifyRepo)
	hub.RegisterHandler(notificationWSHandler)

	// Start hub
	go hub.Run(context.Background())

	// ----- Services (Usecases) -----
	authService := authUsecase.NewAuthService(
		authRepo,
		jwtManager,
		sessionManager,
		rateLimiter,
		emailSender,
		hub,
		redisClient,
		logger,
	)

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

	// ----- Handlers -----
	authHandlerInst := authHandler.NewAuthHandler(authService, logger)
	notifHandler := notifyH.NewNotificationHandler(notifService)
	wsHandlerInst := wsHandler.NewWebSocketHandler(hub, logger)
	catalogHandlerInst := catalogH.NewCatalogHandler(catalogService)
	userHandlerInst := userH.NewUserHandler(userService)

	// ----- Middlewares -----
	authMiddleware := middleware.NewAuthMiddleware(authService)

	s.engine.Use(
		middleware.RecoveryMiddleware(logger),
		middleware.LoggingMiddleware(logger),
		middleware.CORSMiddleware(),
	)

	// ----- Router -----
	handlers := &Handlers{
		AuthHandler:    authHandlerInst,
		NotifHandler:   notifHandler,
		WSHandler:      wsHandlerInst,
		CatalogHandler: catalogHandlerInst,
		UserHandler:    userHandlerInst,
		AuthMiddleware: authMiddleware,
	}
	SetupRouter(s.engine, logger, handlers)

	// ----- Start HTTP -----
	log.Printf("🚀 Server running on %s", s.cfg.HTTPAddr)
	return s.engine.Run(s.cfg.HTTPAddr)
}
