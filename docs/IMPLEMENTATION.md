# Implementation Details

## Core Application Components

### Main Application (`main.go`)
**Key Features:**
- Structured logging with logrus (JSON format)
- OpenTelemetry tracing integration
- Environment-based configuration
- Graceful shutdown handling
- HTTP server with Gin framework

**Initialization Flow:**
1. Configure structured logging
2. Initialize OpenTelemetry tracer
3. Load environment configuration
4. Create in-memory storage
5. Initialize payment service with background health monitoring
6. Setup HTTP routes and middleware
7. Start server on configured port

### Configuration Management (`internal/config/config.go`)
```go
type Config struct {
    ServerPort               string        // Default: "8080"
    DefaultProcessorURL      string        // Default: "http://payment-processor-default:8080"
    FallbackProcessorURL     string        // Default: "http://payment-processor-fallback:8080"
    HealthCheckInterval      time.Duration // Default: 5s
    RequestTimeout          time.Duration // Default: 10s
}
```

**Environment Variables:**
- `SERVER_PORT`: Application port
- `DEFAULT_PROCESSOR_URL`: Primary processor endpoint
- `FALLBACK_PROCESSOR_URL`: Secondary processor endpoint
- `HEALTH_CHECK_INTERVAL`: Health check frequency
- `REQUEST_TIMEOUT`: HTTP client timeout

### Payment Models (`internal/models/payment.go`)

**PaymentRequest:**
```go
type PaymentRequest struct {
    CorrelationID string  `json:"correlationId" binding:"required"`
    Amount        float64 `json:"amount" binding:"required,gt=0"`
}
```

**PaymentRecord:**
```go
type PaymentRecord struct {
    ID            uuid.UUID `json:"id"`
    CorrelationID string    `json:"correlationId"`
    Amount        float64   `json:"amount"`
    Processor     string    `json:"processor"`
    RequestedAt   time.Time `json:"requestedAt"`
    ProcessedAt   time.Time `json:"processedAt"`
}
```

**PaymentSummary:**
```go
type PaymentSummary struct {
    Default  ProcessorSummary `json:"default"`
    Fallback ProcessorSummary `json:"fallback"`
}

type ProcessorSummary struct {
    TotalRequests int     `json:"totalRequests"`
    TotalAmount   float64 `json:"totalAmount"`
}
```

## Business Logic Layer

### Payment Service (`internal/services/payment_service.go`)

**Core Methods:**
- `ProcessPayment`: Main payment processing with smart routing
- `GetPaymentsSummary`: Aggregated payment statistics with time filtering
- `processWithProcessor`: Individual processor communication
- `isProcessorHealthy`: Health status checking
- `monitorProcessorHealth`: Background health monitoring

**Smart Routing Logic:**
```go
func (s *PaymentService) ProcessPayment(req *PaymentRequest) (*PaymentRecord, error) {
    // Check for duplicate payments
    if existingPayment := s.storage.GetPaymentByCorrelationID(req.CorrelationID); existingPayment != nil {
        return existingPayment, nil
    }

    // Create payment record
    record := &PaymentRecord{
        ID:            uuid.New(),
        CorrelationID: req.CorrelationID,
        Amount:        req.Amount,
        RequestedAt:   time.Now(),
    }

    // Try default processor first (lower fee)
    if s.isProcessorHealthy("default") {
        if err := s.processWithProcessor(ctx, req, record, "default"); err == nil {
            return record, nil
        }
    }

    // Fallback to secondary processor (higher fee)
    if s.isProcessorHealthy("fallback") {
        if err := s.processWithProcessor(ctx, req, record, "fallback"); err == nil {
            return record, nil
        }
    }

    // Both processors failed
    record.Processor = "failed"
    s.storage.StorePayment(record)
    return record, fmt.Errorf("both payment processors are unavailable")
}
```

**Health Monitoring:**
- Background goroutine checks processor health every 5 seconds
- Respects rate limits (1 call per 5 seconds per processor)
- Thread-safe health state management with RWMutex
- Automatic recovery when processors come back online

### Storage Layer (`internal/storage/storage.go`)

**Interface:**
```go
type Storage interface {
    StorePayment(payment *PaymentRecord)
    GetPaymentByID(id uuid.UUID) *PaymentRecord
    GetPaymentByCorrelationID(correlationID string) *PaymentRecord
    GetPaymentsSummary(from, to *time.Time) *PaymentSummary
}
```

**In-Memory Implementation:**
```go
type InMemoryStorage struct {
    mu       sync.RWMutex
    payments map[string]*PaymentRecord  // By correlationId
    byID     map[uuid.UUID]*PaymentRecord // By payment ID
}
```

**Key Features:**
- Dual indexing for O(1) lookups by correlationId and UUID
- Thread-safe concurrent access with RWMutex
- Time-based filtering for payment summaries
- Duplicate prevention mechanism

## HTTP Layer

### Request Handlers (`internal/handlers/payment_handler.go`)

**PaymentHandler Structure:**
```go
type PaymentHandler struct {
    service *PaymentService
}
```

**ProcessPayment Handler:**
- Validates request structure using Gin binding
- Processes payment through service layer
- Returns appropriate HTTP status codes
- Includes OpenTelemetry span creation

**GetPaymentsSummary Handler:**
- Parses optional time filter query parameters
- Supports ISO 8601 timestamp format
- Returns aggregated payment statistics
- Handles time parsing errors gracefully

### Middleware (`internal/middleware/middleware.go`)

**Available Middleware:**
- **Recovery**: Panic recovery with error logging
- **Logger**: Request/response logging with timing
- **CORS**: Cross-origin resource sharing headers
- **OpenTelemetry**: Automatic HTTP instrumentation

## Observability Implementation

### OpenTelemetry Tracing (`internal/tracing/tracer.go`)

**Tracer Setup:**
```go
func InitTracer() *trace.TracerProvider {
    exporter, _ := jaeger.New(jaeger.WithCollectorEndpoint())
    
    tp := trace.NewTracerProvider(
        trace.WithBatcher(exporter),
        trace.WithResource(resource.NewWithAttributes(
            semconv.ServiceName("rinha-backend"),
            semconv.ServiceVersion("1.0.0"),
        )),
    )
    
    otel.SetTracerProvider(tp)
    return tp
}
```

**Span Attributes:**
- `payment.correlation_id`: Request correlation ID
- `payment.amount`: Payment amount
- `payment.processor`: Used processor (default/fallback/failed)
- `http.status_code`: Response status code
- Error details and stack traces

### Structured Logging

**Configuration:**
```go
logrus.SetFormatter(&logrus.JSONFormatter{})
logrus.SetLevel(logrus.InfoLevel)
```

**Log Context:**
- Request IDs for tracing
- Payment correlation IDs
- Processing timing information
- Error details and stack traces

## Docker Implementation

### Multi-Stage Dockerfile
```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Runtime stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/
COPY --from=builder /app/main .
CMD ["./main"]
```

**Features:**
- Multi-stage build for minimal image size
- Static binary compilation (CGO_ENABLED=0)
- Security-focused base image (Alpine)
- Timezone data for proper time handling

### Docker Compose Configuration

**Services Architecture:**
```yaml
services:
  jaeger:     # OpenTelemetry backend
  nginx:      # Load balancer (0.3 CPU, 50MB)
  app1:       # Backend instance 1 (0.6 CPU, 150MB)
  app2:       # Backend instance 2 (0.6 CPU, 150MB)
```

**Network Configuration:**
- **backend**: Internal service communication
- **payment-processors_payment-processor**: External processor integration
- Bridge networking (no host mode for security)

### Nginx Load Balancer Configuration

```conf
upstream backend {
    ip_hash;                    # Session affinity for in-memory storage
    server app1:8080;
    server app2:8080;
}

server {
    listen 80;
    location / {
        proxy_pass http://backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_connect_timeout 30s;
        proxy_send_timeout 30s;
        proxy_read_timeout 30s;
    }
}
```

## Testing Infrastructure

### Test Scripts Implementation

**init.sh Features:**
- Docker and Docker Compose availability verification
- Payment processor startup with health verification
- Backend service deployment with readiness checks
- Integration test execution

**test_payments.sh Coverage:**
- Basic payment processing functionality
- Concurrent payment handling
- Input validation testing
- Payment summary verification
- Time-based filtering validation
- Performance measurement

**stress_test.sh Capabilities:**
- Configurable concurrent users and request counts
- Response time percentile calculation (P50, P95, P99)
- Performance bonus calculation (p99 < 11ms target)
- Error rate monitoring and reporting
- Throughput measurement

## Performance Optimizations

### Memory Efficiency
- **Dual Indexing**: Minimal memory overhead for fast lookups
- **Struct Packing**: Optimized data structure layouts
- **Connection Reuse**: HTTP client connection pooling
- **Goroutine Management**: Limited concurrent goroutines

### CPU Efficiency
- **RWMutex Usage**: Optimized for read-heavy workloads
- **Background Processing**: Health checks don't block requests
- **Efficient JSON**: Minimal serialization/deserialization
- **HTTP Keep-Alive**: Persistent connections to processors

### Response Time Targets
- **P99 < 11ms**: Primary performance goal
- **In-Memory Storage**: Sub-millisecond data access
- **Optimized Routing**: Minimal processing overhead
- **Background Health Checks**: Non-blocking health verification

## Integration Points for Future Enhancements

### Database Integration Ready
- **Storage Interface**: Abstract storage layer implemented
- **Transaction Support**: Models designed for ACID compliance
- **Connection Management**: Configuration structure ready
- **Migration Support**: Model versioning considerations

### Redis Cache Integration Ready
- **Health State Sharing**: Distributed health monitoring
- **Session Storage**: Shared session data across instances
- **Payment Caching**: Fast payment lookup cache
- **Pub/Sub Events**: Real-time status updates

### Monitoring Integration Ready
- **Metrics Collection**: Prometheus-compatible metrics
- **Health Endpoints**: Standard health check format
- **Alert Integration**: Structured error reporting
- **Performance Monitoring**: Detailed timing and throughput metrics