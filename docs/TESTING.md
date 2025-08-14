# Testing Guide

Comprehensive testing documentation for the TH Payment Processor, including automated test suites, observability validation, and performance testing.

## Quick Testing Overview

The system includes multiple testing approaches for different scenarios:

### 🎯 **Quick Start Testing**
```bash
# Option 1: Using Makefile (Recommended)
make init    # Initialize and start all services
make test    # Run comprehensive test suite
make clean   # Stop services and cleanup

# Option 2: Using Scripts
./scripts/init.sh              # Start all services
./scripts/test_payments.sh     # Test payment processing
./scripts/test-logging-correlation.sh  # Test observability
./scripts/cleanup.sh           # Clean up

# Option 3: Manual initialization
cd deployments && docker compose up -d
```

### 🧪 **Available Test Suites**
- **Payment API Tests**: Core payment processing functionality
- **Observability Tests**: Logging, tracing, and metrics validation
- **Performance Tests**: Load testing and stress testing
- **Integration Tests**: End-to-end system validation
- **Processor Tests**: Payment processor health and routing

## Makefile Testing Commands

### Available Make Targets

```bash
# View all available commands
make help

# Common testing workflows
make init          # Initialize everything (build + start services)
make test          # Run comprehensive test suite
make test-quick    # Run quick smoke tests only
make test-perf     # Run performance tests
make test-obs      # Run observability tests
make clean         # Stop services and cleanup
make logs          # View service logs
make status        # Check service status
```

### Makefile Benefits

The Makefile provides convenient shortcuts for complex operations:

```makefile
# Example Makefile targets
init:
	@echo "🚀 Initializing TH Payment Processor..."
	./scripts/init.sh
	@echo "✅ Initialization complete"

test: test-health test-payments test-observability
	@echo "🎉 All tests completed successfully!"

test-payments:
	@echo "💳 Testing payment processing..."
	./scripts/test_payments.sh

test-observability:
	@echo "📊 Testing observability stack..."
	./scripts/test-logging-correlation.sh

clean:
	@echo "🧹 Cleaning up..."
	./scripts/cleanup.sh
```

## Core Testing Scripts

### 1. Payment Processing Tests

**Script:** `./scripts/test_payments.sh`

**What it tests:**
- Basic payment processing functionality
- Input validation and error handling
- Multiple payment scenarios
- Payment summary aggregation
- Load testing with concurrent requests
- Performance benchmarking

**Example output:**
```bash
🧪 Starting Payment Processing Tests...

Test 1: Basic Payment Processing
{"correlation_id":"test-payment-001","message":"Payment processed successfully"}
Status: 200, Time: 0.005461s ✅

Test 2: Multiple Payments
Payment 1: Status: 200, Time: 0.043471s ✅
Payment 2: Status: 200, Time: 0.004453s ✅
Payment 3: Status: 200, Time: 0.004901s ✅

Test 3: Invalid Payment Requests
Missing correlationId: Status: 400 ✅
Missing amount: Status: 400 ✅
Zero amount: Status: 400 ✅
Negative amount: Status: 400 ✅

Test 4: Load Testing (10 concurrent payments)
All 10 payments completed successfully ✅
Average response time: 0.012s ✅
```

### 2. Observability Correlation Tests

**Script:** `./scripts/test-logging-correlation.sh`

**What it tests:**
- Correlation ID propagation across services
- Structured logging format validation
- Loki log ingestion and parsing
- Grafana datasource connectivity
- Tempo tracing integration
- Cross-service log correlation

**Example output:**
```bash
🔍 Testing Logging and Tracing Correlation

1. Checking service health...
✓ app1 is running
✓ app2 is running
✓ payment-processor-default is running
✓ payment-processor-fallback is running
✓ loki is running
✓ promtail is running
✓ grafana is running
✓ tempo is running

2. Testing payment processing with correlation...
Using correlation ID: test-1755152670-8460
✓ Payment request successful (HTTP 200)

3. Checking Loki for logs with correlation ID...
✓ Found 3 log streams with correlation ID test-1755152670-8460

Sample log entries:
[info] [th-payment-processor] Payment processed successfully
[info] [th-payment-processor] HTTP request completed
[info] [th-payment-processor] Processing payment request
```

### 3. Payment Processor Tests

**Script:** `./scripts/test_processors.sh`

**What it tests:**
- Direct payment processor connectivity
- Default vs fallback processor routing
- Processor health check endpoints
- Fee calculation accuracy
- Processor failover scenarios
- Administrative configuration endpoints

### 4. Performance & Load Testing

**Script:** `./scripts/stress_test.sh`

**What it tests:**
- High-volume payment processing
- System behavior under load
- Resource utilization monitoring
- Response time consistency (P99 < 11ms target)
- Error rate under stress
- Recovery after load

**Example configuration:**
```bash
# Configurable test parameters
CONCURRENT_USERS=50
REQUESTS_PER_USER=100
RAMP_UP_TIME=30s
TEST_DURATION=300s

# Run stress test
./scripts/stress_test.sh
```

## Test Categories

### 1. Smoke Tests (Quick)

**Purpose:** Verify basic functionality is working
**Duration:** < 30 seconds
**When to run:** After deployment, before releases

```bash
# Quick smoke test
make test-quick

# Or manually:
curl -sf http://localhost:9999/health
curl -X POST http://localhost:9999/payments \
  -H "Content-Type: application/json" \
  -d '{"correlationId":"smoke-test","amount":1.00,"userId":"test"}'
```

### 2. Integration Tests

**Purpose:** Test interaction between components
**Duration:** 2-5 minutes
**When to run:** During development, CI/CD pipelines

```bash
# Full integration test
make test

# Individual components:
./scripts/test_payments.sh        # Payment API integration
./scripts/test_processors.sh      # Processor integration
./scripts/test-logging-correlation.sh  # Observability integration
```

### 3. Performance Tests

**Purpose:** Validate system performance under load
**Duration:** 5-10 minutes
**When to run:** Before releases, capacity planning

```bash
# Standard performance test
make test-perf

# Custom load parameters
CONCURRENT_USERS=100 DURATION=600 ./scripts/stress_test.sh
```

### 4. Observability Tests

**Purpose:** Ensure monitoring and tracing work correctly
**Duration:** 1-2 minutes
**When to run:** After observability changes

```bash
# Test correlation and tracing
make test-obs

# Test specific correlation ID
CORRELATION_ID="custom-test-$(date +%s)" ./scripts/test-logging-correlation.sh
```

## Manual Testing Procedures

### 1. End-to-End Payment Flow

```bash
# Step 1: Start services
make init

# Step 2: Test basic payment
CORRELATION_ID="e2e-$(date +%s)"
curl -X POST http://localhost:9999/payments \
  -H "Content-Type: application/json" \
  -H "X-Correlation-ID: $CORRELATION_ID" \
  -d "{
    \"correlationId\": \"$CORRELATION_ID\",
    \"amount\": 100.50,
    \"userId\": \"test-user-123\"
  }"

# Step 3: Verify in Grafana
echo "Check Grafana logs: {correlation_id=\"$CORRELATION_ID\"}"
open http://localhost:3000

# Step 4: Check payment summary
curl http://localhost:9999/payments-summary
```

### 2. Failover Testing

```bash
# Test default processor failure scenario
docker compose -f deployments/docker-compose.yml stop payment-processor-default

# Make payment (should use fallback)
curl -X POST http://localhost:9999/payments \
  -H "Content-Type: application/json" \
  -d '{"correlationId":"failover-test","amount":100.50,"userId":"test"}'

# Restart default processor
docker compose -f deployments/docker-compose.yml start payment-processor-default
```

## Continuous Integration

### GitHub Actions Example

```yaml
name: Payment Processor Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      
      - name: Initialize Services
        run: make init
        
      - name: Wait for Services
        run: sleep 30
        
      - name: Run Health Checks
        run: make test-health
        
      - name: Run Payment Tests
        run: make test-payments
        
      - name: Run Observability Tests
        run: make test-obs
        
      - name: Cleanup
        run: make clean
        if: always()
```

## Troubleshooting Test Issues

### Common Test Failures

1. **Services not ready**
```bash
# Wait for services to be healthy
./scripts/init.sh
sleep 30  # Allow time for startup
curl http://localhost:9999/health  # Verify readiness
```

2. **Port conflicts**
```bash
# Check for port conflicts
netstat -tulpn | grep :9999
# Kill conflicting processes or change ports
```

3. **Docker issues**
```bash
# Reset Docker environment
make clean
docker system prune -f
make init
```

4. **Observability not working**
```bash
# Check observability services
docker compose -f deployments/docker-compose.yml logs grafana
docker compose -f deployments/docker-compose.yml logs loki
docker compose -f deployments/docker-compose.yml logs promtail

# Restart observability stack
docker compose -f deployments/docker-compose.yml restart grafana loki promtail tempo
```

## Performance Targets & Validation

### Target Metrics
| Metric | Target | Validation Method |
|--------|--------|-------------------|
| Response Time (P99) | < 11ms | `./scripts/stress_test.sh` |
| Response Time (P95) | < 50ms | Load testing |
| Throughput | > 1000 RPS | Stress testing |
| Error Rate | < 0.1% | Integration tests |
| Memory Usage | < 350MB total | Resource monitoring |
| CPU Usage | < 1.5 cores total | Resource monitoring |

### Performance Validation

```bash
# Quick performance check
time curl -X POST http://localhost:9999/payments \
  -H "Content-Type: application/json" \
  -d '{"correlationId":"perf-test","amount":100.50,"userId":"test"}'

# Load test with metrics
ab -n 1000 -c 10 -p payment.json -T application/json http://localhost:9999/payments

# Monitor in Grafana during test
echo "Monitor: http://localhost:3000"
```

## Test Environment Setup

### Prerequisites Validation

```bash
# Check Docker installation
docker --version
docker compose version

# Check available ports
netstat -tulpn | grep -E ":(3000|8001|8002|8081|8082|9090|9999)"

# Check system resources
free -h    # Memory (need at least 2GB)
df -h      # Disk space (need at least 5GB)
```

### Environment Variables

```bash
# Test configuration
export TEST_BASE_URL="http://localhost:9999"
export TEST_TIMEOUT="30s"
export TEST_VERBOSE="true"

# Performance test settings
export STRESS_TEST_USERS="50"
export STRESS_TEST_DURATION="300"
export STRESS_TEST_RAMPUP="30"
```

This comprehensive testing strategy ensures the payment processor maintains high reliability and performance across all deployment scenarios. The combination of Makefile convenience commands and detailed scripts provides both quick validation and thorough testing capabilities.