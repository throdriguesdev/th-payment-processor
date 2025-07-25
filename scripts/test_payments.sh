#!/bin/bash

# Payment Processing Test Suite
# Tests the TH Payment Processor functionality

BASE_URL="http://localhost:9999"
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🧪 Starting Payment Processing Tests...${NC}"

# Test 1: Basic Payment Processing
echo -e "\n${YELLOW}Test 1: Basic Payment Processing${NC}"
curl -X POST "${BASE_URL}/payments" \
  -H "Content-Type: application/json" \
  -d '{
    "correlationId": "test-payment-001",
    "amount": 100.50
  }' \
  -w "\nStatus: %{http_code}\nTime: %{time_total}s\n"

# Test 2: Multiple Payments with Different Amounts
echo -e "\n${YELLOW}Test 2: Multiple Payments${NC}"
for i in {1..5}; do
  echo "Payment $i:"
  curl -X POST "${BASE_URL}/payments" \
    -H "Content-Type: application/json" \
    -d "{
      \"correlationId\": \"test-payment-$(printf "%03d" $i)\",
      \"amount\": $((i * 25)).99
    }" \
    -w "Status: %{http_code}, Time: %{time_total}s\n" \
    -s -o /dev/null
done

# Test 3: Invalid Payment Requests
echo -e "\n${YELLOW}Test 3: Invalid Payment Requests${NC}"

echo "Missing correlationId:"
curl -X POST "${BASE_URL}/payments" \
  -H "Content-Type: application/json" \
  -d '{"amount": 50.00}' \
  -w "Status: %{http_code}\n" \
  -s -o /dev/null

echo "Missing amount:"
curl -X POST "${BASE_URL}/payments" \
  -H "Content-Type: application/json" \
  -d '{"correlationId": "test-missing-amount"}' \
  -w "Status: %{http_code}\n" \
  -s -o /dev/null

echo "Zero amount:"
curl -X POST "${BASE_URL}/payments" \
  -H "Content-Type: application/json" \
  -d '{"correlationId": "test-zero-amount", "amount": 0}' \
  -w "Status: %{http_code}\n" \
  -s -o /dev/null

echo "Negative amount:"
curl -X POST "${BASE_URL}/payments" \
  -H "Content-Type: application/json" \
  -d '{"correlationId": "test-negative-amount", "amount": -10.00}' \
  -w "Status: %{http_code}\n" \
  -s -o /dev/null

# Test 4: Payment Summary
echo -e "\n${YELLOW}Test 4: Payment Summary${NC}"
echo "Getting all payments summary:"
curl "${BASE_URL}/payments-summary" \
  -H "Accept: application/json" \
  -w "\nStatus: %{http_code}\nTime: %{time_total}s\n"

# Test 5: Payment Summary with Time Filtering
echo -e "\n${YELLOW}Test 5: Payment Summary with Time Filters${NC}"
NOW=$(date -u +"%Y-%m-%dT%H:%M:%S.000Z")
YESTERDAY=$(date -u -d "yesterday" +"%Y-%m-%dT%H:%M:%S.000Z")

echo "Getting payments from yesterday to now:"
curl "${BASE_URL}/payments-summary?from=${YESTERDAY}&to=${NOW}" \
  -H "Accept: application/json" \
  -w "\nStatus: %{http_code}\nTime: %{time_total}s\n"

# Test 6: Load Testing (burst of payments)
echo -e "\n${YELLOW}Test 6: Load Testing (10 concurrent payments)${NC}"
for i in {1..10}; do
  (
    curl -X POST "${BASE_URL}/payments" \
      -H "Content-Type: application/json" \
      -d "{
        \"correlationId\": \"load-test-$(date +%s%N)-${i}\",
        \"amount\": 99.99
      }" \
      -w "Payment $i - Status: %{http_code}, Time: %{time_total}s\n" \
      -s -o /dev/null
  ) &
done
wait

# Test 7: Performance Test (measure response times)
echo -e "\n${YELLOW}Test 7: Performance Test (response times)${NC}"
echo "Measuring response times for 20 payments:"
for i in {1..20}; do
  TIME=$(curl -X POST "${BASE_URL}/payments" \
    -H "Content-Type: application/json" \
    -d "{
      \"correlationId\": \"perf-test-$(date +%s%N)-${i}\",
      \"amount\": 50.00
    }" \
    -w "%{time_total}" \
    -s -o /dev/null)
  echo "Payment $i: ${TIME}s"
done

echo -e "\n${GREEN}✅ All tests completed!${NC}"
echo -e "Check the payments summary to verify all payments were processed:"
echo "curl ${BASE_URL}/payments-summary"