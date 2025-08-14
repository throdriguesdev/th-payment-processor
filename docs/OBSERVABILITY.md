# Observability Guide

Comprehensive guide to monitoring, logging, and tracing in the TH Payment Processor system.

## Overview

Our observability platform provides three pillars of insight:
- **📊 Metrics**: Quantitative performance data (Prometheus + Grafana)
- **📋 Logs**: Detailed event and error information (Loki + Promtail)
- **🔍 Traces**: Request flow across distributed services (Tempo + OpenTelemetry)

## Quick Start

### Access Dashboards
- **Grafana**: http://localhost:3000 (admin/admin123)
- **Prometheus**: http://localhost:9090
- **Direct Queries**: Use the endpoints below for programmatic access

### Generate Test Data
```bash
# Create correlated test data
./scripts/test-logging-correlation.sh

# Generate load for metrics
./scripts/stress_test.sh

# View real-time correlation
CORRELATION_ID="obs-test-$(date +%s)"
curl -X POST http://localhost:9999/payments \
  -H "X-Correlation-ID: $CORRELATION_ID" \
  -H "Content-Type: application/json" \
  -d "{\"correlationId\":\"$CORRELATION_ID\",\"amount\":100.50,\"userId\":\"test-user\"}"
```

## 📊 Metrics & Monitoring

### Prometheus Metrics Collected

#### HTTP Metrics
```promql
# Request rate by endpoint
rate(http_requests_total[5m])

# Error rate percentage
(rate(http_requests_total{status=~"4..|5.."}[5m]) / rate(http_requests_total[5m])) * 100

# Response time percentiles
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))

# Active connections
http_active_connections
```

#### Payment Business Metrics
```promql
# Payment success rate
rate(payments_total{status="success"}[5m]) / rate(payments_total[5m])

# Payment amounts by processor
sum(rate(payment_amount_dollars[5m])) by (processor)

# Payment processor health
up{job="payment-processors"}

# Database connection pool
db_connections_active
db_connections_idle
```

#### System Metrics
```promql
# CPU usage
rate(process_cpu_seconds_total[5m])

# Memory usage
process_resident_memory_bytes

# Go garbage collection
rate(go_gc_duration_seconds[5m])

# Goroutines
go_goroutines
```

### Key Performance Indicators (KPIs)

| Metric | Target | Alert Threshold |
|--------|--------|-----------------|
| Response Time (P95) | < 50ms | > 100ms |
| Response Time (P99) | < 100ms | > 200ms |
| Error Rate | < 0.1% | > 1% |
| Payment Success Rate | > 99.9% | < 99% |
| Processor Health | 100% | < 95% |
| Database Connections | < 80% of pool | > 90% of pool |

### Grafana Dashboards

#### Pre-configured Panels

1. **Service Overview**
   - Request rate trends
   - Error rate percentage
   - Response time percentiles
   - Service health status

2. **Payment Analytics**
   - Payment volume by processor
   - Success vs failure rates
   - Payment amount distributions
   - Geographic payment patterns (if enabled)

3. **System Health**
   - Resource utilization
   - Database performance
   - Cache hit rates
   - Connection pool status

4. **SLI/SLO Tracking**
   - Availability percentage
   - Performance targets
   - Error budget burn rate

#### Custom Dashboard Queries

```promql
# Payment processor performance comparison
sum(rate(payments_total{status="success"}[5m])) by (processor) /
sum(rate(payments_total[5m])) by (processor)

# Request latency heatmap
sum(rate(http_request_duration_seconds_bucket[5m])) by (le, service)

# Error rate by endpoint
sum(rate(http_requests_total{status=~"4..|5.."}[5m])) by (path) /
sum(rate(http_requests_total[5m])) by (path)
```

## 📋 Structured Logging

### Log Format and Structure

All logs use structured JSON format for machine readability:

```json
{
  "@timestamp": "2024-01-15T10:30:45.123Z",
  "level": "info",
  "message": "Payment processed successfully",
  "service_name": "th-payment-processor",
  "component": "payment_service",
  "operation": "payment_success",
  "correlation_id": "req-123e4567-e89b-12d3-a456-426614174000",
  "request_id": "87654321-abcd-1234-5678-123456789abc",
  "trace_id": "1234567890abcdef1234567890abcdef",
  "span_id": "abcdef0123456789",
  "payment_id": "pay_12345",
  "amount": 100.50,
  "processor": "default",
  "user_id": "user123",
  "http_method": "POST",
  "http_status": 200,
  "http_url": "/payments",
  "latency_ms": 15,
  "remote_addr": "192.168.1.100"
}
```

### Log Levels and Usage

| Level | Purpose | Examples |
|-------|---------|----------|
| **DEBUG** | Development troubleshooting | Variable values, detailed flow |
| **INFO** | Normal operations | Request/response, successful operations |
| **WARN** | Potential issues | Fallback processor used, high latency |
| **ERROR** | Errors requiring attention | Payment failures, database errors |
| **FATAL** | Critical system errors | Service cannot start, critical resource failure |

### Loki Queries

#### Basic Queries
```logql
# All payment service logs
{service_name="th-payment-processor"}

# Logs for specific correlation ID
{correlation_id="req-123e4567-e89b-12d3-a456-426614174000"}

# Error logs only
{service_name="th-payment-processor"} |= "level":"error"

# Logs from specific container
{container_name="app1"}
```

#### Advanced Queries
```logql
# Payment failures with details
{service_name="th-payment-processor"} 
| json 
| operation="payment_failure" 
| line_format "{{.timestamp}} [{{.level}}] Payment failed: {{.message}} (correlation_id={{.correlation_id}}, amount={{.amount}})"

# High latency requests
{service_name="th-payment-processor"} 
| json 
| latency_ms > 100 
| line_format "Slow request: {{.http_method}} {{.http_url}} took {{.latency_ms}}ms"

# Payment amounts by processor
sum by (processor) (
  sum_over_time(
    {service_name="th-payment-processor"} 
    | json 
    | operation="payment_success" 
    | unwrap amount [5m]
  )
)

# Error rate over time
sum(rate(
  {service_name="th-payment-processor"} 
  | json 
  | level="error" [5m]
)) by (component)
```

#### Log-based Alerts
```logql
# High error rate alert
sum(rate(
  {service_name="th-payment-processor"} 
  | json 
  | level="error" [5m]
)) > 0.1

# Payment processor failures
sum(rate(
  {service_name="th-payment-processor"} 
  | json 
  | operation="payment_failure" [5m]
)) > 0.01
```

## 🔍 Distributed Tracing

### OpenTelemetry Integration

#### Trace Structure
```
Trace: Payment Processing Request
├── Span: HTTP Request /payments
│   ├── Span: Request Validation
│   ├── Span: Database Query (user lookup)
│   ├── Span: Redis Cache Check
│   ├── Span: Payment Processor Call
│   │   ├── Span: HTTP Client Request
│   │   └── Span: Response Processing
│   ├── Span: Database Insert (payment record)
│   └── Span: Response Generation
```

#### Trace Attributes
```json
{
  "trace_id": "1234567890abcdef1234567890abcdef",
  "span_id": "abcdef0123456789",
  "parent_span_id": "fedcba9876543210",
  "operation_name": "payment_processing",
  "start_time": "2024-01-15T10:30:45.100Z",
  "end_time": "2024-01-15T10:30:45.115Z",
  "duration_ms": 15,
  "status": "ok",
  "attributes": {
    "http.method": "POST",
    "http.url": "/payments",
    "http.status_code": 200,
    "payment.correlation_id": "req-123e4567",
    "payment.amount": 100.50,
    "payment.processor": "default",
    "user.id": "user123",
    "service.name": "th-payment-processor",
    "service.version": "1.0.0"
  }
}
```

### Tempo Queries

#### Search Patterns
```
# By correlation ID
{.payment.correlation_id="req-123e4567-e89b-12d3-a456-426614174000"}

# By service and operation
{.service.name="th-payment-processor" && .operation="payment_processing"}

# By duration (slow requests)
{duration > 100ms}

# By status (errors)
{status=error}

# Complex queries
{.service.name="th-payment-processor" && .payment.amount > 1000 && duration > 50ms}
```

#### TraceQL Examples
```traceql
# Find traces with payment failures
{ .service.name = "th-payment-processor" && .status = "error" }

# Find high-value payments
{ .payment.amount > 1000 }

# Find slow database operations
{ .db.operation.name = "SELECT" && duration > 100ms }

# Find traces with specific user
{ .user.id = "user123" }
```

## Cross-Correlation

### Logs ↔ Traces
```logql
# Find logs for a specific trace
{service_name="th-payment-processor"} | json | trace_id="1234567890abcdef1234567890abcdef"
```

### Metrics ↔ Traces
Use correlation IDs to link metrics and traces:
```promql
# Metrics for traces with high latency
rate(http_request_duration_seconds_bucket{trace_id="1234567890abcdef1234567890abcdef"}[5m])
```

### Unified Dashboards
Grafana panels showing correlated data:
1. **Timeline View**: Metrics, logs, and traces on same time axis
2. **Correlation Drill-down**: Click metrics → view logs → explore traces
3. **Error Investigation**: Error metrics → error logs → failed traces

## Alerting Strategy

### Critical Alerts (PagerDuty/Immediate)
```yaml
# High error rate
alert: PaymentErrorRateHigh
expr: (rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m])) * 100 > 1
for: 2m
labels:
  severity: critical
annotations:
  summary: "Payment service error rate above 1%"

# Service down
alert: PaymentServiceDown
expr: up{job="payment-service"} == 0
for: 1m
labels:
  severity: critical
annotations:
  summary: "Payment service is down"
```

### Warning Alerts (Slack/Email)
```yaml
# High latency
alert: PaymentLatencyHigh
expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 0.1
for: 5m
labels:
  severity: warning
annotations:
  summary: "Payment service P95 latency above 100ms"

# Payment processor degraded
alert: PaymentProcessorDegraded
expr: rate(payments_total{status="failed", processor="default"}[5m]) / rate(payments_total{processor="default"}[5m]) > 0.05
for: 5m
labels:
  severity: warning
annotations:
  summary: "Default payment processor failure rate above 5%"
```

## Best Practices

### Correlation ID Strategy
```go
// Always propagate correlation IDs
func processPayment(ctx context.Context, req PaymentRequest) error {
    correlationID := middleware.GetCorrelationID(ctx)
    logger := middleware.GetStructuredLogger(ctx)
    
    // Start tracing span
    ctx, span := tracer.Start(ctx, "payment_processing")
    defer span.End()
    
    // Add correlation to span
    span.SetAttributes(attribute.String("payment.correlation_id", correlationID))
    
    // Log with context
    logger.WithOperation("payment_start").Info("Processing payment")
    
    return nil
}
```

### Structured Logging Guidelines
```go
// Good: Structured with context
logger.WithPaymentFields(logging.PaymentFields{
    CorrelationID: correlationID,
    Amount:        req.Amount,
    Processor:     "default",
    Status:        "success",
}).Info("Payment processed successfully")

// Bad: Unstructured
logger.Info("Payment processed: " + correlationID + " amount: " + fmt.Sprintf("%.2f", req.Amount))
```

### Performance Monitoring
```bash
# Continuous monitoring script
#!/bin/bash
while true; do
    # Check error rate
    ERROR_RATE=$(curl -s 'http://localhost:9090/api/v1/query?query=rate(http_requests_total{status=~"5.."}[5m])' | jq -r '.data.result[0].value[1]')
    
    if (( $(echo "$ERROR_RATE > 0.01" | bc -l) )); then
        echo "ALERT: Error rate too high: $ERROR_RATE"
    fi
    
    sleep 30
done
```

## Troubleshooting Guide

### Common Investigation Workflows

#### 1. High Error Rate Investigation
```bash
# 1. Check error rate metrics
curl 'http://localhost:9090/api/v1/query?query=rate(http_requests_total{status=~"5.."}[5m])'

# 2. Find error logs
# Loki query: {service_name="th-payment-processor"} |= "level":"error"

# 3. Get correlation IDs from error logs
# Extract correlation_ids from logs

# 4. Find traces for failing requests
# Tempo query: {.payment.correlation_id="extracted-correlation-id"}
```

#### 2. Performance Degradation Investigation
```bash
# 1. Check latency percentiles
curl 'http://localhost:9090/api/v1/query?query=histogram_quantile(0.95,rate(http_request_duration_seconds_bucket[5m]))'

# 2. Identify slow operations
# Loki query: {service_name="th-payment-processor"} | json | latency_ms > 100

# 3. Analyze slow traces
# Tempo query: {duration > 100ms}

# 4. Check resource metrics
curl 'http://localhost:9090/api/v1/query?query=process_resident_memory_bytes'
```

#### 3. Payment Processor Issues
```bash
# 1. Check processor health
curl http://localhost:9999/payments/service-health

# 2. Check processor-specific metrics
curl 'http://localhost:9090/api/v1/query?query=up{job="payment-processors"}'

# 3. Find processor error logs
# Loki query: {service_name="th-payment-processor"} | json | processor="default" | level="error"
```

This observability setup provides complete visibility into system behavior, enabling proactive monitoring and rapid issue resolution.