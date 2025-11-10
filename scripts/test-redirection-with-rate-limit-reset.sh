#!/bin/bash

# Redirection Edge Cases Test with Rate Limit Reset
# This version waits for rate limits to reset before running tests

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

BASE_URL="http://localhost:8081"
API_URL="$BASE_URL/api/v1"

# Test counters
TOTAL=0
PASSED=0
FAILED=0

echo -e "${BLUE}🧪 Redirection Edge Cases Test (Rate Limit Safe)${NC}"
echo "============================================================"

# Wait for any existing rate limits to reset
echo -e "${YELLOW}⏳ Waiting 65 seconds for rate limits to reset...${NC}"
sleep 65

echo -e "${YELLOW}🧹 Cleaning database for fresh test run...${NC}"
./scripts/cleanup-db.sh > /dev/null 2>&1

# Helper function to create test URL
create_test_url() {
    local url="$1"
    local alias="$2"
    curl -s -X POST "$API_URL/urls" \
        -H "Content-Type: application/json" \
        -d "{\"original_url\": \"$url\", \"custom_alias\": \"$alias\"}" > /dev/null
    sleep 1  # Wait between creations
}

# Helper function to test redirect with rate limit safety
test_redirect_safe() {
    local name="$1"
    local short_code="$2"
    local expected_status="$3"
    local expected_location="$4"
    local description="$5"
    
    echo -e "${YELLOW}Testing: $name${NC}"
    if [ -n "$description" ]; then
        echo "  Description: $description"
    fi
    
    ((TOTAL++))
    
    # Wait to avoid rate limiting
    sleep 2
    
    # Make request
    response=$(curl -s -D - -o /dev/null -w "HTTPSTATUS:%{http_code}" \
                   "$BASE_URL/$short_code" 2>/dev/null || echo "HTTPSTATUS:000")
    
    # Parse response
    http_status=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
    location=$(echo "$response" | grep -i "^location:" | cut -d' ' -f2- | tr -d '\r\n')
    
    # Check result
    if [ "$http_status" = "$expected_status" ]; then
        if [ "$expected_status" = "302" ] && [ -n "$expected_location" ]; then
            if [ "$location" = "$expected_location" ]; then
                echo -e "  ${GREEN}✓ PASSED${NC} (Status: $http_status, Location: $location)"
                ((PASSED++))
            else
                echo -e "  ${RED}✗ FAILED${NC} (Expected location: $expected_location, Got: $location)"
                ((FAILED++))
            fi
        else
            echo -e "  ${GREEN}✓ PASSED${NC} (Expected: $expected_status, Got: $http_status)"
            ((PASSED++))
        fi
    else
        echo -e "  ${RED}✗ FAILED${NC} (Expected: $expected_status, Got: $http_status)"
        if [ "$http_status" = "429" ]; then
            echo -e "  ${YELLOW}⚠️  Rate limit hit - waiting 30 seconds...${NC}"
            sleep 30
        fi
        ((FAILED++))
    fi
    echo ""
}

# Helper function to test error responses
test_error_safe() {
    local name="$1"
    local short_code="$2"
    local expected_status="$3"
    local description="$4"
    
    echo -e "${YELLOW}Testing: $name${NC}"
    if [ -n "$description" ]; then
        echo "  Description: $description"
    fi
    
    ((TOTAL++))
    
    # Wait to avoid rate limiting
    sleep 2
    
    # Make request
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
                   "$BASE_URL/$short_code" 2>/dev/null || echo "HTTPSTATUS:000")
    
    # Parse response
    http_status=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
    
    # Check result
    if [ "$http_status" = "$expected_status" ]; then
        echo -e "  ${GREEN}✓ PASSED${NC} (Expected: $expected_status, Got: $http_status)"
        ((PASSED++))
    else
        echo -e "  ${RED}✗ FAILED${NC} (Expected: $expected_status, Got: $http_status)"
        if [ "$http_status" = "429" ]; then
            echo -e "  ${YELLOW}⚠️  Rate limit hit - waiting 30 seconds...${NC}"
            sleep 30
        fi
        ((FAILED++))
    fi
    echo ""
}

# Run core tests
echo -e "${YELLOW}📋 1. Basic Redirection Tests${NC}"

create_test_url "https://www.google.com" "google-test"
test_redirect_safe "Valid short code redirect" "google-test" "302" "https://www.google.com" "Should redirect to original URL"

create_test_url "https://example.com/path?param=value" "complex-test"
test_redirect_safe "Complex URL redirect" "complex-test" "302" "https://example.com/path?param=value" "Should handle URLs with paths and query params"

echo -e "${YELLOW}📋 2. Invalid Short Code Tests${NC}"

test_error_safe "Non-existent short code" "nonexistent123" "404" "Should return 404 for non-existent codes"
test_error_safe "Too short code" "ab" "400" "Should reject codes that are too short"
test_error_safe "Too long code" "verylongshortcodethatexceedslimit" "400" "Should reject codes that are too long"

echo -e "${YELLOW}📋 3. Expired URL Tests${NC}"

# Create expired URL
curl -s -X POST "$API_URL/urls" \
    -H "Content-Type: application/json" \
    -d '{"original_url": "https://expired.example.com", "custom_alias": "expired-test", "expires_at": "2023-01-01T00:00:00Z"}' > /dev/null
sleep 1

test_error_safe "Expired URL" "expired-test" "410" "Should return 410 Gone for expired URLs"

echo -e "${YELLOW}📋 4. Case Sensitivity Tests${NC}"

create_test_url "https://case.example.com" "CaseTest"
test_redirect_safe "Exact case match" "CaseTest" "302" "https://case.example.com" "Should match exact case"
test_error_safe "Different case" "casetest" "404" "Should be case sensitive"

echo -e "${YELLOW}📋 5. Performance Test${NC}"

# Test redirect performance
echo -e "${YELLOW}Testing redirect performance...${NC}"
start_time=$(date +%s%3N)
response=$(curl -s -w "TIME:%{time_total}" -o /dev/null "$BASE_URL/google-test")
end_time=$(date +%s%3N)

time_total=$(echo "$response" | grep -o "TIME:[0-9.]*" | cut -d: -f2)
time_ms=$(echo "$time_total * 1000" | awk '{printf "%.0f", $1}')

if [ "$time_ms" -lt 100 ]; then
    echo -e "${GREEN}✓ Performance: ${time_ms}ms (< 100ms)${NC}"
    ((PASSED++))
else
    echo -e "${RED}✗ Performance: ${time_ms}ms (>= 100ms)${NC}"
    ((FAILED++))
fi
((TOTAL++))

# Final results
echo ""
echo "============================================================"
echo -e "${BLUE}📊 Test Results Summary${NC}"
echo "Total Tests: $TOTAL"
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 All redirection tests passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ Some redirection tests failed.${NC}"
    exit 1
fi
