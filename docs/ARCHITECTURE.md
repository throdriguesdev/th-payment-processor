# Technical Architecture - UPDATED ✅

## High-Level Design

The TH Payment Processor is a production-ready Go-based payment processing intermediary service featuring hybrid storage architecture (PostgreSQL + Redis), intelligent payment routing, and high-performance optimizations for handling virtual user loads.

### Core Components

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Client        │───▶│   Nginx         │───▶│   App 1         │
│                 │    │ Load Balancer   │    │ (Go + Hybrid    │
└─────────────────┘    └─────────────────┘    │  Storage)       │
                                │              └─────────────────┘
                                │                      │
                                └─────────▶┌─────────────────┐  │
                                           │   App 2         │  │
                                           │ (Go + Hybrid    │  │
                                           │  Storage)       │  │
                                           └─────────────────┘  │
                                                     │          │
                            ┌────────────────────────┴──────────┘
                            │
                            ▼
              ┌─────────────────┐              ┌─────────────────┐
              │   PostgreSQL    │              │     Redis       │
              │   (Persistence) │◀────────────▶│   (Cache +      │
              │                 │              │    Streams)     │
              └─────────────────┘              └─────────────────┘
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

## Data Storage - HYBRID ARCHITECTURE ✅

### Three-Tier Storage Strategy
```go
// Primary: Hybrid Storage (PostgreSQL + Redis)
type HybridStorage struct {
    postgres *PostgresStorage
    redis    *RedisCache
}

// Fallback 1: PostgreSQL Only
type PostgresStorage struct {
    db *sql.DB
}

// Fallback 2: In-Memory (Emergency)
type InMemoryStorage struct {
    mu       sync.RWMutex
    payments map[string]*PaymentRecord
}
```

**Storage Features:**
- **Write-Through Cache**: PostgreSQL + Redis caching
- **Read Optimization**: Redis-first, PostgreSQL fallback
- **Real-time Streams**: Redis Streams for event publishing  
- **Graceful Degradation**: Automatic fallback on failures
- **ACID Compliance**: PostgreSQL for data integrity
- **Sub-ms Lookups**: Redis cache performance

## Performance Optimizations - ENHANCED ✅

### Target: P99 < 11ms (Achieved through multiple optimizations)

#### 1. Storage Performance
- **Redis Caching**: Sub-millisecond lookups from memory
- **Database Indexing**: Optimized PostgreSQL queries
- **Connection Pooling**: PostgreSQL (25 max) + Redis (10 pool size)
- **Write-Through Strategy**: Fast writes with immediate caching

#### 2. Application Performance  
- **HTTP Connection Pooling**: 100 max connections per host
- **Object Pooling**: Request object reuse (reduces GC pressure)
- **Optimized Timeouts**: 5s connect, 10s read/write
- **Background Health Checks**: Non-blocking monitoring

#### 3. Load Balancer Performance
- **Least Connections**: Better distribution than ip_hash
- **Connection Keepalive**: Persistent upstream connections  
- **Gzip Compression**: Reduced response size
- **Optimized Buffers**: 8x4k proxy buffers

### Resource Constraints (Rinha Requirements) ✅
- **Total CPU**: 1.5 cores exactly distributed:
  - nginx: 0.1 CPU + 30MB
  - app1: 0.45 CPU + 90MB  
  - app2: 0.45 CPU + 90MB
  - postgres: 0.3 CPU + 80MB
  - redis: 0.2 CPU + 60MB
- **Total Memory**: 350MB exactly
- **Networking**: Bridge mode only (no host networking)

## Load Balancing Strategy - OPTIMIZED ✅

### Nginx Configuration  
- **Algorithm**: Least Connections (no session affinity needed)
- **Instances**: Two backend app instances with health checks
- **Timeouts**: Optimized for performance (5s connect, 10s read/write)
- **Keepalive**: 32 upstream connections maintained
- **Health**: max_fails=3, fail_timeout=30s per upstream

### No Session Affinity Required
With PostgreSQL + Redis shared storage, any app instance can handle any request, enabling optimal load distribution and better fault tolerance.

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

## Scalability Achieved ✅

### Production-Ready Features
- **Persistent Storage**: PostgreSQL with ACID compliance
- **Horizontal Scaling**: No session affinity required
- **Shared State**: Redis for distributed caching and health monitoring  
- **Data Durability**: Survives container restarts and failures

### Real-time Capabilities
- **Event Streaming**: Redis Streams for payment events
- **Consumer Groups**: Support for microservices integration
- **Health Broadcasting**: Real-time processor status updates
- **Cache Invalidation**: Efficient cache management

### Monitoring and Observability
- **Storage Health**: Monitoring for PostgreSQL and Redis
- **Performance Metrics**: Database connection pools, cache hit rates
- **Graceful Degradation**: Automatic fallback with logging
- **Tracing Integration**: Full request lifecycle tracking

### Future Enhancements Ready
- **Read Replicas**: PostgreSQL read scaling
- **Redis Clustering**: Redis horizontal scaling
- **Multi-region**: Database replication support
- **Microservices**: Event-driven architecture via Redis Streams