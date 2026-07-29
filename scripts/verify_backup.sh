#!/usr/bin/env bash
#
# verify_backup.sh — prove a backup is actually restorable (a backup you have
# never restored is not a backup). Restores a dump into a SCRATCH database and
# runs sanity checks: key tables exist and are populated, and the double-entry
# ledger balances (sum of debits == sum of credits).
#
# Usage:
#   SCRATCH_DATABASE_URL=postgres://user:pass@host:5432/recurso_verify \
#     ./scripts/verify_backup.sh path/to/recurso-<ts>.dump
#
# SCRATCH_DATABASE_URL MUST point at a throwaway, EMPTY database — its contents
# are replaced. Run this on a schedule (see docs/backups-and-dr.md) so a broken
# backup is caught long before you need it.

set -euo pipefail

DUMP_FILE="${1:?Usage: verify_backup.sh <dump-file>}"
: "${SCRATCH_DATABASE_URL:?SCRATCH_DATABASE_URL is required (a throwaway empty DB)}"

if [ ! -f "$DUMP_FILE" ]; then
  echo "ERROR: dump file not found: $DUMP_FILE" >&2
  exit 1
fi

echo "==> Restoring ${DUMP_FILE} into the scratch database…"
pg_restore \
  --clean --if-exists \
  --no-owner --no-privileges \
  --exit-on-error \
  -d "$SCRATCH_DATABASE_URL" \
  "$DUMP_FILE"

fail() { echo "VERIFY FAILED: $1" >&2; exit 1; }

q() { psql "$SCRATCH_DATABASE_URL" -tAc "$1"; }

echo "==> Checking core tables are present and populated…"
for tbl in tenants customers subscriptions invoices ledger_accounts ledger_transactions; do
  if [ "$(q "SELECT to_regclass('public.${tbl}') IS NOT NULL;")" != "t" ]; then
    fail "table '${tbl}' missing from the restored database"
  fi
  count="$(q "SELECT count(*) FROM ${tbl};")"
  echo "    ${tbl}: ${count} row(s)"
done

# A tenants table with zero rows means the dump restored an empty/wrong DB.
if [ "$(q "SELECT count(*) FROM tenants;")" -eq 0 ]; then
  fail "no tenants in the restored database — this dump looks empty"
fi

echo "==> Checking the double-entry ledger balances…"
# Postgres is the authoritative ledger (ADR-002). Every transaction posts the
# same amount to a debit and a credit account, so across ALL accounts total
# debits_posted must equal total credits_posted. A non-zero net = corrupt/
# partial restore.
NET="$(q "SELECT COALESCE(SUM(debits_posted) - SUM(credits_posted), 0) FROM ledger_accounts;")"
if [ "$NET" != "0" ]; then
  fail "ledger does not balance in the restored DB (sum debits - credits = ${NET})"
fi
echo "    ledger balances (net = 0) ✓"

echo "==> VERIFY OK — backup is restorable and internally consistent."
