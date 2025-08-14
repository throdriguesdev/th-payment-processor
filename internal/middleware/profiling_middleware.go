package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"th_payment_processor/internal/profiling"
)

// ProfilingMiddleware adds profiling context to HTTP requests
func ProfilingMiddleware(profiler *profiling.Profiler) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !profiler.IsEnabled() {
			c.Next()
			return
		}

		// Create profiling tags for this HTTP request
		tags := profiler.ProfileHTTPHandler(c.Request.Method, c.FullPath())
		
		// Add additional context from correlation ID if available
		correlationID := GetCorrelationID(c.Request.Context())
		if correlationID != "" {
			if tags == nil {
				tags = make(profiling.Labels)
			}
			tags["correlation_id"] = correlationID
		}

		// Add user ID if available from context
		if userID, exists := c.Get("user_id"); exists {
			if userIDStr, ok := userID.(string); ok {
				if tags == nil {
					tags = make(profiling.Labels)
				}
				tags["user_id"] = userIDStr
			}
		}

		// Wrap the request context with profiling tags
		ctx := profiler.TagWrapper(c.Request.Context(), tags)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// ProfilePaymentOperation wraps payment operations with profiling context
func ProfilePaymentOperation(profiler *profiling.Profiler, ctx context.Context, processor, operation string, fn func(context.Context) error) error {
	if !profiler.IsEnabled() {
		return fn(ctx)
	}

	tags := profiler.ProfilePaymentOperation(processor, operation)
	
	// Add correlation ID if available
	correlationID := GetCorrelationID(ctx)
	if correlationID != "" {
		if tags == nil {
			tags = make(profiling.Labels)
		}
		tags["correlation_id"] = correlationID
	}

	wrappedCtx := profiler.TagWrapper(ctx, tags)
	return fn(wrappedCtx)
}

// ProfileDatabaseOperation wraps database operations with profiling context
func ProfileDatabaseOperation(profiler *profiling.Profiler, ctx context.Context, dbType, operation, table string, fn func(context.Context) error) error {
	if !profiler.IsEnabled() {
		return fn(ctx)
	}

	tags := profiler.ProfileDatabaseOperation(dbType, operation, table)
	wrappedCtx := profiler.TagWrapper(ctx, tags)
	return fn(wrappedCtx)
}

// AddProfilingTags adds custom profiling tags to the current context
func AddProfilingTags(profiler *profiling.Profiler, ctx context.Context, tags map[string]string) context.Context {
	if !profiler.IsEnabled() {
		return ctx
	}

	return profiler.AddCustomTags(ctx, tags)
}