#!/bin/bash

# Simple Performance Test Suite
# Tests critical performance requirements

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BASE_URL="http://localhost:8081"
API_URL="$BASE_URL/api/v1"
PERFORMANCE_THRESHOLD_MS=100

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Helper functions
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
    ((PASSED_TESTS++))
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
    ((FAILED_TESTS++))
}

log_section() {
    echo -e "\n${YELLOW}📋 $1${NC}"
    echo "=================================================="
}

# Convert seconds to milliseconds using awk instead of bc
seconds_to_ms() {
    echo "$1" | awk '{printf "%.0f", $1 * 1000}'
}

# Test redirect performance
test_redirect_performance() {
    log_section "1. Redirect Performance Tests (<100ms requirement)"
    
    # Clean database and create test URL
    log_info "Setting up test data..."
    ./scripts/cleanup-db.sh > /dev/null 2>&1
    
    curl -s -X POST "$API_URL/urls" \
        -H "Content-Type: application/json" \
        -d '{"original_url": "https://www.google.com", "custom_alias": "perftest"}' > /dev/null
    
    # Test single redirect
    log_info "Testing single redirect performance..."
    ((TOTAL_TESTS++))
    
    local response=$(curl -s -w "HTTPSTATUS:%{http_code};TIME:%{time_total}" -o /dev/null "$BASE_URL/perftest")
    local http_status=$(echo "$response" | cut -d';' -f1 | cut -d':' -f2)
    local time_seconds=$(echo "$response" | cut -d';' -f2 | cut -d':' -f2)
    local time_ms=$(seconds_to_ms "$time_seconds")
    
    if [ "$http_status" = "302" ] && [ "$time_ms" -lt "$PERFORMANCE_THRESHOLD_MS" ]; then
        log_success "Single redirect: ${time_ms}ms (< ${PERFORMANCE_THRESHOLD_MS}ms) ✓"
    else
        log_error "Single redirect: ${time_ms}ms (>= ${PERFORMANCE_THRESHOLD_MS}ms) or status: $http_status"
    fi
    
    # Test multiple redirects
    log_info "Testing 10 consecutive redirects..."
    ((TOTAL_TESTS++))
    
    local total_time=0
    local max_time=0
    local success_count=0
    
    for ((i=1; i<=10; i++)); do
        local resp=$(curl -s -w "HTTPSTATUS:%{http_code};TIME:%{time_total}" -o /dev/null "$BASE_URL/perftest")
        local status=$(echo "$resp" | cut -d';' -f1 | cut -d':' -f2)
        local time_sec=$(echo "$resp" | cut -d';' -f2 | cut -d':' -f2)
        local time_ms=$(seconds_to_ms "$time_sec")
        
        if [ "$status" = "302" ]; then
            ((success_count++))
            total_time=$((total_time + time_ms))
            if [ "$time_ms" -gt "$max_time" ]; then
                max_time=$time_ms
            fi
        fi
    done
    
    if [ "$success_count" -eq 10 ] && [ "$max_time" -lt "$PERFORMANCE_THRESHOLD_MS" ]; then
        local avg_time=$((total_time / success_count))
        log_success "10 redirects: Max ${max_time}ms, Avg ${avg_time}ms (all < ${PERFORMANCE_THRESHOLD_MS}ms) ✓"
    else
        log_error "10 redirects: Max ${max_time}ms, Success: $success_count/10"
    fi
}

# Test URL creation capacity
test_url_creation_capacity() {
    log_section "2. URL Creation Capacity Tests (10,000 URLs/day requirement)"
    
    log_info "Testing URL creation rate for 30 seconds..."
    ((TOTAL_TESTS++))
    
    local start_time=$(date +%s)
    local end_time=$((start_time + 30))
    local created_count=0
    local error_count=0
    
    while [ $(date +%s) -lt $end_time ]; do
        local current_time=$(date +%s)
        local url_id="test_${current_time}_${created_count}"
        
        local response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X POST "$API_URL/urls" \
            -H "Content-Type: application/json" \
            -d "{\"original_url\": \"https://example.com/test-$url_id\"}")
        
        local http_status=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        
        if [ "$http_status" = "201" ]; then
            ((created_count++))
        else
            ((error_count++))
        fi
        
        # Small delay to avoid overwhelming
        sleep 0.1
    done
    
    local actual_duration=$(($(date +%s) - start_time))
    local urls_per_minute=$((created_count * 60 / actual_duration))
    local projected_daily=$((urls_per_minute * 24 * 60))
    
    log_info "Created $created_count URLs in ${actual_duration}s (${error_count} errors)"
    log_info "Rate: ~$urls_per_minute URLs/minute"
    log_info "Projected daily capacity: ~$projected_daily URLs/day"
    
    if [ "$projected_daily" -ge 10000 ]; then
        log_success "Daily capacity: $projected_daily URLs/day (>= 10,000) ✓"
    else
        log_error "Daily capacity: $projected_daily URLs/day (< 10,000)"
    fi
}

# Test basic rate limiting
test_basic_rate_limiting() {
    log_section "3. Basic Rate Limiting Tests (100 requests/minute limit)"
    
    # Test normal rate
    log_info "Testing normal request rate (10 requests over 10 seconds)..."
    ((TOTAL_TESTS++))
    
    local allowed_count=0
    local blocked_count=0
    
    for ((i=1; i<=10; i++)); do
        local response=$(curl -s -w "HTTPSTATUS:%{http_code}" "$API_URL/health")
        local http_status=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        
        if [ "$http_status" = "200" ]; then
            ((allowed_count++))
        elif [ "$http_status" = "429" ]; then
            ((blocked_count++))
        fi
        
        sleep 1
    done
    
    if [ "$allowed_count" -eq 10 ] && [ "$blocked_count" -eq 0 ]; then
        log_success "Normal rate: All 10 requests allowed ✓"
    else
        log_error "Normal rate: $allowed_count allowed, $blocked_count blocked (expected 10/0)"
    fi
    
    # Test burst rate limiting
    log_info "Testing burst requests (120 requests rapidly)..."
    ((TOTAL_TESTS++))
    
    allowed_count=0
    blocked_count=0
    
    for ((i=1; i<=120; i++)); do
        local response=$(curl -s -w "HTTPSTATUS:%{http_code}" "$API_URL/health" 2>/dev/null)
        local http_status=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        
        if [ "$http_status" = "200" ]; then
            ((allowed_count++))
        elif [ "$http_status" = "429" ]; then
            ((blocked_count++))
        fi
        
        sleep 0.01
    done
    
    log_info "Burst test: $allowed_count allowed, $blocked_count blocked"
    
    if [ "$blocked_count" -gt 0 ] && [ "$allowed_count" -le 100 ]; then
        log_success "Rate limiting: Working correctly (blocked $blocked_count excess requests) ✓"
    else
        log_error "Rate limiting: Not working properly (allowed: $allowed_count, blocked: $blocked_count)"
    fi
}

# Main test execution
main() {
    echo -e "${BLUE}🚀 Performance Requirements Test Suite${NC}"
    echo "============================================================"
    echo -e "Testing: ${YELLOW}1. <100ms redirects, 2. 10K URLs/day, 3. Rate limiting${NC}"
    echo ""
    
    # Check if services are running
    log_info "Checking if services are running..."
    if ! curl -s "$BASE_URL/health" > /dev/null; then
        log_error "Services are not running. Please start the application first."
        exit 1
    fi
    
    log_success "Services are running"
    
    # Run tests
    test_redirect_performance
    test_url_creation_capacity
    test_basic_rate_limiting
    
    # Final results
    echo ""
    echo "============================================================"
    echo -e "${BLUE}📊 Performance Test Results${NC}"
    echo "Total Tests: $TOTAL_TESTS"
    echo -e "Passed: ${GREEN}$PASSED_TESTS${NC}"
    echo -e "Failed: ${RED}$FAILED_TESTS${NC}"
    
    if [ $FAILED_TESTS -eq 0 ]; then
        echo -e "${GREEN}🎉 All performance requirements are met!${NC}"
        echo -e "${GREEN}✅ Redirect performance: <100ms${NC}"
        echo -e "${GREEN}✅ Daily capacity: 10,000+ URLs${NC}"
        echo -e "${GREEN}✅ Rate limiting: Working${NC}"
        exit 0
    else
        echo -e "${RED}❌ Some performance requirements are not met.${NC}"
        exit 1
    fi
}

# Run main function
main "$@"
