# Setup & Deployment Guide

Complete installation and configuration guide for the TH Payment Processor with full observability stack.

## Prerequisites

- **Docker**: 20.10+ with Docker Compose v2
- **System Requirements**: 2GB RAM, 2 CPU cores minimum
- **Network**: Ports 3000, 8001-8002, 8081-8082, 9090, 9999 available

## Quick Setup

### 1. Clone and Initialize
```bash
git clone <repository>
cd th_payment_processor

# Start all services with one command
./scripts/init.sh
```

### 2. Verify Installation
```bash
# Check all services are running
docker compose -f deployments/docker-compose.yml ps

# Test payment processing
curl -X POST http://localhost:9999/payments \
  -H "Content-Type: application/json" \
  -d '{"correlationId":"setup-test","amount":100.50,"userId":"test-user"}'

# Verify observability stack
curl http://localhost:3000/api/health  # Grafana
curl http://localhost:9090/-/ready     # Prometheus
curl http://localhost:3100/ready       # Loki
```

### 3. Access Dashboards
- **Grafana**: http://localhost:3000 (admin/admin123)
- **Prometheus**: http://localhost:9090
- **Payment API**: http://localhost:9999

## Detailed Installation

### Manual Setup Steps

1. **Build Application Images**
```bash
# Build main application
docker build -f build/Dockerfile -t th-payment-processor .

# Build payment processors
cd payment-processors
docker build -t payment-processor .
cd ..
```

2. **Start Infrastructure Services**
```bash
cd deployments

# Start databases first
docker compose up -d postgres redis

# Wait for databases to be ready
docker compose up -d --wait postgres redis

# Start observability stack
docker compose up -d prometheus loki tempo grafana promtail otel-collector

# Start application services
docker compose up -d app1 app2 payment-processor-default payment-processor-fallback nginx
```

3. **Verify Each Component**
```bash
# Database connectivity
docker compose exec postgres psql -U postgres -d payments -c "SELECT 1;"

# Redis connectivity  
docker compose exec redis redis-cli ping

# Application health
curl http://localhost:8081/health
curl http://localhost:8082/health

# Payment processors
curl http://localhost:8001/payments/service-health
curl http://localhost:8002/payments/service-health
```

## Configuration

### Environment Variables

#### Application Configuration
```bash
# Server settings
SERVER_PORT=8080
LOG_LEVEL=info

# Database connections
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=password123
POSTGRES_DB=payments
POSTGRES_SSLMODE=disable
POSTGRES_MAX_CONNECTIONS=200

# Redis configuration
REDIS_ADDR=redis:6379
REDIS_PASSWORD=
REDIS_DB=0

# Payment processor URLs
DEFAULT_PROCESSOR_URL=http://payment-processor-default:8080
FALLBACK_PROCESSOR_URL=http://payment-processor-fallback:8080
```

#### Observability Configuration
```bash
# OpenTelemetry
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318
TEMPO_ENDPOINT=http://tempo:4318

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

### Service Configuration Files

#### Database Schema
The application automatically creates required tables:
```sql
CREATE TABLE payments (
    id SERIAL PRIMARY KEY,
    correlation_id VARCHAR(255) UNIQUE NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    amount DECIMAL(10,2) NOT NULL,
    processor VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_payments_correlation_id ON payments(correlation_id);
CREATE INDEX idx_payments_user_id ON payments(user_id);
CREATE INDEX idx_payments_created_at ON payments(created_at);
```

#### Nginx Load Balancer
```nginx
upstream backend {
    server app1:8080;
    server app2:8080;
}

server {
    listen 80;
    location / {
        proxy_pass http://backend;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

## Observability Setup

### Grafana Configuration

1. **Access Grafana**: http://localhost:3000
2. **Login**: admin/admin123
3. **Pre-configured Datasources**:
   - Prometheus (metrics)
   - Loki (logs) 
   - Tempo (traces)

### Key Grafana Queries

#### Loki (Logs)
```logql
# All payment service logs
{service_name="th-payment-processor"}

# Logs by correlation ID
{correlation_id="your-correlation-id"}

# Error logs only
{service_name="th-payment-processor"} |= "level":"error"

# Payment operations
{service_name="th-payment-processor"} | json | operation="payment_success"

# High latency operations
{service_name="th-payment-processor"} | json | latency_ms > 100
```

#### Prometheus (Metrics)
```promql
# Request rate
rate(http_requests_total[5m])

# Error rate
rate(http_requests_total{status=~"4..|5.."}[5m]) / rate(http_requests_total[5m])

# Response time percentiles
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# Payment success rate
rate(payments_total{status="success"}[5m]) / rate(payments_total[5m])
```

#### Tempo (Traces)
- Search by correlation_id
- Browse by service name
- Filter by operation name

## Scaling and Production

### Horizontal Scaling
```yaml
# Add more app instances
services:
  app3:
    build:
      context: ..
      dockerfile: build/Dockerfile
    ports:
      - "8083:8080"
    environment:
      - SERVER_PORT=8080
      # ... same as app1/app2

# Update nginx upstream
upstream backend {
    server app1:8080;
    server app2:8080;
    server app3:8080;
}
```

### Database Scaling
```bash
# PostgreSQL read replicas
POSTGRES_READ_HOST=postgres-read
POSTGRES_WRITE_HOST=postgres-write

# Redis clustering
REDIS_CLUSTER_NODES=redis1:6379,redis2:6379,redis3:6379
```

### Resource Limits
```yaml
services:
  app1:
    deploy:
      resources:
        limits:
          cpus: '0.75'
          memory: 175M
        reservations:
          cpus: '0.5'
          memory: 128M
```

## Troubleshooting

### Common Issues

#### Services Not Starting
```bash
# Check logs
docker compose logs <service-name>

# Check resource usage
docker stats

# Rebuild images
docker compose build --no-cache
```

#### Database Connection Issues
```bash
# Check database health
docker compose exec postgres pg_isready -U postgres

# Check connection from app
docker compose exec app1 pg_isready -h postgres -U postgres

# Reset database
docker compose down
docker volume rm deployments_postgres_data
docker compose up -d postgres
```

#### Observability Issues
```bash
# Check Prometheus targets
curl http://localhost:9090/api/v1/targets

# Check Loki ingestion
curl http://localhost:3100/metrics

# Check Promtail logs
docker compose logs promtail

# Test log correlation
./scripts/test-logging-correlation.sh
```

### Performance Tuning

#### Database Optimization
```sql
-- PostgreSQL configuration
SHOW shared_buffers;          -- Should be ~25% of RAM
SHOW effective_cache_size;    -- Should be ~75% of RAM
SHOW random_page_cost;        -- 1.1 for SSD

-- Query performance
EXPLAIN ANALYZE SELECT * FROM payments WHERE correlation_id = 'test';
```

#### Redis Optimization
```bash
# Redis memory usage
redis-cli INFO memory

# Connection pooling
redis-cli INFO clients

# Hit rate
redis-cli INFO stats | grep keyspace
```

## Security Considerations

### Network Security
```yaml
# Internal networks
networks:
  backend:
    driver: bridge
    internal: true
  frontend:
    driver: bridge
```

### Database Security
```bash
# Environment variables
POSTGRES_PASSWORD_FILE=/run/secrets/postgres_password
REDIS_PASSWORD_FILE=/run/secrets/redis_password

# Connection encryption
POSTGRES_SSLMODE=require
```

### Application Security
```go
// Rate limiting
rateLimiter := rate.NewLimiter(rate.Limit(100), 10)

// Input validation
validator := validator.New()
```

This setup provides a production-ready payment processing system with comprehensive observability and monitoring capabilities.