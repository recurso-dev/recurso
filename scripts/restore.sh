#!/usr/bin/env bash
#
# restore.sh — restore a Recurso backup (from scripts/backup.sh) into a target
# database. DESTRUCTIVE: it drops and recreates the target's schema objects.
#
# Usage:
#   TARGET_DATABASE_URL=postgres://user:pass@host:5432/recurso_restore \
#     ./scripts/restore.sh path/to/recurso-<ts>.dump
#
# Safety:
#   - Refuses to run without an explicit TARGET_DATABASE_URL (never defaults to
#     DATABASE_URL, so you cannot fat-finger a production overwrite).
#   - Requires CONFIRM=yes to proceed (interactive prompt otherwise).
#
# After a production restore, ALWAYS re-verify ledger integrity — run the
# reconciler / invariant harness (see docs/backups-and-dr.md).

set -euo pipefail

DUMP_FILE="${1:?Usage: restore.sh <dump-file>}"
: "${TARGET_DATABASE_URL:?TARGET_DATABASE_URL is required (never defaults to DATABASE_URL, on purpose)}"

if [ ! -f "$DUMP_FILE" ]; then
  echo "ERROR: dump file not found: $DUMP_FILE" >&2
  exit 1
fi

# Redact credentials when echoing the target.
SAFE_TARGET="$(printf '%s' "$TARGET_DATABASE_URL" | sed -E 's#(//[^:]+):[^@]+@#\1:****@#')"
echo "==> Restore target: ${SAFE_TARGET}"
echo "==> Dump file:      ${DUMP_FILE}"
echo "!!! This will OVERWRITE objects in the target database."

if [ "${CONFIRM:-}" != "yes" ]; then
  read -r -p "Type 'yes' to proceed: " reply
  if [ "$reply" != "yes" ]; then
    echo "Aborted."
    exit 1
  fi
fi

echo "==> Restoring…"
# --clean --if-exists  drop objects before recreating (idempotent restore)
# --no-owner/--no-privileges  match the dump flags
# --exit-on-error  fail loudly rather than leaving a half-restored DB
pg_restore \
  --clean --if-exists \
  --no-owner --no-privileges \
  --exit-on-error \
  -d "$TARGET_DATABASE_URL" \
  "$DUMP_FILE"

echo "==> Restore complete."
echo "==> NEXT: verify ledger integrity before trusting this database:"
echo "      TEST_DATABASE_URL=\"$TARGET_DATABASE_URL\" go test ./internal/service/ -run TestLedgerInvariants"
echo "    and/or hit GET /finance/reconciliation and confirm zero discrepancy."
