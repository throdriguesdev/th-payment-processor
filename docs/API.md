# API Documentation

Complete API reference for the TH Payment Processor with request/response examples and integration patterns.

## Base URL
```
http://localhost:9999
```

## Authentication
Currently no authentication required for development. In production, implement API keys or OAuth2.

## Headers
```http
Content-Type: application/json
X-Correlation-ID: optional-correlation-id
```

## Core Payment API

### Create Payment

Process a new payment through the intelligent routing system.

**Endpoint:** `POST /payments`

**Request Body:**
```json
{
  "correlationId": "string (required)",
  "amount": "number (required, > 0)",
  "userId": "string (required)"
}
```

**Response (Success):**
```json
{
  "correlation_id": "payment-12345",
  "message": "Payment processed successfully"
}
```

**Response (Error):**
```json
{
  "correlation_id": "payment-12345",
  "error": "Invalid request format"
}
```

**HTTP Status Codes:**
- `200` - Payment processed successfully
- `400` - Invalid request format or validation error
- `500` - Internal server error or payment processor failure

**Examples:**

```bash
# Basic payment
curl -X POST http://localhost:9999/payments \
  -H "Content-Type: application/json" \
  -d '{
    "correlationId": "pay-001",
    "amount": 100.50,
    "userId": "user123"
  }'

# Payment with custom correlation ID header
curl -X POST http://localhost:9999/payments \
  -H "Content-Type: application/json" \
  -H "X-Correlation-ID: custom-trace-001" \
  -d '{
    "correlationId": "pay-002", 
    "amount": 250.75,
    "userId": "user456"
  }'
```

### Get Payment Summary

Retrieve aggregated payment statistics with optional time filtering.

**Endpoint:** `GET /payments-summary`

**Query Parameters:**
- `from` (optional): ISO 8601 timestamp for start date
- `to` (optional): ISO 8601 timestamp for end date

**Response:**
```json
{
  "default": {
    "totalRequests": 150,
    "totalAmount": 15750.50
  },
  "fallback": {
    "totalRequests": 25,
    "totalAmount": 2500.00
  }
}
```

**Examples:**

```bash
# All payments summary
curl http://localhost:9999/payments-summary

# Time-filtered summary
curl "http://localhost:9999/payments-summary?from=2024-01-01T00:00:00Z&to=2024-01-02T00:00:00Z"

# Today's payments only
curl "http://localhost:9999/payments-summary?from=$(date -u -d 'today 00:00:00' +%Y-%m-%dT%H:%M:%SZ)"
```

## Health & Monitoring API

### Application Health

Check overall application health status.

**Endpoint:** `GET /health`

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "services": {
    "database": "healthy",
    "redis": "healthy",
    "payment_processors": "healthy"
  }
}
```

### Payment Processors Health

Check external payment processor health status.

**Endpoint:** `GET /payments/service-health`

**Response:**
```json
{
  "default": {
    "status": "healthy",
    "response_time_ms": 45,
    "last_check": "2024-01-15T10:30:00Z"
  },
  "fallback": {
    "status": "healthy", 
    "response_time_ms": 62,
    "last_check": "2024-01-15T10:30:00Z"
  }
}
```

### Metrics

Prometheus metrics for monitoring and alerting.

**Endpoint:** `GET /metrics`

**Response:** Prometheus text format
```
# HELP http_requests_total Total HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="POST",path="/payments",status="200"} 1523

# HELP http_request_duration_seconds HTTP request duration
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{le="0.01"} 1200
```

## Direct Payment Processor API

Access payment processors directly for testing or administration.

### Default Processor (1% fee)

**Base URL:** `http://localhost:8001`

### Fallback Processor (5% fee)

**Base URL:** `http://localhost:8002`

### Payment Processing

**Endpoint:** `POST /payments`

**Request:**
```json
{
  "amount": 100.50,
  "correlation_id": "proc-test-001"
}
```

**Response:**
```json
{
  "id": "payment-id-12345",
  "amount": 100.50,
  "fee": 1.00,
  "total": 101.50,
  "processor": "default",
  "status": "completed",
  "correlation_id": "proc-test-001"
}
```

### Processor Configuration

#### Set Processing Delay

**Endpoint:** `PUT /admin/configurations/delay`

**Request:**
```json
{
  "delay_ms": 100
}
```

#### Set Failure Rate

**Endpoint:** `PUT /admin/configurations/failure`

**Request:**
```json
{
  "failure_rate": 0.1
}
```

#### Set Authentication Token

**Endpoint:** `PUT /admin/configurations/token`

**Request:**
```json
{
  "token": "secret-token-123"
}
```

## Error Handling

### Error Response Format

All API errors return JSON with correlation ID for tracing:

```json
{
  "correlation_id": "error-trace-001",
  "error": "Detailed error message",
  "code": "ERROR_CODE",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### Common Error Codes

| HTTP Status | Error Code | Description |
|-------------|------------|-------------|
| 400 | INVALID_REQUEST | Malformed request body or missing required fields |
| 400 | VALIDATION_ERROR | Field validation failed (e.g., negative amount) |
| 429 | RATE_LIMITED | Too many requests from client |
| 500 | INTERNAL_ERROR | Server-side error occurred |
| 502 | PROCESSOR_ERROR | Payment processor unavailable |
| 503 | SERVICE_UNAVAILABLE | Service temporarily unavailable |

### Error Examples

```bash
# Invalid amount
curl -X POST http://localhost:9999/payments \
  -H "Content-Type: application/json" \
  -d '{
    "correlationId": "test-001",
    "amount": -50.00,
    "userId": "user123"
  }'

# Response:
{
  "correlation_id": "test-001",
  "error": "Amount must be greater than 0"
}
```

## Rate Limiting

Current rate limits (configurable):
- **100 requests/minute** per IP address
- **1000 requests/minute** per correlation ID family

Rate limit headers included in response:
```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1642248600
```

## Integration Patterns

### Idempotency

Use correlation IDs to ensure idempotent operations:

```bash
# First request
curl -X POST http://localhost:9999/payments \
  -H "Content-Type: application/json" \
  -d '{"correlationId": "unique-001", "amount": 100.50, "userId": "user123"}'

# Duplicate request (same correlation ID)
curl -X POST http://localhost:9999/payments \
  -H "Content-Type: application/json" \
  -d '{"correlationId": "unique-001", "amount": 100.50, "userId": "user123"}'
# Returns cached result, no duplicate processing
```

### Correlation Tracking

Track requests across services using correlation IDs:

```bash
# Set correlation ID in header
CORRELATION_ID="track-$(date +%s)-$(shuf -i 1000-9999 -n 1)"

curl -X POST http://localhost:9999/payments \
  -H "Content-Type: application/json" \
  -H "X-Correlation-ID: $CORRELATION_ID" \
  -d "{\"correlationId\": \"$CORRELATION_ID\", \"amount\": 100.50, \"userId\": \"user123\"}"

# Query logs for this correlation ID in Grafana:
# {correlation_id="track-1642248600-1234"}
```

### Async Processing with Webhooks (Future)

```json
{
  "correlationId": "async-001",
  "amount": 100.50,
  "userId": "user123",
  "webhook_url": "https://your-app.com/payment-webhook",
  "async": true
}
```

## Client Libraries

### Go Client Example

```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
)

type PaymentRequest struct {
    CorrelationID string  `json:"correlationId"`
    Amount        float64 `json:"amount"`
    UserID        string  `json:"userId"`
}

func processPayment(req PaymentRequest) error {
    jsonData, _ := json.Marshal(req)
    
    resp, err := http.Post(
        "http://localhost:9999/payments",
        "application/json",
        bytes.NewBuffer(jsonData),
    )
    defer resp.Body.Close()
    
    return err
}
```

### JavaScript Client Example

```javascript
class PaymentClient {
    constructor(baseURL = 'http://localhost:9999') {
        this.baseURL = baseURL;
    }
    
    async processPayment({correlationId, amount, userId}) {
        const response = await fetch(`${this.baseURL}/payments`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-Correlation-ID': correlationId
            },
            body: JSON.stringify({
                correlationId,
                amount,
                userId
            })
        });
        
        return response.json();
    }
    
    async getPaymentSummary(from, to) {
        const params = new URLSearchParams();
        if (from) params.set('from', from);
        if (to) params.set('to', to);
        
        const response = await fetch(`${this.baseURL}/payments-summary?${params}`);
        return response.json();
    }
}
```

### Python Client Example

```python
import requests
import json
from datetime import datetime

class PaymentClient:
    def __init__(self, base_url='http://localhost:9999'):
        self.base_url = base_url
        
    def process_payment(self, correlation_id, amount, user_id):
        payload = {
            'correlationId': correlation_id,
            'amount': amount,
            'userId': user_id
        }
        
        headers = {
            'Content-Type': 'application/json',
            'X-Correlation-ID': correlation_id
        }
        
        response = requests.post(
            f'{self.base_url}/payments',
            json=payload,
            headers=headers
        )
        
        return response.json()
        
    def get_payment_summary(self, from_date=None, to_date=None):
        params = {}
        if from_date:
            params['from'] = from_date.isoformat() + 'Z'
        if to_date:
            params['to'] = to_date.isoformat() + 'Z'
            
        response = requests.get(f'{self.base_url}/payments-summary', params=params)
        return response.json()
```

## Performance Considerations

### Request Optimization
- Use HTTP/2 when available
- Implement connection pooling
- Set appropriate timeouts (recommend 5s)
- Use gzip compression for large responses

### Caching Strategy
- Payment summaries are cached for 30 seconds
- Health checks are cached for 5 seconds
- Use ETags for conditional requests

### Monitoring Integration
All API calls generate:
- **Metrics**: Request rate, error rate, latency percentiles
- **Logs**: Structured JSON logs with correlation IDs
- **Traces**: Distributed traces showing full request flow

Query examples:
```bash
# Check API performance in Grafana
# Prometheus: rate(http_requests_total{path="/payments"}[5m])
# Loki: {service_name="th-payment-processor"} | json | operation="payment_request"
```