#!/bin/bash

# TH Payment Processor Initialization Script
# Sets up and starts the complete payment processing environment

set -e  # Exit on any error

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

PROJECT_ROOT="$(dirname "$(pwd)")"  # Go back one level from scripts/
PROCESSORS_PATH="../payment-processors"
DEPLOYMENTS_PATH="${PROJECT_ROOT}/deployments"

echo -e "${BLUE}🚀 Initializing TH Payment Processor Environment...${NC}"

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to wait for service to be ready
wait_for_service() {
    local url="$1"
    local service_name="$2"
    local max_attempts=30
    local attempt=1
    
    echo -e "${YELLOW}⏳ Waiting for ${service_name} to be ready...${NC}"
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s "$url" >/dev/null 2>&1; then
            echo -e "${GREEN}✅ ${service_name} is ready!${NC}"
            return 0
        fi
        
        echo -n "."
        sleep 2
        attempt=$((attempt + 1))
    done
    
    echo -e "\n${RED}❌ ${service_name} failed to start within timeout${NC}"
    return 1
}

# Step 1: Verify prerequisites
echo -e "\n${YELLOW}📋 Checking prerequisites...${NC}"

if ! command_exists docker; then
    echo -e "${RED}❌ Docker is not installed${NC}"
    exit 1
fi

if ! docker compose version >/dev/null 2>&1; then
    echo -e "${RED}❌ Docker Compose is not installed${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Docker and Docker Compose are available${NC}"

# Step 2: Check if payment processors directory exists
if [ ! -d "$PROCESSORS_PATH" ]; then
    echo -e "${RED}❌ Payment processors directory not found at: $PROCESSORS_PATH${NC}"
    echo -e "${YELLOW}Please ensure the payment-processors project is in the parent directory${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Payment processors directory found${NC}"

# Step 3: Start payment processors first (IMPORTANT: correct sequence)
echo -e "\n${YELLOW}🏭 Starting Payment Processors...${NC}"
cd "$PROCESSORS_PATH"

# Stop existing processors if running
docker compose down >/dev/null 2>&1 || true

# Start processors
if ! docker compose up -d; then
    echo -e "${RED}❌ Failed to start payment processors${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Payment processors starting...${NC}"

# Step 4: Wait for processors to be ready
wait_for_service "http://localhost:8001/payments/service-health" "Default Payment Processor"
wait_for_service "http://localhost:8002/payments/service-health" "Fallback Payment Processor"

# Step 5: Start backend application
echo -e "\n${YELLOW}🏗️ Starting TH Payment Processor...${NC}"
cd "$DEPLOYMENTS_PATH"

# Stop existing backend if running
docker compose down >/dev/null 2>&1 || true

# Build and start backend
if ! docker compose up -d --build; then
    echo -e "${RED}❌ Failed to start rinha backend${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Rinha backend starting...${NC}"

# Step 6: Wait for backend to be ready
wait_for_service "http://localhost:9999/payments-summary" "TH Payment Processor"

# Step 7: Verify all services are running
echo -e "\n${YELLOW}🔍 Verifying services...${NC}"

echo "Backend services:"
cd "$DEPLOYMENTS_PATH" && docker compose ps

echo -e "\nPayment processor services:"
cd "$PROCESSORS_PATH"
cd "$DEPLOYMENTS_PATH" && docker compose ps

# Step 8: Run basic health checks
echo -e "\n${YELLOW}🩺 Running health checks...${NC}"
cd "$PROJECT_ROOT"

echo "Backend payments summary:"
if curl -s "http://localhost:9999/payments-summary" | head -1; then
    echo -e "${GREEN}✅ Backend is responding${NC}"
else
    echo -e "${RED}❌ Backend health check failed${NC}"
fi

echo -e "\nDefault processor health:"
if curl -s "http://localhost:8001/payments/service-health" | head -1; then
    echo -e "${GREEN}✅ Default processor is responding${NC}"
else
    echo -e "${RED}❌ Default processor health check failed${NC}"
fi

echo -e "\nFallback processor health:"
if curl -s "http://localhost:8002/payments/service-health" | head -1; then
    echo -e "${GREEN}✅ Fallback processor is responding${NC}"
else
    echo -e "${RED}❌ Fallback processor health check failed${NC}"
fi

# Step 9: Test basic payment processing
echo -e "\n${YELLOW}💳 Testing basic payment processing...${NC}"

TEST_RESPONSE=$(curl -s -w "%{http_code}" -X POST "http://localhost:9999/payments" \
  -H "Content-Type: application/json" \
  -d '{
    "correlationId": "init-test-001",
    "amount": 10.00
  }')

HTTP_CODE="${TEST_RESPONSE: -3}"
if [ "$HTTP_CODE" -ge 200 ] && [ "$HTTP_CODE" -lt 300 ]; then
    echo -e "${GREEN}✅ Payment processing test successful${NC}"
else
    echo -e "${RED}❌ Payment processing test failed (HTTP $HTTP_CODE)${NC}"
fi

# Step 10: Display final status
echo -e "\n${GREEN}🎉 Initialization Complete!${NC}"
echo -e "\n${BLUE}📋 Service Endpoints:${NC}"
echo "• Backend API: http://localhost:9999"
echo "• Default Processor: http://localhost:8001"  
echo "• Fallback Processor: http://localhost:8002"

echo -e "\n${BLUE}🧪 Test Commands:${NC}"
echo "• Run payment tests: ./test_payments.sh"
echo "• Run processor tests: ./test_processors.sh"
echo "• Manual payment test:"
echo '  curl -X POST http://localhost:9999/payments \'
echo '    -H "Content-Type: application/json" \'
echo '    -d '"'"'{"correlationId":"test-123","amount":100.00}'"'"

echo -e "\n${BLUE}📊 Monitor Services:${NC}"
echo "• Backend logs: cd deployments && docker compose logs -f"
echo "• Processor logs: cd ../payment-processors && docker compose logs -f"
echo "• Payments summary: curl http://localhost:9999/payments-summary"

echo -e "\n${YELLOW}💡 Tip: Use ./cleanup.sh to stop all services when done${NC}"