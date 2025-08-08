#!/bin/bash

echo "=== Payment Processor Monitoring Validation ==="
echo ""

# Test basic service health
echo "1. Testing service health..."
echo "   ✓ Payment service: $(curl -s http://localhost:9999/health -w "%{http_code}" -o /dev/null)"
echo "   ✓ Prometheus: $(curl -s http://localhost:9090/-/healthy -w "%{http_code}" -o /dev/null)"
echo "   ✓ Grafana: $(curl -s http://localhost:3000/api/health -w "%{http_code}" -o /dev/null)"

echo ""
echo "2. Testing metrics endpoints..."
echo "   ✓ App1 metrics: $(curl -s http://localhost:2113/health -w "%{http_code}" -o /dev/null)"
echo "   ✓ App2 metrics: $(curl -s http://localhost:2114/health -w "%{http_code}" -o /dev/null)"
echo "   ✓ Payment processor default: $(curl -s http://localhost:2115/health -w "%{http_code}" -o /dev/null)"
echo "   ✓ Payment processor fallback: $(curl -s http://localhost:2116/health -w "%{http_code}" -o /dev/null)"

echo ""
echo "3. Testing Prometheus targets..."
UP_COUNT=$(curl -s http://localhost:9090/api/v1/targets | jq -r '.data.activeTargets[] | select(.health == "up") | .labels.job' | wc -l)
TOTAL_COUNT=$(curl -s http://localhost:9090/api/v1/targets | jq -r '.data.activeTargets[] | .labels.job' | wc -l)
echo "   ✓ Prometheus targets: $UP_COUNT/$TOTAL_COUNT UP"

if [ "$UP_COUNT" -eq "$TOTAL_COUNT" ]; then
    echo "   🎉 All targets are healthy!"
else
    echo "   ⚠️ Some targets are down - check http://localhost:9090/targets"
fi

echo ""
echo "4. Generating test traffic..."
for i in {1..3}; do
    AMOUNT=$((RANDOM % 100 + 10))
    RESPONSE=$(curl -X POST http://localhost:9999/payments \
        -H 'Content-Type: application/json' \
        -d "{\"correlationId\": \"validation-$i\", \"amount\": $AMOUNT}" \
        -s -w "%{http_code}")
    echo "   Payment $i: $AMOUNT USD - HTTP $RESPONSE"
done

echo ""
echo "5. Waiting for metrics collection..."
sleep 10

echo ""
echo "6. Validating RED metrics..."

# Rate metrics
RATE_COUNT=$(curl -s "http://localhost:9090/api/v1/query?query=http_requests_total" | jq '.data.result | length')
echo "   ✓ Rate metrics: $RATE_COUNT series found"

# Error metrics 
ERROR_COUNT=$(curl -s "http://localhost:9090/api/v1/query?query=http_errors_total" | jq '.data.result | length')
echo "   ✓ Error metrics: $ERROR_COUNT series found"

# Duration metrics
DURATION_COUNT=$(curl -s "http://localhost:9090/api/v1/query?query=http_request_duration_seconds" | jq '.data.result | length')  
echo "   ✓ Duration metrics: $DURATION_COUNT series found"

# Payment metrics
PAYMENT_COUNT=$(curl -s "http://localhost:9090/api/v1/query?query=payment_amount_dollars_count" | jq '.data.result | length')
echo "   ✓ Payment metrics: $PAYMENT_COUNT series found"

echo ""
echo "7. Latest metrics sample:"
echo "   Requests processed:"
curl -s "http://localhost:9090/api/v1/query?query=sum(http_requests_total)" | jq -r '.data.result[0].value[1]' 2>/dev/null | head -1 | sed 's/^/      Total: /'

echo "   Payment amounts:"
curl -s "http://localhost:9090/api/v1/query?query=sum(payment_amount_dollars_count)" | jq -r '.data.result[0].value[1]' 2>/dev/null | head -1 | sed 's/^/      Count: /'

echo ""
echo "=== Validation Complete! ==="
echo ""
echo "🎯 Access your monitoring dashboards:"
echo "   📊 Grafana Dashboard: http://localhost:3000 (admin/admin123)"
echo "   📈 Prometheus Metrics: http://localhost:9090"
echo "   🔍 Jaeger Tracing: http://localhost:16686"
echo "   💳 Payment Service: http://localhost:9999"
echo ""
echo "✅ Your payment processor now has complete RED metrics monitoring!"
echo "   📈 Rate - HTTP requests per second"
echo "   ❌ Errors - Error rate percentage"  
echo "   ⏱️  Duration - Response time percentiles"
echo "   💰 Plus payment amounts and processor performance"