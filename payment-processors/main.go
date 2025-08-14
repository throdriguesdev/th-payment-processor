package main

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"payment-processors/handlers"
	"payment-processors/storage"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

func main() {
	// Configure logging
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.InfoLevel)
	
	// Get configuration from environment
	feePercentage := getEnvAsFloat("FEE_PERCENTAGE", 1.0) // 1% default fee
	minResponseTime := getEnvAsInt("MIN_RESPONSE_TIME", 50) // 50ms default
	port := getEnv("PORT", "8080")
	
	// Initialize tracing with unique service name based on fee percentage
	serviceName := "payment-processor-default"
	if feePercentage > 2.0 {
		serviceName = "payment-processor-fallback" 
	}
	
	shutdown, err := initTracer(serviceName)
	if err != nil {
		logrus.Warnf("Failed to initialize tracing: %v", err)
	} else {
		defer shutdown()
	}
	
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
	router.Use(otelgin.Middleware(serviceName))
	
	// Setup routes
	router.POST("/payments", handler.ProcessPayment)
	router.GET("/payments/:id", handler.GetPaymentDetails)
	router.GET("/payments/service-health", handler.GetServiceHealth)
	
	// Start metrics server in a separate goroutine
	go func() {
		metricsMux := http.NewServeMux()
		metricsMux.Handle("/metrics", promhttp.Handler())
		metricsMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
		
		logrus.Info("Metrics server starting on :2112/metrics")
		if err := http.ListenAndServe(":2112", metricsMux); err != nil {
			logrus.Errorf("Failed to start metrics server: %v", err)
		}
	}()
	
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

func initTracer(serviceName string) (func(), error) {
	ctx := context.Background()

	// Create resource with proper service identification
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String("1.0.0"),
			semconv.ServiceNamespaceKey.String("th-payment-system"),
			semconv.DeploymentEnvironmentKey.String("development"),
		),
	)
	if err != nil {
		return nil, err
	}

	// Initialize OTLP exporter 
	otlpEndpoint := getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://otel-collector:4318")
	
	os.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", otlpEndpoint+"/v1/traces")
	os.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")
	
	otlpExp, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, err
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(otlpExp),
		trace.WithResource(res),
		trace.WithSampler(trace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)
	
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	logrus.Infof("OpenTelemetry tracing initialized for %s", serviceName)

	return func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			logrus.Errorf("Error shutting down tracer provider: %v", err)
		}
	}, nil
}
