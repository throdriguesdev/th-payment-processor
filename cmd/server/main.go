package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"th_payment_processor/internal/config"
	"th_payment_processor/internal/handlers"
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

	// configs
	cfg := config.Load()

	//  storage
	storage := storage.NewInMemoryStorage()

	// init services
	paymentService := services.NewPaymentService(cfg, storage)

	//  health monitoring in background
	ctx := context.Background()
	go paymentService.StartHealthMonitoring(ctx)

	// init handlers
	handler := handlers.NewPaymentHandler(paymentService)

	//  Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	//  middleware
	router.Use(gin.Recovery())
	router.Use(gin.Logger())
	router.Use(otelgin.Middleware("rinha-backend"))

	//  routes
	router.POST("/payments", handler.ProcessPayment)
	router.GET("/payments-summary", handler.GetPaymentsSummary)

	logrus.Infof("Starting rinha-backend on port %s", cfg.ServerPort)
	if err := router.Run(":" + cfg.ServerPort); err != nil {
		logrus.Fatalf("Failed to start server: %v", err)
	}
}
