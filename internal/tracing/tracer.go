package tracing

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

const ServiceName = "th-payment-processor"

func InitTracer() (func(), error) {
	ctx := context.Background()

	// Create resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(ServiceName),
			semconv.ServiceVersionKey.String("1.0.0"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Initialize Jaeger for tracing
	jaegerEndpoint := os.Getenv("JAEGER_ENDPOINT")
	if jaegerEndpoint == "" {
		jaegerEndpoint = "http://jaeger:14268/api/traces"
	}

	traceExp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(jaegerEndpoint)))
	if err != nil {
		logrus.Errorf("Failed to create Jaeger exporter: %v", err)
		return nil, err
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(traceExp),
		trace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

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