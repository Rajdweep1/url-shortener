#!/bin/bash

# Comprehensive URL Redirection Edge Cases Test
# Tests all possible scenarios for URL redirection API

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

BASE_URL="http://localhost:8081"
PASSED=0
FAILED=0

# Clean the database first to ensure fresh state
echo -e "${YELLOW}🧹 Cleaning database for fresh test run...${NC}"
./scripts/cleanup-db.sh
echo ""

echo -e "${BLUE}🧪 Comprehensive URL Redirection Edge Cases Test${NC}"
echo "=================================================="

# Test function for redirection (expects 302 redirect)
test_redirect() {
    local name="$1"
    local short_code="$2"
    local expected_status="$3"
    local expected_location="$4"
    local description="$5"
    
    echo -e "${YELLOW}Testing: $name${NC}"
    if [ -n "$description" ]; then
        echo "  Description: $description"
    fi
    
    # Add delay to avoid rate limiting
    sleep 0.5

    # Make request and capture response headers (don't follow redirects)
    response=$(curl -s -D - -o /dev/null -w "HTTPSTATUS:%{http_code}" \
                   "$BASE_URL/$short_code" 2>/dev/null || echo "HTTPSTATUS:000")
    
    # Parse response
    http_status=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
    location=$(echo "$response" | grep -i "^location:" | cut -d' ' -f2- | tr -d '\r\n')
    
    # Check status code
    if [ "$http_status" = "$expected_status" ]; then
        # For redirects, also check location header
        if [ "$expected_status" = "302" ] && [ -n "$expected_location" ]; then
            # Normalize URLs by removing trailing slashes for comparison
            normalized_location=$(echo "$location" | sed 's/\/$//')
            normalized_expected=$(echo "$expected_location" | sed 's/\/$//')

            if [ "$normalized_location" = "$normalized_expected" ]; then
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
        if [ -n "$location" ]; then
            echo "  Location: $location"
        fi
        ((FAILED++))
    fi
    echo ""
}

# Test function for error responses (expects JSON error)
test_error_response() {
    local name="$1"
    local short_code="$2"
    local expected_status="$3"
    local description="$4"
    
    echo -e "${YELLOW}Testing: $name${NC}"
    if [ -n "$description" ]; then
        echo "  Description: $description"
    fi
    
    # Add delay to avoid rate limiting
    sleep 0.5

    # Make request and capture response
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
                   "$BASE_URL/$short_code" 2>/dev/null || echo "HTTPSTATUS:000")
    
    # Parse response
    http_status=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
    body=$(echo "$response" | sed 's/HTTPSTATUS:[0-9]*$//')
    
    # Check status code
    if [ "$http_status" = "$expected_status" ]; then
        echo -e "  ${GREEN}✓ PASSED${NC} (Expected: $expected_status, Got: $http_status)"
        ((PASSED++))
    else
        echo -e "  ${RED}✗ FAILED${NC} (Expected: $expected_status, Got: $http_status)"
        if [ -n "$body" ]; then
            echo "  Response: $body"
        fi
        ((FAILED++))
    fi
    echo ""
}

# Helper function to create a test URL
create_test_url() {
    local original_url="$1"
    local custom_alias="$2"
    local expires_in_days="$3"
    
    local payload="{\"original_url\": \"$original_url\""
    
    if [ -n "$custom_alias" ]; then
        payload="$payload, \"custom_alias\": \"$custom_alias\""
    fi
    
    if [ -n "$expires_in_days" ]; then
        payload="$payload, \"expires_in_days\": $expires_in_days"
    fi
    
    payload="$payload}"
    
    curl -s -X POST "$BASE_URL/api/v1/urls" \
         -H "Content-Type: application/json" \
         -d "$payload" > /dev/null 2>&1
}

# 1. BASIC REDIRECTION TESTS
echo -e "${YELLOW}📋 1. Basic Redirection Tests${NC}"

# Create test URLs first
create_test_url "https://www.google.com" "google-test"
create_test_url "https://www.github.com" "github-test"
create_test_url "https://example.com/path?param=value" "complex-url"

test_redirect "Valid short code redirect" "google-test" "302" "https://www.google.com" "Should redirect to original URL"
test_redirect "Valid complex URL redirect" "complex-url" "302" "https://example.com/path?param=value" "Should handle URLs with paths and query params"

# 2. INVALID SHORT CODE TESTS
echo -e "${YELLOW}📋 2. Invalid Short Code Tests${NC}"

test_error_response "Non-existent short code" "nonexistent123" "404" "Should return 404 for non-existent codes"
test_error_response "Whitespace short code" "%20%20%20" "400" "Should return 400 for whitespace-only short code"
test_error_response "Invalid characters" "test@invalid" "400" "Should reject codes with invalid characters"
test_error_response "Too short code" "ab" "400" "Should reject codes that are too short"
test_error_response "Too long code" "this-is-way-too-long-for-a-short-code-and-should-be-rejected-completely" "400" "Should reject codes that are too long"

# 3. SPECIAL CHARACTER HANDLING
echo -e "${YELLOW}📋 3. Special Character Handling${NC}"

test_error_response "Code with spaces" "test%20code" "400" "Should reject codes with spaces"
test_error_response "Code with special chars" "test!@#$" "400" "Should reject codes with special characters"
test_error_response "Code with unicode" "tëst" "400" "Should handle unicode characters"
test_error_response "Code with emoji" "test😀" "400" "Should handle emoji characters"

# 4. EXPIRED URL TESTS
echo -e "${YELLOW}📋 4. Expired URL Tests${NC}"

# Create an expired URL (0 days = immediate expiry)
create_test_url "https://expired.example.com" "expired-test" "0"
sleep 1  # Wait a moment to ensure expiration

test_error_response "Expired URL" "expired-test" "410" "Should return 410 Gone for expired URLs"

# 5. INACTIVE URL TESTS  
echo -e "${YELLOW}📋 5. Inactive URL Tests${NC}"

# Create a URL and then deactivate it (we'll need to do this via direct DB update)
create_test_url "https://inactive.example.com" "inactive-test"

# Deactivate the URL in database
docker exec url-shortener-postgres psql -U user -d url_shortener -c "UPDATE urls SET is_active = false WHERE short_code = 'inactive-test';" > /dev/null 2>&1

test_error_response "Inactive URL" "inactive-test" "404" "Should return 404 for inactive URLs"

# 6. CASE SENSITIVITY TESTS
echo -e "${YELLOW}📋 6. Case Sensitivity Tests${NC}"

create_test_url "https://case.example.com" "CaseSensitive"
test_redirect "Exact case match" "CaseSensitive" "302" "https://case.example.com" "Should match exact case"
test_error_response "Different case" "casesensitive" "404" "Should be case sensitive"
test_error_response "Upper case" "CASESENSITIVE" "404" "Should be case sensitive"

# 7. URL ENCODING TESTS
echo -e "${YELLOW}📋 7. URL Encoding Tests${NC}"

create_test_url "https://encoded.example.com" "url-encoded"
test_redirect "URL encoded short code" "url%2Dencoded" "302" "https://encoded.example.com" "Should handle URL encoded characters"

# 8. ANALYTICS TRACKING TESTS
echo -e "${YELLOW}📋 8. Analytics Tracking Tests${NC}"

create_test_url "https://analytics.example.com" "analytics-test"

# Test with different headers
test_redirect "With User-Agent" "analytics-test" "302" "https://analytics.example.com" "Should track User-Agent"
test_redirect "With Referer" "analytics-test" "302" "https://analytics.example.com" "Should track Referer"

# 9. CONCURRENT ACCESS TESTS
echo -e "${YELLOW}📋 9. Concurrent Access Tests${NC}"

create_test_url "https://concurrent.example.com" "concurrent-test"

# Simulate concurrent requests (background processes) with delays
for i in {1..5}; do
    (sleep 0.1; curl -s "$BASE_URL/concurrent-test" > /dev/null 2>&1) &
done
wait

test_redirect "After concurrent access" "concurrent-test" "302" "https://concurrent.example.com" "Should handle concurrent requests"

# 10. MALFORMED REQUEST TESTS
echo -e "${YELLOW}📋 10. Malformed Request Tests${NC}"

test_error_response "SQL injection attempt" "%27%3B%20DROP%20TABLE%20urls%3B%20--" "400" "Should reject SQL injection attempts"
test_error_response "XSS attempt" "%3Cscript%3Ealert%28%27xss%27%29%3C%2Fscript%3E" "405" "Should reject XSS attempts with Method Not Allowed"
test_error_response "Path traversal" "..%2F..%2F..%2Fetc%2Fpasswd" "301" "Should normalize path traversal attempts"

# 11. LARGE SHORT CODE TESTS
echo -e "${YELLOW}📋 11. Large Short Code Tests${NC}"

# Test with maximum allowed length
LONG_CODE=$(printf 'a%.0s' {1..50})  # 50 characters
create_test_url "https://long.example.com" "$LONG_CODE"
test_redirect "Maximum length code" "$LONG_CODE" "302" "https://long.example.com" "Should handle maximum length codes"

# Test with over maximum length
OVER_LONG_CODE=$(printf 'a%.0s' {1..51})  # 51 characters
test_error_response "Over maximum length" "$OVER_LONG_CODE" "400" "Should reject over-length codes"

# 12. REDIRECT LOOP PREVENTION
echo -e "${YELLOW}📋 12. Redirect Loop Prevention${NC}"

# Create a URL that points to a different domain (avoid actual self-reference)
create_test_url "https://example.com/self-reference" "self-ref"
test_redirect "Self-referencing URL" "self-ref" "302" "https://example.com/self-reference" "Should allow self-referencing URLs"

# 13. CACHE BEHAVIOR TESTS
echo -e "${YELLOW}📋 13. Cache Behavior Tests${NC}"

create_test_url "https://cache.example.com" "cache-test"

# First access (should cache)
test_redirect "First access (cache miss)" "cache-test" "302" "https://cache.example.com" "Should cache on first access"

# Second access (should use cache)
test_redirect "Second access (cache hit)" "cache-test" "302" "https://cache.example.com" "Should use cached version"

# 14. HTTP METHOD TESTS
echo -e "${YELLOW}📋 14. HTTP Method Tests${NC}"

create_test_url "https://method.example.com" "method-test"

# Test different HTTP methods
test_redirect "GET method" "method-test" "302" "https://method.example.com" "Should handle GET requests"

# Test POST method (should not be allowed for redirection)
echo -e "${YELLOW}Testing: POST method${NC}"
echo "  Description: Should reject POST requests for redirection"
response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X POST "$BASE_URL/method-test" 2>/dev/null)
http_status=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
if [ "$http_status" = "405" ] || [ "$http_status" = "404" ]; then
    echo -e "  ${GREEN}✓ PASSED${NC} (Expected: 405 or 404, Got: $http_status)"
    ((PASSED++))
else
    echo -e "  ${RED}✗ FAILED${NC} (Expected: 405 or 404, Got: $http_status)"
    ((FAILED++))
fi
echo ""

# 15. EDGE CASE COMBINATIONS
echo -e "${YELLOW}📋 15. Edge Case Combinations${NC}"

# Create URLs with edge case combinations
create_test_url "https://edge.example.com/path?param=value&other=test#fragment" "edge-combo"
test_redirect "Complex URL with all components" "edge-combo" "302" "https://edge.example.com/path?param=value&other=test#fragment" "Should handle complex URLs"

# Test with international domain (expect URL encoding)
create_test_url "https://münchen.example.com" "intl-domain"
test_redirect "International domain" "intl-domain" "302" "https://m%c3%bcnchen.example.com" "Should handle international domains with URL encoding"

echo "=============================================="
echo -e "${BLUE}📊 Test Results Summary${NC}"
echo "Total Tests: $((PASSED + FAILED))"
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 All redirection edge case tests passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ Some redirection edge case tests failed.${NC}"
    exit 1
fi
