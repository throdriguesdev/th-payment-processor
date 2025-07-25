# API Endpoints Reference

## Backend API (Port 9999)

### Payment Processing
**POST /payments**
- Process payment with intelligent routing
- Routes to default processor first (1% fee), falls back to fallback (5% fee)
- Validates input and prevents duplicate payments

**Request:**
```json
{
  "correlationId": "test-123",
  "amount": 100.00
}
```

**Response:**
```json
{
  "id": "uuid-here",
  "correlationId": "test-123",
  "amount": 100.00,
  "processor": "default",
  "requestedAt": "2025-01-15T12:34:56.000Z",
  "processedAt": "2025-01-15T12:34:56.123Z"
}
```

### Payment Summary
**GET /payments-summary**
- Get aggregated payment summary with time filtering
- Optional query parameters: `from` and `to` (ISO 8601 format)

**Response:**
```json
{
  "default": {
    "totalRequests": 10,
    "totalAmount": 1000.00
  },
  "fallback": {
    "totalRequests": 2,
    "totalAmount": 200.00
  }
}
```

## Payment Processor Endpoints

### Default Processor (Port 8001) & Fallback Processor (Port 8002)

**POST /payments** - Process payment
**GET /payments/{id}** - Get payment details
**GET /payments/service-health** - Health check (rate limited to 1 call/5s)

### Admin Endpoints (Require X-Rinha-Token header)

**GET /admin/payments-summary** - Get payment summary
**PUT /admin/configurations/token** - Set admin token
**PUT /admin/configurations/delay** - Set response delay
**PUT /admin/configurations/failure** - Set failure mode
**POST /admin/purge-payments** - Clear all payments

### Example Admin Usage

```bash
# Set admin token
curl -X PUT http://localhost:8001/admin/configurations/token \
  -H "Content-Type: application/json" \
  -H "X-Rinha-Token: 123" \
  -d '{"token":"new-token"}'

# Enable failure mode for testing
curl -X PUT http://localhost:8001/admin/configurations/failure \
  -H "Content-Type: application/json" \
  -H "X-Rinha-Token: 123" \
  -d '{"failure":true}'
```