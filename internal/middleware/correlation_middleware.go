package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"th_payment_processor/internal/logging"
)

const (
	CorrelationIDHeader = "X-Correlation-ID"
	RequestIDHeader     = "X-Request-ID"
	CorrelationIDKey    = "correlation_id"
	RequestIDKey        = "request_id"
	LoggerKey           = "structured_logger"
	StartTimeKey        = "start_time"
)

// CorrelationIDMiddleware adds correlation ID to request context and logs
func CorrelationIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		
		// Get or generate correlation ID
		correlationID := c.GetHeader(CorrelationIDHeader)
		if correlationID == "" {
			correlationID = uuid.New().String()
		}
		
		// Get or generate request ID (for tracing individual requests)
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}
		
		// Set response headers for traceability
		c.Header(CorrelationIDHeader, correlationID)
		c.Header(RequestIDHeader, requestID)
		
		// Create enhanced context with IDs
		ctx := context.WithValue(c.Request.Context(), CorrelationIDKey, correlationID)
		ctx = context.WithValue(ctx, RequestIDKey, requestID)
		ctx = context.WithValue(ctx, StartTimeKey, startTime)
		
		// Update request context
		c.Request = c.Request.WithContext(ctx)
		
		// Create structured logger with context
		logger := logging.NewStructuredLogger("http_middleware").WithContext(ctx)
		
		// Set logger in context for handlers to use
		ctx = context.WithValue(ctx, LoggerKey, logger)
		c.Request = c.Request.WithContext(ctx)
		
		// Add trace attributes for better observability
		if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
			span.SetAttributes(
				attribute.String("http.correlation_id", correlationID),
				attribute.String("http.request_id", requestID),
				attribute.String("http.method", c.Request.Method),
				attribute.String("http.url.path", c.Request.URL.Path),
				attribute.String("http.user_agent", c.Request.UserAgent()),
				attribute.String("http.remote_addr", c.ClientIP()),
			)
		}
		
		// Log request start with structured logging
		logger.LogHTTPRequest(c.Request.Method, c.Request.URL.Path, c.ClientIP())
		
		c.Next()
		
		// Calculate request duration
		duration := time.Since(startTime)
		latencyMs := duration.Milliseconds()
		
		// Add response attributes to trace
		if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
			span.SetAttributes(
				attribute.Int("http.status_code", c.Writer.Status()),
				attribute.Int64("http.response.duration_ms", latencyMs),
				attribute.Int("http.response.size_bytes", c.Writer.Size()),
			)
		}
		
		// Log request completion with structured logging
		logger.LogHTTPResponse(c.Request.Method, c.Request.URL.Path, c.Writer.Status(), latencyMs)
	}
}

// GetCorrelationID extracts correlation ID from context
func GetCorrelationID(ctx context.Context) string {
	if correlationID, ok := ctx.Value(CorrelationIDKey).(string); ok {
		return correlationID
	}
	return ""
}

// GetRequestID extracts request ID from context
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// GetStructuredLogger extracts structured logger from context
func GetStructuredLogger(ctx context.Context) *logging.StructuredLogger {
	if logger, ok := ctx.Value(LoggerKey).(*logging.StructuredLogger); ok {
		return logger
	}
	// Fallback to default logger
	return logging.NewStructuredLogger("fallback").WithContext(ctx)
}

// GetLogger extracts structured logger from context (backward compatibility)
func GetLogger(ctx context.Context) *logging.StructuredLogger {
	return GetStructuredLogger(ctx)
}

// GetStartTime extracts request start time from context
func GetStartTime(ctx context.Context) time.Time {
	if startTime, ok := ctx.Value(StartTimeKey).(time.Time); ok {
		return startTime
	}
	return time.Now() // fallback
}

// PropagateContext creates a new context with trace and correlation context propagated
func PropagateContext(parentCtx context.Context) context.Context {
	// Create new context preserving trace and correlation info
	ctx := context.Background()
	
	// Propagate correlation ID
	if correlationID := GetCorrelationID(parentCtx); correlationID != "" {
		ctx = context.WithValue(ctx, CorrelationIDKey, correlationID)
	}
	
	// Propagate request ID
	if requestID := GetRequestID(parentCtx); requestID != "" {
		ctx = context.WithValue(ctx, RequestIDKey, requestID)
	}
	
	// Propagate trace context
	if span := trace.SpanFromContext(parentCtx); span.SpanContext().IsValid() {
		ctx = trace.ContextWithSpan(ctx, span)
	}
	
	return ctx
}