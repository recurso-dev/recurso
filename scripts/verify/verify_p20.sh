#!/bin/bash
set -e

API_URL="http://localhost:8080/v1"
AUTH_HEADER="Authorization: Bearer recurso_secret"

echo "Verifying Phase 20 (Search, Filters, Pagination)..."

SUFFIX=$RANDOM

# 1. Setup Data
echo "--- Structuring Data ---"
# Create Plan
PLAN_RES=$(curl -s -X POST $API_URL/plans -H "$AUTH_HEADER" -H "Content-Type: application/json" -d "{\"name\": \"Filter Test Plan $SUFFIX\", \"code\": \"filter-plan-$SUFFIX\", \"interval_unit\": \"month\", \"interval_count\": 1, \"amount\": 5000, \"currency\": \"USD\"}")
PLAN_ID=$(echo $PLAN_RES | jq -r .id)
echo "Plan Created: $PLAN_ID"

# Create Customer "Alice Searchable"
ALICE_RES=$(curl -s -X POST $API_URL/customers -H "$AUTH_HEADER" -H "Content-Type: application/json" -d "{\"email\": \"alice-$SUFFIX@search.test\", \"name\": \"Alice Searchable\", \"country\": \"US\"}")
ALICE_ID=$(echo $ALICE_RES | jq -r .id)
echo "Customer Alice Created: $ALICE_ID"

# Create Customer "Bob Ignored"
BOB_RES=$(curl -s -X POST $API_URL/customers -H "$AUTH_HEADER" -H "Content-Type: application/json" -d "{\"email\": \"bob-$SUFFIX@ignore.test\", \"name\": \"Bob Ignored\", \"country\": \"CA\"}")
BOB_ID=$(echo $BOB_RES | jq -r .id)
echo "Customer Bob Created: $BOB_ID"

# Create Subscription for Alice
curl -s -X POST $API_URL/subscriptions -H "$AUTH_HEADER" -H "Content-Type: application/json" -d "{ \"customer_id\": \"$ALICE_ID\", \"plan_id\": \"$PLAN_ID\" }" > /dev/null
echo "Subscription for Alice Created"

echo ""
echo "--- Verifying Search & Filters ---"

# 2. Test Customer Search
echo "Testing Customer Search (q=Alice)..."
SEARCH_ALICE=$(curl -s "$API_URL/customers?q=Alice" -H "$AUTH_HEADER")
COUNT_ALICE=$(echo $SEARCH_ALICE | jq '.data | length')
if [ "$COUNT_ALICE" -ge 1 ]; then echo "✅ Found $COUNT_ALICE Alice(s)"; else echo "❌ Failed to find Alice"; exit 1; fi

echo "Testing Customer Search (q=Bob)..."
SEARCH_BOB=$(curl -s "$API_URL/customers?q=Bob" -H "$AUTH_HEADER")
COUNT_BOB=$(echo $SEARCH_BOB | jq '.data | length')
if [ "$COUNT_BOB" -ge 1 ]; then echo "✅ Found $COUNT_BOB Bob(s)"; else echo "❌ Failed to find Bob"; exit 1; fi

echo "Testing Customer Filter (country=US)..."
SEARCH_US=$(curl -s "$API_URL/customers?country=US" -H "$AUTH_HEADER")
# Should find Alice, not Bob (Bob is CA)
US_DATA=$(echo $SEARCH_US | jq '.data[] | select(.id == "'$ALICE_ID'")')
if [ ! -z "$US_DATA" ]; then echo "✅ Found Alice in US filter"; else echo "❌ Failed to find Alice in US filter"; exit 1; fi

# 3. Test Subscription Search (by Customer Name via Join)
echo "Testing Subscription Search (q=Alice)..."
SUB_SEARCH=$(curl -s "$API_URL/subscriptions?q=Alice" -H "$AUTH_HEADER")
SUB_COUNT=$(echo $SUB_SEARCH | jq '.data | length')
if [ "$SUB_COUNT" -ge 1 ]; then 
  echo "✅ Found $SUB_COUNT subscription(s) for Alice" 
else 
  echo "❌ Failed to find subscription for Alice"
  echo "Response: $SUB_SEARCH"
  exit 1
fi

echo "Testing Subscription Search (q=Bob)..."
SUB_SEARCH_BOB=$(curl -s "$API_URL/subscriptions?q=Bob" -H "$AUTH_HEADER")
SUB_COUNT_BOB=$(echo $SUB_SEARCH_BOB | jq '.data | length')
if [ "$SUB_COUNT_BOB" -eq 0 ]; then echo "✅ Found 0 subscriptions for Bob (Correct)"; else echo "❌ Found unexpected subscriptions for Bob"; exit 1; fi

# 4. Test Pagination
echo "Testing Pagination (Limit=1)..."
PAGE_1=$(curl -s "$API_URL/customers?limit=1" -H "$AUTH_HEADER")
COUNT_PAGE_1=$(echo $PAGE_1 | jq '.data | length')
if [ "$COUNT_PAGE_1" -eq 1 ]; then echo "✅ Pagination Limit=1 returned 1 item"; else echo "❌ Pagination failed, returned $COUNT_PAGE_1 items"; exit 1; fi

echo "✅ All Verification Tests Passed!"
