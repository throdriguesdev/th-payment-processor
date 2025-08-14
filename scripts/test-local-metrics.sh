#!/bin/bash

echo "=== Local Metrics Test ==="
echo "Testing metrics functionality before docker deployment"
echo ""

# Kill any existing processes on our ports
echo "Cleaning up any existing processes..."
pkill -f "th_payment_processor" 2>/dev/null || true
pkill -f "payment-processor" 2>/dev/null || true

# Build applications
echo "1. Building applications..."
go build -o build/th_payment_processor ./cmd/server
cd payment-processors && go build -o payment-processor . && cd ..

echo ""
echo "2. Starting payment processor in background..."
cd payment-processors
PORT=8001 FEE_PERCENTAGE=1.0 MIN_RESPONSE_TIME=50 ./payment-processor &
PP_PID=$!
cd ..

sleep 3

echo "3. Testing payment processor metrics..."
curl -s http://localhost:2112/metrics | head -5 || echo "Payment processor metrics not available"
curl -s http://localhost:2112/health || echo "Payment processor health check failed"

echo ""
echo "4. Starting main app in background (with mock external services)..."
# Set environment variables for local testing
export SERVER_PORT=8080
export DEFAULT_PROCESSOR_URL=http://localhost:8001
export FALLBACK_PROCESSOR_URL=http://localhost:8001  # Use same for testing
export JAEGER_ENDPOINT=http://localhost:14268/api/traces
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5432
export POSTGRES_USER=postgres
export POSTGRES_PASSWORD=password123
export POSTGRES_DB=payments
export POSTGRES_SSLMODE=disable
export REDIS_ADDR=localhost:6379
export REDIS_PASSWORD=
export REDIS_DB=0

# Start main app
./build/th_payment_processor &
APP_PID=$!

sleep 5

echo ""
echo "5. Testing main app metrics..."
curl -s http://localhost:2112/metrics | head -5 || echo "Main app metrics not available"

echo ""
echo "6. Making a test payment to generate metrics..."
curl -X POST http://localhost:8080/payments \
  -H 'Content-Type: application/json' \
  -d '{"correlationId": "local-test-123", "amount": 25.00}' \
  -w "\nStatus: %{http_code}\n" || echo "Payment request failed"

sleep 2

echo ""
echo "7. Checking updated metrics..."
echo "Payment processor metrics:"
curl -s http://localhost:2112/metrics | grep -E "(http_requests|payment_)" | head -3 || echo "No metrics found"

echo ""
echo "8. Cleaning up..."
kill $PP_PID 2>/dev/null || true
kill $APP_PID 2>/dev/null || true

echo ""
echo "Local test complete! If metrics appeared above, the setup is working."
echo "You can now run: cd deployments && ./test-monitoring.sh"