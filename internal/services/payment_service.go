package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"net/http"
	"rinha-backend/internal/config"
	"rinha-backend/internal/models"
	"rinha-backend/internal/storage"
	"sync"
	"time"
)

type PaymentService struct {
	config  *config.Config
	storage *storage.InMemoryStorage
	client  *http.Client

	// Health monitoring
	healthMu       sync.RWMutex
	defaultHealth  *models.ProcessorHealth
	fallbackHealth *models.ProcessorHealth

	// Rate limiting for health checks (per processor)
	lastDefaultHealthCheck  time.Time
	lastFallbackHealthCheck time.Time
	healthCheckMu           sync.Mutex
}

func NewPaymentService(cfg *config.Config, storage *storage.InMemoryStorage) *PaymentService {
	return &PaymentService{
		config:  cfg,
		storage: storage,
		client: &http.Client{
			Timeout:   cfg.RequestTimeout,
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
		defaultHealth: &models.ProcessorHealth{
			IsHealthy: true,
			LastCheck: time.Now(),
		},
		fallbackHealth: &models.ProcessorHealth{
			IsHealthy: true,
			LastCheck: time.Now(),
		},
	}
}

func (s *PaymentService) ProcessPayment(req *models.PaymentRequest) (*models.PaymentRecord, error) {
	ctx := context.Background()
	tracer := otel.Tracer("payment-service")
	ctx, span := tracer.Start(ctx, "ProcessPayment")
	defer span.End()

	span.SetAttributes(
		attribute.String("payment.correlation_id", req.CorrelationID),
		attribute.Float64("payment.amount", req.Amount),
	)

	logrus.Infof("Processing payment: correlationId=%s, amount=%.2f", req.CorrelationID, req.Amount)

	// Check if payment already exists
	if existing, exists := s.storage.GetPaymentByCorrelationID(req.CorrelationID); exists {
		logrus.Infof("Payment already exists: %s", req.CorrelationID)
		span.SetAttributes(attribute.Bool("payment.already_exists", true))
		return existing, nil
	}

	// Create payment record
	record := &models.PaymentRecord{
		ID:            uuid.New(),
		CorrelationID: req.CorrelationID,
		Amount:        req.Amount,
		ProcessedAt:   time.Now(),
		Success:       false,
	}

	// try default processor first
	if s.isProcessorHealthy("default") {
		logrus.Infof("Trying default processor for payment: %s", req.CorrelationID)
		span.SetAttributes(attribute.String("payment.processor.attempted", "default"))
		if err := s.processWithProcessor(ctx, req, record, "default"); err == nil {
			logrus.Infof("Payment processed successfully with default processor: %s", req.CorrelationID)
			span.SetAttributes(attribute.String("payment.processor.used", "default"))
			s.storage.StorePayment(record)
			return record, nil
		} else {
			logrus.Errorf("Default processor failed for payment %s: %v", req.CorrelationID, err)
			span.SetAttributes(attribute.String("payment.processor.default.error", err.Error()))
		}
	} else {
		logrus.Warnf("Default processor not healthy for payment: %s", req.CorrelationID)
		span.SetAttributes(attribute.Bool("payment.processor.default.unhealthy", true))
	}

	// try fallback processor
	if s.isProcessorHealthy("fallback") {
		logrus.Infof("Trying fallback processor for payment: %s", req.CorrelationID)
		span.SetAttributes(attribute.String("payment.processor.attempted", "fallback"))
		if err := s.processWithProcessor(ctx, req, record, "fallback"); err == nil {
			logrus.Infof("Payment processed successfully with fallback processor: %s", req.CorrelationID)
			span.SetAttributes(attribute.String("payment.processor.used", "fallback"))
			s.storage.StorePayment(record)
			return record, nil
		} else {
			logrus.Errorf("Fallback processor failed for payment %s: %v", req.CorrelationID, err)
			span.SetAttributes(attribute.String("payment.processor.fallback.error", err.Error()))
		}
	} else {
		logrus.Warnf("Fallback processor not healthy for payment: %s", req.CorrelationID)
		span.SetAttributes(attribute.Bool("payment.processor.fallback.unhealthy", true))
	}

	// if  both  fail, mark as failed but still store
	record.Processor = "failed"
	s.storage.StorePayment(record)
	logrus.Errorf("Both processors failed for payment: %s", req.CorrelationID)

	span.SetStatus(codes.Error, "both payment processors are unavailable")
	span.SetAttributes(attribute.String("payment.processor.used", "failed"))

	return record, fmt.Errorf("both payment processors are unavailable")
}

func (s *PaymentService) processWithProcessor(ctx context.Context, req *models.PaymentRequest, record *models.PaymentRecord, processor string) error {
	_, span := otel.Tracer("payment-service").Start(ctx, "processWithProcessor")
	defer span.End()

	span.SetAttributes(
		attribute.String("payment.processor.name", processor),
		attribute.String("payment.correlation_id", req.CorrelationID),
	)
	var url string
	switch processor {
	case "default":
		url = s.config.DefaultProcessorURL + "/payments"
	case "fallback":
		url = s.config.FallbackProcessorURL + "/payments"
	default:
		return fmt.Errorf("unknown processor: %s", processor)
	}

	// prepare request
	processorReq := models.PaymentProcessorRequest{
		CorrelationID: req.CorrelationID,
		Amount:        req.Amount,
		RequestedAt:   time.Now(),
	}

	jsonData, err := json.Marshal(processorReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// make request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	span.SetAttributes(attribute.String("http.url", url))

	resp, err := s.client.Do(httpReq)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("processor returned status %d", resp.StatusCode)
		span.RecordError(err)
		return err
	}

	// parse response
	var processorResp models.PaymentProcessorResponse
	if err := json.NewDecoder(resp.Body).Decode(&processorResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Update record
	record.Processor = processor
	record.Success = true

	return nil
}

func (s *PaymentService) isProcessorHealthy(processor string) bool {
	s.healthMu.RLock()
	defer s.healthMu.RUnlock()

	switch processor {
	case "default":
		return s.defaultHealth.IsHealthy && !s.defaultHealth.Failing
	case "fallback":
		return s.fallbackHealth.IsHealthy && !s.fallbackHealth.Failing
	default:
		return false
	}
}

func (s *PaymentService) StartHealthMonitoring(ctx context.Context) {
	ticker := time.NewTicker(s.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkProcessorHealth("default")
			s.checkProcessorHealth("fallback")
		}
	}
}

func (s *PaymentService) checkProcessorHealth(processor string) {
	// Rate limiting: only check every 5 seconds per processor
	s.healthCheckMu.Lock()
	var lastCheck time.Time
	switch processor {
	case "default":
		lastCheck = s.lastDefaultHealthCheck
	case "fallback":
		lastCheck = s.lastFallbackHealthCheck
	}

	if time.Since(lastCheck) < 5*time.Second {
		s.healthCheckMu.Unlock()
		return
	}

	// Update the last check time for this processor
	switch processor {
	case "default":
		s.lastDefaultHealthCheck = time.Now()
	case "fallback":
		s.lastFallbackHealthCheck = time.Now()
	}
	s.healthCheckMu.Unlock()

	var url string
	switch processor {
	case "default":
		url = s.config.DefaultProcessorURL + "/payments/service-health"
	case "fallback":
		url = s.config.FallbackProcessorURL + "/payments/service-health"
	default:
		return
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logrus.Errorf("Failed to create health check request for %s: %v", processor, err)
		return
	}

	resp, err := s.client.Do(req)
	if err != nil {
		s.updateProcessorHealth(processor, false, 0, true)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		// Rate limited, don't update health status
		return
	}

	if resp.StatusCode != http.StatusOK {
		s.updateProcessorHealth(processor, false, 0, true)
		return
	}

	var healthResp models.HealthCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		logrus.Errorf("Failed to decode health response for %s: %v", processor, err)
		return
	}

	s.updateProcessorHealth(processor, true, healthResp.MinResponseTime, healthResp.Failing)
}

func (s *PaymentService) GetPaymentsSummary(from, to *time.Time) models.PaymentSummary {
	return s.storage.GetPaymentsSummary(from, to)
}

func (s *PaymentService) updateProcessorHealth(processor string, isHealthy bool, minResponseTime int, failing bool) {
	s.healthMu.Lock()
	defer s.healthMu.Unlock()

	switch processor {
	case "default":
		s.defaultHealth.IsHealthy = isHealthy
		s.defaultHealth.MinResponseTime = minResponseTime
		s.defaultHealth.Failing = failing
		s.defaultHealth.LastCheck = time.Now()
	case "fallback":
		s.fallbackHealth.IsHealthy = isHealthy
		s.fallbackHealth.MinResponseTime = minResponseTime
		s.fallbackHealth.Failing = failing
		s.fallbackHealth.LastCheck = time.Now()
	}

	logrus.Infof("Processor %s health updated: healthy=%v, failing=%v, minResponseTime=%d",
		processor, isHealthy, failing, minResponseTime)
}
