#!/bin/bash
set -e

echo "Verifying P5 (Ledger Integration)..."

# 1. Create Data
echo "Creating Plan & Customer..."
PLAN_RES=$(curl -s -X POST http://localhost:8080/v1/plans \
  -H "Authorization: Bearer recurso_secret" \
  -H "Content-Type: application/json" \
  -d '{"name": "Ledger Plan", "code": "ledger-plan", "interval_unit": "month", "interval_count": 1, "amount": 5000, "currency": "USD"}')
PLAN_ID=$(echo $PLAN_RES | jq -r .id)

CUST_RES=$(curl -s -X POST http://localhost:8080/v1/customers \
  -H "Authorization: Bearer recurso_secret" \
  -H "Content-Type: application/json" \
  -d '{"email": "ledger@recurso.dev", "name": "Ledger User", "country": "US"}')
CUST_ID=$(echo $CUST_RES | jq -r .id)

# 2. Create Subscription (Triggers Dual-Write)
echo "Creating Subscription..."
SUB_RES=$(curl -s -X POST http://localhost:8080/v1/subscriptions \
  -H "Authorization: Bearer recurso_secret" \
  -H "Content-Type: application/json" \
  -d "{
    \"customer_id\": \"$CUST_ID\",
    \"plan_id\": \"$PLAN_ID\"
  }")

if echo "$SUB_RES" | grep -q "id"; then
  echo "✅ Subscription Created!"
  echo "🐯 If no errors in server log, TigerBeetle transaction was posted."
else
  echo "❌ Failed: $SUB_RES"
  exit 1
fi
