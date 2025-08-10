package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
)

const (
	CorrelationIDHeader = "X-Correlation-ID"
	CorrelationIDKey    = "correlation_id"
)

// CorrelationIDMiddleware adds correlation ID to request context and logs
func CorrelationIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		correlationID := c.GetHeader(CorrelationIDHeader)
		
		// Generate correlation ID if not present
		if correlationID == "" {
			correlationID = uuid.New().String()
		}
		
		// Set in response header for traceability
		c.Header(CorrelationIDHeader, correlationID)
		
		// Add to request context
		ctx := context.WithValue(c.Request.Context(), CorrelationIDKey, correlationID)
		c.Request = c.Request.WithContext(ctx)
		
		// Add correlation ID and trace info to all logs in this request
		span := trace.SpanFromContext(ctx)
		logEntry := logrus.WithFields(logrus.Fields{
			"correlation_id": correlationID,
			"service_name":   "th-payment-processor",
		})
		
		if span.SpanContext().IsValid() {
			logEntry = logEntry.WithFields(logrus.Fields{
				"trace_id": span.SpanContext().TraceID().String(),
				"span_id":  span.SpanContext().SpanID().String(),
			})
		}
		
		// Set structured logger in context for use by handlers
		ctx = context.WithValue(ctx, "logger", logEntry)
		c.Request = c.Request.WithContext(ctx)
		
		// Log request start
		logEntry.WithFields(logrus.Fields{
			"method": c.Request.Method,
			"path":   c.Request.URL.Path,
			"remote_addr": c.ClientIP(),
		}).Info("Request started")
		
		c.Next()
		
		// Log request end
		logEntry.WithFields(logrus.Fields{
			"status": c.Writer.Status(),
		}).Info("Request completed")
	}
}

// GetCorrelationID extracts correlation ID from context
func GetCorrelationID(ctx context.Context) string {
	if correlationID, ok := ctx.Value(CorrelationIDKey).(string); ok {
		return correlationID
	}
	return ""
}

// GetLogger extracts structured logger from context
func GetLogger(ctx context.Context) *logrus.Entry {
	if logger, ok := ctx.Value("logger").(*logrus.Entry); ok {
		return logger
	}
	// Fallback to default logger
	return logrus.WithFields(logrus.Fields{
		"service_name": "th-payment-processor",
	})
}