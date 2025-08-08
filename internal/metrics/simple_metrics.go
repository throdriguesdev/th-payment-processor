package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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