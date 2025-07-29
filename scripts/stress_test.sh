#!/bin/bash

# Stress Test Script for Rinha Backend
# Tests performance under load to validate p99 < 11ms target

BASE_URL="http://localhost:9999"
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

CONCURRENT_USERS=${1:-10}
REQUESTS_PER_USER=${2:-100}
TOTAL_REQUESTS=$((CONCURRENT_USERS * REQUESTS_PER_USER))

echo -e "${BLUE}🚀 Stress Testing Rinha Backend${NC}"
echo -e "${YELLOW}Configuration:${NC}"
echo "• Concurrent Users: $CONCURRENT_USERS"
echo "• Requests per User: $REQUESTS_PER_USER"
echo "• Total Requests: $TOTAL_REQUESTS"
echo "• Target: p99 < 11ms"

# Function to generate UUID-like string
generate_uuid() {
    cat /proc/sys/kernel/random/uuid 2>/dev/null || date +%s%N | sha256sum | cut -c1-32
}

# Function to run load test for one user
run_user_load() {
    local user_id="$1"
    local requests="$2"
    local results_file="/tmp/stress_test_user_${user_id}.log"
    
    > "$results_file"  # Clear file
    
    for i in $(seq 1 "$requests"); do
        correlation_id="stress-test-${user_id}-$(printf "%04d" $i)-$(date +%s%N)"
        amount=$(awk "BEGIN {printf \"%.2f\", ($i % 1000) + 1}")
        
        response_time=$(curl -X POST "${BASE_URL}/payments" \
            -H "Content-Type: application/json" \
            -d "{
                \"correlationId\": \"$correlation_id\",
                \"amount\": $amount
            }" \
            -w "%{time_total}" \
            -s -o /dev/null)
        
        # Convert to milliseconds and log
        ms_time=$(awk "BEGIN {printf \"%.2f\", $response_time * 1000}")
        echo "$ms_time" >> "$results_file"
    done
}

# Pre-test: Verify service is up
echo -e "\n${YELLOW}🔍 Verifying service availability...${NC}"
if ! curl -s "${BASE_URL}/payments-summary" >/dev/null; then
    echo -e "${RED}❌ Service is not available at ${BASE_URL}${NC}"
    echo "Please run ./init.sh first to start the services"
    exit 1
fi
echo -e "${GREEN}✅ Service is available${NC}"

# Clear any existing test files
rm -f /tmp/stress_test_user_*.log

# Run the stress test
echo -e "\n${YELLOW}⚡ Starting stress test...${NC}"
START_TIME=$(date +%s)

# Launch concurrent users
for user_id in $(seq 1 "$CONCURRENT_USERS"); do
    echo "Starting user $user_id..."
    run_user_load "$user_id" "$REQUESTS_PER_USER" &
done

echo -e "${YELLOW}⏳ Waiting for all users to complete...${NC}"
wait

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

# Aggregate results
echo -e "\n${YELLOW}📊 Analyzing results...${NC}"

# Combine all result files
cat /tmp/stress_test_user_*.log > /tmp/all_results.log

# Calculate statistics
TOTAL_ACTUAL=$(wc -l < /tmp/all_results.log)
MIN_TIME=$(sort -n /tmp/all_results.log | head -1)
MAX_TIME=$(sort -n /tmp/all_results.log | tail -1)
AVG_TIME=$(awk '{sum+=$1} END {print sum/NR}' /tmp/all_results.log)

# Calculate percentiles
P50=$(sort -n /tmp/all_results.log | awk -v p=50 'BEGIN{c=0} {all[c++]=$1} END{print all[int(c*p/100)]}')
P95=$(sort -n /tmp/all_results.log | awk -v p=95 'BEGIN{c=0} {all[c++]=$1} END{print all[int(c*p/100)]}')
P99=$(sort -n /tmp/all_results.log | awk -v p=99 'BEGIN{c=0} {all[c++]=$1} END{print all[int(c*p/100)]}')

# Display results
echo -e "\n${BLUE}📈 Stress Test Results${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Duration: ${DURATION}s"
echo "Total Requests: $TOTAL_ACTUAL"
if [ "$DURATION" -gt 0 ]; then
    echo "Requests/second: $(awk "BEGIN {printf \"%.2f\", $TOTAL_ACTUAL / $DURATION}")"
else
    echo "Requests/second: $(awk "BEGIN {printf \"%.2f\", $TOTAL_ACTUAL}")"
fi
echo ""
echo "Response Times (ms):"
echo "• Min:  $(printf '%.2f' "$MIN_TIME")"
echo "• Avg:  $(printf '%.2f' "$AVG_TIME")"
echo "• Max:  $(printf '%.2f' "$MAX_TIME")"
echo "• P50:  $(printf '%.2f' "$P50")"
echo "• P95:  $(printf '%.2f' "$P95")"
echo "• P99:  $(printf '%.2f' "$P99")"

# Performance evaluation
echo ""
# Use awk for floating point comparison to avoid bc issues
if awk "BEGIN {exit !($P99 < 11)}"; then
    BONUS=$(awk "BEGIN {printf \"%.2f\", (11 - $P99) * 0.02 * 100}")
    echo -e "${GREEN}🎉 EXCELLENT! P99 < 11ms target achieved!${NC}"
    echo -e "${GREEN}Performance Bonus: ${BONUS}%${NC}"
elif awk "BEGIN {exit !($P99 < 20)}"; then
    echo -e "${YELLOW}✅ Good performance (P99 < 20ms)${NC}"
else
    echo -e "${RED}⚠️ Performance needs improvement (P99 >= 20ms)${NC}"
fi

# Error rate check - simplified approach
ERROR_COUNT=$(find /tmp -name "stress_test_user_*.log" -exec grep -c "error\|Error\|ERROR" {} \; 2>/dev/null | awk '{sum+=$1} END {print sum+0}')

if [ "$TOTAL_ACTUAL" -gt 0 ]; then
    ERROR_RATE=$(awk "BEGIN {printf \"%.4f\", $ERROR_COUNT / $TOTAL_ACTUAL * 100}")
else
    ERROR_RATE="0.0000"
fi
echo "Error Rate: ${ERROR_RATE}%"

# Final summary
echo -e "\n${YELLOW}📋 Final Payment Summary:${NC}"
curl -s "${BASE_URL}/payments-summary" | jq '.' 2>/dev/null || curl -s "${BASE_URL}/payments-summary"

# Cleanup temp files
rm -f /tmp/stress_test_user_*.log /tmp/all_results.log

echo -e "\n${GREEN}✅ Stress test completed!${NC}"

# Usage examples
echo -e "\n${BLUE}💡 Usage Examples:${NC}"
echo "• Default test:     ./stress_test.sh"
echo "• High load test:   ./stress_test.sh 50 200"
echo "• Quick test:       ./stress_test.sh 5 10"