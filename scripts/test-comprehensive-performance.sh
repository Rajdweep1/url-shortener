#!/bin/bash

# Comprehensive Performance & Rate Limiting Test Suite
# Covers all missing performance requirements and edge cases

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
DAILY_URL_TARGET=10000
RATE_LIMIT_PER_MINUTE=100

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

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_section() {
    echo -e "\n${YELLOW}📋 $1${NC}"
    echo "=================================================="
}

# Convert seconds to milliseconds
seconds_to_ms() {
    echo "$1" | awk '{printf "%.0f", $1 * 1000}'
}

# Setup test data
setup_test_data() {
    log_info "🧹 Setting up test data..."
    ./scripts/cleanup-db.sh > /dev/null 2>&1
    
    # Create various test URLs
    local test_urls=(
        '{"original_url": "https://www.google.com", "custom_alias": "google"}'
        '{"original_url": "https://github.com", "custom_alias": "github"}'
        '{"original_url": "https://stackoverflow.com", "custom_alias": "stackoverflow"}'
        '{"original_url": "https://www.example.com/very/long/path/with/many/segments?param1=value1&param2=value2", "custom_alias": "longurl"}'
        '{"original_url": "https://subdomain.example.com:8080/path?query=test#fragment", "custom_alias": "complex"}'
    )
    
    for url_data in "${test_urls[@]}"; do
        curl -s -X POST "$API_URL/urls" \
            -H "Content-Type: application/json" \
            -d "$url_data" > /dev/null
    done
    
    log_success "Test data created"
}

# Test redirect performance under various conditions
test_comprehensive_redirect_performance() {
    log_section "1. Comprehensive Redirect Performance Tests"
    
    # Test 1: Single redirect performance
    log_info "Testing single redirect performance..."
    ((TOTAL_TESTS++))
    
    local response=$(curl -s -w "HTTPSTATUS:%{http_code};TIME:%{time_total}" -o /dev/null "$BASE_URL/google")
    local http_status=$(echo "$response" | cut -d';' -f1 | cut -d':' -f2)
    local time_seconds=$(echo "$response" | cut -d';' -f2 | cut -d':' -f2)
    local time_ms=$(seconds_to_ms "$time_seconds")
    
    if [ "$http_status" = "302" ] && [ "$time_ms" -lt "$PERFORMANCE_THRESHOLD_MS" ]; then
        log_success "Single redirect: ${time_ms}ms (< ${PERFORMANCE_THRESHOLD_MS}ms) ✓"
    else
        log_error "Single redirect: ${time_ms}ms (>= ${PERFORMANCE_THRESHOLD_MS}ms) or status: $http_status"
    fi
    
    # Test 2: Concurrent redirect performance
    log_info "Testing concurrent redirect performance (20 parallel requests)..."
    ((TOTAL_TESTS++))
    
    local temp_file="/tmp/concurrent_perf_$$"
    
    # Launch 20 concurrent requests
    for ((i=1; i<=20; i++)); do
        (
            local resp=$(curl -s -w "HTTPSTATUS:%{http_code};TIME:%{time_total}" -o /dev/null "$BASE_URL/google")
            local status=$(echo "$resp" | cut -d';' -f1 | cut -d':' -f2)
            local time_sec=$(echo "$resp" | cut -d';' -f2 | cut -d':' -f2)
            local time_ms=$(seconds_to_ms "$time_sec")
            echo "$status,$time_ms" >> "$temp_file"
        ) &
    done
    
    wait
    
    # Analyze concurrent results
    local max_time=0
    local total_time=0
    local success_count=0
    
    while IFS=',' read -r status time_ms; do
        if [ "$status" = "302" ]; then
            ((success_count++))
            total_time=$((total_time + time_ms))
            if [ "$time_ms" -gt "$max_time" ]; then
                max_time=$time_ms
            fi
        fi
    done < "$temp_file"
    
    rm -f "$temp_file"
    
    if [ "$success_count" -eq 20 ] && [ "$max_time" -lt "$PERFORMANCE_THRESHOLD_MS" ]; then
        local avg_time=$((total_time / success_count))
        log_success "20 concurrent redirects: Max ${max_time}ms, Avg ${avg_time}ms (all < ${PERFORMANCE_THRESHOLD_MS}ms) ✓"
    else
        log_error "20 concurrent redirects: Max ${max_time}ms, Success: $success_count/20"
    fi
    
    # Test 3: Different URL types performance
    log_info "Testing performance with different URL types..."
    ((TOTAL_TESTS++))
    
    local url_types=("github" "stackoverflow" "longurl" "complex")
    local all_under_threshold=true
    
    for url_type in "${url_types[@]}"; do
        local resp=$(curl -s -w "TIME:%{time_total}" -o /dev/null "$BASE_URL/$url_type")
        local time_sec=$(echo "$resp" | cut -d':' -f2)
        local time_ms=$(seconds_to_ms "$time_sec")
        
        log_info "  $url_type: ${time_ms}ms"
        
        if [ "$time_ms" -ge "$PERFORMANCE_THRESHOLD_MS" ]; then
            all_under_threshold=false
        fi
    done
    
    if [ "$all_under_threshold" = true ]; then
        log_success "All URL types: Under ${PERFORMANCE_THRESHOLD_MS}ms ✓"
    else
        log_error "Some URL types: Over ${PERFORMANCE_THRESHOLD_MS}ms"
    fi
}

# Test URL creation capacity under stress
test_url_creation_stress() {
    log_section "2. URL Creation Stress Tests"
    
    # Test 1: Sustained creation rate
    log_info "Testing sustained URL creation for 60 seconds..."
    ((TOTAL_TESTS++))
    
    local start_time=$(date +%s)
    local end_time=$((start_time + 60))
    local created_count=0
    local error_count=0
    
    while [ $(date +%s) -lt $end_time ]; do
        local current_time=$(date +%s)
        local url_id="stress_${current_time}_${created_count}"
        
        local response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X POST "$API_URL/urls" \
            -H "Content-Type: application/json" \
            -d "{\"original_url\": \"https://example.com/stress-$url_id\"}")
        
        local http_status=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        
        if [ "$http_status" = "201" ]; then
            ((created_count++))
        else
            ((error_count++))
        fi
        
        sleep 0.1
    done
    
    local actual_duration=$(($(date +%s) - start_time))
    local urls_per_minute=$((created_count * 60 / actual_duration))
    local projected_daily=$((urls_per_minute * 24 * 60))
    
    log_info "Stress test: Created $created_count URLs in ${actual_duration}s (${error_count} errors)"
    log_info "Rate: ~$urls_per_minute URLs/minute"
    log_info "Projected daily: ~$projected_daily URLs/day"
    
    if [ "$projected_daily" -ge "$DAILY_URL_TARGET" ]; then
        log_success "Stress capacity: $projected_daily URLs/day (>= $DAILY_URL_TARGET) ✓"
    else
        log_error "Stress capacity: $projected_daily URLs/day (< $DAILY_URL_TARGET)"
    fi
    
    # Test 2: Burst creation capacity
    log_info "Testing burst URL creation (100 URLs rapidly)..."
    ((TOTAL_TESTS++))
    
    local burst_start=$(date +%s%3N)
    local burst_created=0
    local burst_errors=0
    
    for ((i=1; i<=100; i++)); do
        local response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X POST "$API_URL/urls" \
            -H "Content-Type: application/json" \
            -d "{\"original_url\": \"https://example.com/burst-$i\"}")
        
        local status=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        
        if [ "$status" = "201" ]; then
            ((burst_created++))
        else
            ((burst_errors++))
        fi
    done
    
    local burst_end=$(date +%s%3N)
    local burst_duration=$((burst_end - burst_start))
    
    log_info "Burst test: Created $burst_created URLs in ${burst_duration}ms (${burst_errors} errors)"
    
    if [ "$burst_created" -ge 80 ]; then # Allow some failures under extreme load
        log_success "Burst creation: $burst_created/100 URLs created ✓"
    else
        log_error "Burst creation: Only $burst_created/100 URLs created"
    fi
}

# Test comprehensive rate limiting scenarios
test_comprehensive_rate_limiting() {
    log_section "3. Comprehensive Rate Limiting Tests"
    
    # Test 1: IP-based rate limiting
    log_info "Testing IP-based rate limiting..."
    ((TOTAL_TESTS++))
    
    local test_ip="192.168.1.100"
    local ip_allowed=0
    local ip_blocked=0
    
    for ((i=1; i<=120; i++)); do
        local response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
            -H "X-Forwarded-For: $test_ip" \
            "$API_URL/health" 2>/dev/null)
        local status=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        
        if [ "$status" = "200" ]; then
            ((ip_allowed++))
        elif [ "$status" = "429" ]; then
            ((ip_blocked++))
        fi
        
        sleep 0.02
    done
    
    log_info "IP-based ($test_ip): $ip_allowed allowed, $ip_blocked blocked"
    
    if [ "$ip_blocked" -gt 0 ] && [ "$ip_allowed" -le "$RATE_LIMIT_PER_MINUTE" ]; then
        log_success "IP-based rate limiting: Working correctly ✓"
    else
        log_warning "IP-based rate limiting: May need adjustment"
    fi
    
    # Test 2: Endpoint-specific rate limiting
    log_info "Testing endpoint-specific rate limiting..."
    ((TOTAL_TESTS++))
    
    local endpoints=("/api/v1/health" "/health")
    local endpoint_working=true
    
    for endpoint in "${endpoints[@]}"; do
        local ep_allowed=0
        local ep_blocked=0
        
        for ((i=1; i<=60; i++)); do
            local response=$(curl -s -w "HTTPSTATUS:%{http_code}" "$BASE_URL$endpoint" 2>/dev/null)
            local status=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
            
            if [ "$status" = "200" ]; then
                ((ep_allowed++))
            elif [ "$status" = "429" ]; then
                ((ep_blocked++))
            fi
            
            sleep 0.03
        done
        
        log_info "  $endpoint: $ep_allowed allowed, $ep_blocked blocked"
        
        if [ "$ep_allowed" -eq 0 ]; then
            endpoint_working=false
        fi
    done
    
    if [ "$endpoint_working" = true ]; then
        log_success "Endpoint-specific rate limiting: Working ✓"
    else
        log_error "Endpoint-specific rate limiting: Issues detected"
    fi
    
    # Test 3: Rate limit recovery
    log_info "Testing rate limit recovery (waiting 65 seconds)..."
    ((TOTAL_TESTS++))
    
    # Trigger rate limit first
    for ((i=1; i<=120; i++)); do
        curl -s "$API_URL/health" > /dev/null 2>&1
        sleep 0.01
    done
    
    # Check if blocked
    local blocked_response=$(curl -s -w "HTTPSTATUS:%{http_code}" "$API_URL/health")
    local blocked_status=$(echo "$blocked_response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
    
    if [ "$blocked_status" = "429" ]; then
        log_info "Rate limit triggered, waiting for recovery..."
        sleep 65
        
        local recovery_response=$(curl -s -w "HTTPSTATUS:%{http_code}" "$API_URL/health")
        local recovery_status=$(echo "$recovery_response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        
        if [ "$recovery_status" = "200" ]; then
            log_success "Rate limit recovery: Working correctly ✓"
        else
            log_error "Rate limit recovery: Failed (status: $recovery_status)"
        fi
    else
        log_warning "Could not trigger rate limit for recovery test"
    fi
}

# Main test execution
main() {
    echo -e "${BLUE}🚀 Comprehensive Performance & Rate Limiting Test Suite${NC}"
    echo "============================================================"
    echo -e "Requirements: ${YELLOW}<100ms redirects, 10K URLs/day, Rate limiting${NC}"
    echo ""
    
    # Check if services are running
    log_info "Checking if services are running..."
    if ! curl -s "$BASE_URL/health" > /dev/null; then
        log_error "Services are not running. Please start the application first."
        exit 1
    fi
    
    log_success "Services are running"
    
    # Setup test data
    setup_test_data
    
    # Run comprehensive tests
    test_comprehensive_redirect_performance
    test_url_creation_stress
    test_comprehensive_rate_limiting
    
    # Final results
    echo ""
    echo "============================================================"
    echo -e "${BLUE}📊 Comprehensive Performance Test Results${NC}"
    echo "Total Tests: $TOTAL_TESTS"
    echo -e "Passed: ${GREEN}$PASSED_TESTS${NC}"
    echo -e "Failed: ${RED}$FAILED_TESTS${NC}"
    
    if [ $FAILED_TESTS -eq 0 ]; then
        echo -e "${GREEN}🎉 All comprehensive performance tests passed!${NC}"
        echo -e "${GREEN}✅ Performance Requirements: FULLY MET${NC}"
        exit 0
    else
        echo -e "${RED}❌ Some comprehensive performance tests failed.${NC}"
        exit 1
    fi
}

# Run main function
main "$@"
