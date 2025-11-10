package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/rajweepmondal/url-shortener/internal/config"
	grpcHandler "github.com/rajweepmondal/url-shortener/internal/handler/grpc"
	"github.com/rajweepmondal/url-shortener/internal/middleware"
	"github.com/rajweepmondal/url-shortener/internal/repository/postgres"
	"github.com/rajweepmondal/url-shortener/internal/repository/redis"
	"github.com/rajweepmondal/url-shortener/internal/router"
	"github.com/rajweepmondal/url-shortener/internal/service"
	"github.com/rajweepmondal/url-shortener/internal/utils"
	"github.com/rajweepmondal/url-shortener/pkg/auth"
	"github.com/rajweepmondal/url-shortener/pkg/ratelimiter"
	pb "github.com/rajweepmondal/url-shortener/proto/gen/go/url_shortener/v1"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Initialize logger
	logger, err := initLogger(cfg.Log)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Starting URL Shortener Service",
		zap.String("version", "1.0.0"),
		zap.String("port", cfg.Server.Port),
		zap.String("environment", "development"),
	)

	// Initialize database connections
	dbConn, err := utils.NewDatabaseConnection(cfg)
	if err != nil {
		logger.Fatal("Failed to connect to databases", zap.Error(err))
	}
	defer dbConn.Close()

	logger.Info("Database connections established")

	// Initialize repositories
	urlRepo := postgres.NewURLRepository(dbConn.PostgreSQL)
	analyticsRepo := postgres.NewAnalyticsRepository(dbConn.PostgreSQL)
	cacheRepo := redis.NewCacheRepository(dbConn.Redis)
	rateLimitRepo := redis.NewRateLimitRepository(dbConn.Redis)

	// Initialize services
	urlService := service.NewURLService(
		urlRepo,
		analyticsRepo,
		cacheRepo,
		cfg.App.ShortCodeLength,
		cfg.App.BaseURL,
		cfg.App.CacheTTL,
	)

	// Initialize authentication manager
	authConfig := auth.AuthConfig{
		JWTSecret:    cfg.Auth.JWTSecret,
		JWTDuration:  cfg.Auth.JWTDuration,
		JWTIssuer:    cfg.Auth.JWTIssuer,
		AdminAPIKey:  cfg.Auth.AdminAPIKey,
		EnableJWT:    cfg.Auth.EnableJWT,
		EnableAPIKey: cfg.Auth.EnableAPIKey,
	}

	authManager, err := auth.NewAuthManager(authConfig)
	if err != nil {
		logger.Fatal("Failed to initialize auth manager", zap.Error(err))
	}

	// Initialize rate limiter
	rateLimiterConfig := ratelimiter.Config{
		Strategy: ratelimiter.StrategySlidingWindow,
		Limit:    cfg.App.RateLimit,
		Window:   cfg.App.RateWindow,
	}
	rateLimiter := ratelimiter.New(rateLimitRepo, rateLimiterConfig)
	rateLimitMiddleware := ratelimiter.NewMiddleware(rateLimiter)

	// Initialize authentication middleware
	authMiddleware := middleware.NewAuthMiddleware(authManager, logger, cfg.Auth.RequireAuth)

	// Initialize gRPC server
	grpcServer := initGRPCServer(logger, rateLimitMiddleware, authMiddleware)

	// Register gRPC services
	urlHandler := grpcHandler.NewURLHandler(urlService)
	pb.RegisterURLShortenerServiceServer(grpcServer, urlHandler)

	// Enable reflection for development
	reflection.Register(grpcServer)

	// Initialize HTTP router
	httpRouter := router.New(urlService, logger, rateLimitMiddleware, authMiddleware)

	// Calculate ports
	grpcPort := cfg.Server.Port
	httpPortInt, _ := strconv.Atoi(grpcPort)
	httpPort := strconv.Itoa(httpPortInt + 1) // HTTP on port+1

	// Start gRPC server
	grpcListener, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		logger.Fatal("Failed to listen for gRPC", zap.Error(err))
	}

	// Start HTTP server
	httpServer := &http.Server{
		Addr:         ":" + httpPort,
		Handler:      httpRouter.Handler(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start gRPC server in a goroutine
	go func() {
		logger.Info("gRPC server starting",
			zap.String("address", grpcListener.Addr().String()),
			zap.String("port", grpcPort))
		if err := grpcServer.Serve(grpcListener); err != nil {
			logger.Fatal("Failed to serve gRPC", zap.Error(err))
		}
	}()

	// Start HTTP server in a goroutine
	go func() {
		logger.Info("HTTP server starting",
			zap.String("address", httpServer.Addr),
			zap.String("port", httpPort))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to serve HTTP", zap.Error(err))
		}
	}()

	// Log available routes
	routes := httpRouter.GetRoutes()
	logger.Info("HTTP routes registered", zap.Strings("routes", routes))

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.GracefulTimeout)
	defer cancel()

	// Shutdown HTTP server
	logger.Info("Shutting down HTTP server...")
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("HTTP server shutdown error", zap.Error(err))
	}

	// Shutdown gRPC server
	logger.Info("Shutting down gRPC server...")
	grpcServer.GracefulStop()

	// Wait for shutdown to complete or timeout
	done := make(chan struct{})
	go func() {
		grpcServer.Stop()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("Servers stopped gracefully")
	case <-ctx.Done():
		logger.Warn("Server shutdown timeout exceeded")
		grpcServer.Stop()
	}
}

// initLogger initializes the logger based on configuration
func initLogger(cfg config.LogConfig) (*zap.Logger, error) {
	var zapConfig zap.Config

	if cfg.Format == "console" {
		zapConfig = zap.NewDevelopmentConfig()
	} else {
		zapConfig = zap.NewProductionConfig()
	}

	// Set log level
	switch cfg.Level {
	case "debug":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		zapConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	return zapConfig.Build()
}

// initGRPCServer initializes the gRPC server with middleware
func initGRPCServer(logger *zap.Logger, rateLimitMiddleware *ratelimiter.Middleware, authMiddleware *middleware.AuthMiddleware) *grpc.Server {
	// Create unary interceptors
	unaryInterceptors := []grpc.UnaryServerInterceptor{
		middleware.RecoveryInterceptor(logger),
		middleware.LoggingInterceptor(logger),
		middleware.ValidationInterceptor(),
		middleware.RateLimitInterceptor(rateLimitMiddleware),
		authMiddleware.GRPCAuthInterceptor(),
		middleware.MetricsInterceptor(),
	}

	// Create stream interceptors
	streamInterceptors := []grpc.StreamServerInterceptor{
		middleware.StreamRecoveryInterceptor(logger),
		middleware.StreamLoggingInterceptor(logger),
	}

	// Create server options
	opts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(unaryInterceptors...),
		grpc.ChainStreamInterceptor(streamInterceptors...),
		grpc.MaxRecvMsgSize(4 * 1024 * 1024), // 4MB
		grpc.MaxSendMsgSize(4 * 1024 * 1024), // 4MB
	}

	return grpc.NewServer(opts...)
}

// healthCheck performs a basic health check
func healthCheck(dbConn *utils.DatabaseConnection) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status := dbConn.HealthCheck(ctx)
	
	for service, health := range status {
		if health != "healthy" {
			return fmt.Errorf("%s is %s", service, health)
		}
	}

	return nil
}
