package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"chatbox-gin/internal/auth"
	"chatbox-gin/internal/bot"
	"chatbox-gin/internal/channel"
	"chatbox-gin/internal/config"
	"chatbox-gin/internal/database"
	"chatbox-gin/internal/handlers"
	"chatbox-gin/internal/middleware"
	"chatbox-gin/internal/realtime"
	"chatbox-gin/internal/repositories"
	"chatbox-gin/internal/services"
	"chatbox-gin/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	// =========================================================================
	// Load configuration
	// =========================================================================
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// =========================================================================
	// Khởi tạo Logger
	// =========================================================================
	log, err := logger.NewLogger(cfg.Logging.Level, cfg.Logging.Format)
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync()

	log.Info("starting server",
		zap.String("app", cfg.App.Name),
		zap.String("env", cfg.App.Env),
		zap.Int("port", cfg.App.Port),
	)

	// =========================================================================
	// Kết nối Database
	// =========================================================================
	db, err := database.NewConnection(&cfg.Database, log)
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}
	defer database.Close(db)

	// Auto migrate trong development mode
	if cfg.App.IsDevelopment() {
		if err := database.AutoMigrate(db); err != nil {
			log.Warn("auto migrate failed", zap.Error(err))
		} else {
			log.Info("database auto migration completed")
		}
	}

	// =========================================================================
	// Khởi tạo Repositories
	// =========================================================================
	participantRepo := repositories.NewParticipantRepository(db)
	conversationRepo := repositories.NewConversationRepository(db)
	messageRepo := repositories.NewMessageRepository(db)
	ruleRepo := repositories.NewRuleRepository(db)
	userRepo := repositories.NewUserRepository(db)
	channelAccountRepo := repositories.NewChannelAccountRepository(db)

	log.Info("repositories initialized")

	// =========================================================================
	// Khởi tạo Channel Registry và đăng ký channels
	// =========================================================================
	channelRegistry := channel.NewRegistry()

	// Đăng ký Mock Channel (dùng để testing)
	mockChannel := channel.NewMockChannel(log)
	channelRegistry.Register(mockChannel)
	log.Info("registered channel", zap.String("type", mockChannel.Type()))

	// Đăng ký Facebook Channel
	fbChannel := channel.NewFacebookChannel(log)
	channelRegistry.Register(fbChannel)
	log.Info("registered channel", zap.String("type", fbChannel.Type()))

	// TODO: Đăng ký Zalo Channel khi implement

	// =========================================================================
	// Khởi tạo Bot Engine (Rule matching + Response generation)
	// =========================================================================
	ruleEngine := bot.NewRuleEngine(log)
	responseBuilder := bot.NewResponseBuilder()
	botResponder := bot.NewResponder(ruleRepo, ruleEngine, responseBuilder, log)

	log.Info("bot engine initialized")

	// =========================================================================
	// Khởi tạo Realtime Publisher (Centrifugo)
	// =========================================================================
	var publisher realtime.Publisher
	centrifugoURL := os.Getenv("CENTRIFUGO_URL")
	centrifugoAPIKey := os.Getenv("CENTRIFUGO_API_KEY")

	if centrifugoURL != "" && centrifugoAPIKey != "" {
		publisher = realtime.NewCentrifugoClient(centrifugoURL, centrifugoAPIKey, log)
		log.Info("centrifugo publisher initialized", zap.String("url", centrifugoURL))
	} else {
		publisher = realtime.NewNoopPublisher()
		log.Warn("centrifugo not configured, using noop publisher")
	}

	// =========================================================================
	// Khởi tạo Services
	// =========================================================================
	messageService := services.NewMessageService(
		participantRepo,
		conversationRepo,
		messageRepo,
		channelAccountRepo,
		channelRegistry,
		botResponder,
		publisher,
		log,
	)

	log.Info("services initialized")

	// =========================================================================
	// Khởi tạo Handlers
	// =========================================================================
	mockHandler := handlers.NewMockHandler(channelRegistry, messageService, log)
	ruleHandler := handlers.NewRuleHandler(ruleRepo, log)
	conversationHandler := handlers.NewConversationHandler(
		conversationRepo,
		messageRepo,
		participantRepo,
		channelAccountRepo,
		channelRegistry,
		publisher,
		log,
	)

	// Auth handler
	jwtService := auth.NewJWTService(cfg.JWT)
	authService := services.NewAuthService(userRepo, jwtService, log)
	authHandler := handlers.NewAuthHandler(authService, log)
	authMiddleware := middleware.AuthMiddleware(jwtService)

	// Webhook handler
	fbVerifyToken := os.Getenv("FB_VERIFY_TOKEN")
	if fbVerifyToken == "" {
		fbVerifyToken = "chatbox_fb_verify_2024"
	}
	webhookHandler := handlers.NewWebhookHandler(
		channelRegistry,
		channelAccountRepo,
		messageService,
		fbVerifyToken,
		log,
	)

	log.Info("handlers initialized")

	// =========================================================================
	// Thiết lập Gin Router
	// =========================================================================
	if cfg.App.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Middleware
	router.Use(middleware.RequestID())
	router.Use(middleware.Recovery(log))
	router.Use(middleware.Logging(log))
	router.Use(middleware.CORS([]string{"*"}))
	// CSRF protection - exempt auth và webhook routes
	router.Use(middleware.CSRFMiddlewareWithExempt([]string{
		"/api/v1/auth/",    // Login, refresh không cần CSRF ban đầu
		"/api/v1/mock/",    // Mock webhooks
		"/api/v1/webhook/", // FB, Zalo webhooks từ external services
		"/health",          // Health check
	}))

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":   "healthy",
			"service":  cfg.App.Name,
			"version":  "1.0.0",
			"channels": channelRegistry.GetAll(),
		})
	})

	// =========================================================================
	// API Routes
	// =========================================================================
	api := router.Group("/api/v1")
	{
		// Ping endpoint (public)
		api.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "pong"})
		})

		// Auth routes (login, refresh: public | me, logout: protected)
		authHandler.RegisterRoutes(api, authMiddleware)

		// Mock channel routes (public - for testing only)
		// TODO: Disable in production hoặc thêm API key
		mockHandler.RegisterRoutes(api)

		// Webhook routes (public - for FB, Zalo webhooks)
		webhookHandler.RegisterRoutes(api)

		// =====================================================================
		// Protected routes - Require authentication
		// =====================================================================
		protected := api.Group("")
		protected.Use(authMiddleware)
		{
			// Conversation & Message routes
			conversationHandler.RegisterRoutes(protected)

			// Rule management routes (dashboard)
			ruleHandler.RegisterRoutes(protected)
		}
	}

	log.Info("routes registered",
		zap.Strings("endpoints", []string{
			"/api/v1/mock/*",
			"/api/v1/conversations",
			"/api/v1/conversations/:id",
			"/api/v1/conversations/:id/messages",
			"/api/v1/rules",
		}),
	)

	// =========================================================================
	// Khởi động HTTP Server
	// =========================================================================
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.App.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("server listening", zap.Int("port", cfg.App.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("failed to start server", zap.Error(err))
		}
	}()

	// =========================================================================
	// Graceful Shutdown
	// =========================================================================
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("server forced to shutdown", zap.Error(err))
	}

	log.Info("server exited")
}