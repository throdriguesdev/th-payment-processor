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
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
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

	var exporters []trace.SpanExporter

	// Initialize Jaeger for tracing (legacy support)
	jaegerEndpoint := os.Getenv("JAEGER_ENDPOINT")
	if jaegerEndpoint == "" {
		jaegerEndpoint = "http://jaeger:14268/api/traces"
	}

	jaegerExp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(jaegerEndpoint)))
	if err != nil {
		logrus.Warnf("Failed to create Jaeger exporter: %v", err)
	} else {
		exporters = append(exporters, jaegerExp)
		logrus.Info("Jaeger trace exporter initialized")
	}

	// Initialize Tempo OTLP exporter using environment variables approach
	// Clear any existing OTEL environment variables that might conflict
	os.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "http://tempo:4318/v1/traces")
	os.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")
	
	logrus.Info("Creating Tempo OTLP exporter using environment variables")
	
	tempoExp, err := otlptracehttp.New(ctx)
	if err != nil {
		logrus.Warnf("Failed to create Tempo exporter: %v", err)
	} else {
		exporters = append(exporters, tempoExp)
		logrus.Info("Tempo trace exporter initialized")
	}

	if len(exporters) == 0 {
		return nil, fmt.Errorf("no trace exporters available")
	}

	// Create trace provider with multiple exporters
	var batchOptions []trace.TracerProviderOption
	for _, exp := range exporters {
		batchOptions = append(batchOptions, trace.WithBatcher(exp))
	}
	batchOptions = append(batchOptions, trace.WithResource(res))

	tp := trace.NewTracerProvider(batchOptions...)

	otel.SetTracerProvider(tp)
	
	// Set up comprehensive context propagation for distributed tracing
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, // W3C Trace Context
		propagation.Baggage{},      // W3C Baggage
		propagation.TraceContext{}, // Duplicate for fallback support
	))

	logrus.Info("OpenTelemetry tracing initialized with multiple exporters")

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