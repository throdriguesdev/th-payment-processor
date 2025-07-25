package main

import (
	"os"
	"strconv"
	"payment-processors/handlers"
	"payment-processors/storage"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	// Configure logging
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.InfoLevel)
	
	// Get configuration from environment
	feePercentage := getEnvAsFloat("FEE_PERCENTAGE", 1.0) // 1% default fee
	minResponseTime := getEnvAsInt("MIN_RESPONSE_TIME", 50) // 50ms default
	port := getEnv("PORT", "8080")
	
	// Initialize storage
	storage := storage.NewInMemoryStorage(feePercentage, minResponseTime)
	
	// Initialize handlers
	handler := handlers.NewPaymentHandler(storage)
	
	// Setup Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	
	// Add middleware
	router.Use(gin.Recovery())
	router.Use(gin.Logger())
	
	// Setup routes
	router.POST("/payments", handler.ProcessPayment)
	router.GET("/payments/:id", handler.GetPaymentDetails)
	router.GET("/payments/service-health", handler.GetServiceHealth)
	
	// Admin routes
	admin := router.Group("/admin")
	{
		admin.GET("/payments-summary", handler.GetPaymentsSummary)
		admin.PUT("/configurations/token", handler.SetToken)
		admin.PUT("/configurations/delay", handler.SetDelay)
		admin.PUT("/configurations/failure", handler.SetFailure)
		admin.POST("/purge-payments", handler.PurgePayments)
	}
	
	logrus.Infof("Starting payment processor on port %s with %.2f%% fee", port, feePercentage)
	if err := router.Run(":" + port); err != nil {
		logrus.Fatalf("Failed to start server: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}
