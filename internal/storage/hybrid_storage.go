package storage

import (
	"context"
	"time"
	"th_payment_processor/internal/models"

	"github.com/sirupsen/logrus"
)

type HybridStorage struct {
	postgres *PostgresStorage
	redis    *RedisCache
}

func NewHybridStorage(postgres *PostgresStorage, redis *RedisCache) *HybridStorage {
	return &HybridStorage{
		postgres: postgres,
		redis:    redis,
	}
}

func (h *HybridStorage) StorePayment(ctx context.Context, record *models.PaymentRecord) error {
	// Store in PostgreSQL for persistence
	pgErr := h.postgres.StorePayment(ctx, record)
	if pgErr != nil {
		logrus.Errorf("Failed to store payment in PostgreSQL: %v", pgErr)
	}

	// Cache in Redis for fast access
	if err := h.redis.CachePayment(record); err != nil {
		logrus.Errorf("Failed to cache payment in Redis: %v", err)
		// Continue even if Redis fails, we have PostgreSQL persistence
	}

	// Publish to Redis stream for real-time processing
	if err := h.redis.PublishPaymentEvent(record); err != nil {
		logrus.Errorf("Failed to publish payment event: %v", err)
	}

	// Return PostgreSQL error if it failed, since that's our primary store
	return pgErr
}

func (h *HybridStorage) GetPaymentByCorrelationID(correlationID string) (*models.PaymentRecord, bool) {
	// Try Redis cache first for speed
	if record, found := h.redis.GetCachedPayment(correlationID); found {
		return record, true
	}

	// Fallback to PostgreSQL
	record, found := h.postgres.GetPaymentByCorrelationID(correlationID)
	if found {
		// Cache the result for next time
		if err := h.redis.CachePayment(record); err != nil {
			logrus.Errorf("Failed to cache payment from PostgreSQL: %v", err)
		}
	}

	return record, found
}

func (h *HybridStorage) GetPaymentsSummary(from, to *time.Time) models.PaymentSummary {
	// Try Redis cache first
	if summary, found := h.redis.GetCachedPaymentSummary(from, to); found {
		return *summary
	}

	// Fallback to PostgreSQL
	summary := h.postgres.GetPaymentsSummary(from, to)

	// Cache the result
	if err := h.redis.CachePaymentSummary(summary, from, to); err != nil {
		logrus.Errorf("Failed to cache payment summary: %v", err)
	}

	return summary
}

func (h *HybridStorage) GetAllPayments() []*models.PaymentRecord {
	// This operation always goes to PostgreSQL for consistency
	return h.postgres.GetAllPayments()
}

// Health check methods
func (h *HybridStorage) IsPostgresHealthy() bool {
	// Simple ping to check if PostgreSQL is available
	records := h.postgres.GetAllPayments()
	return records != nil
}

func (h *HybridStorage) IsRedisHealthy() bool {
	return h.redis.IsHealthy()
}

func (h *HybridStorage) GetHealthStatus() map[string]bool {
	return map[string]bool{
		"postgres": h.IsPostgresHealthy(),
		"redis":    h.IsRedisHealthy(),
	}
}

func (h *HybridStorage) Close() error {
	var pgErr, redisErr error
	
	if h.postgres != nil {
		pgErr = h.postgres.Close()
	}
	
	if h.redis != nil {
		redisErr = h.redis.Close()
	}
	
	if pgErr != nil {
		return pgErr
	}
	return redisErr
}