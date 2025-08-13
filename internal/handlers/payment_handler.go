package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"th_payment_processor/internal/logging"
	"th_payment_processor/internal/metrics"
	"th_payment_processor/internal/middleware"
	"th_payment_processor/internal/models"
	"th_payment_processor/internal/services"
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
	ctx := c.Request.Context()
	logger := middleware.GetStructuredLogger(ctx)
	correlationID := middleware.GetCorrelationID(ctx)
	
	var req models.PaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.WithError(err).WithField("operation", "request_binding").Error("Invalid payment request format")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":          "Invalid request format",
			"correlation_id": correlationID,
		})
		return
	}

	// Set correlation ID in request if not present
	if req.CorrelationID == "" {
		req.CorrelationID = correlationID
	}

	logger.WithPaymentFields(logging.PaymentFields{
		Amount:        req.Amount,
		CorrelationID: req.CorrelationID,
	}).WithField("operation", "payment_request").Info("Processing payment request")

	// Process payment with context
	response, err := h.paymentService.ProcessPayment(ctx, &req)
	if err != nil {
		logger.WithPaymentFields(logging.PaymentFields{
			Amount:        req.Amount,
			CorrelationID: req.CorrelationID,
		}).WithError(err).WithField("operation", "payment_processing").Error("Payment processing failed")
		
		// Record failed payment with trace context
		metrics.RecordPaymentAmountWithContext(ctx, req.Amount, "unknown", "failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":          "Payment processing failed",
			"correlation_id": correlationID,
		})
		return
	}

	// Record successful payment metrics with trace context
	if response != nil {
		metrics.RecordPaymentAmountWithContext(ctx, req.Amount, response.Processor, "success")
		
		logger.WithPaymentFields(logging.PaymentFields{
			Amount:        req.Amount,
			CorrelationID: req.CorrelationID,
			Processor:     response.Processor,
			Status:        "success",
		}).WithField("operation", "payment_success").Info("Payment processed successfully")
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
