# Quick Start Guide - HYBRID STORAGE ✅

## Prerequisites
- Docker and Docker Compose installed
- Payment processors project available (separate repository)

## 30-Second Setup (With Database & Cache)

```bash
# Clone and initialize everything (now includes PostgreSQL + Redis)
./scripts/init.sh

# Test the payment API
curl -X POST http://localhost:9999/payments \
  -H "Content-Type: application/json" \
  -d '{"correlationId":"quick-test","amount":100.00}'

# Check payment summary (cached in Redis)
curl http://localhost:9999/payments-summary
```

## What Just Happened?

The `init.sh` script automatically:
1. ✅ Started PostgreSQL database with automatic schema creation
2. ✅ Started Redis cache and streaming server
3. ✅ Started payment processors (default + fallback)
4. ✅ Deployed backend with optimized load balancer
5. ✅ Verified all services are healthy with dependencies
6. ✅ Ready to process payments with persistence + caching

## Service Status
```bash
# Check all services are running
cd deployments && docker compose ps

# Expected output:
# postgres   Up (healthy)   5432/tcp
# redis      Up (healthy)   6379/tcp
# nginx      Up             0.0.0.0:9999->80/tcp
# app1       Up             8080/tcp
# app2       Up             8080/tcp
# jaeger     Up             16686/tcp, 14268/tcp
```

## Available Endpoints
- **Backend API**: http://localhost:9999 (Load balanced, 2 instances)
- **Payment Summary**: http://localhost:9999/payments-summary (Redis cached)
- **Health Check**: http://localhost:9999/health (New endpoint)
- **Default Processor**: http://localhost:8001 (1% fee)
- **Fallback Processor**: http://localhost:8002 (5% fee)
- **Jaeger UI**: http://localhost:16686
- **PostgreSQL**: localhost:5432 (postgres/password123)
- **Redis**: localhost:6379 (no password)

## Test Payment Processing
```bash
# Process a payment
curl -X POST http://localhost:9999/payments \
  -H "Content-Type: application/json" \
  -d '{"correlationId":"test-001","amount":250.50}'

# Expected response:
{
  "id": "uuid-here",
  "correlationId": "test-001",
  "amount": 250.50,
  "processor": "default",
  "requestedAt": "2025-01-15T12:34:56.000Z",
  "processedAt": "2025-01-15T12:34:56.123Z"
}
```

## Test Summary Endpoint
```bash
# Get payment summary
curl http://localhost:9999/payments-summary

# Expected response:
{
  "default": {
    "totalRequests": 1,
    "totalAmount": 250.50
  },
  "fallback": {
    "totalRequests": 0,
    "totalAmount": 0
  }
}
```

## Run Tests
```bash
# Quick API tests
./scripts/test_payments.sh

# Integration tests  
./scripts/test_processors.sh

# Performance tests (p99 < 11ms target)
./scripts/stress_test.sh 10 50

# Full test suite
make test
```

## Clean Up
```bash
# Stop all services and clean up
./scripts/cleanup.sh
```

## What's Running?

### Architecture
```
Client → Nginx → App1/App2 → PostgreSQL + Redis
                    ↓
               Payment Processors
```

### Resource Usage (Rinha Compliant)
- **Total**: 1.5 CPU, 350MB memory exactly
- **postgres**: 0.3 CPU, 80MB
- **redis**: 0.2 CPU, 60MB
- **nginx**: 0.1 CPU, 30MB
- **app1**: 0.45 CPU, 90MB  
- **app2**: 0.45 CPU, 90MB

### Smart Routing + Storage
1. **Routing**: Default processor first (1% fee) → Fallback (5% fee)
2. **Storage**: Write to PostgreSQL + cache in Redis 
3. **Reads**: Try Redis cache first → fallback to PostgreSQL
4. **Streaming**: Real-time events via Redis Streams
5. **Health**: Background monitoring every 5 seconds
6. **Fallback**: PostgreSQL-only → In-memory if needed

## Advanced Features

### Database Inspection
```bash
# Connect to PostgreSQL
docker exec -it deployments_postgres_1 psql -U postgres -d payments

# View payments table
SELECT * FROM payments ORDER BY processed_at DESC LIMIT 10;

# View table structure
\d payments
```

### Redis Inspection  
```bash
# Connect to Redis
docker exec -it deployments_redis_1 redis-cli

# View cached payments
KEYS payment:*

# View payment events stream
XREAD STREAMS payment_events 0
```

### Performance Testing
```bash
# Run stress test with custom parameters
./scripts/stress_test.sh 50 100  # 50 users, 100 requests each

# Monitor with htop during stress test
htop

# View Redis metrics
docker exec -it deployments_redis_1 redis-cli INFO memory
```

## Next Steps
- Explore the [API endpoints](ENDPOINTS.md)
- Run comprehensive [tests](TESTING.md)  
- Understand the [hybrid architecture](ARCHITECTURE.md)
- Review [database integration](DATABASE_INTEGRATION.md)
- Check [deployment options](DEPLOYMENT.md)