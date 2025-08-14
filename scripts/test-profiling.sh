#!/bin/bash

set -e

echo "🔥 Testing Pyroscope Profiling Integration"
echo "==========================================="

# Generate some load to create profiling data
echo "📊 Generating load for profiling data..."
for i in {1..20}; do
    CORRELATION_ID="prof-load-$(date +%s)-$i"
    curl -s -X POST http://localhost:9999/payments \
         -H "Content-Type: application/json" \
         -H "X-Correlation-ID: $CORRELATION_ID" \
         -d "{\"correlationId\":\"$CORRELATION_ID\",\"amount\":$(( (RANDOM % 1000) + 100 )).$(( RANDOM % 100 ))\",\"userId\":\"load-test-$i\"}" \
         > /dev/null
    if [ $((i % 5)) -eq 0 ]; then
        echo "  ✓ Processed $i payments..."
    fi
    sleep 0.5
done

echo "✅ Generated 20 payment requests for profiling"
echo ""

# Wait for profiling data to be collected
echo "⏳ Waiting for profiling data collection (30 seconds)..."
sleep 30

# Check available applications in Pyroscope
echo "📈 Checking Pyroscope applications..."
APPS_JSON=$(curl -s "http://localhost:4040/render?format=json&query=th-payment-processor-app1%7Benvironment%3Ddevelopment%7D&from=now-1h&until=now")

if echo "$APPS_JSON" | jq '.flamebearer.numTicks' > /dev/null 2>&1; then
    TICKS=$(echo "$APPS_JSON" | jq '.flamebearer.numTicks')
    echo "  ✓ Profile data available: $TICKS ticks"
    
    if [ "$TICKS" -gt 0 ]; then
        echo "  ✅ CPU profiling data is being collected!"
    else
        echo "  ⚠️  No CPU profiling data yet, trying memory profiling..."
        
        # Try memory profiling
        MEM_JSON=$(curl -s "http://localhost:4040/render?format=json&query=th-payment-processor-app1%7Benvironment%3Ddevelopment%7D.inuse_objects&from=now-1h&until=now")
        MEM_TICKS=$(echo "$MEM_JSON" | jq '.flamebearer.numTicks' 2>/dev/null || echo "0")
        
        if [ "$MEM_TICKS" -gt 0 ]; then
            echo "  ✅ Memory profiling data is being collected: $MEM_TICKS ticks"
        else
            echo "  ⚠️  No profiling data found. Let's check what's available..."
        fi
    fi
else
    echo "  ❌ Error parsing Pyroscope response"
fi

echo ""

# Check if Grafana can connect to Pyroscope
echo "🔍 Testing Grafana datasource connection..."
GRAFANA_RESPONSE=$(curl -s -u admin:admin123 "http://localhost:3000/api/datasources/uid/pyroscope/proxy/api/v1/label-values?label=__name__")

if echo "$GRAFANA_RESPONSE" | grep -q "error"; then
    echo "  ❌ Grafana datasource connection failed"
    echo "  Response: $GRAFANA_RESPONSE"
else
    echo "  ✅ Grafana can connect to Pyroscope datasource"
fi

echo ""
echo "🚀 Profiling Integration Status:"
echo "  • Pyroscope Server: http://localhost:4040"
echo "  • Grafana Explore: http://localhost:3000/explore?panes=%7B%22aN4%22:%7B%22datasource%22:%22pyroscope%22%7D%7D"
echo "  • Profile Query: th-payment-processor-app1{environment=\"development\"}"
echo ""
echo "💡 To view profiles:"
echo "  1. Go to http://localhost:4040 (Pyroscope UI)"
echo "  2. Or use Grafana Explore with Pyroscope datasource"
echo "  3. Query: th-payment-processor-app1{environment=\"development\"}"
echo ""