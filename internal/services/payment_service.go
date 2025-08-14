package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/semconv/v1.21.0"
	"th_payment_processor/internal/config"
	"th_payment_processor/internal/logging"
	"th_payment_processor/internal/models"
	"th_payment_processor/internal/profiling"
	"th_payment_processor/internal/storage"
)

type PaymentService struct {
	config  *config.Config
	storage storage.Storage
	client  *http.Client
	profiler *profiling.Profiler

	// Health monitoring
	healthMu       sync.RWMutex
	defaultHealth  *models.ProcessorHealth
	fallbackHealth *models.ProcessorHealth

	// Rate limiting for health checks (per processor)
	lastDefaultHealthCheck  time.Time
	lastFallbackHealthCheck time.Time
	healthCheckMu           sync.Mutex

	// Performance optimizations
	requestPool sync.Pool
}

func NewPaymentService(cfg *config.Config, storage storage.Storage, profiler *profiling.Profiler) *PaymentService {
	service := &PaymentService{
		config:  cfg,
		storage: storage,
		profiler: profiler,
		client: &http.Client{
			Timeout:   cfg.RequestTimeout,
			Transport: otelhttp.NewTransport(&http.Transport{
				MaxIdleConns:          300,
				MaxConnsPerHost:       300,
				MaxIdleConnsPerHost:   300,
				IdleConnTimeout:       30 * time.Second,
				DisableKeepAlives:     false,
				DisableCompression:    true,
				TLSHandshakeTimeout:   1 * time.Second,
				ResponseHeaderTimeout: 1 * time.Second,
				ExpectContinueTimeout: 200 * time.Millisecond,
			}),
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

	// Initialize object pool for request reuse
	service.requestPool = sync.Pool{
		New: func() interface{} {
			return &models.PaymentProcessorRequest{}
		},
	}

	return service
}

func (s *PaymentService) ProcessPayment(ctx context.Context, req *models.PaymentRequest) (*models.PaymentRecord, error) {
	tracer := otel.Tracer("payment-service")
	ctx, span := tracer.Start(ctx, "ProcessPayment")
	defer span.End()

	// Create structured logger with context
	logger := logging.NewStructuredLogger("payment_service").WithContext(ctx)
	startTime := time.Now()

	span.SetAttributes(
		attribute.String("payment.correlation_id", req.CorrelationID),
		attribute.Float64("payment.amount", req.Amount),
		attribute.String("service.operation", "process_payment"),
		semconv.ServiceNameKey.String("th-payment-processor"),
		semconv.ServiceVersionKey.String("1.0.0"),
		semconv.HTTPMethodKey.String("POST"),
		attribute.String("business.payment.type", "standard"),
	)

	// Log payment start with structured logging
	logger.LogPaymentStart(req.CorrelationID, req.Amount)

	// Skip duplicate check for performance - database handles uniqueness constraint
	// This reduces latency by avoiding extra database lookups during high load

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
		logger.WithOperation("try_default_processor").Debug("Attempting payment with default processor")
		span.SetAttributes(attribute.String("payment.processor.attempted", "default"))
		if err := s.processWithProcessor(ctx, req, record, "default"); err == nil {
			latencyMs := time.Since(startTime).Milliseconds()
			logger.LogPaymentSuccess(req.CorrelationID, req.Amount, "default", latencyMs)
			span.SetAttributes(attribute.String("payment.processor.used", "default"))
			if err := s.storage.StorePayment(ctx, record); err != nil {
				logger.WithError(err).Error("Failed to store payment record")
			}
			return record, nil
		} else {
			logger.WithPaymentFields(logging.PaymentFields{
				CorrelationID: req.CorrelationID,
				Amount:        req.Amount,
				Processor:     "default",
			}).WithError(err).Error("Default processor failed")
			span.SetAttributes(attribute.String("payment.processor.default.error", err.Error()))
		}
	} else {
		logger.WithPaymentFields(logging.PaymentFields{
			CorrelationID: req.CorrelationID,
			Processor:     "default",
		}).Warn("Default processor not healthy, skipping")
		span.SetAttributes(attribute.Bool("payment.processor.default.unhealthy", true))
	}

	// try fallback processor
	if s.isProcessorHealthy("fallback") {
		logger.WithOperation("try_fallback_processor").Debug("Attempting payment with fallback processor")
		span.SetAttributes(attribute.String("payment.processor.attempted", "fallback"))
		if err := s.processWithProcessor(ctx, req, record, "fallback"); err == nil {
			latencyMs := time.Since(startTime).Milliseconds()
			logger.LogPaymentSuccess(req.CorrelationID, req.Amount, "fallback", latencyMs)
			span.SetAttributes(attribute.String("payment.processor.used", "fallback"))
			if err := s.storage.StorePayment(ctx, record); err != nil {
				logger.WithError(err).Error("Failed to store payment record")
			}
			return record, nil
		} else {
			logger.WithPaymentFields(logging.PaymentFields{
				CorrelationID: req.CorrelationID,
				Amount:        req.Amount,
				Processor:     "fallback",
			}).WithError(err).Error("Fallback processor failed")
			span.SetAttributes(attribute.String("payment.processor.fallback.error", err.Error()))
		}
	} else {
		logger.WithPaymentFields(logging.PaymentFields{
			CorrelationID: req.CorrelationID,
			Processor:     "fallback",
		}).Warn("Fallback processor not healthy, skipping")
		span.SetAttributes(attribute.Bool("payment.processor.fallback.unhealthy", true))
	}

	// if both fail, mark as failed but still store
	record.Processor = "failed"
	s.storage.StorePayment(ctx, record)
	
	latencyMs := time.Since(startTime).Milliseconds()
	err := fmt.Errorf("both payment processors are unavailable")
	logger.LogPaymentFailure(req.CorrelationID, req.Amount, "failed", err, latencyMs)

	span.SetStatus(codes.Error, "both payment processors are unavailable")
	span.SetAttributes(attribute.String("payment.processor.used", "failed"))

	return record, err
}

func (s *PaymentService) processWithProcessor(ctx context.Context, req *models.PaymentRequest, record *models.PaymentRecord, processor string) error {
	_, span := otel.Tracer("payment-service").Start(ctx, "processWithProcessor")
	defer span.End()

	logger := logging.NewStructuredLogger("payment_processor").WithContext(ctx)
	startTime := time.Now()

	span.SetAttributes(
		attribute.String("payment.processor.name", processor),
		attribute.String("payment.correlation_id", req.CorrelationID),
		attribute.Float64("payment.amount", req.Amount),
		attribute.String("service.operation", "external_processor_call"),
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

	// Get request object from pool
	processorReq := s.requestPool.Get().(*models.PaymentProcessorRequest)
	defer s.requestPool.Put(processorReq)
	
	// Reset and populate request
	processorReq.CorrelationID = req.CorrelationID
	processorReq.Amount = req.Amount
	processorReq.RequestedAt = time.Now()

	jsonData, err := json.Marshal(processorReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// make request with enhanced tracing
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		span.RecordError(err)
		logger.WithPaymentFields(logging.PaymentFields{
			CorrelationID: req.CorrelationID,
			Processor:     processor,
		}).WithError(err).Error("Failed to create HTTP request")
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add correlation ID to request headers for upstream tracing
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Correlation-ID", req.CorrelationID)
	
	span.SetAttributes(
		attribute.String("http.url", url),
		attribute.String("http.method", "POST"),
		attribute.String("http.request.correlation_id", req.CorrelationID),
	)

	logger.WithHTTPFields(logging.HTTPFields{
		Method: "POST",
		URL:    url,
	}).WithField("operation", "external_processor_request").Debug("Making request to payment processor")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		span.RecordError(err)
		latencyMs := time.Since(startTime).Milliseconds()
		logger.WithPaymentFields(logging.PaymentFields{
			CorrelationID: req.CorrelationID,
			Processor:     processor,
		}).WithError(err).WithField("latency_ms", latencyMs).Error("HTTP request to processor failed")
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	latencyMs := time.Since(startTime).Milliseconds()
	span.SetAttributes(
		attribute.Int("http.status_code", resp.StatusCode),
		attribute.Int64("http.response.duration_ms", latencyMs),
	)

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("processor returned status %d", resp.StatusCode)
		span.RecordError(err)
		logger.WithPaymentFields(logging.PaymentFields{
			CorrelationID: req.CorrelationID,
			Processor:     processor,
		}).WithField("http_status", resp.StatusCode).WithField("latency_ms", latencyMs).Error("Processor returned non-200 status")
		return err
	}

	// parse response
	var processorResp models.PaymentProcessorResponse
	if err := json.NewDecoder(resp.Body).Decode(&processorResp); err != nil {
		logger.WithPaymentFields(logging.PaymentFields{
			CorrelationID: req.CorrelationID,
			Processor:     processor,
		}).WithError(err).Error("Failed to decode processor response")
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Update record
	record.Processor = processor
	record.Success = true

	logger.WithPaymentFields(logging.PaymentFields{
		CorrelationID: req.CorrelationID,
		Processor:     processor,
		Amount:        req.Amount,
		Status:        "success",
	}).WithField("latency_ms", latencyMs).WithField("operation", "processor_success").Debug("Payment processed successfully by external processor")

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
	ctx := context.Background()
	logger := logging.NewStructuredLogger("health_monitor").WithContext(ctx)
	startTime := time.Now()

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

	logger.WithOperation("health_check_start").Debug("Starting health check for processor")

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		logger.WithPaymentFields(logging.PaymentFields{
			Processor: processor,
		}).WithError(err).Error("Failed to create health check request")
		return
	}

	resp, err := s.client.Do(req)
	if err != nil {
		logger.WithPaymentFields(logging.PaymentFields{
			Processor: processor,
		}).WithError(err).Error("Health check request failed")
		s.updateProcessorHealth(processor, false, 0, true)
		return
	}
	defer resp.Body.Close()

	responseTime := time.Since(startTime).Milliseconds()

	if resp.StatusCode == http.StatusTooManyRequests {
		// Rate limited, don't update health status
		logger.WithPaymentFields(logging.PaymentFields{
			Processor: processor,
		}).Warn("Health check rate limited, skipping status update")
		return
	}

	if resp.StatusCode != http.StatusOK {
		logger.WithPaymentFields(logging.PaymentFields{
			Processor: processor,
		}).WithField("http_status", resp.StatusCode).Warn("Health check returned non-200 status")
		s.updateProcessorHealth(processor, false, 0, true)
		return
	}

	var healthResp models.HealthCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		logger.WithPaymentFields(logging.PaymentFields{
			Processor: processor,
		}).WithError(err).Error("Failed to decode health check response")
		return
	}

	logger.LogHealthCheck(processor, true, responseTime)
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
