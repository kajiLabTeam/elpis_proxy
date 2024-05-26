#!/bin/bash

SERVER_URL="http://localhost:8080"

# 連合登録のテスト
REGISTER_ENDPOINT="$SERVER_URL/api/register"
REGISTER_PAYLOAD='{
  "organization_id": "org123",
  "system_uri": "http://example.com/system"
}'

echo "Testing /api/register endpoint..."
REGISTER_RESPONSE=$(curl -s -X POST -H "Content-Type: application/json" -d "$REGISTER_PAYLOAD" $REGISTER_ENDPOINT)
echo "Response: $REGISTER_RESPONSE"

# 確信度照会依頼のテスト
INQUIRY_ENDPOINT="$SERVER_URL/api/inquiry"

echo "Testing /api/inquiry endpoint..."
INQUIRY_RESPONSE=$(curl -s -X POST -F "wifi_data=@wifi_data.csv" -F "ble_data=@ble_data.csv" $INQUIRY_ENDPOINT)
echo "Response: $INQUIRY_RESPONSE"
