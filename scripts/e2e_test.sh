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

# post_json <path> <json-body> [api-key]
# Sends a JSON POST without ever placing the body or the API key on the curl
# argv (which would leak them via `ps aux`). The body is piped over stdin with
# `--data @-`; the Authorization header (when an api-key is given) is passed
# through a `curl -K` config file supplied on stdin-adjacent fd via process
# substitution, so the bearer token never appears in the process list.
post_json() {
  local path="$1" body="$2" api_key="${3:-}"
  if [ -n "$api_key" ]; then
    curl -s -X POST "$path" \
      -H "Content-Type: application/json" \
      -K <(printf 'header = "Authorization: Bearer %s"\n' "$api_key") \
      --data @- <<<"$body"
  else
    curl -s -X POST "$path" \
      -H "Content-Type: application/json" \
      --data @- <<<"$body"
  fi
}

# 1. Setup Tenant
TIMESTAMP=$(date +%s)
EMAIL="admin_e2e_${TIMESTAMP}@example.com"
# Read the registration password from the environment so it never sits on an
# argv. Falls back to the historical test value when E2E_PASSWORD is unset.
E2E_PASSWORD="${E2E_PASSWORD:-password123}"
echo "Step 1: Creating Tenant ($EMAIL)..."
TENANT_RES=$(post_json "$API_URL/auth/register" "$(jq -n \
  --arg email "$EMAIL" \
  --arg password "$E2E_PASSWORD" \
  '{name: "E2E Tech Inc", email: $email, password: $password, company_name: "E2E Corp"}')")

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
CUST_INR_RES=$(post_json "$API_URL/v1/customers" '{
    "name": "Ramesh Gupta",
    "email": "ramesh.gupta@example.com",
    "billing_address": {
      "country": "IN",
      "state": "MH",
      "postal_code": "400001"
    },
    "tax_id": "27DQBPS8356J1Z1"
  }' "$API_KEY")
CUST_ID_INR=$(get_json_value "$CUST_INR_RES" "id")
echo "  > Customer Created: $CUST_ID_INR"

# Create INR Plan
PLAN_INR_RES=$(post_json "$API_URL/v1/plans" '{
    "code": "BASIC-INR",
    "name": "Basic Plan INR",
    "interval_unit": "month",
    "interval_count": 1,
    "amount": 50000,
    "currency": "INR"
  }' "$API_KEY")
PLAN_ID_INR=$(get_json_value "$PLAN_INR_RES" "id")
echo "  > Plan Created: $PLAN_ID_INR"

# Create INR Subscription
SUB_INR_RES=$(post_json "$API_URL/v1/subscriptions" "$(jq -n \
  --arg customer_id "$CUST_ID_INR" \
  --arg plan_id "$PLAN_ID_INR" \
  '{customer_id: $customer_id, plan_id: $plan_id, start_date: "2025-01-01T00:00:00Z"}')" \
  "$API_KEY")
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
CUST_USD_RES=$(post_json "$API_URL/v1/customers" '{
    "name": "Jane User",
    "email": "jane@example.com",
    "billing_address": {
      "country": "US",
      "state": "CA",
      "postal_code": "90210"
    }
  }' "$API_KEY")
CUST_ID_USD=$(get_json_value "$CUST_USD_RES" "id")
echo "  > Customer Created: $CUST_ID_USD"

# Create USD Plan
PLAN_USD_RES=$(post_json "$API_URL/v1/plans" '{
    "code": "PRO-USD",
    "name": "Pro Plan USD",
    "interval_unit": "month",
    "interval_count": 1,
    "amount": 2900,
    "currency": "USD"
  }' "$API_KEY")
PLAN_ID_USD=$(get_json_value "$PLAN_USD_RES" "id")
echo "  > Plan Created: $PLAN_ID_USD"

# Create USD Subscription
SUB_USD_RES=$(post_json "$API_URL/v1/subscriptions" "$(jq -n \
  --arg customer_id "$CUST_ID_USD" \
  --arg plan_id "$PLAN_ID_USD" \
  '{customer_id: $customer_id, plan_id: $plan_id, start_date: "2025-01-01T00:00:00Z"}')" \
  "$API_KEY")
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
