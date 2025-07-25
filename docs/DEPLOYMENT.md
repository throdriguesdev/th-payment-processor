# Deployment Guide

## Prerequisites
- Docker and Docker Compose
- Access to payment-processors project (separate repository)

## Quick Start (Automated)
```bash
# Initialize entire environment automatically
./init.sh

# Verify deployment
curl http://localhost:9999/payments-summary

# Clean up when done
./cleanup.sh
```

## Manual Deployment

### 1. Start Payment Processors (External Project)
```bash
cd ../payment-processors
docker-compose up -d
```

### 2. Start Backend Application
```bash
cd ../rinha-backend
docker-compose up -d
```

### 3. Verify Deployment
```bash
# Check service status
docker-compose ps

# Test backend connectivity
curl http://localhost:9999/payments-summary

# Test payment processing
curl -X POST http://localhost:9999/payments \
  -H "Content-Type: application/json" \
  -d '{"correlationId":"deploy-test","amount":50.00}'
```

## Architecture

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

## Service Configuration

### Backend Environment Variables
- `SERVER_PORT`: Backend service port (default: 8080)
- `DEFAULT_PROCESSOR_URL`: Default processor URL
- `FALLBACK_PROCESSOR_URL`: Fallback processor URL
- `HEALTH_CHECK_INTERVAL`: Health check frequency (default: 5s)
- `REQUEST_TIMEOUT`: Processor request timeout (default: 10s)

### Resource Limits (Enforced)
- **nginx**: 0.3 CPU, 50MB memory
- **app1**: 0.6 CPU, 150MB memory
- **app2**: 0.6 CPU, 150MB memory
- **Total**: 1.5 CPU, 350MB memory (exactly as required)

### Network Configuration
- **Backend Network**: Internal service communication
- **External Network**: `payment-processors_payment-processor`
- **Port 9999**: Public API access
- **Bridge Networking**: No host mode (security requirement)

## Monitoring

### Service Status
```bash
# View all services
docker-compose ps

# Monitor backend logs
docker-compose logs -f

# Monitor specific service
docker-compose logs -f app1
```

### Health Checks
```bash
# Backend health via payment summary
curl http://localhost:9999/payments-summary

# Default processor health
curl http://localhost:8001/payments/service-health

# Fallback processor health
curl http://localhost:8002/payments/service-health
```

### Performance Monitoring
- OpenTelemetry tracing available at Jaeger UI
- Structured JSON logs for debugging
- Background health monitoring every 5 seconds

## Service Endpoints
- **Backend API**: http://localhost:9999
- **Default Processor**: http://localhost:8001 (1% fee)
- **Fallback Processor**: http://localhost:8002 (5% fee)
- **Jaeger UI**: http://localhost:16686 (tracing)

## Production Considerations
- **Smart Routing**: Always tries default processor first (lower fee)
- **Automatic Failover**: Falls back when default processor fails
- **Session Affinity**: Nginx ip_hash for consistent routing
- **Performance Target**: P99 < 11ms response time
- **Audit Trail**: Complete payment record tracking