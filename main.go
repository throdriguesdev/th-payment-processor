package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"rinha-backend/internal/config"
	"rinha-backend/internal/handlers"
	"rinha-backend/internal/services"
	"rinha-backend/internal/storage"
)

func main() {
	// logs
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.InfoLevel)

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

	//  routes
	router.POST("/payments", handler.ProcessPayment)
	router.GET("/payments-summary", handler.GetPaymentsSummary)

	logrus.Infof("Starting rinha-backend on port %s", cfg.ServerPort)
	if err := router.Run(":" + cfg.ServerPort); err != nil {
		logrus.Fatalf("Failed to start server: %v", err)
	}
}
