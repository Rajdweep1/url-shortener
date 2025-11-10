#!/bin/bash

# Comprehensive URL Creation Edge Cases Test
# Tests all possible scenarios for short URL creation API

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

# Test function
test_case() {
    local name="$1"
    local expected_status="$2"
    local payload="$3"
    local description="$4"
    
    echo -e "${YELLOW}Testing: $name${NC}"
    if [ -n "$description" ]; then
        echo "  Description: $description"
    fi
    
    response=$(curl -s -w "%{http_code}" -X POST "$BASE_URL/api/v1/urls" \
        -H "Content-Type: application/json" \
        -d "$payload")
    
    status_code="${response: -3}"
    response_body="${response%???}"
    
    if [ "$status_code" = "$expected_status" ]; then
        echo -e "  ${GREEN}✓ PASSED${NC} (Expected: $expected_status, Got: $status_code)"
        ((PASSED++))
    else
        echo -e "  ${RED}✗ FAILED${NC} (Expected: $expected_status, Got: $status_code)"
        echo "  Response: $response_body"
        ((FAILED++))
    fi
    echo
}

echo -e "${BLUE}🧪 Comprehensive URL Creation Edge Cases Test${NC}"
echo "=============================================="
echo

# 1. BASIC VALIDATION TESTS
echo -e "${YELLOW}📋 1. Basic Input Validation${NC}"

test_case "Empty JSON" "400" '{}' "Missing required original_url field"
test_case "Empty URL" "400" '{"original_url": ""}' "Empty URL should be rejected"
test_case "Null URL" "400" '{"original_url": null}' "Null URL should be rejected"
test_case "Non-string URL" "400" '{"original_url": 123}' "Non-string URL should be rejected"
test_case "Invalid JSON" "400" '{"original_url": "https://example.com"' "Malformed JSON should be rejected"

# 2. URL FORMAT VALIDATION
echo -e "${YELLOW}📋 2. URL Format Validation${NC}"

test_case "Valid HTTPS URL" "201" '{"original_url": "https://example.com"}' "Standard HTTPS URL should work"
test_case "Valid HTTP URL" "201" '{"original_url": "http://example.com"}' "Standard HTTP URL should work"
test_case "URL with path" "201" '{"original_url": "https://example.com/path/to/page"}' "URL with path should work"
test_case "URL with query params" "201" '{"original_url": "https://example.com?param=value&other=123"}' "URL with query parameters should work"
test_case "URL with fragment" "201" '{"original_url": "https://example.com#section"}' "URL with fragment should work"
test_case "URL with port" "201" '{"original_url": "https://example.com:8080"}' "URL with port should work"
test_case "URL with subdomain" "201" '{"original_url": "https://sub.example.com"}' "URL with subdomain should work"

# 3. INVALID URL SCHEMES
echo -e "${YELLOW}📋 3. Invalid URL Schemes${NC}"

test_case "FTP scheme" "400" '{"original_url": "ftp://example.com"}' "FTP URLs should be rejected"
test_case "File scheme" "400" '{"original_url": "file:///etc/passwd"}' "File URLs should be rejected"
test_case "JavaScript scheme" "400" '{"original_url": "javascript:alert(1)"}' "JavaScript URLs should be rejected"
test_case "Data scheme" "400" '{"original_url": "data:text/html,<script>alert(1)</script>"}' "Data URLs should be rejected"
test_case "No scheme" "400" '{"original_url": "example.com"}' "URLs without scheme should be rejected"
test_case "Invalid scheme" "400" '{"original_url": "custom://example.com"}' "Custom schemes should be rejected"

# 4. URL LENGTH VALIDATION
echo -e "${YELLOW}📋 4. URL Length Validation${NC}"

test_case "Very short URL" "400" '{"original_url": "http://a"}' "URLs too short should be rejected"
test_case "Minimum valid URL" "201" '{"original_url": "http://example.com"}' "Minimum valid URL should work"

# Generate very long URL (over 2048 characters)
LONG_PATH=$(printf 'very-long-path-segment/%.0s' {1..100})
LONG_URL="https://example.com/$LONG_PATH"
test_case "Very long URL" "400" "{\"original_url\": \"$LONG_URL\"}" "URLs over 2048 chars should be rejected"

# 5. SECURITY VALIDATION
echo -e "${YELLOW}📋 5. Security Validation${NC}"

test_case "Localhost URL" "400" '{"original_url": "http://localhost"}' "Localhost URLs should be rejected"
test_case "127.0.0.1 URL" "400" '{"original_url": "http://127.0.0.1"}' "127.0.0.1 URLs should be rejected"
test_case "Private IP 192.168" "400" '{"original_url": "http://192.168.1.1"}' "Private IP 192.168.x.x should be rejected"
test_case "Private IP 10.x" "400" '{"original_url": "http://10.0.0.1"}' "Private IP 10.x.x.x should be rejected"
test_case "Private IP 172.16" "400" '{"original_url": "http://172.16.0.1"}' "Private IP 172.16-31.x.x should be rejected"

# 6. CUSTOM ALIAS VALIDATION
echo -e "${YELLOW}📋 6. Custom Alias Validation${NC}"

test_case "Valid custom alias" "201" '{"original_url": "https://example.com/custom", "custom_alias": "my-link"}' "Valid custom alias should work"
test_case "Custom alias with numbers" "201" '{"original_url": "https://example.com/numbers", "custom_alias": "link123"}' "Custom alias with numbers should work"
test_case "Custom alias with underscores" "201" '{"original_url": "https://example.com/underscores", "custom_alias": "my_link"}' "Custom alias with underscores should work"
test_case "Empty custom alias" "400" '{"original_url": "https://example.com/empty", "custom_alias": ""}' "Empty custom alias should be rejected"
test_case "Null custom alias" "201" '{"original_url": "https://example.com/null", "custom_alias": null}' "Null custom alias should default to generated"

test_case "Custom alias too short" "400" '{"original_url": "https://example.com", "custom_alias": "ab"}' "Custom alias under 3 chars should be rejected"
test_case "Custom alias too long" "400" '{"original_url": "https://example.com", "custom_alias": "this-is-a-very-very-very-very-very-long-custom-alias-that-exceeds-the-maximum-allowed-length"}' "Custom alias over 50 chars should be rejected"
test_case "Custom alias with spaces" "400" '{"original_url": "https://example.com", "custom_alias": "my link"}' "Custom alias with spaces should be rejected"
test_case "Custom alias with special chars" "400" '{"original_url": "https://example.com", "custom_alias": "my@link"}' "Custom alias with special chars should be rejected"
test_case "Reserved word alias" "400" '{"original_url": "https://example.com", "custom_alias": "api"}' "Reserved words as alias should be rejected"
test_case "Reserved word admin" "400" '{"original_url": "https://example.com", "custom_alias": "admin"}' "Reserved word 'admin' should be rejected"

# 7. EXPIRATION VALIDATION
echo -e "${YELLOW}📋 7. Expiration Validation${NC}"

test_case "Valid expiration" "201" '{"original_url": "https://example.com", "expires_in_days": 30}' "Valid expiration should work"
test_case "Zero expiration" "201" '{"original_url": "https://example.com", "expires_in_days": 0}' "Zero expiration should work (immediate expiry)"
test_case "Negative expiration" "400" '{"original_url": "https://example.com", "expires_in_days": -1}' "Negative expiration should be rejected"
test_case "Very large expiration" "201" '{"original_url": "https://example.com", "expires_in_days": 36500}' "Large expiration (100 years) should work"
test_case "Non-integer expiration" "400" '{"original_url": "https://example.com", "expires_in_days": "30"}' "Non-integer expiration should be rejected"
test_case "Float expiration" "400" '{"original_url": "https://example.com", "expires_in_days": 30.5}' "Float expiration should be rejected"

# 8. USER ID VALIDATION
echo -e "${YELLOW}📋 8. User ID Validation${NC}"

test_case "Valid user ID" "201" '{"original_url": "https://example.com", "user_id": "user-123"}' "Valid user ID should work"
test_case "Empty user ID" "201" '{"original_url": "https://example.com", "user_id": ""}' "Empty user ID should work (anonymous)"
test_case "Null user ID" "201" '{"original_url": "https://example.com", "user_id": null}' "Null user ID should work (anonymous)"
test_case "Very long user ID" "201" '{"original_url": "https://example.com", "user_id": "user-with-very-long-identifier-that-might-cause-issues-in-some-systems"}' "Long user ID should work"
test_case "Non-string user ID" "400" '{"original_url": "https://example.com", "user_id": 123}' "Non-string user ID should be rejected"

# 9. UNICODE AND INTERNATIONAL DOMAINS
echo -e "${YELLOW}📋 9. Unicode and International Domains${NC}"

test_case "Unicode domain" "201" '{"original_url": "https://测试.com"}' "Unicode domain should work"
test_case "Punycode domain" "201" '{"original_url": "https://xn--nxasmq6b.com"}' "Punycode domain should work"
test_case "Unicode path" "201" '{"original_url": "https://example.com/测试/页面"}' "Unicode in path should work"
test_case "Unicode query" "201" '{"original_url": "https://example.com?query=测试"}' "Unicode in query should work"

# 10. EDGE CASE COMBINATIONS
echo -e "${YELLOW}📋 10. Edge Case Combinations${NC}"

test_case "All valid fields" "201" '{"original_url": "https://example.com/test", "custom_alias": "combo-test", "expires_in_days": 7, "user_id": "user-123"}' "All valid fields together should work"
test_case "URL with credentials" "201" '{"original_url": "https://user:pass@example.com"}' "URL with credentials should work"
test_case "URL with unusual port" "201" '{"original_url": "https://example.com:65535"}' "URL with max port should work"
test_case "URL with encoded chars" "201" '{"original_url": "https://example.com/path%20with%20spaces"}' "URL with encoded characters should work"

# 11. DUPLICATE HANDLING
echo -e "${YELLOW}📋 11. Duplicate Handling${NC}"

# Create a URL first
UNIQUE_URL="https://example.com/duplicate-test-$(date +%s)"
test_case "First creation" "201" "{\"original_url\": \"$UNIQUE_URL\"}" "First creation should work"
test_case "Duplicate URL" "201" "{\"original_url\": \"$UNIQUE_URL\"}" "Duplicate URL should return existing (idempotency)"

# Try duplicate custom alias
test_case "First custom alias" "201" '{"original_url": "https://example.com/first", "custom_alias": "dup-test"}' "First custom alias should work"
test_case "Duplicate custom alias" "409" '{"original_url": "https://example.com/second", "custom_alias": "dup-test"}' "Duplicate custom alias should be rejected"

# 12. MALFORMED REQUEST BODY
echo -e "${YELLOW}📋 12. Malformed Request Body${NC}"

test_case "Extra fields" "201" '{"original_url": "https://example.com", "extra_field": "should_be_ignored"}' "Extra fields should be ignored"
test_case "Nested objects" "400" '{"original_url": {"nested": "https://example.com"}}' "Nested objects should be rejected"
test_case "Array as URL" "400" '{"original_url": ["https://example.com"]}' "Array as URL should be rejected"

# 13. CONTENT-TYPE VALIDATION
echo -e "${YELLOW}📋 13. Content-Type Validation${NC}"

# Test with wrong content type
echo -e "${YELLOW}Testing: Wrong Content-Type${NC}"
response=$(curl -s -w "%{http_code}" -X POST "$BASE_URL/api/v1/urls" \
    -H "Content-Type: text/plain" \
    -d '{"original_url": "https://example.com"}')
status_code="${response: -3}"
if [ "$status_code" = "415" ]; then
    echo -e "  ${GREEN}✓ PASSED${NC} (Expected: 415, Got: $status_code)"
    ((PASSED++))
else
    echo -e "  ${RED}✗ FAILED${NC} (Expected: 415, Got: $status_code)"
    ((FAILED++))
fi
echo

# 14. LARGE PAYLOAD HANDLING
echo -e "${YELLOW}📋 14. Large Payload Handling${NC}"

# Create a very large JSON payload
LARGE_PAYLOAD='{"original_url": "https://example.com", "user_id": "'$(printf 'x%.0s' {1..10000})'"}'
test_case "Very large payload" "400" "$LARGE_PAYLOAD" "Large payload with oversized user_id should be rejected"

# Summary
echo "=============================================="
echo -e "${BLUE}📊 Test Results Summary${NC}"
echo "Total Tests: $((PASSED + FAILED))"
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 All edge case tests passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ Some edge case tests failed.${NC}"
    exit 1
fi
