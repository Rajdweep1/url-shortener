#!/bin/bash

# Quick Performance & Rate Limiting Test
set -e

BASE_URL="http://localhost:8081"
API_URL="$BASE_URL/api/v1"

echo "🚀 Quick Performance & Rate Limiting Test"
echo "=========================================="

# Test 1: Redirect Performance
echo ""
echo "📋 1. Testing Redirect Performance (<100ms requirement)"
echo "Setting up test URL..."
./scripts/cleanup-db.sh > /dev/null 2>&1

curl -s -X POST "$API_URL/urls" \
    -H "Content-Type: application/json" \
    -d '{"original_url": "https://www.google.com", "custom_alias": "quicktest"}' > /dev/null

echo "Testing redirect speed..."
for i in {1..5}; do
    response=$(curl -s -w "TIME:%{time_total}" -o /dev/null "$BASE_URL/quicktest")
    time_seconds=$(echo "$response" | cut -d':' -f2)
    time_ms=$(echo "$time_seconds * 1000" | awk '{printf "%.0f", $1}')
    
    if [ "$time_ms" -lt 100 ]; then
        echo "✅ Redirect $i: ${time_ms}ms (< 100ms)"
    else
        echo "❌ Redirect $i: ${time_ms}ms (>= 100ms)"
    fi
done

# Test 2: URL Creation Rate
echo ""
echo "📋 2. Testing URL Creation Rate (10,000/day requirement)"
echo "Creating URLs for 10 seconds..."

start_time=$(date +%s)
end_time=$((start_time + 10))
created_count=0

while [ $(date +%s) -lt $end_time ]; do
    url_id="rate_test_${created_count}"
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X POST "$API_URL/urls" \
        -H "Content-Type: application/json" \
        -d "{\"original_url\": \"https://example.com/test-$url_id\"}")
    
    status=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
    if [ "$status" = "201" ]; then
        ((created_count++))
    fi
    
    sleep 0.1
done

actual_duration=$(($(date +%s) - start_time))
urls_per_minute=$((created_count * 60 / actual_duration))
projected_daily=$((urls_per_minute * 24 * 60))

echo "Created $created_count URLs in ${actual_duration}s"
echo "Rate: ~$urls_per_minute URLs/minute"
echo "Projected daily: ~$projected_daily URLs/day"

if [ "$projected_daily" -ge 10000 ]; then
    echo "✅ Daily capacity: $projected_daily URLs/day (>= 10,000)"
else
    echo "❌ Daily capacity: $projected_daily URLs/day (< 10,000)"
fi

# Test 3: Basic Rate Limiting
echo ""
echo "📋 3. Testing Basic Rate Limiting (100 requests/minute)"
echo "Testing normal rate (5 requests over 5 seconds)..."

allowed=0
for i in {1..5}; do
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" "$API_URL/health")
    status=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
    
    if [ "$status" = "200" ]; then
        ((allowed++))
    fi
    sleep 1
done

echo "Normal rate: $allowed/5 requests allowed"
if [ "$allowed" -eq 5 ]; then
    echo "✅ Normal rate limiting: Working"
else
    echo "❌ Normal rate limiting: Issues detected"
fi

echo "Testing burst rate (50 rapid requests)..."
allowed=0
blocked=0

for i in {1..50}; do
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" "$API_URL/health" 2>/dev/null)
    status=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
    
    if [ "$status" = "200" ]; then
        ((allowed++))
    elif [ "$status" = "429" ]; then
        ((blocked++))
    fi
    
    sleep 0.02
done

echo "Burst rate: $allowed allowed, $blocked blocked"
if [ "$blocked" -gt 0 ]; then
    echo "✅ Burst rate limiting: Working (blocked $blocked excess requests)"
else
    echo "⚠️  Burst rate limiting: May not be working (no requests blocked)"
fi

echo ""
echo "=========================================="
echo "🎉 Quick performance test completed!"
