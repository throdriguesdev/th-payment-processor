#!/bin/bash

echo "=== Quick Metrics Test ==="

# Kill any existing processes
pkill -f "payment-processor" 2>/dev/null || true
sleep 2

echo "1. Starting payment processor..."
cd payment-processors
PORT=8080 FEE_PERCENTAGE=1.0 ./payment-processor &
PP_PID=$!
cd ..

echo "2. Waiting for startup..."
sleep 5

echo "3. Testing metrics endpoint:"
curl -s http://localhost:2112/metrics | head -10

echo ""
echo "4. Testing health endpoint:"
curl -s http://localhost:2112/health

echo ""
echo "5. Making a payment request to generate metrics..."
curl -X POST http://localhost:8080/payments \
  -H 'Content-Type: application/json' \
  -d '{"correlationId": "test-123", "amount": 100.50}'

sleep 2

echo ""
echo "6. Checking updated metrics:"
curl -s http://localhost:2112/metrics | grep -E "(http_requests|payment)"

echo ""
echo "7. Cleaning up..."
kill $PP_PID 2>/dev/null || true

echo "Test complete!"