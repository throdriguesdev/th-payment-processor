package storage

import (
	"database/sql"
	"fmt"
	"time"
	"th_payment_processor/internal/models"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(host, port, user, password, dbname, sslmode string) (*PostgresStorage, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	storage := &PostgresStorage{db: db}
	if err := storage.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	// Aggressive connection pool for sub-11ms performance
	db.SetMaxOpenConns(50)  // More connections for high concurrency
	db.SetMaxIdleConns(10)  // Keep more idle connections ready
	db.SetConnMaxLifetime(2 * time.Minute)  // Shorter lifetime for fresh connections
	db.SetConnMaxIdleTime(30 * time.Second) // Close idle connections faster

	logrus.Info("Connected to PostgreSQL database")
	return storage, nil
}

func (s *PostgresStorage) createTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS payments (
		id UUID PRIMARY KEY,
		correlation_id VARCHAR(255) UNIQUE NOT NULL,
		amount DECIMAL(10,2) NOT NULL,
		processor VARCHAR(50) NOT NULL,
		processed_at TIMESTAMP WITH TIME ZONE NOT NULL,
		success BOOLEAN NOT NULL DEFAULT false,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_payments_correlation_id ON payments(correlation_id);
	CREATE INDEX IF NOT EXISTS idx_payments_processed_at ON payments(processed_at);
	CREATE INDEX IF NOT EXISTS idx_payments_processor ON payments(processor);
	CREATE INDEX IF NOT EXISTS idx_payments_success ON payments(success);
	`

	_, err := s.db.Exec(query)
	return err
}

func (s *PostgresStorage) StorePayment(record *models.PaymentRecord) error {
	query := `
		INSERT INTO payments (id, correlation_id, amount, processor, processed_at, success)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (correlation_id) DO NOTHING
	`

	_, err := s.db.Exec(query,
		record.ID,
		record.CorrelationID,
		record.Amount,
		record.Processor,
		record.ProcessedAt,
		record.Success,
	)

	if err != nil {
		logrus.Errorf("Failed to store payment: %v", err)
		return fmt.Errorf("failed to store payment: %w", err)
	}

	return nil
}

func (s *PostgresStorage) GetPaymentByCorrelationID(correlationID string) (*models.PaymentRecord, bool) {
	query := `
		SELECT id, correlation_id, amount, processor, processed_at, success
		FROM payments
		WHERE correlation_id = $1
	`

	var record models.PaymentRecord
	err := s.db.QueryRow(query, correlationID).Scan(
		&record.ID,
		&record.CorrelationID,
		&record.Amount,
		&record.Processor,
		&record.ProcessedAt,
		&record.Success,
	)

	if err == sql.ErrNoRows {
		return nil, false
	}
	if err != nil {
		logrus.Errorf("Failed to get payment by correlation ID: %v", err)
		return nil, false
	}

	return &record, true
}

func (s *PostgresStorage) GetPaymentsSummary(from, to *time.Time) models.PaymentSummary {
	query := `
		SELECT 
			processor,
			COUNT(*) as total_requests,
			SUM(amount) as total_amount
		FROM payments
		WHERE success = true
	`

	args := []interface{}{}
	if from != nil {
		query += " AND processed_at >= $" + fmt.Sprintf("%d", len(args)+1)
		args = append(args, *from)
	}
	if to != nil {
		query += " AND processed_at <= $" + fmt.Sprintf("%d", len(args)+1)
		args = append(args, *to)
	}

	query += " GROUP BY processor"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		logrus.Errorf("Failed to get payments summary: %v", err)
		return models.PaymentSummary{}
	}
	defer rows.Close()

	summary := models.PaymentSummary{
		Default:  models.ProcessorSummary{},
		Fallback: models.ProcessorSummary{},
	}

	for rows.Next() {
		var processor string
		var totalRequests int
		var totalAmount float64

		if err := rows.Scan(&processor, &totalRequests, &totalAmount); err != nil {
			logrus.Errorf("Failed to scan summary row: %v", err)
			continue
		}

		switch processor {
		case "default":
			summary.Default.TotalRequests = totalRequests
			summary.Default.TotalAmount = totalAmount
		case "fallback":
			summary.Fallback.TotalRequests = totalRequests
			summary.Fallback.TotalAmount = totalAmount
		}
	}

	return summary
}

func (s *PostgresStorage) GetAllPayments() []*models.PaymentRecord {
	query := `
		SELECT id, correlation_id, amount, processor, processed_at, success
		FROM payments
		ORDER BY processed_at DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		logrus.Errorf("Failed to get all payments: %v", err)
		return []*models.PaymentRecord{}
	}
	defer rows.Close()

	var records []*models.PaymentRecord
	for rows.Next() {
		var record models.PaymentRecord
		if err := rows.Scan(
			&record.ID,
			&record.CorrelationID,
			&record.Amount,
			&record.Processor,
			&record.ProcessedAt,
			&record.Success,
		); err != nil {
			logrus.Errorf("Failed to scan payment record: %v", err)
			continue
		}
		records = append(records, &record)
	}

	return records
}

func (s *PostgresStorage) Close() error {
	return s.db.Close()
}