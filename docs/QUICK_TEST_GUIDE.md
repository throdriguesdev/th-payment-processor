# Quick Observability Testing Guide

## 🚀 Quick Start

### 1. Run the Test Script
```bash
cd /home/th/dev/th/th_payment_processor
./scripts/test_observability.sh
```

This script will:
- Generate various payment requests with unique correlation IDs
- Test error scenarios
- Perform load testing
- Create structured log entries with trace context

### 2. Access Grafana
Open your browser and go to: **http://localhost:3000**
- Login: `admin` / `admin123`

### 3. Immediate Verification Queries

#### In Grafana Explore → Loki:

**View all test logs:**
```logql
{service_name="th-payment-processor"} | json
```

**Find logs with correlation IDs:**
```logql
{service_name="th-payment-processor"} | json | correlation_id != ""
```

**Payment processing logs:**
```logql
{service_name="th-payment-processor"} | json | operation=~"payment_.*"
```

**Error logs:**
```logql
{service_name="th-payment-processor"} | json | level="error"
```

#### In Grafana Explore → Tempo:

Search for traces by:
- Service Name: `th-payment-processor`
- Tags: `http.correlation_id` (use correlation IDs from test script output)

## 🔍 What to Look For

### ✅ Structured Logging Success Indicators

1. **JSON Format**: All logs should be properly formatted JSON
2. **Correlation IDs**: Present in all request-related logs
3. **Trace Context**: `trace_id` and `span_id` fields populated
4. **Payment Fields**: `amount`, `processor`, `status` fields
5. **Timing Data**: `latency_ms` in operation completion logs
6. **Component Info**: `component` and `operation` fields

### ✅ Example Expected Log Entry
```json
{
  "@timestamp": "2024-01-15T10:30:45.123Z",
  "level": "info",
  "message": "Payment processed successfully",
  "service_name": "th-payment-processor",
  "service_version": "1.0.0",
  "component": "payment_service", 
  "operation": "payment_success",
  "correlation_id": "basic-payment-success-1705318245",
  "trace_id": "1234567890abcdef1234567890abcdef",
  "span_id": "abcdef0123456789",
  "amount": 100.50,
  "processor": "default",
  "status": "success",
  "latency_ms": 45
}
```

### ✅ Tracing Success Indicators

1. **Single Trace ID**: All spans in a request share the same trace_id
2. **Multiple Spans**: Different operations have different span_ids
3. **Span Attributes**: Enhanced attributes like correlation_id, amount
4. **Service Graph**: Shows payment processor interactions

## 📊 Dashboard Import

Import the pre-built dashboard:

1. Go to Grafana → Dashboards → Import
2. Upload: `deployments/grafana/dashboards/structured-logging-dashboard.json`
3. The dashboard includes:
   - Request volume metrics
   - Payment success rates
   - Latency percentiles
   - Error rates by component
   - Live payment logs
   - Health check status

## 🐛 Troubleshooting

### Issue: No logs appearing
**Check:**
- Payment processor is running: `curl http://localhost:8080/health`
- Loki datasource configured in Grafana
- Logs are being sent to Loki (check docker-compose logs)

### Issue: Missing trace context
**Check:**
- OpenTelemetry properly initialized
- Tempo datasource configured
- Trace exporters running (Jaeger/Tempo)

### Issue: Correlation IDs not working
**Check:**
- Middleware properly configured in Gin router
- Headers being sent with requests
- Context propagation working

## 📈 Performance Testing

For higher volume testing:
```bash
# Generate 100 requests over 30 seconds
for i in {1..100}; do
  curl -X POST http://localhost:8080/payments \
    -H "Content-Type: application/json" \
    -H "X-Correlation-ID: perf-test-$i" \
    -d '{"correlation_id": "perf-test-'$i'", "amount": 100.00}' &
  
  if (( $i % 10 == 0 )); then
    sleep 3
  fi
done
```

## 🎯 Key Verification Points

After running tests, verify in Grafana:

1. **Log Volume**: Increase in log ingestion rate
2. **Correlation Coverage**: All requests have correlation IDs
3. **Trace Completeness**: Traces show full request lifecycle
4. **Error Context**: Error logs include detailed context
5. **Performance Impact**: Minimal impact on response times
6. **Dashboard Functionality**: All panels showing data

This streamlined approach ensures your structured logging and distributed tracing enhancements are working correctly!