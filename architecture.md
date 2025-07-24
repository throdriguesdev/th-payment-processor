
## 🏗️ **Architecture Overview**

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

## 📁 **Project Structure**

```
rinha-backend/
├── main.go                    # Main application entry point
├── go.mod & go.sum           # Go dependencies
├── Dockerfile                 # Multi-stage Docker build
├── docker-compose.yml         # Complete orchestration
├── nginx.conf                 # Load balancer configuration
├── README.md                  # Comprehensive documentation
├── .gitignore                 # Git ignore rules
└── internal/
    ├── config/                # Configuration management
    ├── models/                # Data structures
    ├── storage/               # In-memory storage
    ├── services/              # Business logic
    ├── handlers/              # HTTP handlers
    └── middleware/            # HTTP middleware

../payment-processors/
├── main.go                    # Payment processor entry point
├── go.mod & go.sum           # Dependencies
├── Dockerfile                 # Docker build
├── docker-compose.yml         # Processor orchestration
├── README.md                  # Documentation
└── [models, storage, handlers]/ # Processor components
```

## 🎯 **Key Features Implemented**

### **Backend API**

- ✅ **POST /payments** - Process payments with intelligent routing
- ✅ **GET /payments-summary** - Audit trail with time filtering
- ✅ **Health Monitoring** - Continuous processor health checks (rate-limited)
- ✅ **Load Balancing** - Two app instances with nginx
- ✅ **Resource Limits** - 1.5 CPU, 350MB total memory
- ✅ **Docker Compose** - Complete containerization
- ✅ **Network Integration** - Connects to payment-processor network

### **Payment Processors (payment-processors)**

- ✅ **Default Processor** - 1% fee, 50ms min response time
- ✅ **Fallback Processor** - 5% fee, 100ms min response time
- ✅ **Health Endpoints** - `/payments/service-health` (rate-limited)
- ✅ **Admin Endpoints** - Token, delay, failure configuration
- ✅ **Payment Summary** - `/admin/payments-summary` for audit
- ✅ **Docker Setup** - Ready for deployment

## 🔧 **Technical Implementation**

### **Smart Payment Routing**

- Always tries default processor first (lower fee)
- Falls back to fallback processor if default fails
- Health monitoring with rate limiting (1 call/5s)
- Handles both processors being down

### **Consistency & Audit**

- Complete payment record tracking
- Time-based summary filtering
- Matches processor summary format exactly
- Thread-safe in-memory storage

### **Performance Optimization**

- In-memory storage for speed
- Efficient JSON handling
- Minimal memory footprint
- Optimized for p99 < 11ms

### **Docker Configuration**

- Multi-stage builds for small images
- Resource limits enforced
- Network isolation
- Bridge networking (no host mode)
- No privileged containers

## 🚀 **Deployment Instructions**

### **Start Payment Processors:**

```bash
cd ../payment-processors
docker-compose up -d
```

### **Start Backend:**

```bash
cd ../rinha-backend
docker-compose up -d
```

## 🧪 **Testing Commands**

### **Test Payment Processing:**

```bash
# Test payment processing
curl -X POST http://localhost:9999/payments \
  -H "Content-Type: application/json" \
  -d '{"correlationId":"test-123","amount":100.00}'
```

### **Test Summary Endpoint:**

```bash
# Test summary
curl http://localhost:9999/payments-summary
```

### **Test Processor Health:**

```bash
# Test default processor health
curl http://localhost:8001/payments/service-health

# Test fallback processor health
curl http://localhost:8002/payments/service-health
```

### **Test Payment Processors Directly:**

```bash
# Test default processor
curl -X POST http://localhost:8001/payments \
  -H "Content-Type: application/json" \
  -d '{"correlationId":"test-123","amount":100.00,"requestedAt":"2025-01-15T12:34:56.000Z"}'

# Test fallback processor
curl -X POST http://localhost:8002/payments \
  -H "Content-Type: application/json" \
  -d '{"correlationId":"test-456","amount":100.00,"requestedAt":"2025-01-15T12:34:56.000Z"}'
```

### **Test Administrative Endpoints:**

```bash
# Set admin token
curl -X PUT http://localhost:8001/admin/configurations/token \
  -H "Content-Type: application/json" \
  -H "X-Rinha-Token: 123" \
  -d '{"token":"new-token"}'

# Set response delay
curl -X PUT http://localhost:8001/admin/configurations/delay \
  -H "Content-Type: application/json" \
  -H "X-Rinha-Token: 123" \
  -d '{"delay":1000}'

# Enable failure mode
curl -X PUT http://localhost:8001/admin/configurations/failure \
  -H "Content-Type: application/json" \
  -H "X-Rinha-Token: 123" \
  -d '{"failure":true}'

# Get payment summary
curl -X GET "http://localhost:8001/admin/payments-summary?from=2025-01-15T00:00:00.000Z&to=2025-01-15T23:59:59.000Z" \
  -H "X-Rinha-Token: 123"

# Purge all payments
curl -X POST http://localhost:8001/admin/purge-payments \
  -H "X-Rinha-Token: 123"
```

## 📊 **Scoring Optimization**

- **Profit Maximization**: Always uses lowest fee processor when available
- **Consistency**: Complete audit trail prevents penalties
- **Performance**: Optimized for p99 < 11ms (bonus eligible)
- **Reliability**: Handles all failure scenarios gracefully

## 🔍 **API Endpoints Reference**

### **Backend Endpoints**

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/payments` | Process payment with intelligent routing |
| GET | `/payments-summary` | Get payment summary with time filtering |

### **Payment Processor Endpoints**

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/payments` | Process payment |
| GET | `/payments/{id}` | Get payment details |
| GET | `/payments/service-health` | Health check (rate limited) |
| GET | `/admin/payments-summary` | Get payment summary (admin) |
| PUT | `/admin/configurations/token` | Set admin token |
| PUT | `/admin/configurations/delay` | Set response delay |
| PUT | `/admin/configurations/failure` | Set failure mode |
| POST | `/admin/purge-payments` | Clear all payments |
