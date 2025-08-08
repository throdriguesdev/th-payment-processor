package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type Metrics struct {
	// RED Metrics
	requestCounter    metric.Int64Counter
	requestDuration   metric.Float64Histogram
	errorCounter      metric.Int64Counter
	
	// Additional metrics
	activeConnections metric.Int64UpDownCounter
	paymentAmount     metric.Float64Histogram
}

func NewMetrics() (*Metrics, error) {
	meter := otel.Meter("th-payment-processor")

	requestCounter, err := meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
	)
	if err != nil {
		return nil, err
	}

	requestDuration, err := meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("Duration of HTTP requests in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	errorCounter, err := meter.Int64Counter(
		"http_errors_total",
		metric.WithDescription("Total number of HTTP errors"),
	)
	if err != nil {
		return nil, err
	}

	activeConnections, err := meter.Int64UpDownCounter(
		"http_active_connections",
		metric.WithDescription("Number of active HTTP connections"),
	)
	if err != nil {
		return nil, err
	}

	paymentAmount, err := meter.Float64Histogram(
		"payment_amount_dollars",
		metric.WithDescription("Payment amounts processed in dollars"),
		metric.WithUnit("USD"),
	)
	if err != nil {
		return nil, err
	}

	return &Metrics{
		requestCounter:    requestCounter,
		requestDuration:   requestDuration,
		errorCounter:      errorCounter,
		activeConnections: activeConnections,
		paymentAmount:     paymentAmount,
	}, nil
}

func (m *Metrics) RecordRequest(ctx context.Context, method, path, status string, duration time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("method", method),
		attribute.String("path", path),
		attribute.String("status", status),
	}

	m.requestCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.requestDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

func (m *Metrics) RecordError(ctx context.Context, method, path, errorType string) {
	attrs := []attribute.KeyValue{
		attribute.String("method", method),
		attribute.String("path", path),
		attribute.String("error_type", errorType),
	}

	m.errorCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
}

func (m *Metrics) RecordPaymentAmount(ctx context.Context, amount float64, processor, status string) {
	attrs := []attribute.KeyValue{
		attribute.String("processor", processor),
		attribute.String("status", status),
	}

	m.paymentAmount.Record(ctx, amount, metric.WithAttributes(attrs...))
}

func (m *Metrics) IncActiveConnections(ctx context.Context) {
	m.activeConnections.Add(ctx, 1)
}

func (m *Metrics) DecActiveConnections(ctx context.Context) {
	m.activeConnections.Add(ctx, -1)
}