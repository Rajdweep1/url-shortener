#!/bin/bash

# Comprehensive Authentication Test Script
set -e

BASE_URL="http://localhost:8081"

echo "🔐 Comprehensive Authentication Testing"
echo "======================================"

# Test 1: Health check (public endpoint)
echo ""
echo "Test 1: Health Check (Public Endpoint)"
health_response=$(curl -s -X GET "$BASE_URL/health" -w "\nHTTP_STATUS:%{http_code}")
health_status=$(echo "$health_response" | grep "HTTP_STATUS:" | cut -d: -f2)
echo "Result: HTTP $health_status"

if [ "$health_status" = "200" ]; then
    echo "✅ PASS: Health check accessible without authentication"
else
    echo "❌ FAIL: Health check should be public"
fi

# Test 2: Try accessing protected endpoint without auth
echo ""
echo "Test 2: Protected Endpoint Without Authentication"
echo "Testing: GET /api/v1/urls"
urls_response=$(curl -s -X GET "$BASE_URL/api/v1/urls" -w "\nHTTP_STATUS:%{http_code}")
urls_status=$(echo "$urls_response" | grep "HTTP_STATUS:" | cut -d: -f2)
echo "Result: HTTP $urls_status"

if [ "$urls_status" = "401" ]; then
    echo "✅ PASS: Protected endpoint correctly requires authentication"
else
    echo "❌ FAIL: Expected 401, got $urls_status"
fi

# Test 3: Admin login
echo ""
echo "Test 3: Admin Login"
login_response=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
  -H 'Content-Type: application/json' \
  -d '{"username": "admin", "password": "admin"}' \
  -w "\nHTTP_STATUS:%{http_code}")

login_status=$(echo "$login_response" | grep "HTTP_STATUS:" | cut -d: -f2)
echo "Result: HTTP $login_status"

if [ "$login_status" = "200" ]; then
    jwt_token=$(echo "$login_response" | sed '/HTTP_STATUS:/d' | grep -o '"token":"[^"]*' | cut -d'"' -f4)
    echo "✅ PASS: Admin login successful"
    echo "JWT Token: ${jwt_token:0:30}..."
else
    echo "❌ FAIL: Admin login failed"
    exit 1
fi

# Test 4: Invalid login credentials
echo ""
echo "Test 4: Invalid Login Credentials"
invalid_login=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
  -H 'Content-Type: application/json' \
  -d '{"username": "admin", "password": "wrong"}' \
  -w "\nHTTP_STATUS:%{http_code}")

invalid_status=$(echo "$invalid_login" | grep "HTTP_STATUS:" | cut -d: -f2)
echo "Result: HTTP $invalid_status"

if [ "$invalid_status" = "401" ]; then
    echo "✅ PASS: Invalid credentials correctly rejected"
else
    echo "❌ FAIL: Expected 401, got $invalid_status"
fi

# Test 5: JWT Token Validation
echo ""
echo "Test 5: JWT Token Validation"
validate_response=$(curl -s -X POST "$BASE_URL/api/v1/auth/validate" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $jwt_token" \
  -d '{"token": "'$jwt_token'"}' \
  -w "\nHTTP_STATUS:%{http_code}")

validate_status=$(echo "$validate_response" | grep "HTTP_STATUS:" | cut -d: -f2)
echo "Result: HTTP $validate_status"

if [ "$validate_status" = "200" ]; then
    echo "✅ PASS: JWT token validation successful"
else
    echo "❌ FAIL: JWT token validation failed"
fi

# Test 6: Access protected endpoint with JWT
echo ""
echo "Test 6: Access Protected Endpoint with JWT"
protected_response=$(curl -s -X GET "$BASE_URL/api/v1/urls" \
  -H "Authorization: Bearer $jwt_token" \
  -w "\nHTTP_STATUS:%{http_code}")

protected_status=$(echo "$protected_response" | grep "HTTP_STATUS:" | cut -d: -f2)
echo "Result: HTTP $protected_status"

if [ "$protected_status" = "200" ]; then
    echo "✅ PASS: Protected endpoint accessible with JWT"
else
    echo "❌ FAIL: Expected 200, got $protected_status"
fi

# Test 7: Create URL with JWT authentication
echo ""
echo "Test 7: Create URL with JWT Authentication"
create_response=$(curl -s -X POST "$BASE_URL/api/v1/urls" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $jwt_token" \
  -d '{"original_url": "https://www.github.com", "custom_short_code": "github-test"}' \
  -w "\nHTTP_STATUS:%{http_code}")

create_status=$(echo "$create_response" | grep "HTTP_STATUS:" | cut -d: -f2)
echo "Result: HTTP $create_status"

if [ "$create_status" = "201" ]; then
    short_code=$(echo "$create_response" | sed '/HTTP_STATUS:/d' | grep -o '"short_code":"[^"]*' | cut -d'"' -f4)
    echo "✅ PASS: URL created with JWT auth, short code: $short_code"
else
    echo "❌ FAIL: Expected 201, got $create_status"
fi

# Test 8: Create API Key
echo ""
echo "Test 8: Create API Key"
api_key_response=$(curl -s -X POST "$BASE_URL/api/v1/auth/api-keys" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $jwt_token" \
  -d '{"name": "Test API Key", "permissions": ["url:create", "url:read"]}' \
  -w "\nHTTP_STATUS:%{http_code}")

api_key_status=$(echo "$api_key_response" | grep "HTTP_STATUS:" | cut -d: -f2)
echo "Result: HTTP $api_key_status"

if [ "$api_key_status" = "201" ]; then
    api_key=$(echo "$api_key_response" | sed '/HTTP_STATUS:/d' | grep -o '"api_key":"[^"]*' | cut -d'"' -f4)
    echo "✅ PASS: API key created successfully"
    echo "API Key: ${api_key:0:20}..."
else
    echo "❌ FAIL: Expected 201, got $api_key_status"
    api_key=""
fi

# Test 9: Use API Key for authentication
if [ -n "$api_key" ]; then
    echo ""
    echo "Test 9: Use API Key for Authentication"
    api_auth_response=$(curl -s -X POST "$BASE_URL/api/v1/urls" \
      -H 'Content-Type: application/json' \
      -H "Authorization: Bearer $api_key" \
      -d '{"original_url": "https://www.stackoverflow.com", "custom_short_code": "stack-test"}' \
      -w "\nHTTP_STATUS:%{http_code}")

    api_auth_status=$(echo "$api_auth_response" | grep "HTTP_STATUS:" | cut -d: -f2)
    echo "Result: HTTP $api_auth_status"

    if [ "$api_auth_status" = "201" ]; then
        echo "✅ PASS: API key authentication successful"
    else
        echo "❌ FAIL: Expected 201, got $api_auth_status"
    fi
fi

# Test 10: Test URL redirection (public endpoint)
if [ -n "$short_code" ]; then
    echo ""
    echo "Test 10: URL Redirection (Public Access)"
    redirect_response=$(curl -s -X GET "$BASE_URL/$short_code" -w "\nHTTP_STATUS:%{http_code}")
    redirect_status=$(echo "$redirect_response" | grep "HTTP_STATUS:" | cut -d: -f2)
    echo "Result: HTTP $redirect_status"

    if [ "$redirect_status" = "302" ] || [ "$redirect_status" = "301" ]; then
        echo "✅ PASS: URL redirection works without authentication"
    else
        echo "❌ FAIL: Expected 301/302, got $redirect_status"
    fi
fi

# Test 11: Access admin endpoint with JWT
echo ""
echo "Test 11: Access Admin Profile Endpoint"
profile_response=$(curl -s -X GET "$BASE_URL/api/v1/auth/profile" \
  -H "Authorization: Bearer $jwt_token" \
  -w "\nHTTP_STATUS:%{http_code}")

profile_status=$(echo "$profile_response" | grep "HTTP_STATUS:" | cut -d: -f2)
echo "Result: HTTP $profile_status"

if [ "$profile_status" = "200" ]; then
    echo "✅ PASS: Admin profile accessible with JWT"
else
    echo "❌ FAIL: Expected 200, got $profile_status"
fi

# Test 12: Invalid JWT token
echo ""
echo "Test 12: Invalid JWT Token"
invalid_jwt_response=$(curl -s -X GET "$BASE_URL/api/v1/urls" \
  -H "Authorization: Bearer invalid-token-12345" \
  -w "\nHTTP_STATUS:%{http_code}")

invalid_jwt_status=$(echo "$invalid_jwt_response" | grep "HTTP_STATUS:" | cut -d: -f2)
echo "Result: HTTP $invalid_jwt_status"

if [ "$invalid_jwt_status" = "401" ]; then
    echo "✅ PASS: Invalid JWT token correctly rejected"
else
    echo "❌ FAIL: Expected 401, got $invalid_jwt_status"
fi

echo ""
echo "======================================"
echo "🎯 Comprehensive Authentication Test Complete"
echo "======================================"
