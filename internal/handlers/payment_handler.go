package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"th_payment_processor/internal/metrics"
	"th_payment_processor/internal/middleware"
	"th_payment_processor/internal/models"
	"th_payment_processor/internal/services"
	"time"
)

type PaymentHandler struct {
	paymentService *services.PaymentService
}

func NewPaymentHandler(paymentService *services.PaymentService) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
	}
}

func (h *PaymentHandler) ProcessPayment(c *gin.Context) {
	logger := middleware.GetLogger(c.Request.Context())
	correlationID := middleware.GetCorrelationID(c.Request.Context())
	
	var req models.PaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.WithError(err).Error("Invalid payment request format")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":          "Invalid request format",
			"correlation_id": correlationID,
		})
		return
	}

	logger.WithFields(logrus.Fields{
		"payment_amount":      req.Amount,
		"payment_correlation": req.CorrelationID,
	}).Info("Processing payment request")

	// Process payment
	response, err := h.paymentService.ProcessPayment(&req)
	if err != nil {
		logger.WithError(err).WithFields(logrus.Fields{
			"payment_amount":      req.Amount,
			"payment_correlation": req.CorrelationID,
		}).Error("Payment processing failed")
		
		// Record failed payment with trace context
		metrics.RecordPaymentAmountWithContext(c.Request.Context(), req.Amount, "unknown", "failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":          "Payment processing failed",
			"correlation_id": correlationID,
		})
		return
	}

	// Record successful payment metrics with trace context
	if response != nil {
		metrics.RecordPaymentAmountWithContext(c.Request.Context(), req.Amount, response.Processor, "success")
		
		logger.WithFields(logrus.Fields{
			"payment_amount":      req.Amount,
			"payment_correlation": req.CorrelationID,
			"processor_used":      response.Processor,
		}).Info("Payment processed successfully")
	}

	// Return success response (any 2XX status is valid)
	c.JSON(http.StatusOK, gin.H{
		"message":        "Payment processed successfully",
		"correlation_id": correlationID,
	})
}

func (h *PaymentHandler) GetPaymentsSummary(c *gin.Context) {
	// Parse query parameters
	fromStr := c.Query("from")
	toStr := c.Query("to")

	var from, to *time.Time

	if fromStr != "" {
		if parsed, err := time.Parse(time.RFC3339, fromStr); err == nil {
			from = &parsed
		} else {
			logrus.Errorf("Invalid 'from' parameter: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'from' parameter format"})
			return
		}
	}

	if toStr != "" {
		if parsed, err := time.Parse(time.RFC3339, toStr); err == nil {
			to = &parsed
		} else {
			logrus.Errorf("Invalid 'to' parameter: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'to' parameter format"})
			return
		}
	}

	// Get summary from storage
	summary := h.paymentService.GetPaymentsSummary(from, to)

	c.JSON(http.StatusOK, summary)
}

func (h *PaymentHandler) GetHealthStatus(c *gin.Context) {
	// Get storage health status if using hybrid storage
	status := map[string]interface{}{
		"status": "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, status)
}
