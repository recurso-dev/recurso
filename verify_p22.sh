#!/bin/bash
set -e

echo "Testing Ledger..."

# 1. Get API Key (Assuming one exists or default dev secret)
API_KEY="recurso_secret" # Replace if dynamic

# 2. Query Ledger Entries (Expecting empty or error if no account_id, so passing a dummy uuid)
# Using a random UUID
RANDOM_UUID="00000000-0000-0000-0000-000000000001"

echo "Querying ledger for account $RANDOM_UUID..."
RESPONSE=$(curl -s -H "Authorization: Bearer $API_KEY" "http://localhost:8080/v1/ledger/entries?account_id=$RANDOM_UUID")

echo "Response: $RESPONSE"

if echo "$RESPONSE" | grep -q "data"; then
    echo "✅ Ledger API returned data structure"
else
    echo "❌ Ledger API failed or returned unexpected format"
    exit 1
fi
