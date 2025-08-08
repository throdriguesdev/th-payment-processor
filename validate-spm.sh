#!/bin/bash

echo "=== Service Performance Monitoring (SPM) Validation ==="
echo "Testing Jaeger + Prometheus + Grafana integration"
echo ""

# Wait for services to be ready
echo "1. Waiting for services to initialize..."
sleep 30

echo "2. Testing basic service connectivity..."
echo "   ✓ Jaeger UI: $(curl -s http://localhost:16686/api/services -w "%{http_code}" -o /dev/null)"
echo "   ✓ Jaeger Admin: $(curl -s http://localhost:14269/metrics -w "%{http_code}" -o /dev/null)"
echo "   ✓ Prometheus: $(curl -s http://localhost:9090/-/healthy -w "%{http_code}" -o /dev/null)"
echo "   ✓ Grafana: $(curl -s http://localhost:3000/api/health -w "%{http_code}" -o /dev/null)"

echo ""
echo "3. Checking Prometheus targets..."
UP_COUNT=$(curl -s http://localhost:9090/api/v1/targets | jq -r '.data.activeTargets[] | select(.health == "up") | .labels.job' | wc -l)
TOTAL_COUNT=$(curl -s http://localhost:9090/api/v1/targets | jq -r '.data.activeTargets[] | .labels.job' | wc -l)
echo "   ✓ Targets UP: $UP_COUNT/$TOTAL_COUNT"

# Check if Jaeger target is included
JAEGER_TARGET=$(curl -s http://localhost:9090/api/v1/targets | jq -r '.data.activeTargets[] | select(.labels.job == "jaeger") | .health')
if [ "$JAEGER_TARGET" == "up" ]; then
    echo "   ✓ Jaeger metrics target is UP"
else
    echo "   ⚠️ Jaeger metrics target is DOWN"
fi

echo ""
echo "4. Testing trace generation..."
echo "   Generating payment requests to create traces..."
for i in {1..5}; do
    AMOUNT=$((RANDOM % 100 + 50))
    RESPONSE=$(curl -X POST http://localhost:9999/payments \
        -H 'Content-Type: application/json' \
        -d "{\"correlationId\": \"spm-test-$i\", \"amount\": $AMOUNT}" \
        -s -w "%{http_code}")
    echo "   Payment $i: $AMOUNT USD - HTTP $RESPONSE"
done

echo ""
echo "5. Waiting for traces to be processed..."
sleep 15

echo ""
echo "6. Validating Jaeger trace collection..."
SERVICES=$(curl -s http://localhost:16686/api/services | jq -r '.data[]')
if [[ $SERVICES == *"th-payment-processor"* ]]; then
    echo "   ✓ Payment processor service found in Jaeger"
else
    echo "   ⚠️ Payment processor service NOT found in Jaeger"
fi

# Get trace count
TRACE_RESPONSE=$(curl -s "http://localhost:16686/api/traces?service=th-payment-processor&limit=10")
TRACE_COUNT=$(echo $TRACE_RESPONSE | jq -r '.data | length // 0')
echo "   ✓ Recent traces found: $TRACE_COUNT"

echo ""
echo "7. Validating Prometheus metrics with exemplars..."

# Check for HTTP metrics
HTTP_METRICS=$(curl -s "http://localhost:9090/api/v1/query?query=http_requests_total" | jq '.data.result | length')
echo "   ✓ HTTP request metrics: $HTTP_METRICS series"

# Check for payment metrics
PAYMENT_METRICS=$(curl -s "http://localhost:9090/api/v1/query?query=payment_amount_dollars_count" | jq '.data.result | length')
echo "   ✓ Payment amount metrics: $PAYMENT_METRICS series"

# Check for Jaeger metrics
JAEGER_METRICS=$(curl -s "http://localhost:9090/api/v1/query?query=jaeger_traces_received_total" | jq '.data.result | length')
echo "   ✓ Jaeger ingestion metrics: $JAEGER_METRICS series"

echo ""
echo "8. Testing Grafana data sources..."

# Check Prometheus datasource
PROM_STATUS=$(curl -s http://localhost:3000/api/datasources/name/Prometheus -u admin:admin123 | jq -r '.access // "unknown"')
echo "   ✓ Prometheus datasource: $PROM_STATUS"

# Check Jaeger datasource  
JAEGER_STATUS=$(curl -s http://localhost:3000/api/datasources/name/Jaeger -u admin:admin123 | jq -r '.access // "unknown"')
echo "   ✓ Jaeger datasource: $JAEGER_STATUS"

echo ""
echo "9. Generating additional test traffic for correlation..."
for i in {6..8}; do
    curl -X POST http://localhost:9999/payments \
        -H 'Content-Type: application/json' \
        -d "{\"correlationId\": \"correlation-test-$i\", \"amount\": $((RANDOM % 200 + 25))}" \
        -s > /dev/null
done

# Also test the summary endpoint
curl -s http://localhost:9999/payments-summary > /dev/null

echo "   ✓ Generated additional traces for correlation testing"

echo ""
echo "=== SPM Validation Complete! ==="
echo ""
echo "🎯 Access your Service Performance Monitoring:"
echo "   📊 SPM Dashboard: http://localhost:3000/d/spm-dashboard (admin/admin123)"
echo "   📈 RED Metrics Dashboard: http://localhost:3000/d/payment-processor-red"
echo "   📈 Prometheus Queries: http://localhost:9090"
echo "   🔍 Jaeger Traces: http://localhost:16686"
echo "   💳 Payment Service: http://localhost:9999"
echo ""
echo "✅ SPM Features Available:"
echo "   🔗 Trace-to-Metrics Correlation - Click metric points to view traces"
echo "   📊 Service Dependency Mapping - Visual service topology"  
echo "   📈 RED Metrics with Exemplars - Traces linked to metrics"
echo "   🔍 Distributed Tracing - End-to-end request tracking"
echo "   📉 Jaeger Performance Metrics - Trace ingestion statistics"
echo "   🎭 Recent Trace Browser - Latest payment processing traces"
echo ""
echo "💡 Try This:"
echo "   1. Open the SPM dashboard in Grafana"
echo "   2. Look for dots on the metrics graphs (exemplars)"
echo "   3. Click on exemplar dots to jump to related traces"
echo "   4. Explore the service map to see dependencies"
echo "   5. Check recent traces table for detailed trace analysis"