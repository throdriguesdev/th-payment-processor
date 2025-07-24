package models

import (
	"time"
	"github.com/google/uuid"
)

type PaymentRequest struct {
	CorrelationID string  `json:"correlationId" binding:"required"`
	Amount        float64 `json:"amount" binding:"required,gt=0"`
}

type PaymentProcessorRequest struct {
	CorrelationID string    `json:"correlationId"`
	Amount        float64   `json:"amount"`
	RequestedAt   time.Time `json:"requestedAt"`
}

type PaymentProcessorResponse struct {
	Message string `json:"message"`
}

type HealthCheckResponse struct {
	Failing        bool `json:"failing"`
	MinResponseTime int  `json:"minResponseTime"`
}

type PaymentSummary struct {
	Default  ProcessorSummary `json:"default"`
	Fallback ProcessorSummary `json:"fallback"`
}

type ProcessorSummary struct {
	TotalRequests int     `json:"totalRequests"`
	TotalAmount   float64 `json:"totalAmount"`
}

type PaymentRecord struct {
	ID            uuid.UUID `json:"id"`
	CorrelationID string    `json:"correlationId"`
	Amount        float64   `json:"amount"`
	Processor     string    `json:"processor"`
	ProcessedAt   time.Time `json:"processedAt"`
	Success       bool      `json:"success"`
}

type ProcessorHealth struct {
	IsHealthy     bool
	MinResponseTime int
	LastCheck     time.Time
	Failing       bool
} 