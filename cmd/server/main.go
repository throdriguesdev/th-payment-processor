package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"th_payment_processor/internal/config"
	"th_payment_processor/internal/handlers"
	"th_payment_processor/internal/middleware"
	"th_payment_processor/internal/services"
	"th_payment_processor/internal/storage"
	"th_payment_processor/internal/tracing"
)

func main() {
	// logs
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.InfoLevel)

	// tracing
	shutdown, err := tracing.InitTracer()
	if err != nil {
		logrus.Fatalf("Failed to initialize tracing: %v", err)
	}
	defer shutdown()

	// start metrics server
	tracing.StartMetricsServer()

	// configs
	cfg := config.Load()

	// initialize storage based on configuration
	var storageImpl storage.Storage
	
	// Try to initialize hybrid storage (PostgreSQL + Redis)
	postgresStorage, err := storage.NewPostgresStorage(
		cfg.PostgresHost,
		cfg.PostgresPort,
		cfg.PostgresUser,
		cfg.PostgresPassword,
		cfg.PostgresDB,
		cfg.PostgresSSLMode,
	)
	if err != nil {
		logrus.Warnf("Failed to initialize PostgreSQL: %v. Falling back to in-memory storage", err)
		storageImpl = storage.NewInMemoryStorage()
	} else {
		redisCache, err := storage.NewRedisCache(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
		if err != nil {
			logrus.Warnf("Failed to initialize Redis: %v. Using PostgreSQL only", err)
			storageImpl = postgresStorage
		} else {
			logrus.Info("Using hybrid storage (PostgreSQL + Redis)")
			storageImpl = storage.NewHybridStorage(postgresStorage, redisCache)
		}
	}

	// init services
	paymentService := services.NewPaymentService(cfg, storageImpl)

	//  health monitoring in background
	ctx := context.Background()
	go paymentService.StartHealthMonitoring(ctx)

	// init handlers
	handler := handlers.NewPaymentHandler(paymentService)

	//  Gin router with performance optimizations
	gin.SetMode(gin.ReleaseMode)
	gin.DisableConsoleColor() // Reduce logging overhead
	router := gin.New()

	//  Minimal middleware for maximum performance
	router.Use(gin.Recovery())
	// Skip gin.Logger() for performance - we have structured logging
	router.Use(middleware.CorrelationIDMiddleware())
	router.Use(otelgin.Middleware("th-payment-processor"))
	router.Use(middleware.SimpleMetricsMiddleware())

	//  routes
	router.POST("/payments", handler.ProcessPayment)
	router.GET("/payments-summary", handler.GetPaymentsSummary)
	router.GET("/health", handler.GetHealthStatus)

	logrus.Infof("Starting rinha-backend on port %s", cfg.ServerPort)
	if err := router.Run(":" + cfg.ServerPort); err != nil {
		logrus.Fatalf("Failed to start server: %v", err)
	}
}
