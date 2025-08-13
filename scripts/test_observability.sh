#!/bin/bash

# Observability Testing Script for Payment Processor
# This script generates various test requests to verify structured logging and tracing

set -e

# Configuration
BASE_URL="http://localhost:8080"
TIMESTAMP=$(date +%s)
LOG_FILE="observability_test_${TIMESTAMP}.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Utility functions
log() {
    echo -e "${BLUE}[$(date '+%H:%M:%S')]${NC} $1" | tee -a "$LOG_FILE"
}

success() {
    echo -e "${GREEN}✅ $1${NC}" | tee -a "$LOG_FILE"
}

warning() {
    echo -e "${YELLOW}⚠️  $1${NC}" | tee -a "$LOG_FILE"
}

error() {
    echo -e "${RED}❌ $1${NC}" | tee -a "$LOG_FILE"
}

# Check if service is running
check_service() {
    log "Checking if payment processor service is running..."
    if curl -s "$BASE_URL/health" > /dev/null; then
        success "Payment processor is running at $BASE_URL"
    else
        error "Payment processor is not running at $BASE_URL"
        exit 1
    fi
}

# Test 1: Basic Payment Processing
test_basic_payments() {
    log "🧪 TEST 1: Basic Payment Processing Flow"
    
    local correlation_ids=(
        "basic-payment-success-${TIMESTAMP}"
        "basic-payment-medium-${TIMESTAMP}"  
        "basic-payment-large-${TIMESTAMP}"
    )
    
    local amounts=(100.50 500.00 1500.75)
    
    for i in "${!correlation_ids[@]}"; do
        local corr_id="${correlation_ids[$i]}"
        local amount="${amounts[$i]}"
        
        log "Processing payment: $corr_id (Amount: $amount)"
        
        local response=$(curl -s -w "\nHTTP_CODE:%{http_code}\nRESPONSE_TIME:%{time_total}" \
            -X POST "$BASE_URL/payments" \
            -H "Content-Type: application/json" \
            -H "X-Correlation-ID: $corr_id" \
            -d "{
                \"correlation_id\": \"$corr_id\",
                \"amount\": $amount
            }")
        
        local http_code=$(echo "$response" | grep "HTTP_CODE:" | cut -d: -f2)
        local response_time=$(echo "$response" | grep "RESPONSE_TIME:" | cut -d: -f2)
        
        if [[ "$http_code" == "200" ]]; then
            success "Payment processed successfully - Correlation ID: $corr_id, Response Time: ${response_time}s"
        else
            error "Payment failed - Correlation ID: $corr_id, HTTP Code: $http_code"
        fi
        
        sleep 1
    done
}

# Test 2: Error Scenarios
test_error_scenarios() {
    log "🧪 TEST 2: Error Handling Scenarios"
    
    # Invalid JSON
    log "Testing invalid JSON format..."
    local error_corr_id="error-invalid-json-${TIMESTAMP}"
    curl -s -w "\nHTTP_CODE:%{http_code}" \
        -X POST "$BASE_URL/payments" \
        -H "Content-Type: application/json" \
        -H "X-Correlation-ID: $error_corr_id" \
        -d '{"invalid": "json" missing_quote}' > /dev/null
    warning "Sent invalid JSON - Correlation ID: $error_corr_id"
    
    # Missing required fields
    log "Testing missing required fields..."
    local missing_corr_id="error-missing-fields-${TIMESTAMP}"
    curl -s -w "\nHTTP_CODE:%{http_code}" \
        -X POST "$BASE_URL/payments" \
        -H "Content-Type: application/json" \
        -H "X-Correlation-ID: $missing_corr_id" \
        -d '{"invalid_field": "test"}' > /dev/null
    warning "Sent request with missing fields - Correlation ID: $missing_corr_id"
    
    # Negative amount
    log "Testing negative amount..."
    local negative_corr_id="error-negative-amount-${TIMESTAMP}"
    curl -s -w "\nHTTP_CODE:%{http_code}" \
        -X POST "$BASE_URL/payments" \
        -H "Content-Type: application/json" \
        -H "X-Correlation-ID: $negative_corr_id" \
        -d "{
            \"correlation_id\": \"$negative_corr_id\",
            \"amount\": -100.00
        }" > /dev/null
    warning "Sent negative amount - Correlation ID: $negative_corr_id"
    
    sleep 2
}

# Test 3: Load Testing
test_load_scenario() {
    log "🧪 TEST 3: Load Testing Scenario"
    
    local num_requests=25
    local batch_size=5
    
    log "Generating $num_requests concurrent payment requests in batches of $batch_size..."
    
    for ((batch=0; batch<$((num_requests/batch_size)); batch++)); do
        log "Starting batch $((batch+1))..."
        
        for ((i=0; i<batch_size; i++)); do
            local request_id=$((batch * batch_size + i + 1))
            local corr_id="load-test-${TIMESTAMP}-$(printf "%03d" $request_id)"
            local amount=$((RANDOM % 900 + 100)).$((RANDOM % 100))
            
            curl -s -X POST "$BASE_URL/payments" \
                -H "Content-Type: application/json" \
                -H "X-Correlation-ID: $corr_id" \
                -d "{
                    \"correlation_id\": \"$corr_id\",
                    \"amount\": $amount
                }" > /dev/null &
        done
        
        wait # Wait for batch to complete
        success "Batch $((batch+1)) completed"
        sleep 0.5
    done
    
    success "Load test completed: $num_requests requests sent"
}

# Test 4: Health and Summary Endpoints
test_health_endpoints() {
    log "🧪 TEST 4: Health and Summary Endpoints"
    
    # Health check
    log "Testing health endpoint..."
    local health_corr_id="health-check-${TIMESTAMP}"
    local health_response=$(curl -s -w "\nHTTP_CODE:%{http_code}" \
        -X GET "$BASE_URL/health" \
        -H "X-Correlation-ID: $health_corr_id")
    
    local health_code=$(echo "$health_response" | grep "HTTP_CODE:" | cut -d: -f2)
    if [[ "$health_code" == "200" ]]; then
        success "Health check successful - Correlation ID: $health_corr_id"
    else
        warning "Health check returned HTTP $health_code"
    fi
    
    # Payments summary
    log "Testing payments summary endpoint..."
    local summary_corr_id="summary-check-${TIMESTAMP}"
    local summary_response=$(curl -s -w "\nHTTP_CODE:%{http_code}" \
        -X GET "$BASE_URL/payments-summary" \
        -H "X-Correlation-ID: $summary_corr_id")
    
    local summary_code=$(echo "$summary_response" | grep "HTTP_CODE:" | cut -d: -f2)
    if [[ "$summary_code" == "200" ]]; then
        success "Payments summary successful - Correlation ID: $summary_corr_id"
    else
        warning "Payments summary returned HTTP $summary_code"
    fi
}

# Test 5: Unique Correlation ID Scenarios
test_correlation_scenarios() {
    log "🧪 TEST 5: Correlation ID Propagation Test"
    
    # Test with external correlation ID
    log "Testing with external correlation ID..."
    local ext_corr_id="external-system-${TIMESTAMP}-abc123"
    curl -s -X POST "$BASE_URL/payments" \
        -H "Content-Type: application/json" \
        -H "X-Correlation-ID: $ext_corr_id" \
        -d "{
            \"correlation_id\": \"$ext_corr_id\",
            \"amount\": 250.00
        }" > /dev/null
    success "External correlation ID test - ID: $ext_corr_id"
    
    # Test without correlation ID header (should generate one)
    log "Testing without correlation ID header..."
    local response=$(curl -s -X POST "$BASE_URL/payments" \
        -H "Content-Type: application/json" \
        -d "{
            \"amount\": 150.00
        }")
    
    success "Auto-generated correlation ID test completed"
    
    # Test correlation ID consistency across multiple calls
    log "Testing correlation ID consistency..."
    local consistent_corr_id="consistency-test-${TIMESTAMP}"
    
    for i in {1..3}; do
        curl -s -X POST "$BASE_URL/payments" \
            -H "Content-Type: application/json" \
            -H "X-Correlation-ID: $consistent_corr_id" \
            -d "{
                \"correlation_id\": \"$consistent_corr_id\",
                \"amount\": $((100 + i * 50)).00
            }" > /dev/null
        
        sleep 0.5
    done
    success "Consistency test completed - Correlation ID: $consistent_corr_id"
}

# Generate Grafana queries based on test data
generate_grafana_queries() {
    log "📊 GENERATING GRAFANA QUERIES"
    
    local queries_file="grafana_queries_${TIMESTAMP}.md"
    
    cat > "$queries_file" << EOF
# Grafana Queries for Observability Test Session: ${TIMESTAMP}

## Loki Log Queries

### 1. All Test Logs from This Session
\`\`\`logql
{service_name="th-payment-processor"} |= "${TIMESTAMP}"
\`\`\`

### 2. Basic Payment Flow Logs
\`\`\`logql
{service_name="th-payment-processor"} |= "basic-payment" |= "${TIMESTAMP}" | json
\`\`\`

### 3. Error Scenario Logs
\`\`\`logql
{service_name="th-payment-processor"} |= "error-" |= "${TIMESTAMP}" | json | level="error"
\`\`\`

### 4. Load Test Logs
\`\`\`logql
{service_name="th-payment-processor"} |= "load-test-${TIMESTAMP}" | json
\`\`\`

### 5. Health Check Logs
\`\`\`logql
{service_name="th-payment-processor"} |= "${TIMESTAMP}" | json | operation="health_check"
\`\`\`

### 6. Payment Success Rate for This Session
\`\`\`logql
sum by (status) (count_over_time({service_name="th-payment-processor"} |= "${TIMESTAMP}" | json | operation="payment_success" or operation="payment_failure" [5m]))
\`\`\`

### 7. Average Latency for Test Payments
\`\`\`logql
avg_over_time({service_name="th-payment-processor"} |= "${TIMESTAMP}" | json | latency_ms != "" | unwrap latency_ms [5m])
\`\`\`

### 8. Top Correlation IDs from This Test
\`\`\`logql
topk(10, count by (correlation_id) (count_over_time({service_name="th-payment-processor"} |= "${TIMESTAMP}" | json [1h])))
\`\`\`

## Tempo Trace Queries

Search for traces with these correlation IDs in Grafana Explore → Tempo:

- basic-payment-success-${TIMESTAMP}
- error-invalid-json-${TIMESTAMP}  
- load-test-${TIMESTAMP}-001 through load-test-${TIMESTAMP}-025
- health-check-${TIMESTAMP}
- external-system-${TIMESTAMP}-abc123

## Expected Results

✅ **Structured Logs**: All logs should be in JSON format with consistent fields
✅ **Correlation IDs**: Present in all logs related to requests
✅ **Trace Context**: trace_id and span_id should be present and consistent
✅ **Payment Fields**: amount, processor, status should be captured
✅ **Latency Data**: Response times should be recorded
✅ **Error Context**: Error logs should include detailed context and stack traces

EOF

    success "Grafana queries saved to: $queries_file"
}

# Display test summary
display_summary() {
    log "📋 TEST SUMMARY"
    echo ""
    echo "Test completed at: $(date)"
    echo "Log file: $LOG_FILE"
    echo "Timestamp used: $TIMESTAMP"
    echo ""
    echo "🔍 NEXT STEPS:"
    echo "1. Open Grafana at http://localhost:3000"
    echo "2. Go to Explore → Loki"
    echo "3. Use the queries from: grafana_queries_${TIMESTAMP}.md"
    echo "4. Verify structured logging and trace correlation"
    echo ""
    echo "📊 KEY VERIFICATION POINTS:"
    echo "- Correlation IDs present in all request logs"
    echo "- Trace IDs consistent across request lifecycle"  
    echo "- JSON structured format with payment fields"
    echo "- Latency measurements in response logs"
    echo "- Error logs contain detailed context"
    echo ""
}

# Main execution
main() {
    echo ""
    log "🚀 STARTING OBSERVABILITY TESTING"
    echo "This script will test structured logging and distributed tracing features"
    echo ""
    
    check_service
    echo ""
    
    test_basic_payments
    echo ""
    
    test_error_scenarios  
    echo ""
    
    test_load_scenario
    echo ""
    
    test_health_endpoints
    echo ""
    
    test_correlation_scenarios
    echo ""
    
    # Wait for logs to be processed
    log "⏳ Waiting for logs to be processed by Loki..."
    sleep 5
    
    generate_grafana_queries
    echo ""
    
    display_summary
}

# Execute main function
main "$@"