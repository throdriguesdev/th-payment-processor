#!/bin/bash

# Cleanup Script for Rinha Backend Environment
# Stops all services and cleans up Docker resources

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

PROJECT_ROOT="$(pwd)"
PROCESSORS_PATH="../payment-processors"

echo -e "${BLUE}🧹 Cleaning up Rinha Backend Environment...${NC}"

# Function to safely run commands
safe_run() {
    local cmd="$1"
    local description="$2"
    
    echo -e "${YELLOW}$description...${NC}"
    if eval "$cmd" >/dev/null 2>&1; then
        echo -e "${GREEN}✅ $description completed${NC}"
    else
        echo -e "${YELLOW}⚠️ $description had issues (may already be stopped)${NC}"
    fi
}

# Stop backend services
echo -e "\n${YELLOW}🛑 Stopping Rinha Backend services...${NC}"
cd "$PROJECT_ROOT"
safe_run "docker compose down" "Stopping backend containers"
safe_run "docker compose down --volumes" "Removing backend volumes"

# Stop payment processor services
if [ -d "$PROCESSORS_PATH" ]; then
    echo -e "\n${YELLOW}🛑 Stopping Payment Processor services...${NC}"
    cd "$PROCESSORS_PATH"
    safe_run "docker compose down" "Stopping processor containers"
    safe_run "docker compose down --volumes" "Removing processor volumes"
else
    echo -e "${YELLOW}⚠️ Payment processors directory not found, skipping...${NC}"
fi

# Return to project root
cd "$PROJECT_ROOT"

# Optional: Clean up Docker resources (commented out by default)
echo -e "\n${YELLOW}🗑️ Optional cleanup (uncomment if needed):${NC}"
echo "# Remove unused Docker images:"
echo "# docker image prune -f"
echo "# Remove unused Docker networks:"  
echo "# docker network prune -f"
echo "# Remove all stopped containers:"
echo "# docker container prune -f"

# Show remaining Docker processes
echo -e "\n${BLUE}📋 Remaining Docker processes:${NC}"
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

# Check if any rinha-related containers are still running
RUNNING_CONTAINERS=$(docker ps --filter "name=rinha" --filter "name=payment-processor" --format "{{.Names}}" | wc -l)

if [ "$RUNNING_CONTAINERS" -eq 0 ]; then
    echo -e "\n${GREEN}✅ All Rinha services stopped successfully!${NC}"
else
    echo -e "\n${YELLOW}⚠️ Some containers may still be running. Check with: docker ps${NC}"
fi

echo -e "\n${BLUE}💡 To restart services, run: ./init.sh${NC}"