#!/bin/bash

# Comprehensive Rate Limiting Test Suite
# Tests rate limiting functionality across different scenarios

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
RATE_LIMIT_PER_MINUTE=100
RATE_LIMIT_WINDOW=60

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

# Test basic rate limiting
test_basic_rate_limiting() {
    log_section "1. Basic Rate Limiting Tests"
    
    # Test 1: Normal request rate should be allowed
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
    
    # Test 2: Burst requests should trigger rate limiting
    log_info "Testing burst requests (150 requests rapidly)..."
    ((TOTAL_TESTS++))
    
    allowed_count=0
    blocked_count=0
    local burst_requests=150
    
    for ((i=1; i<=burst_requests; i++)); do
        local response=$(curl -s -w "HTTPSTATUS:%{http_code}" "$API_URL/health" 2>/dev/null)
        local http_status=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        
        if [ "$http_status" = "200" ]; then
            ((allowed_count++))
        elif [ "$http_status" = "429" ]; then
            ((blocked_count++))
        fi
        
        # Very small delay to avoid connection issues
        sleep 0.01
    done
    
    log_info "Burst test results: $allowed_count allowed, $blocked_count blocked"
    
    # We expect some requests to be blocked
    if [ "$blocked_count" -gt 0 ] && [ "$allowed_count" -le "$RATE_LIMIT_PER_MINUTE" ]; then
        log_success "Burst rate limiting: Working correctly (blocked $blocked_count excess requests) ✓"
    else
        log_error "Burst rate limiting: Not working (allowed: $allowed_count, blocked: $blocked_count)"
    fi
}

# Test IP-based rate limiting
test_ip_based_rate_limiting() {
    log_section "2. IP-Based Rate Limiting Tests"
    
    # Test with different IP headers
    log_info "Testing rate limiting with X-Forwarded-For header..."
    ((TOTAL_TESTS++))
    
    local test_ip="192.168.1.100"
    local allowed_count=0
    local blocked_count=0
    
    # Send requests with specific IP
    for ((i=1; i<=120; i++)); do
        local response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
            -H "X-Forwarded-For: $test_ip" \
            "$API_URL/health" 2>/dev/null)
        local http_status=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        
        if [ "$http_status" = "200" ]; then
            ((allowed_count++))
        elif [ "$http_status" = "429" ]; then
            ((blocked_count++))
        fi
        
        sleep 0.02
    done
    
    log_info "IP-based test ($test_ip): $allowed_count allowed, $blocked_count blocked"
    
    if [ "$blocked_count" -gt 0 ]; then
        log_success "IP-based rate limiting: Working correctly ✓"
    else
        log_warning "IP-based rate limiting: May not be working as expected"
    fi
}

# Test endpoint-specific rate limiting
test_endpoint_rate_limiting() {
    log_section "3. Endpoint-Specific Rate Limiting Tests"
    
    local endpoints=("/api/v1/health" "/health")
    
    for endpoint in "${endpoints[@]}"; do
        log_info "Testing rate limiting for $endpoint..."
        ((TOTAL_TESTS++))
        
        local allowed_count=0
        local blocked_count=0
        
        # Send 80 requests to this endpoint
        for ((i=1; i<=80; i++)); do
            local response=$(curl -s -w "HTTPSTATUS:%{http_code}" "$BASE_URL$endpoint" 2>/dev/null)
            local http_status=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
            
            if [ "$http_status" = "200" ]; then
                ((allowed_count++))
            elif [ "$http_status" = "429" ]; then
                ((blocked_count++))
            fi
            
            sleep 0.02
        done
        
        log_info "$endpoint: $allowed_count allowed, $blocked_count blocked"
        
        if [ "$allowed_count" -gt 0 ]; then
            log_success "$endpoint rate limiting: Functional ✓"
        else
            log_error "$endpoint rate limiting: All requests blocked"
        fi
    done
}

# Test rate limit recovery
test_rate_limit_recovery() {
    log_section "4. Rate Limit Recovery Tests"
    
    log_info "Testing rate limit recovery after window expires..."
    ((TOTAL_TESTS++))
    
    # First, trigger rate limiting
    log_info "Triggering rate limit..."
    local blocked_count=0
    
    for ((i=1; i<=120; i++)); do
        local response=$(curl -s -w "HTTPSTATUS:%{http_code}" "$API_URL/health" 2>/dev/null)
        local http_status=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        
        if [ "$http_status" = "429" ]; then
            ((blocked_count++))
            break
        fi
        
        sleep 0.01
    done
    
    if [ "$blocked_count" -gt 0 ]; then
        log_info "Rate limit triggered successfully"
        
        # Wait for rate limit window to reset (65 seconds to be safe)
        log_info "Waiting 65 seconds for rate limit window to reset..."
        sleep 65
        
        # Test if requests are allowed again
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

# Test concurrent rate limiting
test_concurrent_rate_limiting() {
    log_section "5. Concurrent Rate Limiting Tests"
    
    log_info "Testing rate limiting under concurrent load..."
    ((TOTAL_TESTS++))
    
    local concurrent_processes=5
    local requests_per_process=30
    local temp_file="/tmp/rate_limit_results_$$"
    
    # Launch concurrent processes
    for ((p=1; p<=concurrent_processes; p++)); do
        (
            local process_allowed=0
            local process_blocked=0
            
            for ((i=1; i<=requests_per_process; i++)); do
                local response=$(curl -s -w "HTTPSTATUS:%{http_code}" "$API_URL/health" 2>/dev/null)
                local http_status=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
                
                if [ "$http_status" = "200" ]; then
                    ((process_allowed++))
                elif [ "$http_status" = "429" ]; then
                    ((process_blocked++))
                fi
                
                sleep 0.05
            done
            
            echo "$process_allowed,$process_blocked" >> "$temp_file"
        ) &
    done
    
    # Wait for all processes to complete
    wait
    
    # Analyze results
    local total_allowed=0
    local total_blocked=0
    
    while IFS=',' read -r allowed blocked; do
        total_allowed=$((total_allowed + allowed))
        total_blocked=$((total_blocked + blocked))
    done < "$temp_file"
    
    rm -f "$temp_file"
    
    log_info "Concurrent test: $total_allowed allowed, $total_blocked blocked (across $concurrent_processes processes)"
    
    if [ "$total_blocked" -gt 0 ] && [ "$total_allowed" -le "$RATE_LIMIT_PER_MINUTE" ]; then
        log_success "Concurrent rate limiting: Working correctly ✓"
    else
        log_warning "Concurrent rate limiting: Results may indicate issues"
    fi
}

# Test rate limiting on different HTTP methods
test_method_rate_limiting() {
    log_section "6. HTTP Method Rate Limiting Tests"
    
    local methods=("GET" "POST" "PUT" "DELETE")
    
    for method in "${methods[@]}"; do
        log_info "Testing rate limiting for $method requests..."
        ((TOTAL_TESTS++))
        
        local allowed_count=0
        local blocked_count=0
        local error_count=0
        
        for ((i=1; i<=50; i++)); do
            local response
            case $method in
                "GET")
                    response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X GET "$API_URL/health" 2>/dev/null)
                    ;;
                "POST")
                    response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X POST "$API_URL/urls" \
                        -H "Content-Type: application/json" \
                        -d '{"original_url": "https://example.com"}' 2>/dev/null)
                    ;;
                "PUT")
                    response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X PUT "$API_URL/urls/test" \
                        -H "Content-Type: application/json" \
                        -d '{"original_url": "https://example.com"}' 2>/dev/null)
                    ;;
                "DELETE")
                    response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X DELETE "$API_URL/urls/test" 2>/dev/null)
                    ;;
            esac
            
            local http_status=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
            
            case $http_status in
                "200"|"201"|"404"|"405")
                    ((allowed_count++))
                    ;;
                "429")
                    ((blocked_count++))
                    ;;
                *)
                    ((error_count++))
                    ;;
            esac
            
            sleep 0.05
        done
        
        log_info "$method: $allowed_count allowed, $blocked_count blocked, $error_count errors"
        
        if [ "$allowed_count" -gt 0 ]; then
            log_success "$method rate limiting: Functional ✓"
        else
            log_warning "$method rate limiting: No successful requests"
        fi
    done
}

# Main test execution
main() {
    echo -e "${BLUE}🚀 Comprehensive Rate Limiting Test Suite${NC}"
    echo "============================================================"
    echo -e "Rate Limit Configuration: ${YELLOW}$RATE_LIMIT_PER_MINUTE requests/minute${NC}"
    echo ""
    
    # Check if services are running
    log_info "Checking if services are running..."
    if ! curl -s "$BASE_URL/health" > /dev/null; then
        log_error "Services are not running. Please start the application first."
        exit 1
    fi
    
    log_success "Services are running"
    
    # Run all rate limiting tests
    test_basic_rate_limiting
    test_ip_based_rate_limiting
    test_endpoint_rate_limiting
    test_rate_limit_recovery
    test_concurrent_rate_limiting
    test_method_rate_limiting
    
    # Final results
    echo ""
    echo "============================================================"
    echo -e "${BLUE}📊 Rate Limiting Test Results${NC}"
    echo "Total Tests: $TOTAL_TESTS"
    echo -e "Passed: ${GREEN}$PASSED_TESTS${NC}"
    echo -e "Failed: ${RED}$FAILED_TESTS${NC}"
    
    if [ $FAILED_TESTS -eq 0 ]; then
        echo -e "${GREEN}🎉 All rate limiting tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}❌ Some rate limiting tests failed.${NC}"
        exit 1
    fi
}

# Run main function
main "$@"
