#!/bin/bash

echo "=== Payment Processor Monitoring Stack Test ==="
echo "This script will start the services and test the monitoring setup"
echo ""

# Start the services
echo "Starting services with docker-compose..."
docker compose up -d

echo "Waiting for services to be ready..."
sleep 30

echo "Testing service endpoints:"
echo ""

# Test main services
echo "1. Testing main payment service health..."
curl -s http://localhost:9999/health | jq '.' 2>/dev/null || echo "Service not ready"

echo ""
echo "2. Testing Prometheus metrics endpoints:"
echo "   - App1 metrics:"
curl -s http://localhost:9999/metrics | head -5 2>/dev/null || echo "Metrics not available"

echo ""
echo "3. Testing Prometheus UI:"
curl -s http://localhost:9090/-/healthy 2>/dev/null && echo "Prometheus is healthy" || echo "Prometheus not ready"

echo ""
echo "4. Testing Grafana UI:"
curl -s http://localhost:3000/api/health 2>/dev/null | jq '.' 2>/dev/null || echo "Grafana not ready"

echo ""
echo "5. Testing Jaeger UI:"
curl -s http://localhost:16686/api/services 2>/dev/null | jq '.' 2>/dev/null || echo "Jaeger not ready"

echo ""
echo "=== Access URLs ==="
echo "Grafana Dashboard: http://localhost:3000 (admin/admin123)"
echo "Prometheus: http://localhost:9090"
echo "Jaeger Tracing: http://localhost:16686"
echo "Payment Service: http://localhost:9999"
echo ""

echo "=== Sample Payment Request ==="
echo "curl -X POST http://localhost:9999/payments \\"
echo "  -H 'Content-Type: application/json' \\"
echo "  -d '{\"correlationId\": \"test-123\", \"amount\": 100.50}'"
echo ""

echo "To stop all services: docker-compose down"
