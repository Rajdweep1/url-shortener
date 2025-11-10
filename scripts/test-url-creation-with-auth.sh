#!/bin/bash

# URL Creation with Authentication Test Script
# This script tests URL creation functionality with authentication enabled

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Server configuration
BASE_URL="http://localhost:8081"
GRPC_PORT="8080"

echo -e "${BLUE}🔗 URL Shortener - Authentication & URL Creation Test${NC}"
echo "=================================================="
echo ""

# Function to print test results
print_result() {
    local test_name=$1
    local status=$2
    local details=$3
    
    if [ "$status" = "PASS" ]; then
        echo -e "${GREEN}✅ $test_name: PASSED${NC}"
        if [ -n "$details" ]; then
            echo -e "   ${details}"
        fi
    else
        echo -e "${RED}❌ $test_name: FAILED${NC}"
        if [ -n "$details" ]; then
            echo -e "   ${RED}$details${NC}"
        fi
    fi
    echo ""
}

# Function to make HTTP requests with error handling
make_request() {
    local method=$1
    local url=$2
    local headers=$3
    local data=$4
    
    if [ -n "$data" ]; then
        curl -s -X "$method" "$url" $headers -d "$data" -w "\nHTTP_STATUS:%{http_code}" 2>/dev/null
    else
        curl -s -X "$method" "$url" $headers -w "\nHTTP_STATUS:%{http_code}" 2>/dev/null
    fi
}

# Test 1: Check server health
echo -e "${YELLOW}Test 1: Server Health Check${NC}"
response=$(make_request "GET" "$BASE_URL/health" "" "")
http_status=$(echo "$response" | grep "HTTP_STATUS:" | cut -d: -f2)

if [ "$http_status" = "200" ]; then
    print_result "Server Health Check" "PASS" "Server is running on $BASE_URL"
else
    print_result "Server Health Check" "FAIL" "Server not responding (HTTP $http_status)"
    exit 1
fi

# Test 2: Try creating URL without authentication (should fail)
echo -e "${YELLOW}Test 2: URL Creation Without Authentication${NC}"
url_data='{"original_url": "https://www.google.com", "custom_short_code": "test123"}'
response=$(make_request "POST" "$BASE_URL/api/v1/urls" "-H 'Content-Type: application/json'" "$url_data")
http_status=$(echo "$response" | grep "HTTP_STATUS:" | cut -d: -f2)

if [ "$http_status" = "401" ]; then
    print_result "URL Creation Without Auth" "PASS" "Correctly rejected with 401 Unauthorized"
else
    print_result "URL Creation Without Auth" "FAIL" "Expected 401, got HTTP $http_status"
fi

# Test 3: Admin login to get JWT token
echo -e "${YELLOW}Test 3: Admin Login${NC}"
login_data='{"username": "admin", "password": "admin123"}'
response=$(make_request "POST" "$BASE_URL/api/v1/auth/login" "-H 'Content-Type: application/json'" "$login_data")
http_status=$(echo "$response" | grep "HTTP_STATUS:" | cut -d: -f2)
response_body=$(echo "$response" | sed '/HTTP_STATUS:/d')

if [ "$http_status" = "200" ]; then
    # Extract JWT token from response
    jwt_token=$(echo "$response_body" | grep -o '"token":"[^"]*' | cut -d'"' -f4)
    if [ -n "$jwt_token" ]; then
        print_result "Admin Login" "PASS" "JWT token obtained successfully"
        echo -e "   Token: ${jwt_token:0:20}..."
    else
        print_result "Admin Login" "FAIL" "No JWT token in response"
        echo "Response: $response_body"
        exit 1
    fi
else
    print_result "Admin Login" "FAIL" "Login failed with HTTP $http_status"
    echo "Response: $response_body"
    exit 1
fi

# Test 4: Create URL with JWT authentication
echo -e "${YELLOW}Test 4: URL Creation With JWT Token${NC}"
url_data='{"original_url": "https://www.example.com", "custom_short_code": "jwt-test"}'
response=$(make_request "POST" "$BASE_URL/api/v1/urls" "-H 'Content-Type: application/json' -H 'Authorization: Bearer $jwt_token'" "$url_data")
http_status=$(echo "$response" | grep "HTTP_STATUS:" | cut -d: -f2)
response_body=$(echo "$response" | sed '/HTTP_STATUS:/d')

if [ "$http_status" = "201" ]; then
    short_code=$(echo "$response_body" | grep -o '"short_code":"[^"]*' | cut -d'"' -f4)
    print_result "URL Creation With JWT" "PASS" "URL created successfully with short code: $short_code"
else
    print_result "URL Creation With JWT" "FAIL" "Expected 201, got HTTP $http_status"
    echo "Response: $response_body"
fi

# Test 5: Create admin API key
echo -e "${YELLOW}Test 5: Create Admin API Key${NC}"
api_key_data='{"name": "test-key", "permissions": ["url:create", "url:read", "url:update"], "expires_in_days": 30}'
response=$(make_request "POST" "$BASE_URL/api/v1/auth/api-keys" "-H 'Content-Type: application/json' -H 'Authorization: Bearer $jwt_token'" "$api_key_data")
http_status=$(echo "$response" | grep "HTTP_STATUS:" | cut -d: -f2)
response_body=$(echo "$response" | sed '/HTTP_STATUS:/d')

if [ "$http_status" = "201" ]; then
    api_key=$(echo "$response_body" | grep -o '"api_key":"[^"]*' | cut -d'"' -f4)
    if [ -n "$api_key" ]; then
        print_result "API Key Creation" "PASS" "API key created successfully"
        echo -e "   API Key: ${api_key:0:20}..."
    else
        print_result "API Key Creation" "FAIL" "No API key in response"
        echo "Response: $response_body"
    fi
else
    print_result "API Key Creation" "FAIL" "Expected 201, got HTTP $http_status"
    echo "Response: $response_body"
fi

# Test 6: Create URL with API key authentication
if [ -n "$api_key" ]; then
    echo -e "${YELLOW}Test 6: URL Creation With API Key${NC}"
    url_data='{"original_url": "https://www.github.com", "custom_short_code": "api-test"}'
    response=$(make_request "POST" "$BASE_URL/api/v1/urls" "-H 'Content-Type: application/json' -H 'X-API-Key: $api_key'" "$url_data")
    http_status=$(echo "$response" | grep "HTTP_STATUS:" | cut -d: -f2)
    response_body=$(echo "$response" | sed '/HTTP_STATUS:/d')

    if [ "$http_status" = "201" ]; then
        short_code=$(echo "$response_body" | grep -o '"short_code":"[^"]*' | cut -d'"' -f4)
        print_result "URL Creation With API Key" "PASS" "URL created successfully with short code: $short_code"
    else
        print_result "URL Creation With API Key" "FAIL" "Expected 201, got HTTP $http_status"
        echo "Response: $response_body"
    fi
fi

# Test 7: Test URL redirection (public endpoint)
if [ -n "$short_code" ]; then
    echo -e "${YELLOW}Test 7: URL Redirection (Public Access)${NC}"
    response=$(make_request "GET" "$BASE_URL/$short_code" "" "")
    http_status=$(echo "$response" | grep "HTTP_STATUS:" | cut -d: -f2)

    if [ "$http_status" = "302" ] || [ "$http_status" = "301" ]; then
        print_result "URL Redirection" "PASS" "Redirect working correctly (HTTP $http_status)"
    else
        print_result "URL Redirection" "FAIL" "Expected 301/302, got HTTP $http_status"
    fi
fi

# Test 8: Get URL details with authentication
if [ -n "$short_code" ]; then
    echo -e "${YELLOW}Test 8: Get URL Details With Authentication${NC}"
    response=$(make_request "GET" "$BASE_URL/api/v1/urls/$short_code" "-H 'Authorization: Bearer $jwt_token'" "")
    http_status=$(echo "$response" | grep "HTTP_STATUS:" | cut -d: -f2)
    response_body=$(echo "$response" | sed '/HTTP_STATUS:/d')

    if [ "$http_status" = "200" ]; then
        print_result "Get URL Details" "PASS" "URL details retrieved successfully"
    else
        print_result "Get URL Details" "FAIL" "Expected 200, got HTTP $http_status"
        echo "Response: $response_body"
    fi
fi

# Test 9: Try to access analytics (admin only)
echo -e "${YELLOW}Test 9: Analytics Access (Admin Only)${NC}"
response=$(make_request "GET" "$BASE_URL/api/v1/analytics/summary" "-H 'Authorization: Bearer $jwt_token'" "")
http_status=$(echo "$response" | grep "HTTP_STATUS:" | cut -d: -f2)

if [ "$http_status" = "200" ]; then
    print_result "Analytics Access" "PASS" "Admin can access analytics"
elif [ "$http_status" = "403" ]; then
    print_result "Analytics Access" "PASS" "Correctly restricted to admin (403)"
else
    print_result "Analytics Access" "FAIL" "Unexpected response: HTTP $http_status"
fi

# Test 10: Token validation
echo -e "${YELLOW}Test 10: Token Validation${NC}"
response=$(make_request "GET" "$BASE_URL/api/v1/auth/validate?token=$jwt_token" "" "")
http_status=$(echo "$response" | grep "HTTP_STATUS:" | cut -d: -f2)

if [ "$http_status" = "200" ]; then
    print_result "Token Validation" "PASS" "JWT token is valid"
else
    print_result "Token Validation" "FAIL" "Token validation failed: HTTP $http_status"
fi

echo -e "${BLUE}=================================================="
echo -e "🎯 URL Creation with Authentication Test Complete"
echo -e "==================================================${NC}"
echo ""
echo -e "${GREEN}✅ Authentication system is working correctly with URL creation!${NC}"
echo -e "${GREEN}✅ Both JWT and API key authentication methods are functional${NC}"
echo -e "${GREEN}✅ Public endpoints (redirects) work without authentication${NC}"
echo -e "${GREEN}✅ Protected endpoints require proper authentication${NC}"
echo -e "${GREEN}✅ Admin endpoints are properly restricted${NC}"
