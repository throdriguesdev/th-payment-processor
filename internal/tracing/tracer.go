package tracing

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

const ServiceName = "th-payment-processor"

func InitTracer() (func(), error) {
	ctx := context.Background()

	// Create resource with proper service identification for service graphs
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(ServiceName),
			semconv.ServiceVersionKey.String("1.0.0"),
			semconv.ServiceNamespaceKey.String("th-payment-system"),
			semconv.DeploymentEnvironmentKey.String("development"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Initialize OTLP exporter to OpenTelemetry Collector
	otlpEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if otlpEndpoint == "" {
		otlpEndpoint = "http://otel-collector:4318"
	}
	
	// Set OTEL environment variables for proper configuration
	os.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", otlpEndpoint+"/v1/traces")
	os.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")
	
	logrus.Infof("Creating OTLP exporter to collector: %s", otlpEndpoint)
	
	otlpExp, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}
	logrus.Info("OTLP trace exporter initialized")

	// Create trace provider with proper sampling for production
	tp := trace.NewTracerProvider(
		trace.WithBatcher(otlpExp),
		trace.WithResource(res),
		trace.WithSampler(trace.AlwaysSample()), // Use ParentBased(TraceIDRatioBased(0.1)) for production
	)

	otel.SetTracerProvider(tp)
	
	// Set up comprehensive context propagation for distributed tracing
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, // W3C Trace Context
		propagation.Baggage{},      // W3C Baggage
	))

	logrus.Info("OpenTelemetry tracing initialized successfully")

	return func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			logrus.Errorf("Error shutting down tracer provider: %v", err)
		}
	}, nil
}

// StartMetricsServer starts the Prometheus metrics server
func StartMetricsServer() {
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
}

func GetTracer() *trace.TracerProvider {
	return trace.NewTracerProvider()
}