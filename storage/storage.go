package storage

import (
	"sync"
	"time"
	"payment-processors/models"
	"github.com/google/uuid"
)

type InMemoryStorage struct {
	mu       sync.RWMutex
	payments map[uuid.UUID]*models.PaymentRecord
	config   *models.Config
}

func NewInMemoryStorage(feePercentage float64, minResponseTime int) *InMemoryStorage {
	return &InMemoryStorage{
		payments: make(map[uuid.UUID]*models.PaymentRecord),
		config: &models.Config{
			Token:           "123", // Default token
			Delay:           0,
			Failure:         false,
			FeePercentage:   feePercentage,
			MinResponseTime: minResponseTime,
		},
	}
}

func (s *InMemoryStorage) StorePayment(record *models.PaymentRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.payments[record.ID] = record
}

func (s *InMemoryStorage) GetPayment(id uuid.UUID) (*models.PaymentRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, exists := s.payments[id]
	return record, exists
}

func (s *InMemoryStorage) GetPaymentsSummary(from, to *time.Time) models.PaymentSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	summary := models.PaymentSummary{
		FeePerTransaction: s.config.FeePercentage,
	}
	
	for _, record := range s.payments {
		// Filter by time range if provided
		if from != nil && record.ProcessedAt.Before(*from) {
			continue
		}
		if to != nil && record.ProcessedAt.After(*to) {
			continue
		}
		
		summary.TotalRequests++
		summary.TotalAmount += record.Amount
		summary.TotalFee += record.Fee
	}
	
	return summary
}

func (s *InMemoryStorage) GetConfig() *models.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

func (s *InMemoryStorage) UpdateConfig(config *models.Config) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = config
}

func (s *InMemoryStorage) PurgePayments() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.payments = make(map[uuid.UUID]*models.PaymentRecord)
} 