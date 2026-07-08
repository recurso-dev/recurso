#!/usr/bin/env bash
#
# cloud-billing-setup.sh — provision the "Recurso Cloud" billing catalog on a
# Recurso instance, so Recurso Cloud can bill its own customers with Recurso
# (dogfooding). Idempotent: skips catalog items that already exist by code.
#
# Usage:
#   API_URL=http://localhost:8080/v1 API_KEY=sk_test_12345 ./scripts/cloud-billing-setup.sh
#
# The base fee and overage rates come from the public pricing page
# (https://recurso.dev/pricing): $299/mo including 10k invoices + 5k active
# subscriptions, then $0.02/invoice and $0.05/subscription overage. Add-on
# prices are PLACEHOLDERS — a founder pricing decision (override via env).
#
set -euo pipefail

API_URL="${API_URL:-http://localhost:8080/v1}"
API_KEY="${API_KEY:-sk_test_12345}"
CURRENCY="${CLOUD_CURRENCY:-USD}"

# Amounts in minor units (cents). Add-on prices are placeholders.
BASE_AMOUNT="${CLOUD_BASE_AMOUNT:-29900}"      # $299.00 base/mo
ADDON_CHURN="${CLOUD_ADDON_CHURN:-4900}"       # $49.00
ADDON_ANALYTICS="${CLOUD_ADDON_ANALYTICS:-7900}" # $79.00
ADDON_ACCOUNTING="${CLOUD_ADDON_ACCOUNTING:-3900}" # $39.00
ADDON_FX="${CLOUD_ADDON_FX:-2900}"             # $29.00

auth=(-H "Authorization: Bearer ${API_KEY}" -H "Content-Type: application/json")

echo "→ Recurso Cloud catalog setup on ${API_URL}"

# Existing plan codes (so re-runs are safe).
existing=$(curl -s "${auth[@]}" "${API_URL}/plans" | python3 -c \
  'import sys,json; d=json.load(sys.stdin); print("\n".join(p.get("code","") for p in (d.get("data") or [])))' 2>/dev/null || true)

# create_plan <code> <name> <amount> — creates the plan unless its code exists.
create_plan() {
  local code="$1" name="$2" amount="$3"
  if echo "${existing}" | grep -qx "${code}"; then
    echo "  ✓ ${code} already exists — skipping"
    return
  fi
  local body
  body=$(printf '{"name":"%s","code":"%s","currency":"%s","amount":%s,"interval_unit":"month","interval_count":1}' \
    "${name}" "${code}" "${CURRENCY}" "${amount}")
  local resp
  resp=$(curl -s -w '\n%{http_code}' "${auth[@]}" -X POST "${API_URL}/plans" -d "${body}")
  local status="${resp##*$'\n'}"
  if [ "${status}" = "200" ] || [ "${status}" = "201" ]; then
    echo "  + created ${code} (${name})"
  else
    echo "  ✗ failed to create ${code} (HTTP ${status}): ${resp%$'\n'*}" >&2
  fi
}

echo "Base plan:"
create_plan "cloud_base" "Recurso Cloud" "${BASE_AMOUNT}"

echo "Add-on plans (placeholder pricing — set via CLOUD_ADDON_* env):"
create_plan "cloud_addon_churn"      "Cloud Add-on: Churn Prediction"  "${ADDON_CHURN}"
create_plan "cloud_addon_analytics"  "Cloud Add-on: Advanced Analytics" "${ADDON_ANALYTICS}"
create_plan "cloud_addon_accounting" "Cloud Add-on: Accounting Sync"    "${ADDON_ACCOUNTING}"
create_plan "cloud_addon_fx"         "Cloud Add-on: Multi-currency FX"  "${ADDON_FX}"

cat <<EOF

✓ Catalog ready.

Overage is metered, not a plan — report it monthly on these usage dimensions:
  • invoices              (\$0.02 each beyond the 10,000 included)
  • active_subscriptions  (\$0.05 each beyond the 5,000 included)

See docs/cloud-dogfooding-runbook.md for the per-customer onboarding and the
monthly usage → overage-charge → invoice flow.
EOF
