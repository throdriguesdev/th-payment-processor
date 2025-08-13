# Observability Testing Guide

## Overview
This guide provides step-by-step instructions to test the enhanced structured logging and distributed tracing features in your payment processor application.

## Prerequisites
- Payment processor application running
- Grafana accessible at http://localhost:3000
- Loki configured for log aggregation
- Tempo configured for distributed tracing
- Prometheus for metrics

## Test Scenarios

### Test 1: Basic Payment Processing Flow

#### Test Purpose
Verify structured logging captures complete payment flow with correlation IDs, trace context, and timing information.

#### Test Steps
```bash
# 1. Process a successful payment
curl -X POST http://localhost:8080/payments \
  -H "Content-Type: application/json" \
  -H "X-Correlation-ID: test-payment-001" \
  -d '{
    "correlation_id": "test-payment-001",
    "amount": 100.50
  }'

# Expected Response: 200 OK with correlation_id in response
```

#### What to Check in Grafana

**Loki Logs (Explore → Loki):**
```logql
{service_name="th-payment-processor"} |= "test-payment-001"
```

**Expected Log Entries:**
1. HTTP request received log
2. Payment processing started log  
3. Payment processor attempt logs
4. Payment success/failure log
5. HTTP response sent log

**Key Fields to Verify:**
- `correlation_id`: "test-payment-001"
- `trace_id`: Present and consistent across logs
- `span_id`: Different for each operation
- `amount`: 100.50
- `operation`: Various (payment_start, payment_success, etc.)
- `latency_ms`: Present in response logs

---

### Test 2: Payment Processor Failover

#### Test Purpose
Test logging during processor failover scenarios and health check monitoring.

#### Test Steps
```bash
# 1. Make multiple payments to trigger processor selection
for i in {1..5}; do
  curl -X POST http://localhost:8080/payments \
    -H "Content-Type: application/json" \
    -H "X-Correlation-ID: failover-test-$i" \
    -d '{
      "correlation_id": "failover-test-'$i'",
      "amount": 50.00
    }'
  sleep 1
done
```

#### What to Check in Grafana

**Query for Processor Selection:**
```logql
{service_name="th-payment-processor"} |= "failover-test" | json | component="payment_service"
```

**Expected Patterns:**
- Default processor attempts
- Fallback processor usage (if default fails)
- Health check logs for both processors
- Processor success/failure patterns

---

### Test 3: Error Handling and Edge Cases

#### Test Purpose
Verify error logging captures detailed context and maintains trace correlation.

#### Test Steps
```bash
# 1. Invalid request format
curl -X POST http://localhost:8080/payments \
  -H "Content-Type: application/json" \
  -H "X-Correlation-ID: error-test-001" \
  -d '{
    "invalid": "request"
  }'

# 2. Large payment amount (if validation exists)
curl -X POST http://localhost:8080/payments \
  -H "Content-Type: application/json" \
  -H "X-Correlation-ID: error-test-002" \
  -d '{
    "correlation_id": "error-test-002",
    "amount": 999999.99
  }'
```

#### What to Check in Grafana

**Query for Errors:**
```logql
{service_name="th-payment-processor"} |= "error-test" | json | level="error"
```

**Expected Error Logs:**
- Request binding errors
- Validation errors  
- Error context with correlation IDs
- Stack traces (in debug mode)

---

### Test 4: High-Volume Load Testing

#### Test Purpose
Test logging performance and correlation ID consistency under load.

#### Test Steps
```bash
# Create a load test script
cat > load_test.sh << 'EOF'
#!/bin/bash
for i in {1..50}; do
  (
    curl -X POST http://localhost:8080/payments \
      -H "Content-Type: application/json" \
      -H "X-Correlation-ID: load-test-$(date +%s)-$i" \
      -d '{
        "correlation_id": "load-test-'$(date +%s)'-'$i'",
        "amount": '$((RANDOM % 1000 + 10))'.50
      }' &
  )
  if (( $i % 10 == 0 )); then
    wait
    sleep 0.5
  fi
done
wait
EOF

chmod +x load_test.sh
./load_test.sh
```

#### What to Check in Grafana

**Query for Load Test:**
```logql
{service_name="th-payment-processor"} |= "load-test" | json
```

**Performance Analysis:**
- Log volume and indexing performance
- Latency distribution across requests
- Correlation ID uniqueness and consistency
- No missing trace context

---

### Test 5: Health Check Monitoring

#### Test Purpose
Verify health check logging and processor status monitoring.

#### Test Steps
```bash
# 1. Check application health
curl -X GET http://localhost:8080/health \
  -H "X-Correlation-ID: health-check-001"

# 2. Wait for background health checks to run
sleep 30

# 3. Check payments summary
curl -X GET http://localhost:8080/payments-summary \
  -H "X-Correlation-ID: summary-check-001"
```

#### What to Check in Grafana

**Query for Health Checks:**
```logql
{service_name="th-payment-processor"} | json | operation="health_check"
```

**Expected Health Logs:**
- Periodic health check attempts
- Processor response times
- Health status changes
- Rate limiting messages (if applicable)

---

## Grafana Dashboard Queries

### Log Analysis Queries

#### 1. Payment Success Rate Over Time
```logql
sum by (status) (count_over_time({service_name="th-payment-processor"} | json | operation="payment_success" or operation="payment_failure" [5m]))
```

#### 2. Average Payment Processing Latency
```logql
avg_over_time({service_name="th-payment-processor"} | json | operation="payment_success" | unwrap latency_ms [5m])
```

#### 3. Error Rate by Component
```logql
sum by (component) (count_over_time({service_name="th-payment-processor"} | json | level="error" [5m]))
```

#### 4. Top Correlation IDs by Request Volume
```logql
topk(10, count by (correlation_id) (count_over_time({service_name="th-payment-processor"} | json [1h])))
```

#### 5. Payment Amount Distribution
```logql
histogram_quantile(0.95, sum(rate({service_name="th-payment-processor"} | json | operation="payment_start" | unwrap amount [5m])) by (le))
```

### Trace Analysis (Tempo/Jaeger)

#### 1. Find Traces by Correlation ID
In Grafana Explore → Tempo:
- Query Type: Search
- Service Name: th-payment-processor  
- Tags: http.correlation_id="test-payment-001"

#### 2. Trace Analysis Points
- **Trace Duration**: Total request processing time
- **Span Count**: Number of operations per request
- **Service Graph**: Visualize service interactions
- **Error Spans**: Identify failed operations

---

## Expected Results Summary

### Successful Implementation Indicators

#### ✅ Structured Logging
- [ ] All logs in JSON format with consistent fields
- [ ] Correlation IDs present in all request-related logs
- [ ] Trace IDs and Span IDs automatically included
- [ ] Payment-specific fields (amount, processor, status)
- [ ] Latency measurements in all operation logs
- [ ] Error logs include stack traces and context

#### ✅ Trace Context Propagation
- [ ] Single trace ID spans entire request lifecycle
- [ ] Different span IDs for each operation
- [ ] Correlation IDs propagated to external service calls
- [ ] Enhanced span attributes (correlation_id, amount, etc.)
- [ ] Service graph shows complete request flow

#### ✅ Integration with Monitoring Stack
- [ ] Logs queryable in Loki with structured fields
- [ ] Traces searchable in Tempo by correlation ID
- [ ] Log-to-trace correlation works bidirectionally
- [ ] Metrics correlate with log events

---

## Troubleshooting Common Issues

### Issue: Missing Correlation IDs
**Symptoms:** Logs don't contain correlation_id field
**Solution:** Verify middleware is properly configured in Gin router

### Issue: Incomplete Trace Context
**Symptoms:** trace_id or span_id missing from logs
**Solution:** Check OpenTelemetry initialization and context propagation

### Issue: High Log Volume
**Symptoms:** Too many debug logs in production
**Solution:** Set LOG_LEVEL=info or LOG_LEVEL=warn in environment

### Issue: Query Performance
**Symptoms:** Slow log queries in Grafana
**Solution:** Add indexes on commonly queried fields in Loki configuration

---

## Performance Monitoring

### Key Metrics to Monitor

1. **Log Ingestion Rate**: Logs per second processed by Loki
2. **Query Response Time**: Time to execute log queries
3. **Trace Collection Rate**: Traces per second sent to Tempo
4. **Storage Usage**: Log and trace storage consumption
5. **Application Overhead**: Impact of logging on request latency

### Optimization Recommendations

1. **Log Level Management**: Use appropriate log levels for different environments
2. **Sampling**: Implement trace sampling for high-volume scenarios
3. **Field Selection**: Include only necessary fields in structured logs
4. **Async Logging**: Consider async log writing for high-throughput scenarios

This comprehensive testing approach will validate that your structured logging and tracing enhancements are working correctly and providing the observability improvements you need.