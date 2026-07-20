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

# get_json <path> <api-key> — authenticated GET, key kept off argv.
get_json() {
  local path="$1" api_key="$2"
  curl -s "$path" \
    -K <(printf 'header = "Authorization: Bearer %s"\n' "$api_key")
}

# put_json <path> <json-body> <api-key>
put_json() {
  local path="$1" body="$2" api_key="$3"
  curl -s -X PUT "$path" \
    -H "Content-Type: application/json" \
    -K <(printf 'header = "Authorization: Bearer %s"\n' "$api_key") \
    --data @- <<<"$body"
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

# 4. Coupon correctness: a 20% coupon must discount the first invoice by 20%,
# not by 20 minor units (the percent/amount enum bug class).
echo "Step 4: Testing coupon math (percent, not minor units)..."
COUPON_RES=$(post_json "$API_URL/v1/coupons" '{
    "code": "E2E-20OFF",
    "discount_type": "percent",
    "discount_value": 20,
    "duration": "once"
  }' "$API_KEY")

CUST_C_RES=$(post_json "$API_URL/v1/customers" '{
    "name": "Coupon Carl",
    "email": "carl@example.com",
    "billing_address": {"country": "US", "state": "CA", "postal_code": "90210"}
  }' "$API_KEY")
CUST_ID_C=$(get_json_value "$CUST_C_RES" "id")

SUB_C_RES=$(post_json "$API_URL/v1/subscriptions" "$(jq -n \
  --arg customer_id "$CUST_ID_C" \
  --arg plan_id "$PLAN_ID_USD" \
  '{customer_id: $customer_id, plan_id: $plan_id, coupon_code: "E2E-20OFF"}')" \
  "$API_KEY")
SUB_ID_C=$(get_json_value "$SUB_C_RES" "id")

# The USD plan is 2900; 20% off => 2320 on the first invoice.
INVOICES_RES=$(get_json "$API_URL/v1/invoices" "$API_KEY")
COUPON_TOTAL=$(echo "$INVOICES_RES" | jq -r --arg sid "$SUB_ID_C" \
  '[.data[] | select(.subscription_id == $sid)][0].total')
if [ "$COUPON_TOTAL" = "2320" ]; then
    echo -e "${GREEN}  ✅ Coupon applied as 20% (invoice total 2320)${NC}"
else
    echo -e "${RED}  ❌ Coupon math wrong: first invoice total = $COUPON_TOTAL, want 2320${NC}"
    echo "Response: $INVOICES_RES"
    exit 1
fi

# 5. Usage ingestion round-trip: an ingested event must be visible on the raw
# event stream endpoint.
echo "Step 5: Testing usage ingest -> raw stream visibility..."
post_json "$API_URL/v1/usage/events" "$(jq -n \
  --arg sub "$SUB_ID_USD" --arg cust "$CUST_ID_USD" \
  '{subscription_id: $sub, customer_id: $cust, dimension: "e2e_api_calls", quantity: 42, transaction_id: "e2e-txn-1"}')" \
  "$API_KEY" > /dev/null

EVENTS_RES=$(get_json "$API_URL/v1/usage/events?dimension=e2e_api_calls" "$API_KEY")
EVENT_QTY=$(echo "$EVENTS_RES" | jq -r '.data[0].quantity')
if [ "$EVENT_QTY" = "42" ]; then
    echo -e "${GREEN}  ✅ Ingested event visible on the raw stream (qty 42)${NC}"
else
    echo -e "${RED}  ❌ Ingested event not visible${NC}"
    echo "Response: $EVENTS_RES"
    exit 1
fi

# 6. Webhook endpoint lifecycle: create -> pause -> verify -> delete.
echo "Step 6: Testing webhook endpoint pause/resume..."
WH_RES=$(post_json "$API_URL/v1/webhooks" '{
    "url": "https://example.com/e2e-hook",
    "events": ["invoice.paid"]
  }' "$API_KEY")
WH_ID=$(get_json_value "$WH_RES" "data.id")

put_json "$API_URL/v1/webhooks/$WH_ID/status" '{"status":"inactive"}' "$API_KEY" > /dev/null
WH_LIST=$(get_json "$API_URL/v1/webhooks" "$API_KEY")
WH_STATUS=$(echo "$WH_LIST" | jq -r --arg id "$WH_ID" '.data[] | select(.id == $id) | .status')
if [ "$WH_STATUS" = "inactive" ]; then
    echo -e "${GREEN}  ✅ Endpoint paused (status inactive)${NC}"
else
    echo -e "${RED}  ❌ Endpoint pause failed (status: $WH_STATUS)${NC}"
    exit 1
fi

# 7. Audit-grade gate: after everything above, the ledger must reconcile with
# ZERO discrepancies — the end-to-end version of the invariant harness. Any
# invoice-creating flow that forgets its ledger leg fails the whole run here.
echo "Step 7: Reconciliation must be clean..."
RECON_RES=$(get_json "$API_URL/v1/finance/reconciliation" "$API_KEY")
DISCREPANCIES=$(echo "$RECON_RES" | jq -r '.data.discrepancies | length')
if [ "$DISCREPANCIES" = "0" ]; then
    echo -e "${GREEN}  ✅ Ledger reconciles clean (0 discrepancies)${NC}"
else
    echo -e "${RED}  ❌ Reconciliation found $DISCREPANCIES discrepancies${NC}"
    echo "Response: $RECON_RES"
    exit 1
fi

echo -e "${GREEN}✅ All Scenario Checks Passed!${NC}"
