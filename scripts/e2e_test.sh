#!/bin/bash
set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

API_URL="http://localhost:8080"

echo "Starting End-to-End Verification..."

# Helper function for JSON extraction using jq (more robust)
get_json_value() {
  echo "$1" | jq -r ".$2"
}

# 1. Setup Tenant
TIMESTAMP=$(date +%s)
EMAIL="admin_e2e_${TIMESTAMP}@example.com"
echo "Step 1: Creating Tenant ($EMAIL)..."
TENANT_RES=$(curl -s -X POST "$API_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "E2E Tech Inc",
    "email": "'"$EMAIL"'",
    "password": "password123",
    "org_name": "E2E Corp"
  }')

if echo "$TENANT_RES" | grep -q "api_key"; then
  API_KEY=$(get_json_value "$TENANT_RES" "api_key")
  TENANT_ID=$(get_json_value "$TENANT_RES" "tenant.id")
  echo -e "${GREEN}✅ Tenant Created (ID: $TENANT_ID)${NC}"
else
  echo -e "${RED}❌ Failed to create tenant${NC}"
  echo "Response: $TENANT_RES"
  exit 1
fi

# 2. Test INR Flow (Razorpay + GST)
echo "Step 2: Testing INR/Razorpay Flow (India Stack)..."

# Create INR Customer
CUST_INR_RES=$(curl -s -X POST "$API_URL/v1/customers" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $API_KEY" \
  -d '{
    "name": "Ramesh Gupta",
    "email": "ramesh.gupta@example.com",
    "billing_address": {
      "country": "IN",
      "state": "MH",
      "postal_code": "400001"
    },
    "tax_id": "27DQBPS8356J1Z1"
  }')
CUST_ID_INR=$(get_json_value "$CUST_INR_RES" "id")
echo "  > Customer Created: $CUST_ID_INR"

# Create INR Plan
PLAN_INR_RES=$(curl -s -X POST "$API_URL/v1/plans" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $API_KEY" \
  -d '{
    "code": "BASIC-INR",
    "name": "Basic Plan INR",
    "interval_unit": "month",
    "interval_count": 1,
    "amount": 50000,
    "currency": "INR"
  }')
PLAN_ID_INR=$(get_json_value "$PLAN_INR_RES" "id")
echo "  > Plan Created: $PLAN_ID_INR"

# Create INR Subscription
SUB_INR_RES=$(curl -s -X POST "$API_URL/v1/subscriptions" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $API_KEY" \
  -d '{
    "customer_id": "'"$CUST_ID_INR"'",
    "plan_id": "'"$PLAN_ID_INR"'",
    "start_date": "2025-01-01T00:00:00Z"
  }')
SUB_ID_INR=$(get_json_value "$SUB_INR_RES" "id")
RAZORPAY_ID=$(get_json_value "$SUB_INR_RES" "razorpay_subscription_id")

if [[ "$RAZORPAY_ID" == sub_* ]]; then
    echo -e "${GREEN}  ✅ INR Subscription routing verified (Razorpay ID: $RAZORPAY_ID)${NC}"
else
    echo -e "${RED}  ❌ INR Subscription failed to populate Razorpay ID${NC}"
    echo "Response: $SUB_INR_RES"
    exit 1
fi


# 3. Test USD Flow (Stripe + Global)
echo "Step 3: Testing USD/Stripe Flow (Global)..."

# Create USD Customer
CUST_USD_RES=$(curl -s -X POST "$API_URL/v1/customers" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $API_KEY" \
  -d '{
    "name": "Jane User",
    "email": "jane@example.com",
    "billing_address": {
      "country": "US",
      "state": "CA",
      "postal_code": "90210"
    }
  }')
CUST_ID_USD=$(get_json_value "$CUST_USD_RES" "id")
echo "  > Customer Created: $CUST_ID_USD"

# Create USD Plan
PLAN_USD_RES=$(curl -s -X POST "$API_URL/v1/plans" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $API_KEY" \
  -d '{
    "code": "PRO-USD",
    "name": "Pro Plan USD",
    "interval_unit": "month",
    "interval_count": 1,
    "amount": 2900,
    "currency": "USD"
  }')
PLAN_ID_USD=$(get_json_value "$PLAN_USD_RES" "id")
echo "  > Plan Created: $PLAN_ID_USD"

# Create USD Subscription
SUB_USD_RES=$(curl -s -X POST "$API_URL/v1/subscriptions" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $API_KEY" \
  -d '{
    "customer_id": "'"$CUST_ID_USD"'",
    "plan_id": "'"$PLAN_ID_USD"'",
    "start_date": "2025-01-01T00:00:00Z"
  }')
SUB_ID_USD=$(get_json_value "$SUB_USD_RES" "id")
STRIPE_ID=$(get_json_value "$SUB_USD_RES" "stripe_subscription_id")

if [[ "$STRIPE_ID" == sub_* ]]; then
    echo -e "${GREEN}  ✅ USD Subscription routing verified (Stripe ID: $STRIPE_ID)${NC}"
else
    echo -e "${RED}  ❌ USD Subscription failed to populate Stripe ID${NC}"
    echo "Response: $SUB_USD_RES"
    exit 1
fi

echo -e "${GREEN}✅ All Scenario Checks Passed!${NC}"
