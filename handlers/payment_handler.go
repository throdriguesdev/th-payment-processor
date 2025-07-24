package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"net/http"
	"payment-processors/models"
	"payment-processors/storage"
	"time"
)

type PaymentHandler struct {
	storage *storage.InMemoryStorage
}

func NewPaymentHandler(storage *storage.InMemoryStorage) *PaymentHandler {
	return &PaymentHandler{
		storage: storage,
	}
}

// ProcessPayment handles POST /payments
func (h *PaymentHandler) ProcessPayment(c *gin.Context) {
	var req models.PaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.Errorf("Invalid payment request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	config := h.storage.GetConfig()

	// Check failures
	if config.Failure {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Payment processor is failing"})
		return
	}

	//delay if configured
	if config.Delay > 0 {
		time.Sleep(time.Duration(config.Delay) * time.Millisecond)
	}

	// calc fee
	fee := req.Amount * config.FeePercentage / 100

	// create payment rec
	record := &models.PaymentRecord{
		ID:            uuid.New(),
		CorrelationID: req.CorrelationID,
		Amount:        req.Amount,
		RequestedAt:   req.RequestedAt,
		ProcessedAt:   time.Now(),
		Fee:           fee,
	}

	// store payment
	h.storage.StorePayment(record)

	logrus.Infof("Payment processed: %s, amount: %.2f, fee: %.2f",
		record.CorrelationID, record.Amount, record.Fee)

	c.JSON(http.StatusOK, models.PaymentResponse{
		Message: "payment processed successfully",
	})
}

// GetPaymentDetails handles GET /payments/{id}
func (h *PaymentHandler) GetPaymentDetails(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payment ID"})
		return
	}

	record, exists := h.storage.GetPayment(id)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
		return
	}

	c.JSON(http.StatusOK, record)
}

// GetServiceHealth GET  /payments/service-health
func (h *PaymentHandler) GetServiceHealth(c *gin.Context) {
	config := h.storage.GetConfig()

	c.JSON(http.StatusOK, models.HealthCheckResponse{
		Failing:         config.Failure,
		MinResponseTime: config.MinResponseTime,
	})
}

// GetPaymentsSummary handles GET /admin/payments-summary
func (h *PaymentHandler) GetPaymentsSummary(c *gin.Context) {
	// Check admin token
	token := c.GetHeader("X-Rinha-Token")
	config := h.storage.GetConfig()
	if token != config.Token {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// Parse query parameters
	fromStr := c.Query("from")
	toStr := c.Query("to")

	var from, to *time.Time

	if fromStr != "" {
		if parsed, err := time.Parse(time.RFC3339, fromStr); err == nil {
			from = &parsed
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'from' parameter format"})
			return
		}
	}

	if toStr != "" {
		if parsed, err := time.Parse(time.RFC3339, toStr); err == nil {
			to = &parsed
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'to' parameter format"})
			return
		}
	}

	summary := h.storage.GetPaymentsSummary(from, to)
	c.JSON(http.StatusOK, summary)
}

// SetToken handles PUT /admin/configurations/token
func (h *PaymentHandler) SetToken(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	config := h.storage.GetConfig()
	config.Token = req.Token
	h.storage.UpdateConfig(config)

	c.Status(http.StatusNoContent)
}

// SetDelay handles PUT /admin/configurations/delay
func (h *PaymentHandler) SetDelay(c *gin.Context) {
	var req struct {
		Delay int `json:"delay" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	config := h.storage.GetConfig()
	config.Delay = req.Delay
	h.storage.UpdateConfig(config)

	c.Status(http.StatusNoContent)
}

// SetFailure handles PUT /admin/configurations/failure
func (h *PaymentHandler) SetFailure(c *gin.Context) {
	var req struct {
		Failure bool `json:"failure" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	config := h.storage.GetConfig()
	config.Failure = req.Failure
	h.storage.UpdateConfig(config)

	c.Status(http.StatusNoContent)
}

// PurgePayments handles POST /admin/purge-payments
func (h *PaymentHandler) PurgePayments(c *gin.Context) {
	// Check admin token
	token := c.GetHeader("X-Rinha-Token")
	config := h.storage.GetConfig()
	if token != config.Token {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	h.storage.PurgePayments()

	c.JSON(http.StatusOK, gin.H{"message": "All payments purged."})
}
