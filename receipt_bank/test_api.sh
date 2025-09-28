#!/bin/bash

# Test script for Receipt Bank API
# Usage: ./test_api.sh (assumes server is running on port 4403)

BASE_URL="http://localhost:4403"

echo "Testing Receipt Bank API..."

# Test health endpoint
echo "1. Testing health endpoint..."
curl -s -X GET "$BASE_URL/health" | jq '.' || echo "Health check failed or jq not available"
echo

# Test submit receipt
echo "2. Testing submit receipt..."
SUBMIT_RESPONSE=$(curl -s -X POST "$BASE_URL/submit" \
  -H "Content-Type: application/json" \
  -d '{
    "ephemeral_key": "AwHr8L0AKZqGWxUqR8Ao4qoO+0LzW+5OXQ==",
    "encrypted_data": "dGVzdF9lbmNyeXB0ZWRfZGF0YQ==",
    "receipt_id": "test-receipt-123",
    "webhook_url": "http://localhost:8080/webhook"
  }')

echo "Submit response: $SUBMIT_RESPONSE"
echo

# Test collect receipt  
echo "3. Testing collect receipt..."
COLLECT_RESPONSE=$(curl -s -X GET "$BASE_URL/collect/AwHr8L0AKZqGWxUqR8Ao4qoO+0LzW+5OXQ==")
echo "Collect response: $COLLECT_RESPONSE"
echo

# Test collect again (should fail - already collected)
echo "4. Testing collect again (should fail)..."
COLLECT_RESPONSE2=$(curl -s -X GET "$BASE_URL/collect/AwHr8L0AKZqGWxUqR8Ao4qoO+0LzW+5OXQ==")
echo "Second collect response: $COLLECT_RESPONSE2"
echo

echo "API testing complete."
