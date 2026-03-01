// internal/app/server.go
package app

import (
	"context"
	"fmt"
	"log"

	"diary-service/internal/config"
	"diary-service/internal/db"
	authHandler "diary-service/internal/handlers/auth"
	notifyH "diary-service/internal/handlers/notification"
	wsHandler "diary-service/internal/handlers/websocket"
	"diary-service/internal/middleware"
	"diary-service/internal/pkg/jwt"
	"diary-service/internal/pkg/session"
	"diary-service/internal/repository/postgres"
	"diary-service/internal/service/email"
	authUsecase "diary-service/internal/service/auth"
	notifyUsecase "diary-service/internal/service/notification"
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

	// ----- Handlers -----
	authHandlerInst := authHandler.NewAuthHandler(authService, logger)
	notifHandler := notifyH.NewNotificationHandler( notifService)
	wsHandlerInst := wsHandler.NewWebSocketHandler(hub, logger)

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
		AuthMiddleware: authMiddleware,
	}
	SetupRouter(s.engine, logger, handlers)

	// ----- Start HTTP -----
	log.Printf("🚀 Server running on %s", s.cfg.HTTPAddr)
	return s.engine.Run(s.cfg.HTTPAddr)
}