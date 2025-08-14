#!/bin/bash

# Test script to verify logging and tracing correlation
# This script tests the complete observability stack

set -e

echo "🔍 Testing Logging and Tracing Correlation"
echo "==========================================="

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

# Check if services are running
echo ""
echo "1. Checking service health..."

services=("app1" "app2" "payment-processor-default" "payment-processor-fallback" "loki" "promtail" "grafana" "tempo")

for service in "${services[@]}"; do
    if docker compose -f deployments/docker-compose.yml ps $service | grep -q "Up"; then
        print_status "$service is running"
    else
        print_error "$service is not running"
        echo "Please start the services with: docker compose -f deployments/docker-compose.yml up -d"
        exit 1
    fi
done

echo ""
echo "2. Testing payment processing with correlation..."

# Generate a correlation ID for testing
CORRELATION_ID="test-$(date +%s)-$(shuf -i 1000-9999 -n 1)"
echo "Using correlation ID: $CORRELATION_ID"

# Test payment processing with correlation ID
echo ""
echo "Making payment request with correlation ID..."
RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}\n" \
  -H "Content-Type: application/json" \
  -H "X-Correlation-ID: $CORRELATION_ID" \
  -d "{
    \"correlationId\": \"$CORRELATION_ID\",
    \"amount\": 100.50,
    \"userId\": \"test-user-123\"
  }" \
  http://localhost:9999/payments)

HTTP_STATUS=$(echo "$RESPONSE" | grep "HTTP_STATUS" | cut -d: -f2)

if [ "$HTTP_STATUS" = "200" ] || [ "$HTTP_STATUS" = "201" ]; then
    print_status "Payment request successful (HTTP $HTTP_STATUS)"
    PAYMENT_DATA=$(echo "$RESPONSE" | sed '/HTTP_STATUS/d')
    echo "Response: $PAYMENT_DATA"
    
    # Extract payment ID from response
    PAYMENT_ID=$(echo "$PAYMENT_DATA" | jq -r '.payment_id // .id // "unknown"' 2>/dev/null || echo "unknown")
    echo "Payment ID: $PAYMENT_ID"
else
    print_error "Payment request failed (HTTP $HTTP_STATUS)"
    echo "Response: $RESPONSE"
fi

echo ""
echo "3. Waiting for logs to be processed (10 seconds)..."
sleep 10

echo ""
echo "4. Checking Loki for logs with correlation ID..."

# Query Loki for logs with our correlation ID
LOKI_QUERY="{correlation_id=\"$CORRELATION_ID\"}"
LOKI_URL="http://localhost:3100/loki/api/v1/query_range"

echo "Querying Loki: $LOKI_QUERY"

# Calculate time range (last 2 minutes)
END_TIME=$(date +%s)000000000  # nanoseconds
START_TIME=$((END_TIME - 120000000000))  # 2 minutes ago

LOKI_RESPONSE=$(curl -s -G "$LOKI_URL" \
  --data-urlencode "query=$LOKI_QUERY" \
  --data-urlencode "start=$START_TIME" \
  --data-urlencode "end=$END_TIME" \
  --data-urlencode "limit=100")

# Check if we got logs
LOG_COUNT=$(echo "$LOKI_RESPONSE" | jq -r '.data.result | length' 2>/dev/null || echo "0")

if [ "$LOG_COUNT" -gt 0 ]; then
    print_status "Found $LOG_COUNT log streams with correlation ID $CORRELATION_ID"
    
    # Show sample logs
    echo ""
    echo "Sample log entries:"
    echo "$LOKI_RESPONSE" | jq -r '.data.result[] | .stream as $stream | .values[] | "[\($stream.level // "INFO")] [\($stream.service_name // "unknown")] \(.[1])"' 2>/dev/null | head -5
else
    print_warning "No logs found with correlation ID $CORRELATION_ID"
    echo "This might be due to:"
    echo "  - Logs not yet processed by Promtail"
    echo "  - Loki indexing delay"
    echo "  - Configuration issues"
fi

echo ""
echo "5. Checking Tempo for traces..."

# Query for traces (this is more complex and depends on having trace data)
TEMPO_URL="http://localhost:3200/api/search"

echo "Checking if Tempo is responding..."
TEMPO_HEALTH=$(curl -s -w "%{http_code}" "http://localhost:3200/ready" -o /dev/null)

if [ "$TEMPO_HEALTH" = "200" ]; then
    print_status "Tempo is healthy and ready"
else
    print_warning "Tempo health check returned HTTP $TEMPO_HEALTH"
fi

echo ""
echo "6. Testing Grafana connectivity..."

GRAFANA_URL="http://localhost:3000/api/health"
GRAFANA_HEALTH=$(curl -s "$GRAFANA_URL" | jq -r '.database' 2>/dev/null || echo "unknown")

if [ "$GRAFANA_HEALTH" = "ok" ]; then
    print_status "Grafana is healthy"
    echo "Access Grafana at: http://localhost:3000 (admin/admin123)"
    echo "Try these queries in Grafana Explore:"
    echo "  Loki: {correlation_id=\"$CORRELATION_ID\"}"
    echo "  Loki: {service_name=\"th-payment-processor\"}"
    echo "  Loki: {container_name=~\"app[12]|payment-processor.*\"}"
else
    print_warning "Grafana health check failed"
fi

echo ""
echo "7. Testing log correlation across services..."

# Make another request to test cross-service correlation
echo "Making second request to test cross-service logging..."

SECOND_RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}\n" \
  -H "Content-Type: application/json" \
  -H "X-Correlation-ID: $CORRELATION_ID" \
  -d "{
    \"correlationId\": \"$CORRELATION_ID\",
    \"amount\": 75.25,
    \"userId\": \"test-user-456\"
  }" \
  http://localhost:9999/payments)

SECOND_HTTP_STATUS=$(echo "$SECOND_RESPONSE" | grep "HTTP_STATUS" | cut -d: -f2)

if [ "$SECOND_HTTP_STATUS" = "200" ] || [ "$SECOND_HTTP_STATUS" = "201" ]; then
    print_status "Second payment request successful"
else
    print_warning "Second payment request failed (HTTP $SECOND_HTTP_STATUS)"
fi

echo ""
echo "8. Summary and recommendations..."

print_status "Logging and tracing correlation test completed!"

echo ""
echo "📊 Next steps to verify your setup:"
echo "1. Open Grafana: http://localhost:3000"
echo "2. Go to Explore"
echo "3. Select Loki datasource"
echo "4. Use query: {correlation_id=\"$CORRELATION_ID\"}"
echo "5. Check for logs from multiple services with the same correlation ID"
echo ""
echo "🔍 For trace correlation:"
echo "1. In Grafana, select Tempo datasource"
echo "2. Look for traces containing your correlation ID"
echo "3. Verify spans from different services are connected"
echo ""
echo "📋 Useful Loki queries:"
echo "  - All payment logs: {service_name=\"th-payment-processor\"}"
echo "  - Error logs only: {level=\"error\"}"
echo "  - Specific container: {container_name=\"app1\"}"
echo "  - Payment operations: {operation=\"payment_start\"}"
echo ""
echo "Correlation ID used in this test: $CORRELATION_ID"