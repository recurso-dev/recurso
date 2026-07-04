#!/bin/bash
set -e

echo "Testing Credit Notes..."

# 1. Get API Key (Assuming one exists or default dev secret)
API_KEY="recurso_secret" # Replace if dynamic

# 2. Get a Customer ID (List customers)
echo "Fetching customer list..."
CUSTOMERS_RESPONSE=$(curl -s -H "Authorization: Bearer $API_KEY" "http://localhost:8080/v1/customers")
# Extract first customer ID using regex or python/jq if available. 
# For simplicity, we assume we might need to create one if none exist, or just grab the first UUID found.
CUSTOMER_ID=$(echo "$CUSTOMERS_RESPONSE" | grep -oE '[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}' | head -n 1)

if [ -z "$CUSTOMER_ID" ]; then
    echo "No customer found. Creating one..."
    # Create customer
    CREATE_RES=$(curl -s -X POST -H "Authorization: Bearer $API_KEY" -H "Content-Type: application/json" -d '{"name": "Test Customer", "email": "test@example.com", "plan_id": "00000000-0000-0000-0000-000000000000"}' "http://localhost:8080/v1/customers")
    CUSTOMER_ID=$(echo "$CREATE_RES" | grep -oE '[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}' | head -n 1)
fi

echo "Using Customer ID: $CUSTOMER_ID"

# 3. Create Credit Note
echo "Creating Credit Note..."
CREATE_CN_RESPONSE=$(curl -s -X POST -H "Authorization: Bearer $API_KEY" -H "Content-Type: application/json" -d "{
    \"customer_id\": \"$CUSTOMER_ID\",
    \"amount\": 5000,
    \"currency\": \"USD\",
    \"reason\": \"Refund for downtime\"
}" "http://localhost:8080/v1/credit-notes")

echo "Create Response: $CREATE_CN_RESPONSE"

if echo "$CREATE_CN_RESPONSE" | grep -q "data"; then
    echo "✅ Credit Note Created"
else
    echo "❌ Failed to create Credit Note"
    exit 1
fi

# 4. List Credit Notes
echo "Listing Credit Notes..."
LIST_RESPONSE=$(curl -s -H "Authorization: Bearer $API_KEY" "http://localhost:8080/v1/credit-notes")

if echo "$LIST_RESPONSE" | grep -q "Refund for downtime"; then
     echo "✅ Credit Note found in list"
else
     echo "❌ Credit Note NOT found in list"
     exit 1
fi

echo "Credit Note Verification Complete!"
