#!/bin/bash

# Configuration
API_URL="http://localhost:8080/v1"
ADMIN_KEY="admin-secret-key"

echo "Step 1: Create Tenant"
TENANT_RESP=$(curl -s -X POST "$API_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"Billify India Pvt Ltd\",
    \"email\": \"admin-$(date +%s)@billify.in\",
    \"password\": \"password123\"
  }")
echo "Tenant Response: $TENANT_RESP"
TOKEN=$(echo $TENANT_RESP | jq -r '.token')

echo "Step 2: Create Inter-State Customer (KA)"
CUST_KA_RESP=$(curl -s -X POST "$API_URL/customers" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Tech Startup Bangalore",
    "email": "ceo@bangalore.com",
    "place_of_supply": "KA",
    "gstin": "29ABCD1234E1Z5"
  }')
CUST_KA_ID=$(echo $CUST_KA_RESP | jq -r '.id')
echo "Customer KA ID: $CUST_KA_ID"

echo "Step 3: Create Plan (Standard)"
PLAN_RESP=$(curl -s -X POST "$API_URL/plans" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Pro Plan",
    "code": "pro-monthly",
    "interval_unit": "month",
    "interval_count": 1,
    "prices": [{"amount": 100000, "currency": "INR"}]
  }')
PLAN_ID=$(echo $PLAN_RESP | jq -r '.id')
echo "Plan ID: $PLAN_ID"

echo "Step 4: Create Subscription for Inter-State Customer"
SUB_RESP=$(curl -s -X POST "$API_URL/subscriptions" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    "customer_id": \"$CUST_KA_ID\",
    "plan_id": \"$PLAN_ID\"
  }")
SUB_ID=$(echo $SUB_RESP | jq -r '.id')
RZP_SUB_ID=$(echo $SUB_RESP | jq -r '.razorpay_subscription_id')
echo "Subscription ID: $SUB_ID"
echo "Razorpay Subscription ID: $RZP_SUB_ID"

if [[ "$RZP_SUB_ID" == sub_mock_* ]]; then
  echo "✅ Razorpay Subscription Created (Mock)"
else
  echo "❌ Razorpay Subscription ID Missing or Invalid"
fi

# Fetch Invoice
echo "Step 5: Fetch Invoice and Verify IGST"
# Currently Subscription creates an invoice automatically. Let's find it.
INVOICES_RESP=$(curl -s -X GET "$API_URL/invoices?customer_id=$CUST_KA_ID" \
  -H "Authorization: Bearer $TOKEN")
INVOICE=$(echo $INVOICES_RESP | jq '.data[0]')

echo "Invoice Data: $INVOICE"
IGST=$(echo $INVOICE | jq -r '.igst_amount')
TOTAL=$(echo $INVOICE | jq -r '.total')

echo "IGST Amount: $IGST (Expected 18000)"
echo "Total Amount: $TOTAL (Expected 118000)"

if [ "$IGST" == "18000" ]; then
  echo "✅ IGST Calculation Correct for Inter-State"
else
  echo "❌ IGST Calculation Failed"
fi

echo "Step 6: Create Intra-State Customer (TN)"
CUST_TN_RESP=$(curl -s -X POST "$API_URL/customers" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Local Business Chennai",
    "email": "shop@chennai.com",
    "place_of_supply": "TN",
    "gstin": "33ABCD1234E1Z5"
  }')
CUST_TN_ID=$(echo $CUST_TN_RESP | jq -r '.id')

echo "Step 7: Create Subscription for Intra-State Customer"
SUB_TN_RESP=$(curl -s -X POST "$API_URL/subscriptions" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    "customer_id": \"$CUST_TN_ID\",
    "plan_id": \"$PLAN_ID\"
  }")

echo "Step 8: Fetch Invoice and Verify CGST/SGST"
INVOICES_TN_RESP=$(curl -s -X GET "$API_URL/invoices?customer_id=$CUST_TN_ID" \
  -H "Authorization: Bearer $TOKEN")
INVOICE_TN=$(echo $INVOICES_TN_RESP | jq '.data[0]')

CGST=$(echo $INVOICE_TN | jq -r '.cgst_amount')
SGST=$(echo $INVOICE_TN | jq -r '.sgst_amount')
IGST_TN=$(echo $INVOICE_TN | jq -r '.igst_amount')

echo "CGST: $CGST (Expected 9000)"
echo "SGST: $SGST (Expected 9000)"
echo "IGST: $IGST_TN (Expected 0)"

if [ "$CGST" == "9000" ] && [ "$SGST" == "9000" ]; then
  echo "✅ CGST/SGST Calculation Correct for Intra-State"
else
  echo "❌ CGST/SGST Calculation Failed"
fi
