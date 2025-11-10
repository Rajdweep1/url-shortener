#!/bin/bash

# Authentication Testing Script for URL Shortener
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
BASE_URL="http://localhost:8081"
API_BASE="$BASE_URL/api/v1"

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Function to make HTTP requests with error handling
make_request() {
    local method=$1
    local url=$2
    local data=$3
    local headers=$4
    local expected_status=$5
    
    print_status "Making $method request to $url"
    
    if [ -n "$data" ]; then
        if [ -n "$headers" ]; then
            response=$(curl -s -w "\n%{http_code}" -X "$method" "$url" \
                -H "Content-Type: application/json" \
                -H "$headers" \
                -d "$data")
        else
            response=$(curl -s -w "\n%{http_code}" -X "$method" "$url" \
                -H "Content-Type: application/json" \
                -d "$data")
        fi
    else
        if [ -n "$headers" ]; then
            response=$(curl -s -w "\n%{http_code}" -X "$method" "$url" \
                -H "$headers")
        else
            response=$(curl -s -w "\n%{http_code}" -X "$method" "$url")
        fi
    fi
    
    # Extract status code and body
    status_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n -1)
    
    echo "Response Status: $status_code"
    echo "Response Body: $body"
    
    if [ "$status_code" = "$expected_status" ]; then
        print_success "Request successful (Status: $status_code)"
        echo "$body"
    else
        print_error "Request failed. Expected: $expected_status, Got: $status_code"
        echo "$body"
        return 1
    fi
}

# Test 1: Health Check (Public endpoint)
test_health_check() {
    print_status "Testing health check endpoint..."
    make_request "GET" "$API_BASE/health" "" "" "200"
    echo ""
}

# Test 2: Login with valid credentials
test_login_success() {
    print_status "Testing login with valid credentials..."
    login_data='{"username": "admin", "password": "admin"}'
    
    response=$(make_request "POST" "$API_BASE/auth/login" "$login_data" "" "200")
    
    # Extract JWT token from response
    JWT_TOKEN=$(echo "$response" | jq -r '.token')
    if [ "$JWT_TOKEN" = "null" ] || [ -z "$JWT_TOKEN" ]; then
        print_error "Failed to extract JWT token from login response"
        return 1
    fi
    
    print_success "Login successful. JWT Token: ${JWT_TOKEN:0:50}..."
    echo ""
}

# Test 3: Login with invalid credentials
test_login_failure() {
    print_status "Testing login with invalid credentials..."
    login_data='{"username": "admin", "password": "wrong"}'
    
    make_request "POST" "$API_BASE/auth/login" "$login_data" "" "401"
    echo ""
}

# Test 4: Validate JWT token
test_validate_token() {
    print_status "Testing token validation..."
    
    if [ -z "$JWT_TOKEN" ]; then
        print_error "No JWT token available for validation"
        return 1
    fi
    
    make_request "POST" "$API_BASE/auth/validate" "" "Authorization: Bearer $JWT_TOKEN" "200"
    echo ""
}

# Test 5: Access protected endpoint with JWT
test_protected_endpoint_jwt() {
    print_status "Testing protected endpoint with JWT..."
    
    if [ -z "$JWT_TOKEN" ]; then
        print_error "No JWT token available"
        return 1
    fi
    
    make_request "GET" "$API_BASE/auth/profile" "" "Authorization: Bearer $JWT_TOKEN" "200"
    echo ""
}

# Test 6: Create API Key
test_create_api_key() {
    print_status "Testing API key creation..."
    
    if [ -z "$JWT_TOKEN" ]; then
        print_error "No JWT token available"
        return 1
    fi
    
    api_key_data='{"name": "Test API Key", "permissions": ["urls:read", "urls:write"]}'
    
    response=$(make_request "POST" "$API_BASE/auth/api-keys" "$api_key_data" "Authorization: Bearer $JWT_TOKEN" "201")
    
    # Extract API key from response
    API_KEY=$(echo "$response" | jq -r '.api_key')
    if [ "$API_KEY" = "null" ] || [ -z "$API_KEY" ]; then
        print_error "Failed to extract API key from response"
        return 1
    fi
    
    print_success "API Key created: ${API_KEY:0:20}..."
    echo ""
}

# Test 7: Access protected endpoint with API Key
test_protected_endpoint_api_key() {
    print_status "Testing protected endpoint with API Key..."
    
    if [ -z "$API_KEY" ]; then
        print_error "No API key available"
        return 1
    fi
    
    make_request "GET" "$API_BASE/auth/profile" "" "X-API-Key: $API_KEY" "200"
    echo ""
}

# Test 8: Create URL with JWT authentication
test_create_url_jwt() {
    print_status "Testing URL creation with JWT..."
    
    if [ -z "$JWT_TOKEN" ]; then
        print_error "No JWT token available"
        return 1
    fi
    
    url_data='{"original_url": "https://example.com", "custom_alias": "test-jwt"}'
    
    make_request "POST" "$API_BASE/urls" "$url_data" "Authorization: Bearer $JWT_TOKEN" "201"
    echo ""
}

# Test 9: Create URL with API Key authentication
test_create_url_api_key() {
    print_status "Testing URL creation with API Key..."
    
    if [ -z "$API_KEY" ]; then
        print_error "No API key available"
        return 1
    fi
    
    url_data='{"original_url": "https://example.com/api", "custom_alias": "test-api"}'
    
    make_request "POST" "$API_BASE/urls" "$url_data" "X-API-Key: $API_KEY" "201"
    echo ""
}

# Test 10: Access protected endpoint without authentication
test_unauthorized_access() {
    print_status "Testing unauthorized access to protected endpoint..."
    
    make_request "GET" "$API_BASE/auth/profile" "" "" "401"
    echo ""
}

# Test 11: Access admin endpoint with invalid token
test_invalid_token() {
    print_status "Testing access with invalid token..."
    
    make_request "GET" "$API_BASE/auth/profile" "" "Authorization: Bearer invalid-token" "401"
    echo ""
}

# Main test execution
main() {
    print_status "Starting Authentication System Tests"
    print_status "===================================="
    echo ""
    
    # Check if jq is available
    if ! command -v jq &> /dev/null; then
        print_error "jq is required for JSON parsing. Please install jq."
        exit 1
    fi
    
    # Check if server is running
    if ! curl -s "$API_BASE/health" > /dev/null; then
        print_error "Server is not running at $BASE_URL"
        print_status "Please start the server with: docker-compose up"
        exit 1
    fi
    
    # Run tests
    test_health_check
    test_login_success
    test_login_failure
    test_validate_token
    test_protected_endpoint_jwt
    test_create_api_key
    test_protected_endpoint_api_key
    test_create_url_jwt
    test_create_url_api_key
    test_unauthorized_access
    test_invalid_token
    
    print_success "All authentication tests completed!"
    print_status "JWT Token: ${JWT_TOKEN:0:50}..."
    print_status "API Key: ${API_KEY:0:20}..."
}

# Run main function
main "$@"
