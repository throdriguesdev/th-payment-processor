package services

import (
	"th_payment_processor/internal/config"
	"th_payment_processor/internal/models"
	"th_payment_processor/internal/storage"
	"testing"
	"time"
)

func TestPaymentService_ProcessPayment(t *testing.T) {
	cfg := &config.Config{
		DefaultProcessorURL:  "http://localhost:8001",
		FallbackProcessorURL: "http://localhost:8002",
		RequestTimeout:       10 * time.Second,
	}

	storage := storage.NewInMemoryStorage()
	service := NewPaymentService(cfg, storage)

	req := &models.PaymentRequest{
		CorrelationID: "test-123",
		Amount:        100.00,
	}

	record, err := service.ProcessPayment(req)

	if err == nil {
		t.Error("Expected error when payment processors are unavailable")
	}

	if record == nil {
		t.Error("Expected record to be created even on failure")
	}
}

func TestPaymentService_GetPaymentsSummary(t *testing.T) {
	cfg := &config.Config{
		DefaultProcessorURL:  "http://localhost:8001",
		FallbackProcessorURL: "http://localhost:8002",
		RequestTimeout:       10 * time.Second,
	}

	storage := storage.NewInMemoryStorage()
	service := NewPaymentService(cfg, storage)

	summary := service.GetPaymentsSummary(nil, nil)

	// Should return empty summary
	if summary.Default.TotalRequests != 0 || summary.Fallback.TotalRequests != 0 {
		t.Error("Expected empty summary for new service")
	}
}
