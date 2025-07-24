
### Prerequisites

- Docker and Docker Compose

### Quick Start with Automated Scripts

**Option 1: Automated Setup (Recommended)**

```bash
# Initialize entire environment automatically
./init.sh

# Run comprehensive tests
./test_payments.sh      # Test payment API functionality
./test_processors.sh    # Test processor integration
./stress_test.sh        # Performance testing (p99 < 11ms)

# Clean up when done
./cleanup.sh
```

**Option 2: Manual Setup**

1. **Start the payment processors first** *(separate project)*:

   ```bash
   cd ../payment-processors
   docker-compose up -d
   ```

2. **Start the backend application** *(this project)*:

   ```bash
   cd ../rinha-backend  
   docker-compose up -d
   ```

3. **Verify backend is running:**

   ```bash
   docker-compose ps
   curl http://localhost:9999/payments-summary
   ```

4. **Test payment processing:**

   ```bash
   # Test backend payment processing (routes to processors)
   curl -X POST http://localhost:9999/payments \
     -H "Content-Type: application/json" \
     -d '{"correlationId":"test-123","amount":100.00}'
   
   # Test payment summary
   curl http://localhost:9999/payments-summary
   ```

### Backend Environment Variables

- `SERVER_PORT`: Backend service port (default: 8080)
- `DEFAULT_PROCESSOR_URL`: Default processor URL (default: <http://payment-processor-default:8080>)
- `FALLBACK_PROCESSOR_URL`: Fallback processor URL (default: <http://payment-processor-fallback:8080>)
- `HEALTH_CHECK_INTERVAL`: Health check frequency (default: 5s)
- `REQUEST_TIMEOUT`: Processor request timeout (default: 10s)

### Payment Processor Environment Variables *(separate project)*

- `FEE_PERCENTAGE`: Fee percentage (1.0% for default, 5.0% for fallback)
- `MIN_RESPONSE_TIME`: Minimum response time (50ms for default, 100ms for fallback)

### Testing Payment Processor Configuration *(for development)*

1. **Set admin token on processors:**

   ```bash
   curl -X PUT http://localhost:8001/admin/configurations/token \
     -H "Content-Type: application/json" \
     -H "X-Rinha-Token: 123" \
     -d '{"token":"new-token"}'
   ```

2. **Set response delay on processors:**

   ```bash
   curl -X PUT http://localhost:8001/admin/configurations/delay \
     -H "Content-Type: application/json" \
     -H "X-Rinha-Token: 123" \
     -d '{"delay":1000}'
   ```

3. **Enable failure mode on processors:**

   ```bash
   curl -X PUT http://localhost:8001/admin/configurations/failure \
     -H "Content-Type: application/json" \
     -H "X-Rinha-Token: 123" \
     -d '{"failure":true}'
   ```

### Health Monitoring

- Background health monitoring of payment processors
- Rate-limited health checks (1 call per 5 seconds)
- Automatic processor failover when unhealthy

### Network Configuration

- Backend connects to external `payment-processor` network
- Nginx load balancer distributes requests between app instances
- Bridge networking (no host mode) for security

### Smart Payment Routing

- Always tries default processor first (lower 1% fee)
- Automatically falls back to fallback processor (5% fee)
- Handles both processors being unavailable
- Complete audit trail for consistency verification

## Automated Testing Suite

### Available Test Scripts

| Script | Description | Usage |
|--------|-------------|-------|
| `./test_payments.sh` | **Payment API Tests** - Basic functionality, validation, load testing | `./test_payments.sh` |
| `./test_processors.sh` | **Processor Integration** - Health checks, failure simulation, admin endpoints | `./test_processors.sh` |
| `./stress_test.sh` | **Performance Testing** - Load testing to validate p99 < 11ms target | `./stress_test.sh [users] [requests]` |

### Test Coverage

**Payment API Tests (`test_payments.sh`):**
- ✅ Basic payment processing
- ✅ Multiple concurrent payments
- ✅ Input validation (missing fields, invalid amounts)
- ✅ Payment summary endpoints
- ✅ Time-based filtering
- ✅ Load testing (concurrent requests)
- ✅ Response time measurement

**Processor Integration Tests (`test_processors.sh`):**
- ✅ Health check endpoints
- ✅ Direct processor communication
- ✅ Admin endpoints (summary, configuration)
- ✅ Failure simulation scenarios
- ✅ Response delay testing
- ✅ Rate limiting validation

**Performance Tests (`stress_test.sh`):**
- ✅ Configurable concurrent users and requests
- ✅ Response time percentiles (P50, P95, P99)
- ✅ Performance bonus calculation
- ✅ Error rate monitoring
- ✅ Throughput measurement

### Example Usage

```bash
# Quick functional test
./test_payments.sh

# Test with failure scenarios
./test_processors.sh

# Performance test with custom load
./stress_test.sh 20 100  # 20 users, 100 requests each

# Light performance test
./stress_test.sh 5 10    # 5 users, 10 requests each
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

  - ✅ POST /payments: Successfully processes payments and routes to default processor first
  - ✅ GET /payments-summary: Returns correct summary with default/fallback breakdown
  - ✅ Time Filtering: Works correctly with from/to parameters
  - ✅ Load Balancing: Nginx distributes requests between 2 app instances
  - ✅ Fallback Processing: Automatically switches to fallback processor when default fails
  - ✅ Health Monitoring: Background health checks working every 5 seconds

  Architecture Compliance:

  - ✅ Port 9999: Backend accessible via <http://localhost:9999>
  - ✅ Two Web Servers: app1 and app2 instances running behind nginx
  - ✅ Load Balancer: Nginx with IP hash for session affinity
  - ✅ Resource Limits:
    - nginx: 2.5MB/50MB, 0.3 CPU
    - app1: 5.7MB/150MB, 0.6 CPU
    - app2: 6.0MB/150MB, 0.6 CPU
    - Total: 1.5 CPU, 350MB (exactly as required)

  Integration Tests:

  - ✅ Payment Processors: Both default (1% fee) and fallback (5% fee) working
  - ✅ Network Integration: Backend connects to payment-processor network
  - ✅ Smart Routing: Always tries default first, falls back automatically
  - ✅ Consistency Tracking: Proper audit trail for Central Bank verification

  Key Fixes Applied:

  1. Fixed health monitoring: Now checks both processors separately with proper rate limiting
  2. Fixed shared storage: Implemented nginx ip_hash for session affinity
  3. Fixed payment routing: Proper default→fallback logic with health checks
  4. Fixed requirements format: All endpoints match requisitos.md specifications

  Performance Optimizations:

  - In-memory storage for speed (targeting p99 < 11ms)
  - Background health monitoring
  - Efficient JSON processing
  - Minimal resource footprint

## Available Scripts

| Script | Purpose | Description |
|--------|---------|-------------|
| `./init.sh` | **Environment Setup** | Automatically starts payment processors and backend with proper sequence |
| `./cleanup.sh` | **Environment Cleanup** | Stops all services and cleans up Docker resources |
| `./test_payments.sh` | **API Testing** | Comprehensive payment API functionality tests |
| `./test_processors.sh` | **Integration Testing** | Payment processor integration and failure scenario tests |
| `./stress_test.sh` | **Performance Testing** | Load testing with p99 performance measurement |

### Service Endpoints

- **Backend API:** http://localhost:9999
- **Default Processor:** http://localhost:8001 (1% fee)
- **Fallback Processor:** http://localhost:8002 (5% fee)

### Monitoring Commands

```bash
# View service status
docker-compose ps

# Monitor backend logs
docker-compose logs -f

# Monitor processor logs
cd ../payment-processors && docker-compose logs -f

# Check payment summary
curl http://localhost:9999/payments-summary
```
