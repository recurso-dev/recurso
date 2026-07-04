#!/bin/bash
set -e

echo "Verifying Authentication..."

# 1. Unauthenticated Request (Expect 401)
echo "1. Testing Unauthorized Request..."
STATUS_CODE=$(curl -o /dev/null -s -w "%{http_code}" -X POST http://localhost:8080/v1/plans)
if [ "$STATUS_CODE" -ne 401 ]; then
  echo "Error: Expected 401, got $STATUS_CODE"
  exit 1
fi
echo "Success: Blocked (401)"

# 2. Authorized Request (Expect 201 or 400 - just not 401)
# Using default secret "recurso_secret"
echo "2. Testing Authorized Request..."
STATUS_CODE=$(curl -o /dev/null -s -w "%{http_code}" -X POST http://localhost:8080/v1/plans \
  -H "Authorization: Bearer recurso_secret" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Auth Test Plan",
    "code": "auth-plan",
    "interval_unit": "month",
    "interval_count": 1,
    "amount": 100,
    "currency": "USD"
  }')

if [ "$STATUS_CODE" -eq 401 ]; then
  echo "Error: Still 401 despite token"
  exit 1
fi
echo "Success: Allowed ($STATUS_CODE)"
