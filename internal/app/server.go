// internal/app/server.go
package app

import (
	"context"
	"fmt"
	"log"

	"diary-service/internal/config"
	"diary-service/internal/db"
	authHandler "diary-service/internal/handlers/auth"
	catalogH "diary-service/internal/handlers/catalog"
	notifyH "diary-service/internal/handlers/notification"
	userH "diary-service/internal/handlers/user"
	wsHandler "diary-service/internal/handlers/websocket"
	"diary-service/internal/middleware"
	"diary-service/internal/pkg/jwt"
	"diary-service/internal/pkg/session"
	"diary-service/internal/repository/postgres"
	"diary-service/internal/service/email"
	authUsecase "diary-service/internal/service/auth"
	catalogUsecase "diary-service/internal/service/catalog"
	notifyUsecase "diary-service/internal/service/notification"
	userUsecase "diary-service/internal/service/user"
	"diary-service/internal/websocket"
	wsHandlers "diary-service/internal/websocket/handler"

	"github.com/gin-gonic/gin"
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

	// ----- Email -----
	emailSender := email.NewEmailSender(
		s.cfg.SMTPHost,
		s.cfg.SMTPPort,
		s.cfg.SMTPUser,
		s.cfg.SMTPPass,
		s.cfg.SMTPFromName,
		s.cfg.SMTPSecure,
	)

	// ----- Repositories -----
	authRepo := postgres.NewAuthRepository(pool)
	notifyRepo := postgres.NewNotificationRepository(pool)

	// Catalog and user repositories
	categoryRepo := postgres.NewCategoryRepository(pool)
	brandRepo := postgres.NewBrandRepository(pool)
	tagRepo := postgres.NewTagRepository(pool)
	productRepo := postgres.NewProductRepository(pool)
	attributeRepo := postgres.NewAttributeRepository(pool)
	variantRepo := postgres.NewVariantRepository(pool)
	userAddressRepo := postgres.NewUserAddressRepository(pool)

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