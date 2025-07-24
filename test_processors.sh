#!/bin/bash

# Payment Processors Integration Tests
# Tests integration with both default and fallback payment processors

DEFAULT_PROCESSOR="http://localhost:8001"
FALLBACK_PROCESSOR="http://localhost:8002"
ADMIN_TOKEN="123"  # Default admin token
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}🔌 Testing Payment Processors Integration...${NC}"

# Test 1: Check Processor Health
echo -e "\n${YELLOW}Test 1: Payment Processor Health Checks${NC}"

echo "Default Processor Health:"
curl "${DEFAULT_PROCESSOR}/payments/service-health" \
  -H "Accept: application/json" \
  -w "\nStatus: %{http_code}\nTime: %{time_total}s\n"

echo -e "\nFallback Processor Health:"
curl "${FALLBACK_PROCESSOR}/payments/service-health" \
  -H "Accept: application/json" \
  -w "\nStatus: %{http_code}\nTime: %{time_total}s\n"

# Test 2: Direct Payment to Processors
echo -e "\n${YELLOW}Test 2: Direct Payment Processing${NC}"

echo "Sending payment to Default Processor:"
curl -X POST "${DEFAULT_PROCESSOR}/payments" \
  -H "Content-Type: application/json" \
  -d '{
    "correlationId": "direct-default-001",
    "amount": 100.00,
    "requestedAt": "'$(date -u +"%Y-%m-%dT%H:%M:%S.000Z")'"
  }' \
  -w "\nStatus: %{http_code}\nTime: %{time_total}s\n"

echo -e "\nSending payment to Fallback Processor:"
curl -X POST "${FALLBACK_PROCESSOR}/payments" \
  -H "Content-Type: application/json" \
  -d '{
    "correlationId": "direct-fallback-001", 
    "amount": 100.00,
    "requestedAt": "'$(date -u +"%Y-%m-%dT%H:%M:%S.000Z")'"
  }' \
  -w "\nStatus: %{http_code}\nTime: %{time_total}s\n"

# Test 3: Admin Endpoints (for testing scenarios)
echo -e "\n${YELLOW}Test 3: Admin Endpoints${NC}"

echo "Default Processor Summary:"
curl "${DEFAULT_PROCESSOR}/admin/payments-summary" \
  -H "X-Rinha-Token: ${ADMIN_TOKEN}" \
  -w "\nStatus: %{http_code}\n"

echo -e "\nFallback Processor Summary:"
curl "${FALLBACK_PROCESSOR}/admin/payments-summary" \
  -H "X-Rinha-Token: ${ADMIN_TOKEN}" \
  -w "\nStatus: %{http_code}\n"

# Test 4: Failure Simulation
echo -e "\n${YELLOW}Test 4: Failure Simulation${NC}"

echo "Setting Default Processor to failure mode:"
curl -X PUT "${DEFAULT_PROCESSOR}/admin/configurations/failure" \
  -H "Content-Type: application/json" \
  -H "X-Rinha-Token: ${ADMIN_TOKEN}" \
  -d '{"failure": true}' \
  -w "Status: %{http_code}\n" \
  -s -o /dev/null

echo "Testing payment with default processor failing (should fail):"
curl -X POST "${DEFAULT_PROCESSOR}/payments" \
  -H "Content-Type: application/json" \
  -d '{
    "correlationId": "test-failure-001",
    "amount": 50.00,
    "requestedAt": "'$(date -u +"%Y-%m-%dT%H:%M:%S.000Z")'"
  }' \
  -w "Status: %{http_code}\n" \
  -s -o /dev/null

echo "Restoring Default Processor to normal mode:"
curl -X PUT "${DEFAULT_PROCESSOR}/admin/configurations/failure" \
  -H "Content-Type: application/json" \
  -H "X-Rinha-Token: ${ADMIN_TOKEN}" \
  -d '{"failure": false}' \
  -w "Status: %{http_code}\n" \
  -s -o /dev/null

# Test 5: Delay Simulation
echo -e "\n${YELLOW}Test 5: Delay Simulation${NC}"

echo "Setting 2000ms delay on Default Processor:"
curl -X PUT "${DEFAULT_PROCESSOR}/admin/configurations/delay" \
  -H "Content-Type: application/json" \
  -H "X-Rinha-Token: ${ADMIN_TOKEN}" \
  -d '{"delay": 2000}' \
  -w "Status: %{http_code}\n" \
  -s -o /dev/null

echo "Testing payment with delay (should take ~2+ seconds):"
curl -X POST "${DEFAULT_PROCESSOR}/payments" \
  -H "Content-Type: application/json" \
  -d '{
    "correlationId": "test-delay-001",
    "amount": 75.00,
    "requestedAt": "'$(date -u +"%Y-%m-%dT%H:%M:%S.000Z")'"
  }' \
  -w "Status: %{http_code}, Time: %{time_total}s\n"

echo "Removing delay from Default Processor:"
curl -X PUT "${DEFAULT_PROCESSOR}/admin/configurations/delay" \
  -H "Content-Type: application/json" \
  -H "X-Rinha-Token: ${ADMIN_TOKEN}" \
  -d '{"delay": 0}' \
  -w "Status: %{http_code}\n" \
  -s -o /dev/null

# Test 6: Rate Limiting Test
echo -e "\n${YELLOW}Test 6: Health Check Rate Limiting${NC}"
echo "Testing health check rate limiting (should get 429 after first call):"

for i in {1..3}; do
  echo "Health check attempt $i:"
  curl "${DEFAULT_PROCESSOR}/payments/service-health" \
    -w "Status: %{http_code}\n" \
    -s -o /dev/null
  sleep 1
done

echo -e "\n${GREEN}✅ Processor integration tests completed!${NC}"