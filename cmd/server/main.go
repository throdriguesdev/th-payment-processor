package main

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"th_payment_processor/internal/config"
	"th_payment_processor/internal/handlers"
	"th_payment_processor/internal/logging"
	"th_payment_processor/internal/middleware"
	"th_payment_processor/internal/profiling"
	"th_payment_processor/internal/services"
	"th_payment_processor/internal/storage"
	"th_payment_processor/internal/tracing"
)

// ServiceMetadataHook adds service metadata to all log entries
type ServiceMetadataHook struct {
	ServiceName    string
	ServiceVersion string
}

func (hook *ServiceMetadataHook) Fire(entry *logrus.Entry) error {
	entry.Data["service_name"] = hook.ServiceName
	entry.Data["service_version"] = hook.ServiceVersion
	return nil
}

func (hook *ServiceMetadataHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func main() {
	// Initialize structured logging from environment configuration
	logConfig := logging.LoadConfigFromEnv()
	logging.ConfigureGlobalLogger(logConfig)
	
	// Add service metadata to all logs
	logrus.AddHook(&ServiceMetadataHook{
		ServiceName:    "th-payment-processor",
		ServiceVersion: "1.0.0",
	})
	
	logrus.WithFields(logrus.Fields{
		"log_level":  logConfig.Level,
		"log_format": logConfig.Format,
	}).Info("Application starting with structured logging enabled")

	// Initialize profiling
	profiler := profiling.NewProfiler(logrus.StandardLogger())
	if err := profiler.Start(); err != nil {
		logrus.Warnf("Failed to start profiler: %v", err)
	} else {
		defer func() {
			if err := profiler.Stop(); err != nil {
				logrus.Errorf("Failed to stop profiler: %v", err)
			}
		}()
	}

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
	paymentService := services.NewPaymentService(cfg, storageImpl, profiler)

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
	router.Use(middleware.ProfilingMiddleware(profiler))
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
