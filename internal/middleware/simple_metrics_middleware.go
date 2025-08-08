package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"th_payment_processor/internal/metrics"
)

func SimpleMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		
		// Increment active connections
		metrics.IncActiveConnections()
		
		// Process request
		c.Next()
		
		// Record metrics with trace context for exemplars
		duration := time.Since(start)
		status := strconv.Itoa(c.Writer.Status())
		
		metrics.RecordRequestWithContext(
			c.Request.Context(),
			c.Request.Method,
			c.FullPath(),
			status,
			duration,
		)
		
		// Record errors for 4xx and 5xx status codes
		if c.Writer.Status() >= 400 {
			errorType := "client_error"
			if c.Writer.Status() >= 500 {
				errorType = "server_error"
			}
			
			metrics.RecordError(
				c.Request.Method,
				c.FullPath(),
				errorType,
			)
		}
		
		// Decrement active connections
		metrics.DecActiveConnections()
	}
}