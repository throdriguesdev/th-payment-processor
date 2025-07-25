# Quick Start Guide

## Prerequisites
- Docker and Docker Compose installed
- Payment processors project available (separate repository)

## 30-Second Setup

```bash
# Clone and initialize everything
./init.sh

# Test the payment API
curl -X POST http://localhost:9999/payments \
  -H "Content-Type: application/json" \
  -d '{"correlationId":"quick-test","amount":100.00}'

# Check payment summary
curl http://localhost:9999/payments-summary
```

## What Just Happened?

The `init.sh` script automatically:
1. ✅ Started payment processors (default + fallback)
2. ✅ Deployed backend with load balancer
3. ✅ Verified all services are healthy
4. ✅ Ready to process payments

## Service Status
```bash
# Check all services are running
docker-compose ps

# Expected output:
# nginx    Up      0.0.0.0:9999->80/tcp
# app1     Up      8080/tcp
# app2     Up      8080/tcp
# jaeger   Up      16686/tcp, 14268/tcp
```

## Available Endpoints
- **Backend API**: http://localhost:9999
- **Payment Summary**: http://localhost:9999/payments-summary
- **Default Processor**: http://localhost:8001 (1% fee)
- **Fallback Processor**: http://localhost:8002 (5% fee)
- **Jaeger UI**: http://localhost:16686

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
./test_payments.sh

# Integration tests  
./test_processors.sh

# Performance tests (p99 < 11ms target)
./stress_test.sh 10 50
```

## Clean Up
```bash
# Stop all services and clean up
./cleanup.sh
```

## What's Running?

### Architecture
```
Client → Nginx Load Balancer → App1/App2 → Payment Processors
```

### Resource Usage
- **Total**: 1.5 CPU, 350MB memory (competition requirement)
- **nginx**: 0.3 CPU, 50MB
- **app1**: 0.6 CPU, 150MB  
- **app2**: 0.6 CPU, 150MB

### Smart Routing
1. Always tries default processor first (1% fee)
2. Falls back to fallback processor (5% fee) if needed
3. Background health monitoring every 5 seconds
4. Complete audit trail for all payments

## Next Steps
- Explore the [API endpoints](ENDPOINTS.md)
- Run comprehensive [tests](TESTING.md)
- Understand the [architecture](ARCHITECTURE.md)
- Review [deployment options](DEPLOYMENT.md)