package models

import (
	"time"
	"github.com/google/uuid"
)

type PaymentRequest struct {
	CorrelationID string    `json:"correlationId" binding:"required"`
	Amount        float64   `json:"amount" binding:"required,gt=0"`
	RequestedAt   time.Time `json:"requestedAt" binding:"required"`
}

type PaymentResponse struct {
	Message string `json:"message"`
}

type PaymentRecord struct {
	ID            uuid.UUID `json:"id"`
	CorrelationID string    `json:"correlationId"`
	Amount        float64   `json:"amount"`
	RequestedAt   time.Time `json:"requestedAt"`
	ProcessedAt   time.Time `json:"processedAt"`
	Fee           float64   `json:"fee"`
}

type HealthCheckResponse struct {
	Failing        bool `json:"failing"`
	MinResponseTime int  `json:"minResponseTime"`
}

type PaymentSummary struct {
	TotalRequests     int     `json:"totalRequests"`
	TotalAmount       float64 `json:"totalAmount"`
	TotalFee          float64 `json:"totalFee"`
	FeePerTransaction float64 `json:"feePerTransaction"`
}

type Config struct {
	Token           string        `json:"token"`
	Delay           int           `json:"delay"`
	Failure         bool          `json:"failure"`
	FeePercentage   float64       `json:"feePercentage"`
	MinResponseTime int           `json:"minResponseTime"`
} 