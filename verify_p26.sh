#!/bin/bash

# Configuration
API_URL="http://localhost:8080/v1"
API_KEY="test_api_key_123" 

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

echo "Starting Phase 26 Verification (Dual Currency & Smart Routing)..."

# 1. Register Tenant (Global Corp)
EMAIL="global_admin_p26_$(date +%s)@example.com"
echo "Step 1: Create Tenant ($EMAIL)"
REGISTER_RESP=$(curl -s -X POST "http://localhost:8080/auth/register" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"Global Corp\",
    \"email\": \"$EMAIL\",
    \"password\": \"password123\",
    \"country\": \"US\" 
  }")

API_KEY=$(echo $REGISTER_RESP | jq -r '.api_key')
TENANT_ID=$(echo $REGISTER_RESP | jq -r '.tenant.id')

if [ "$API_KEY" == "null" ] || [ -z "$API_KEY" ]; then
  echo -e "${RED}❌ Failed to register tenant${NC}"
  echo $REGISTER_RESP
  exit 1
fi
echo -e "${GREEN}✅ Tenant Created (ID: $TENANT_ID)${NC}"

# 2. Create Customer (US)
echo "Step 2: Create Customer (US)"
CUST_EMAIL="us_customer_$(date +%s)@example.com"
CUST_RESP=$(curl -s -X POST "$API_URL/customers" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"John Doe US\",
    \"email\": \"$CUST_EMAIL\",
    \"country\": \"US\"
  }")

CUST_ID=$(echo $CUST_RESP | jq -r '.id')
if [ "$CUST_ID" == "null" ]; then
  echo -e "${RED}❌ Failed to create customer${NC}"
  echo $CUST_RESP
  exit 1
fi
echo -e "${GREEN}✅ US Customer Created (ID: $CUST_ID)${NC}"

# 3. Create USD Plan
echo "Step 3: Create USD Plan"
PLAN_RESP=$(curl -s -X POST "$API_URL/plans" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Global SaaS Plan",
    "code": "GLOBAL-USD",
    "description": "USD Plan",
    "interval_unit": "month",
    "interval_count": 1,
    "amount": 2900,
    "currency": "USD"
  }')
PLAN_ID=$(echo $PLAN_RESP | jq -r '.id')
echo -e "${GREEN}✅ USD Plan Created (ID: $PLAN_ID)${NC}"

# 4. Create Subscription (Should route to Stripe)
echo "Step 4: Create Subscription (Expected: Stripe Routing)"
SUB_RESP=$(curl -s -X POST "$API_URL/subscriptions" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d "{
    \"customer_id\": \"$CUST_ID\",
    \"plan_id\": \"$PLAN_ID\"
  }")
SUB_ID=$(echo $SUB_RESP | jq -r '.id')
STRIPE_SUB_ID=$(echo $SUB_RESP | jq -r '.stripe_subscription_id')
RAZORPAY_SUB_ID=$(echo $SUB_RESP | jq -r '.razorpay_subscription_id')

if [ "$SUB_ID" == "null" ]; then
  echo -e "${RED}❌ Failed to create subscription${NC}"
  echo $SUB_RESP
  exit 1
fi

echo "Created Subscription: $SUB_ID"
echo "Stripe Sub ID: $STRIPE_SUB_ID"
echo "Razorpay Sub ID: $RAZORPAY_SUB_ID"

if [[ "$STRIPE_SUB_ID" == sub_* ]]; then
    echo -e "${GREEN}✅ Validated Stripe ID format (Starts with sub_)${NC}"
else
    echo -e "${RED}❌ Invalid Stripe ID format or missing${NC}"
    exit 1
fi

if [ -z "$RAZORPAY_SUB_ID" ] || [ "$RAZORPAY_SUB_ID" == "null" ]; then
    echo -e "${GREEN}✅ Razorpay ID is empty (Correct Routing)${NC}"
else
    echo -e "${RED}❌ Razorpay ID should be empty for USD${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Phase 26 Verification Passed!${NC}"
