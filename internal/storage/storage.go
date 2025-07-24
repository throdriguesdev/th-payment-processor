package storage

import (
	"fmt"
	"rinha-backend/internal/models"
	"sync"
	"time"

	"github.com/google/uuid"
)

type InMemoryStorage struct {
	mu       sync.RWMutex
	payments map[string]*models.PaymentRecord
	byID     map[uuid.UUID]*models.PaymentRecord
}

func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		payments: make(map[string]*models.PaymentRecord),
		byID:     make(map[uuid.UUID]*models.PaymentRecord),
	}
}
func (s *InMemoryStorage) StorePayment(record *models.PaymentRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.payments[record.CorrelationID] = record
	s.byID[record.ID] = record

	// Debug logging
	// fmt.Printf("[DEBUG] Stored payment: ID=%s, CorrelationID=%s, Amount=%.2f, Processor=%s, Success=%v\n",
	// 	record.ID, record.CorrelationID, record.Amount, record.Processor, record.Success)
	// fmt.Printf("[DEBUG] Total payments in storage: %d\n", len(s.payments))
}

func (s *InMemoryStorage) GetPaymentByCorrelationID(correlationID string) (*models.PaymentRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	record, exists := s.payments[correlationID]
	return record, exists
}

func (s *InMemoryStorage) GetPaymentsSummary(from, to *time.Time) models.PaymentSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	summary := models.PaymentSummary{
		Default:  models.ProcessorSummary{},
		Fallback: models.ProcessorSummary{},
	}

	fmt.Printf("[DEBUG] GetPaymentsSummary: Total payments in storage: %d\n", len(s.payments))

	for _, record := range s.payments {
		fmt.Printf("[DEBUG] Processing record: ID=%s, Processor=%s, Success=%v, Amount=%.2f\n",
			record.ID, record.Processor, record.Success, record.Amount)

		// filter by time range if provided
		if from != nil && record.ProcessedAt.Before(*from) {
			fmt.Printf("[DEBUG] Skipping record due to 'from' filter: %s\n", record.ID)
			continue
		}
		if to != nil && record.ProcessedAt.After(*to) {
			fmt.Printf("[DEBUG] Skipping record due to 'to' filter: %s\n", record.ID)
			continue
		}

		if record.Success {
			switch record.Processor {
			case "default":
				summary.Default.TotalRequests++
				summary.Default.TotalAmount += record.Amount
				fmt.Printf("[DEBUG] Added to default summary: requests=%d, amount=%.2f\n",
					summary.Default.TotalRequests, summary.Default.TotalAmount)
			case "fallback":
				summary.Fallback.TotalRequests++
				summary.Fallback.TotalAmount += record.Amount
				fmt.Printf("[DEBUG] Added to fallback summary: requests=%d, amount=%.2f\n",
					summary.Fallback.TotalRequests, summary.Fallback.TotalAmount)
			}
		} else {
			fmt.Printf("[DEBUG] Skipping unsuccessful payment: %s\n", record.ID)
		}
	}

	fmt.Printf("[DEBUG] Final summary - Default: %+v, Fallback: %+v\n", summary.Default, summary.Fallback)
	return summary
}

func (s *InMemoryStorage) GetAllPayments() []*models.PaymentRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := make([]*models.PaymentRecord, 0, len(s.payments))
	for _, record := range s.payments {
		records = append(records, record)
	}
	return records
}
