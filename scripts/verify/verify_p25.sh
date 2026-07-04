#!/bin/bash

# Configuration
API_URL="http://localhost:8080/v1"
API_KEY="test_api_key_123" # Will be replaced by tenant creation

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

echo "Starting Phase 25 Verification (E-Invoicing)..."

# 1. Register Tenant (Recurso India)
EMAIL="billify_admin_p25_$(date +%s)@example.com"
echo "Step 1: Create Tenant ($EMAIL)"
REGISTER_RESP=$(curl -s -X POST "http://localhost:8080/auth/register" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"Billify India Pvt Ltd\",
    \"email\": \"$EMAIL\",
    \"password\": \"password123\",
    \"country\": \"IN\" 
  }")

API_KEY=$(echo $REGISTER_RESP | jq -r '.api_key')
TENANT_ID=$(echo $REGISTER_RESP | jq -r '.tenant.id')

if [ "$API_KEY" == "null" ] || [ -z "$API_KEY" ]; then
  echo -e "${RED}❌ Failed to register tenant${NC}"
  echo $REGISTER_RESP
  exit 1
fi
echo -e "${GREEN}✅ Tenant Created (ID: $TENANT_ID)${NC}"

# 2. Create B2B Customer (India, GST Registered)
echo "Step 2: Create B2B Customer (India, GSTIN)"
CUST_EMAIL="b2b_customer_$(date +%s)@example.com"
CUST_RESP=$(curl -s -X POST "$API_URL/customers" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"TechCorp India\",
    \"email\": \"$CUST_EMAIL\",
    \"country\": \"India\",
    \"state\": \"KA\",
    \"tax_id\": \"29ABCDE1234F1Z5\",
    \"gstin\": \"29ABCDE1234F1Z5\",
    \"tax_type\": \"business\",
    \"place_of_supply\": \"KA\"
  }")

CUST_ID=$(echo $CUST_RESP | jq -r '.id')
if [ "$CUST_ID" == "null" ]; then
  echo -e "${RED}❌ Failed to create customer${NC}"
  echo $CUST_RESP
  exit 1
fi
echo -e "${GREEN}✅ B2B Customer Created (ID: $CUST_ID)${NC}"

# 3. Create Plan
echo "Step 3: Create Plan"
PLAN_RESP=$(curl -s -X POST "$API_URL/plans" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Enterprise Plan (GST)",
    "code": "ENT-GST",
    "description": "Plan with GST",
    "interval_unit": "month",
    "interval_count": 1,
    "prices": [
      {
        "amount": 100000, 
        "currency": "INR"
      }
    ]
  }')
PLAN_ID=$(echo $PLAN_RESP | jq -r '.id')
echo -e "${GREEN}✅ Plan Created (ID: $PLAN_ID)${NC}"

# 4. Create Subscription (Triggers Invoice)
echo "Step 4: Create Subscription for B2B Customer"
SUB_RESP=$(curl -s -X POST "$API_URL/subscriptions" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d "{
    \"customer_id\": \"$CUST_ID\",
    \"plan_id\": \"$PLAN_ID\"
  }")
SUB_ID=$(echo $SUB_RESP | jq -r '.id')
if [ "$SUB_ID" == "null" ]; then
  echo -e "${RED}❌ Failed to create subscription${NC}"
  echo $SUB_RESP
  exit 1
fi
echo -e "${GREEN}✅ Subscription Created (ID: $SUB_ID)${NC}"

# 5. Fetch Invoice and Verify E-Invoicing
echo "Step 5: Fetch Invoice and Verify IRN/QR"
# List invoices for sub
INVOICES_RESP=$(curl -s -X GET "$API_URL/invoices?subscription_id=$SUB_ID" \
  -H "Authorization: Bearer $API_KEY")

# Get first invoice ID
INV_ID=$(echo $INVOICES_RESP | jq -r '.data[0].id')
echo "Invoice ID: $INV_ID"

# Fetch full details
INV_RESP=$(curl -s -X GET "$API_URL/invoices" -H "Authorization: Bearer $API_KEY") # This lists all, but let's assume filtering or just check the list response
# The list endpoint returns array.
IRN=$(echo $INVOICES_RESP | jq -r '.data[0].irn')
QR=$(echo $INVOICES_RESP | jq -r '.data[0].signed_qr_code')
STATUS=$(echo $INVOICES_RESP | jq -r '.data[0].e_invoice_status')

echo "IRN: $IRN"
echo "Status: $STATUS"
echo "QR Length: ${#QR}"

if [ "$STATUS" == "GENERATED" ] && [ -n "$IRN" ] && [ "$IRN" != "null" ]; then
  echo -e "${GREEN}✅ E-Invoice Generated Successfully (IRN Present)${NC}"
else
  echo -e "${RED}❌ E-Invoice Generation Failed${NC}"
  exit 1
fi
