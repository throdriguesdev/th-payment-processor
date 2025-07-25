# Technical Architecture

## High-Level Design

The rinha-backend is a Go-based payment processing intermediary service designed for high performance and reliability. It implements intelligent payment routing with automatic failover capabilities.

### Core Components

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Client        │───▶│   Nginx         │───▶│   App 1         │
│                 │    │ Load Balancer   │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
                                └─────────▶┌─────────────────┐
                                           │   App 2         │
                                           │                 │
                                           └─────────────────┘
```

## Project Structure

```
rinha-backend/
├── main.go                    # Application entry point
├── internal/                  # Internal application packages
│   ├── config/               # Configuration management
│   ├── handlers/             # HTTP request handlers
│   ├── middleware/           # HTTP middleware (logging, CORS)
│   ├── models/               # Data structures and models
│   ├── services/             # Business logic layer
│   ├── storage/              # In-memory storage implementation
│   └── tracing/              # OpenTelemetry tracing setup
├── docker-compose.yml        # Container orchestration
├── Dockerfile                # Multi-stage container build
├── nginx.conf                # Load balancer configuration
└── docs/                     # Documentation
```

## Smart Payment Routing

### Processing Logic
1. **Health Check**: Verify default processor availability
2. **Primary Route**: Attempt payment with default processor (1% fee)
3. **Failover**: If default fails, try fallback processor (5% fee)
4. **Audit Trail**: Record all attempts for consistency verification

### Health Monitoring
- **Background Monitoring**: Continuous health checks every 5 seconds
- **Rate Limiting**: Respects processor limits (1 call/5s per processor)
- **Thread Safety**: Concurrent health state management
- **Automatic Recovery**: Processors automatically come back online

## Data Storage

### In-Memory Storage Design
```go
type InMemoryStorage struct {
    mu       sync.RWMutex
    payments map[string]*PaymentRecord  // By correlationId
    byID     map[uuid.UUID]*PaymentRecord // By payment ID
}
```

**Features:**
- **Dual Indexing**: Fast lookups by correlationId and UUID
- **Thread Safety**: RWMutex for concurrent operations
- **Duplicate Prevention**: Checks existing payments
- **Audit Trail**: Stores successful and failed payments

## Performance Optimizations

### Target: P99 < 11ms
- **In-Memory Storage**: O(1) lookups for speed
- **HTTP Client Reuse**: Persistent connections to processors
- **Background Health Checks**: Non-blocking health monitoring
- **Efficient JSON Processing**: Minimal serialization overhead

### Resource Constraints (Competition Requirements)
- **Total CPU**: 1.5 cores (0.3 nginx + 0.6 app1 + 0.6 app2)
- **Total Memory**: 350MB (50MB nginx + 150MB app1 + 150MB app2)
- **Networking**: Bridge mode only (no host networking)

## Load Balancing Strategy

### Nginx Configuration
- **Algorithm**: IP Hash for session affinity
- **Instances**: Two backend app instances
- **Timeouts**: 30s connect/send/read timeouts
- **Health**: Automatic upstream health detection

### Session Affinity Reasoning
With in-memory storage, session affinity ensures consistent data access across requests from the same client.

## Observability

### OpenTelemetry Tracing
- **Backend**: Jaeger integration
- **Spans**: Payment processing, HTTP requests, processor calls
- **Attributes**: CorrelationId, amount, processor, errors
- **Performance**: Detailed timing analysis

### Structured Logging
- **Format**: JSON via logrus
- **Context**: Payment correlationIds for request tracking
- **Levels**: Info for success, Error for failures
- **Performance**: Request timing and status

## Security Considerations

### Network Security
- **Bridge Networking**: No host mode usage
- **Internal Networks**: Services communicate via Docker networks
- **Port Exposure**: Only port 9999 exposed publicly
- **External Integration**: Connects to payment-processor network

### Input Validation
- **Request Validation**: Required fields and positive amounts
- **Duplicate Prevention**: CorrelationId uniqueness enforcement
- **Error Handling**: Graceful failure with proper error responses

## Integration Architecture

### Payment Processor Integration
- **Default Processor**: Primary choice (1% fee, port 8001)
- **Fallback Processor**: Secondary choice (5% fee, port 8002)
- **Health Endpoints**: `/payments/service-health` monitoring
- **Admin Endpoints**: Configuration and testing capabilities

### Docker Networking
- **Internal Network**: `backend` for app communication
- **External Network**: `payment-processors_payment-processor`
- **Service Discovery**: Static hostname resolution
- **Container Communication**: Standard Docker DNS resolution

## Design Patterns

### Clean Architecture
- **Handlers**: HTTP layer (request/response)
- **Services**: Business logic and external integration
- **Storage**: Data persistence abstraction
- **Models**: Domain entities and data transfer objects

### Dependency Injection
- **Configuration**: Environment-based configuration
- **Service Dependencies**: Clear dependency graph
- **Interface-Based**: Storage and service abstractions

### Circuit Breaker (Implicit)
- **Health Monitoring**: Continuous processor assessment
- **Automatic Failover**: Route based on health status
- **Graceful Degradation**: Record failed payments for audit

## Scalability Considerations

### Current Limitations (By Design)
- **In-Memory Storage**: Limited to single instance memory
- **Session Affinity**: Required for data consistency
- **No Persistence**: Data lost on restart

### Future Database Integration Points
- **Storage Interface**: Ready for database implementation
- **Payment Models**: Database-ready structures
- **Transaction Support**: Designed for ACID compliance
- **Connection Pooling**: Configuration ready for DB connections

### Redis Cache Integration Points
- **Health State**: Redis for shared health monitoring
- **Session Data**: Distributed session storage
- **Payment Cache**: Fast payment lookup cache
- **Pub/Sub**: Real-time health status updates