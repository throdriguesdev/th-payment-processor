# Database & Redis Integration Guide

## Current Architecture State

The rinha-backend is designed with database integration in mind. The storage layer uses an interface-based approach that makes switching to persistent storage straightforward.

## Current Storage Implementation

### Interface Design (`internal/storage/storage.go`)

```go
type Storage interface {
    StorePayment(payment *PaymentRecord)
    GetPaymentByID(id uuid.UUID) *PaymentRecord
    GetPaymentByCorrelationID(correlationID string) *PaymentRecord
    GetPaymentsSummary(from, to *time.Time) *PaymentSummary
}
```

### Current In-Memory Implementation

```go
type InMemoryStorage struct {
    mu       sync.RWMutex
    payments map[string]*PaymentRecord  // By correlationId
    byID     map[uuid.UUID]*PaymentRecord // By payment ID
}
```

## Database Integration Plan

### 1. PostgreSQL Integration

**New Implementation Structure:**
```go
type PostgreSQLStorage struct {
    db *sql.DB
}

func NewPostgreSQLStorage(connectionString string) (*PostgreSQLStorage, error) {
    db, err := sql.Open("postgres", connectionString)
    if err != nil {
        return nil, err
    }
    
    return &PostgreSQLStorage{db: db}, nil
}
```

**Required Database Schema:**
```sql
CREATE TABLE payments (
    id UUID PRIMARY KEY,
    correlation_id VARCHAR(255) UNIQUE NOT NULL,
    amount DECIMAL(10,2) NOT NULL,
    processor VARCHAR(50) NOT NULL,
    requested_at TIMESTAMP NOT NULL,
    processed_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_payments_correlation_id ON payments(correlation_id);
CREATE INDEX idx_payments_processed_at ON payments(processed_at);
CREATE INDEX idx_payments_processor ON payments(processor);
```

**Environment Configuration Updates:**
```go
type Config struct {
    // Existing fields...
    DatabaseURL              string        // New: PostgreSQL connection string
    DatabaseMaxConnections   int          // New: Connection pool size
    DatabaseConnTimeout      time.Duration // New: Connection timeout
}
```

### 2. Storage Interface Implementation

**StorePayment Method:**
```go
func (p *PostgreSQLStorage) StorePayment(payment *PaymentRecord) {
    query := `
        INSERT INTO payments (id, correlation_id, amount, processor, requested_at, processed_at)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (correlation_id) DO NOTHING`
    
    _, err := p.db.Exec(query, 
        payment.ID, 
        payment.CorrelationID, 
        payment.Amount, 
        payment.Processor, 
        payment.RequestedAt, 
        payment.ProcessedAt)
    
    if err != nil {
        log.WithError(err).Error("Failed to store payment")
    }
}
```

**GetPaymentsSummary Method:**
```go
func (p *PostgreSQLStorage) GetPaymentsSummary(from, to *time.Time) *PaymentSummary {
    query := `
        SELECT processor, COUNT(*), SUM(amount)
        FROM payments 
        WHERE ($1::timestamp IS NULL OR processed_at >= $1)
        AND ($2::timestamp IS NULL OR processed_at <= $2)
        AND processor IN ('default', 'fallback')
        GROUP BY processor`
    
    rows, err := p.db.Query(query, from, to)
    // Process results...
}
```

### 3. Configuration Changes

**Docker Compose Addition:**
```yaml
services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: rinha_backend
      POSTGRES_USER: rinha
      POSTGRES_PASSWORD: rinha123
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./sql/init.sql:/docker-entrypoint-initdb.d/init.sql
    networks:
      - backend
    deploy:
      resources:
        limits:
          cpus: "0.5"
          memory: "256MB"

volumes:
  postgres_data:
```

**Environment Variables:**
```bash
DATABASE_URL=postgres://rinha:rinha123@postgres:5432/rinha_backend?sslmode=disable
DATABASE_MAX_CONNECTIONS=25
DATABASE_CONN_TIMEOUT=30s
```

## Redis Integration Plan

### 1. Health Status Caching

**Current Problem:** Each app instance maintains separate health status for processors, causing potential inconsistencies.

**Redis Solution:**
```go
type RedisHealthMonitor struct {
    redis *redis.Client
}

func (r *RedisHealthMonitor) SetProcessorHealth(processor string, healthy bool) error {
    key := fmt.Sprintf("processor:health:%s", processor)
    value := "false"
    if healthy {
        value = "true"
    }
    
    return r.redis.Set(context.Background(), key, value, 30*time.Second).Err()
}

func (r *RedisHealthMonitor) IsProcessorHealthy(processor string) (bool, error) {
    key := fmt.Sprintf("processor:health:%s", processor)
    val, err := r.redis.Get(context.Background(), key).Result()
    if err == redis.Nil {
        return false, nil // Default to unhealthy if no data
    }
    return val == "true", err
}
```

### 2. Session Storage Replacement

**Current Issue:** Nginx `ip_hash` required for session affinity with in-memory storage.

**Redis Solution:**
```go
type RedisSessionStorage struct {
    redis *redis.Client
}

func (r *RedisSessionStorage) StorePayment(payment *PaymentRecord) {
    // Store in both PostgreSQL and Redis cache
    data, _ := json.Marshal(payment)
    
    // Cache for 1 hour for fast access
    r.redis.Set(context.Background(), 
        fmt.Sprintf("payment:correlation:%s", payment.CorrelationID),
        data, time.Hour)
    
    r.redis.Set(context.Background(),
        fmt.Sprintf("payment:id:%s", payment.ID.String()),
        data, time.Hour)
}
```

### 3. Pub/Sub for Real-time Updates

**Health Status Broadcasting:**
```go
func (s *PaymentService) broadcastHealthUpdate(processor string, healthy bool) {
    message := map[string]interface{}{
        "processor": processor,
        "healthy":   healthy,
        "timestamp": time.Now(),
    }
    
    data, _ := json.Marshal(message)
    s.redis.Publish(context.Background(), "processor:health:updates", data)
}

func (s *PaymentService) subscribeToHealthUpdates() {
    pubsub := s.redis.Subscribe(context.Background(), "processor:health:updates")
    defer pubsub.Close()
    
    for msg := range pubsub.Channel() {
        var update HealthUpdate
        json.Unmarshal([]byte(msg.Payload), &update)
        s.updateLocalHealthStatus(update.Processor, update.Healthy)
    }
}
```

## Migration Strategy

### Phase 1: Database Backend
1. **Add PostgreSQL dependencies** to `go.mod`
2. **Implement PostgreSQLStorage** following the Storage interface
3. **Add database configuration** to Config struct
4. **Update docker-compose.yml** with PostgreSQL service
5. **Create database initialization scripts**
6. **Update service initialization** to use database storage
7. **Remove nginx ip_hash** dependency

### Phase 2: Redis Caching Layer
1. **Add Redis dependencies** to `go.mod`
2. **Implement Redis health monitoring** 
3. **Add Redis session caching** for fast lookups
4. **Update docker-compose.yml** with Redis service
5. **Implement pub/sub for health status**
6. **Remove in-memory health state**

### Phase 3: Performance Optimization
1. **Add connection pooling** for database
2. **Implement read replicas** if needed
3. **Add database indexes** for query optimization
4. **Monitor query performance**
5. **Implement caching strategies** for frequent queries

## Configuration Updates Required

### Environment Variables
```bash
# Database Configuration
DATABASE_URL=postgres://user:pass@postgres:5432/rinha_backend
DATABASE_MAX_CONNECTIONS=25
DATABASE_CONN_TIMEOUT=30s

# Redis Configuration  
REDIS_URL=redis://redis:6379
REDIS_DB=0
REDIS_PASSWORD=
REDIS_CONN_TIMEOUT=5s

# Feature Flags
USE_DATABASE_STORAGE=true
USE_REDIS_HEALTH_MONITORING=true
USE_REDIS_SESSION_CACHE=true
```

### Docker Compose Resource Updates
```yaml
# Adjust existing resource limits to accommodate new services
services:
  nginx:
    deploy:
      resources:
        limits:
          cpus: "0.2"
          memory: "32MB"
  
  app1:
    deploy:
      resources:
        limits:
          cpus: "0.4"
          memory: "128MB"
          
  app2:
    deploy:
      resources:
        limits:
          cpus: "0.4"
          memory: "128MB"
          
  postgres:
    deploy:
      resources:
        limits:
          cpus: "0.3"
          memory: "256MB"
          
  redis:
    deploy:
      resources:
        limits:
          cpus: "0.2"
          memory: "64MB"
```

## Benefits After Integration

### Scalability
- **Horizontal Scaling**: Multiple app instances without session affinity
- **Data Persistence**: Payments survive service restarts
- **Shared State**: Consistent health monitoring across instances

### Performance
- **Fast Lookups**: Redis caching for frequent queries
- **Database Optimization**: Proper indexing and query optimization
- **Connection Pooling**: Efficient database resource usage

### Reliability
- **Data Durability**: PostgreSQL ACID compliance
- **Backup Support**: Standard database backup procedures
- **Monitoring**: Database and Redis metrics available

### Development
- **Testing**: Database transactions for isolated tests
- **Debugging**: SQL queries for payment investigation
- **Analytics**: Complex queries for business intelligence

This migration maintains the existing API contract while providing a solid foundation for production deployment and future scaling requirements.