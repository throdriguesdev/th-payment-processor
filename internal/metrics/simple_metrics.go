package metrics

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel/trace"
)

var (
	// HTTP metrics
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	httpErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_errors_total",
			Help: "Total number of HTTP errors",
		},
		[]string{"method", "path", "error_type"},
	)

	activeConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_active_connections",
			Help: "Number of active HTTP connections",
		},
	)

	// Payment metrics
	paymentAmounts = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "payment_amount_dollars",
			Help:    "Payment amounts processed in dollars",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000},
		},
		[]string{"processor", "status"},
	)
)

// RecordRequest records HTTP request metrics
func RecordRequest(method, path, status string, duration time.Duration) {
	labels := prometheus.Labels{
		"method": method,
		"path":   path,
		"status": status,
	}
	
	httpRequestsTotal.With(labels).Inc()
	httpRequestDuration.With(labels).Observe(duration.Seconds())
}

// RecordRequestWithContext records HTTP request metrics with trace context for exemplars
func RecordRequestWithContext(ctx context.Context, method, path, status string, duration time.Duration) {
	labels := prometheus.Labels{
		"method": method,
		"path":   path,
		"status": status,
	}
	
	// Extract trace ID for exemplars
	span := trace.SpanFromContext(ctx)
	var exemplar prometheus.Labels
	if span.SpanContext().IsValid() {
		exemplar = prometheus.Labels{
			"traceID": span.SpanContext().TraceID().String(),
		}
	}
	
	httpRequestsTotal.With(labels).Inc()
	
	// Record duration with exemplar if trace ID is available
	observer := httpRequestDuration.With(labels)
	if exemplar != nil {
		if observerWithExemplar, ok := observer.(prometheus.ExemplarObserver); ok {
			observerWithExemplar.ObserveWithExemplar(duration.Seconds(), exemplar)
		} else {
			observer.Observe(duration.Seconds())
		}
	} else {
		observer.Observe(duration.Seconds())
	}
}

// RecordPaymentAmountWithContext records payment amount metrics with trace context for exemplars
func RecordPaymentAmountWithContext(ctx context.Context, amount float64, processor, status string) {
	labels := prometheus.Labels{
		"processor": processor,
		"status":    status,
	}
	
	// Extract trace ID for exemplars
	span := trace.SpanFromContext(ctx)
	var exemplar prometheus.Labels
	if span.SpanContext().IsValid() {
		exemplar = prometheus.Labels{
			"traceID": span.SpanContext().TraceID().String(),
		}
	}
	
	observer := paymentAmounts.With(labels)
	if exemplar != nil {
		if observerWithExemplar, ok := observer.(prometheus.ExemplarObserver); ok {
			observerWithExemplar.ObserveWithExemplar(amount, exemplar)
		} else {
			observer.Observe(amount)
		}
	} else {
		observer.Observe(amount)
	}
}

// RecordError records HTTP error metrics
func RecordError(method, path, errorType string) {
	httpErrorsTotal.With(prometheus.Labels{
		"method":     method,
		"path":       path,
		"error_type": errorType,
	}).Inc()
}

// RecordPaymentAmount records payment amount metrics
func RecordPaymentAmount(amount float64, processor, status string) {
	paymentAmounts.With(prometheus.Labels{
		"processor": processor,
		"status":    status,
	}).Observe(amount)
}

// IncActiveConnections increments active connections
func IncActiveConnections() {
	activeConnections.Inc()
}

// DecActiveConnections decrements active connections
func DecActiveConnections() {
	activeConnections.Dec()
}