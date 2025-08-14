#!/bin/bash

echo "Testing individual service metrics endpoints..."
echo ""

echo "1. Building main application..."
go build -o build/th_payment_processor ./cmd/server

echo "2. Building payment processors..."
cd payment-processors && go build -o payment-processor . && cd ..

echo ""
echo "3. Starting services in background for metrics test..."
cd deployments

echo "Starting docker-compose services..."
docker-compose up -d

echo "Waiting 45 seconds for services to initialize..."
sleep 45

echo ""
echo "4. Testing metrics endpoints:"

echo "- App1 metrics (via docker network):"
docker exec deployments-prometheus-1 wget -qO- http://app1:2112/metrics | head -3

echo "- App2 metrics (via docker network):"  
docker exec deployments-prometheus-1 wget -qO- http://app2:2112/metrics | head -3

echo "- Payment Processor Default metrics:"
docker exec deployments-prometheus-1 wget -qO- http://payment-processor-default:2112/metrics | head -3

echo "- Payment Processor Fallback metrics:"
docker exec deployments-prometheus-1 wget -qO- http://payment-processor-fallback:2112/metrics | head -3

echo ""
echo "5. Testing Prometheus targets:"
curl -s http://localhost:9090/api/v1/targets | jq '.data.activeTargets[] | {job: .labels.job, health: .health, lastError: .lastError}'

echo ""
echo "6. Making sample payment to generate metrics:"
curl -X POST http://localhost:9999/payments \
  -H 'Content-Type: application/json' \
  -d '{"correlationId": "test-metrics-123", "amount": 50.00}' \
  -w "\nHTTP Status: %{http_code}\n"

sleep 10

echo ""
echo "7. Checking for payment metrics in Prometheus:"
curl -s "http://localhost:9090/api/v1/query?query=http_requests_total" | jq '.data.result[] | {metric: .metric, value: .value[1]}'

echo ""
echo "Setup complete! Access monitoring at:"
echo "- Grafana: http://localhost:3000 (admin/admin123)"
echo "- Prometheus: http://localhost:9090"
echo "- Jaeger: http://localhost:16686"
echo ""
echo "To stop: docker-compose down"