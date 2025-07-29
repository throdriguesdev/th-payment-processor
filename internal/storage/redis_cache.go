package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
	"th_payment_processor/internal/models"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type RedisCache struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisCache(addr, password string, db int) (*RedisCache, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:            addr,
		Password:        password,
		DB:              db,
		PoolSize:        20,  // More connections
		MinIdleConns:    5,   // Keep more idle connections
		MaxRetries:      1,   // Faster failure for sub-11ms
		ReadTimeout:     1 * time.Second,  // Aggressive timeouts
		WriteTimeout:    1 * time.Second,
		DialTimeout:     500 * time.Millisecond,
		PoolTimeout:     500 * time.Millisecond,
	})

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logrus.Info("Connected to Redis cache")
	return &RedisCache{
		client: rdb,
		ctx:    ctx,
	}, nil
}

func (r *RedisCache) CachePayment(record *models.PaymentRecord) error {
	key := fmt.Sprintf("payment:%s", record.CorrelationID)
	
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal payment record: %w", err)
	}

	// Cache for 2 hours for better performance
	if err := r.client.Set(r.ctx, key, data, 2*time.Hour).Err(); err != nil {
		logrus.Errorf("Failed to cache payment: %v", err)
		return fmt.Errorf("failed to cache payment: %w", err)
	}

	return nil
}

func (r *RedisCache) GetCachedPayment(correlationID string) (*models.PaymentRecord, bool) {
	key := fmt.Sprintf("payment:%s", correlationID)
	
	data, err := r.client.Get(r.ctx, key).Result()
	if err == redis.Nil {
		return nil, false
	}
	if err != nil {
		logrus.Errorf("Failed to get cached payment: %v", err)
		return nil, false
	}

	var record models.PaymentRecord
	if err := json.Unmarshal([]byte(data), &record); err != nil {
		logrus.Errorf("Failed to unmarshal cached payment: %v", err)
		return nil, false
	}

	return &record, true
}

func (r *RedisCache) CachePaymentSummary(summary models.PaymentSummary, from, to *time.Time) error {
	key := r.buildSummaryKey(from, to)
	
	data, err := json.Marshal(summary)
	if err != nil {
		return fmt.Errorf("failed to marshal payment summary: %w", err)
	}

	// Cache summary for 60 seconds for better performance
	if err := r.client.Set(r.ctx, key, data, 60*time.Second).Err(); err != nil {
		logrus.Errorf("Failed to cache payment summary: %v", err)
		return fmt.Errorf("failed to cache payment summary: %w", err)
	}

	return nil
}

func (r *RedisCache) GetCachedPaymentSummary(from, to *time.Time) (*models.PaymentSummary, bool) {
	key := r.buildSummaryKey(from, to)
	
	data, err := r.client.Get(r.ctx, key).Result()
	if err == redis.Nil {
		return nil, false
	}
	if err != nil {
		logrus.Errorf("Failed to get cached payment summary: %v", err)
		return nil, false
	}

	var summary models.PaymentSummary
	if err := json.Unmarshal([]byte(data), &summary); err != nil {
		logrus.Errorf("Failed to unmarshal cached payment summary: %v", err)
		return nil, false
	}

	return &summary, true
}

func (r *RedisCache) buildSummaryKey(from, to *time.Time) string {
	key := "summary"
	if from != nil {
		key += fmt.Sprintf(":from:%d", from.Unix())
	}
	if to != nil {
		key += fmt.Sprintf(":to:%d", to.Unix())
	}
	return key
}

// Streaming functionality using Redis Streams
func (r *RedisCache) PublishPaymentEvent(record *models.PaymentRecord) error {
	streamKey := "payment_events"
	
	fields := map[string]interface{}{
		"id":            record.ID.String(),
		"correlation_id": record.CorrelationID,
		"amount":        record.Amount,
		"processor":     record.Processor,
		"processed_at":  record.ProcessedAt.Format(time.RFC3339),
		"success":       record.Success,
	}

	if err := r.client.XAdd(r.ctx, &redis.XAddArgs{
		Stream: streamKey,
		MaxLen: 10000, // Keep last 10k events
		Approx: true,
		Values: fields,
	}).Err(); err != nil {
		logrus.Errorf("Failed to publish payment event: %v", err)
		return fmt.Errorf("failed to publish payment event: %w", err)
	}

	return nil
}

func (r *RedisCache) ConsumePaymentEvents(consumerGroup, consumer string, handler func(*models.PaymentRecord)) error {
	streamKey := "payment_events"
	
	// Create consumer group if it doesn't exist
	r.client.XGroupCreateMkStream(r.ctx, streamKey, consumerGroup, "$")

	for {
		streams, err := r.client.XReadGroup(r.ctx, &redis.XReadGroupArgs{
			Group:    consumerGroup,
			Consumer: consumer,
			Streams:  []string{streamKey, ">"},
			Count:    10,
			Block:    time.Second,
		}).Result()

		if err != nil && err != redis.Nil {
			logrus.Errorf("Failed to read from stream: %v", err)
			time.Sleep(time.Second)
			continue
		}

		for _, stream := range streams {
			for _, message := range stream.Messages {
				record, err := r.parsePaymentEventMessage(message.Values)
				if err != nil {
					logrus.Errorf("Failed to parse payment event: %v", err)
					continue
				}

				handler(record)

				// Acknowledge the message
				r.client.XAck(r.ctx, streamKey, consumerGroup, message.ID)
			}
		}
	}
}

func (r *RedisCache) parsePaymentEventMessage(values map[string]interface{}) (*models.PaymentRecord, error) {
	record := &models.PaymentRecord{}
	
	if id, ok := values["id"].(string); ok {
		if parsed, err := uuid.Parse(id); err == nil {
			record.ID = parsed
		}
	}
	
	if correlationID, ok := values["correlation_id"].(string); ok {
		record.CorrelationID = correlationID
	}
	
	if amount, ok := values["amount"].(string); ok {
		if parsed, err := strconv.ParseFloat(amount, 64); err == nil {
			record.Amount = parsed
		}
	}
	
	if processor, ok := values["processor"].(string); ok {
		record.Processor = processor
	}
	
	if processedAt, ok := values["processed_at"].(string); ok {
		if parsed, err := time.Parse(time.RFC3339, processedAt); err == nil {
			record.ProcessedAt = parsed
		}
	}
	
	if success, ok := values["success"].(string); ok {
		if parsed, err := strconv.ParseBool(success); err == nil {
			record.Success = parsed
		}
	}
	
	return record, nil
}

// Health check for Redis
func (r *RedisCache) IsHealthy() bool {
	return r.client.Ping(r.ctx).Err() == nil
}

func (r *RedisCache) Close() error {
	return r.client.Close()
}