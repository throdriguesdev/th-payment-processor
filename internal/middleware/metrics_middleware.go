package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"th_payment_processor/internal/metrics"
)

func MetricsMiddleware(m *metrics.Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		
		// Increment active connections
		m.IncActiveConnections(c.Request.Context())
		
		// Process request
		c.Next()
		
		// Record metrics
		duration := time.Since(start)
		status := strconv.Itoa(c.Writer.Status())
		
		m.RecordRequest(
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
			
			m.RecordError(
				c.Request.Context(),
				c.Request.Method,
				c.FullPath(),
				errorType,
			)
		}
		
		// Decrement active connections
		m.DecActiveConnections(c.Request.Context())
	}
}