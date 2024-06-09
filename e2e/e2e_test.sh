#!/bin/bash

SERVER_URL="http://localhost:8080"

# 連合登録のテスト
REGISTER_ENDPOINT="$SERVER_URL/api/register"
REGISTER_PAYLOAD='{
  "system_uri": "http://example.com/system"
}'

echo "Testing /api/register endpoint..."
REGISTER_RESPONSE=$(curl -s -X POST -H "Content-Type: application/json" -d "$REGISTER_PAYLOAD" $REGISTER_ENDPOINT)
echo "Response: $REGISTER_RESPONSE"

# キャッシュの確認
echo "Testing /api/register GET endpoint..."
REGISTER_GET_RESPONSE=$(curl -s -X GET $REGISTER_ENDPOINT)
echo "Cache state: $REGISTER_GET_RESPONSE"

# 確信度照会依頼のテスト
INQUIRY_ENDPOINT="$SERVER_URL/api/inquiry"

# Ensure the required CSV files exist for testing
if [ ! -f "wifi_data.csv" ]; then
  echo "Error: wifi_data.csv file not found!"
  exit 1
fi

if [ ! -f "ble_data.csv" ]; then
  echo "Error: ble_data.csv file not found!"
  exit 1
fi

echo "Testing /api/inquiry endpoint..."
INQUIRY_RESPONSE=$(curl -s -X POST -F "wifi_data=@wifi_data.csv" -F "ble_data=@ble_data.csv" $INQUIRY_ENDPOINT)
echo "Response: $INQUIRY_RESPONSE"
