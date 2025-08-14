# Payment Processor Monitoring Setup

## Overview
Complete service performance monitoring with RED metrics (Rate, Errors, Duration) using OpenTelemetry, Prometheus, and Grafana.

## Architecture

### Services
- **Main Payment Service** - Handles payment requests with load balancing
- **Payment Processors** - Default (1% fee) and Fallback (5% fee) processors  
- **Prometheus** - Metrics collection and storage
- **Grafana** - Visualization dashboards
- **Jaeger** - Distributed tracing (existing)
- **PostgreSQL** - Payment data storage
- **Redis** - Caching layer

### Metrics Endpoints
- Main App 1: `http://localhost:2113/metrics`
- Main App 2: `http://localhost:2114/metrics`  
- Payment Processor Default: `http://localhost:2115/metrics`
- Payment Processor Fallback: `http://localhost:2116/metrics`

## RED Metrics Collected

### Rate (Requests per Second)
- `http_requests_total` - Total HTTP requests by method, path, status
- Rate calculation: `rate(http_requests_total[5m])`

### Errors (Error Rate)  
- `http_errors_total` - HTTP errors by type (4xx, 5xx)
- Error rate: `rate(http_errors_total[5m]) / rate(http_requests_total[5m]) * 100`

### Duration (Response Time)
- `http_request_duration_seconds` - Request latency histogram
- Percentiles: `histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))`

### Additional Metrics
- `http_active_connections` - Current active HTTP connections
- `payment_amount_dollars` - Payment amounts processed by processor and status

## Dashboard Access

### URLs
- **Grafana**: http://localhost:3000 (admin/admin123)
- **Prometheus**: http://localhost:9090  
- **Jaeger**: http://localhost:16686
- **Payment Service**: http://localhost:9999

### Grafana Dashboard
Pre-configured dashboard: "Payment Processor - Service Performance Monitoring"

#### Panels:
1. **Request Rate** - Requests per second by endpoint
2. **Error Rate** - Error percentage over time  
3. **Response Duration** - 95th and 50th percentile latencies
4. **Active Connections** - Real-time connection count
5. **Payment Amounts** - Payment value distributions
6. **Success vs Failure Rate** - Payment processing outcomes

## Usage Instructions

### 1. Start All Services
```bash
cd deployments
docker-compose up -d
```

### 2. Test the Setup
```bash
# Run comprehensive test
./test-monitoring.sh

# Or test locally first  
./test-local-metrics.sh
```

### 3. Generate Sample Traffic
```bash
# Single payment
curl -X POST http://localhost:9999/payments \
  -H 'Content-Type: application/json' \
  -d '{"correlationId": "test-123", "amount": 100.50}'

# Load test
./scripts/stress_test.sh
```

### 4. View Metrics
- Open Grafana at http://localhost:3000
- Login with admin/admin123  
- Navigate to the Payment Processor dashboard
- Observe real-time metrics as traffic flows

## Troubleshooting

### Metrics Not Appearing
1. Check service logs: `docker-compose logs <service_name>`
2. Verify metrics endpoints: `curl http://localhost:2113/metrics`
3. Check Prometheus targets: http://localhost:9090/targets
4. Ensure all services are healthy: `docker-compose ps`

### Connection Refused Errors
- Services may still be starting up (wait 30-60 seconds)
- Check port conflicts: `netstat -tulpn | grep :2112`
- Rebuild images: `docker-compose build --no-cache`

### No Data in Grafana
1. Verify Prometheus is collecting data: http://localhost:9090
2. Check datasource configuration in Grafana
3. Generate traffic to populate metrics
4. Refresh dashboard and check time range

## Files Modified/Added

### Core Monitoring
- `internal/tracing/tracer.go` - Added OpenTelemetry metrics
- `internal/metrics/metrics.go` - RED metrics collection
- `internal/middleware/metrics_middleware.go` - HTTP metrics middleware
- `internal/handlers/payment_handler.go` - Payment metrics integration

### Configuration  
- `deployments/docker-compose.yml` - Added Prometheus, Grafana, metrics ports
- `deployments/prometheus.yml` - Prometheus scraping configuration
- `deployments/grafana/` - Grafana datasources and dashboard config

### Testing
- `test-local-metrics.sh` - Local testing script
- `test-metrics.sh` - Docker testing script  
- `deployments/test-monitoring.sh` - Comprehensive test

### Dependencies
- Updated `go.mod` with Prometheus and OpenTelemetry metrics libraries

## Monitoring Best Practices

### Alerting (Future Enhancement)
- Set up alerts for error rates > 5%
- Alert on 95th percentile latency > 1s
- Monitor payment processing failures
- Track service availability

### Performance Optimization
- Use metrics to identify bottlenecks
- Monitor resource utilization
- Track payment processor performance
- Optimize based on RED metrics trends

### Maintenance
- Regular Prometheus data cleanup
- Grafana dashboard updates
- Monitor disk usage for metrics storage
- Update metrics collection as features evolve

## Next Steps

1. **Deploy to Production**: Configure external Prometheus/Grafana
2. **Add Alerting**: Set up AlertManager for critical metrics
3. **Custom Dashboards**: Create business-specific monitoring views  
4. **Log Correlation**: Integrate logs with traces and metrics
5. **SLI/SLO Definition**: Establish service level objectives based on RED metrics