# Testing Guide

## Automated Testing Scripts

### Quick Start
```bash
# Initialize entire environment
./init.sh

# Run all tests
./test_payments.sh      # API functionality tests
./test_processors.sh    # Integration tests
./stress_test.sh        # Performance tests (p99 < 11ms)

# Clean up
./cleanup.sh
```

## Test Scripts Overview

### `./test_payments.sh` - Payment API Tests
- ✅ Basic payment processing
- ✅ Multiple concurrent payments
- ✅ Input validation (missing fields, invalid amounts)
- ✅ Payment summary endpoints
- ✅ Time-based filtering
- ✅ Load testing (concurrent requests)
- ✅ Response time measurement

### `./test_processors.sh` - Processor Integration Tests
- ✅ Health check endpoints
- ✅ Direct processor communication
- ✅ Admin endpoints (summary, configuration)
- ✅ Failure simulation scenarios
- ✅ Response delay testing
- ✅ Rate limiting validation

### `./stress_test.sh` - Performance Tests
- ✅ Configurable concurrent users and requests
- ✅ Response time percentiles (P50, P95, P99)
- ✅ Performance bonus calculation (p99 < 11ms target)
- ✅ Error rate monitoring
- ✅ Throughput measurement

**Usage:**
```bash
./stress_test.sh 20 100  # 20 users, 100 requests each
./stress_test.sh 5 10    # Light test: 5 users, 10 requests each
```

## Manual Testing Examples

### Basic Payment Test
```bash
curl -X POST http://localhost:9999/payments \
  -H "Content-Type: application/json" \
  -d '{"correlationId":"test-123","amount":100.00}'
```

### Summary Test
```bash
curl http://localhost:9999/payments-summary
```

### Time-filtered Summary
```bash
curl "http://localhost:9999/payments-summary?from=2025-01-15T00:00:00.000Z&to=2025-01-15T23:59:59.000Z"
```

## Testing Scenarios

### Normal Operation
- Both processors respond normally
- Default processor has lower fee (1%)
- Fallback processor has higher fee (5%)

### Failure Scenarios
- Set `failure: true` to simulate processor failure
- Set `delay` to simulate slow responses
- Both processors can fail simultaneously

### Consistency Testing
- Use `/admin/payments-summary` to verify payment records
- Compare with backend's `/payments-summary` endpoint
- Purge payments to reset state for testing

## Expected Results

All tests should pass with:
- ✅ Payment processing working correctly
- ✅ Smart routing (default → fallback)
- ✅ Health monitoring active
- ✅ Load balancing between app1/app2
- ✅ Performance target: p99 < 11ms
- ✅ Resource limits: 1.5 CPU, 350MB memory