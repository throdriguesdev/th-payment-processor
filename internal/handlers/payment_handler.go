package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"rinha-backend/internal/models"
	"rinha-backend/internal/services"
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
	var req models.PaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.Errorf("Invalid payment request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Process payment
	_, err := h.paymentService.ProcessPayment(&req)
	if err != nil {
		logrus.Errorf("Payment processing failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Payment processing failed"})
		return
	}

	// Return success response (any 2XX status is valid)
	c.JSON(http.StatusOK, gin.H{"message": "Payment processed successfully"})
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
