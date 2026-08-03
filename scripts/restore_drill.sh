#!/usr/bin/env bash
# Restore drill (Evidence artifact) — proves a backup restores completely and
# the restored books still balance. Run against ANY source database (a local
# instance or a downloaded production backup):
#
#   ./scripts/restore_drill.sh "postgres://.../source_db" [report.md]
#
# What it proves, in order:
#   1. RESTORABILITY  — pg_dump of the source restores into a fresh database.
#   2. COMPLETENESS   — per-table row counts match source exactly.
#   3. FIDELITY       — an order-independent checksum over every invoice's
#                       money columns (id|total|tax|paid|status) matches.
#   4. CONSERVATION   — double-entry holds in the RESTORED books: total posted
#                       debits equal total posted credits, globally and for
#                       every tenant.
# Timings for dump and restore are recorded (your RTO evidence).
#
# v2 (planned): drive the in-product reconciler per tenant via the API for the
# full leg-completeness/abnormal-balance sweep on the restored instance.
set -euo pipefail

SRC_URL="${1:?usage: restore_drill.sh <source_db_url> [report.md]}"
REPORT="${2:-docs/drills/$(date +%Y-%m-%d)-restore-drill.md}"
STAMP=$(date +%Y%m%d%H%M%S)
DUMP="/tmp/recurso_drill_${STAMP}.dump"
RESTORED_DB="recurso_drill_${STAMP}"

say() { echo "[drill] $*"; }
psql_src() { psql "$SRC_URL" -t -A -c "$1"; }
psql_dst() { psql "$RESTORED_DB" -t -A -c "$1"; }

say "source: $SRC_URL"

# --- 1. Dump (timed) ---------------------------------------------------------
t0=$(date +%s)
pg_dump -Fc "$SRC_URL" -f "$DUMP"
t1=$(date +%s)
DUMP_SECS=$((t1 - t0))
DUMP_SIZE=$(du -h "$DUMP" | cut -f1)
say "dump: ${DUMP_SIZE} in ${DUMP_SECS}s"

# --- 2. Restore into a FRESH database (timed) --------------------------------
createdb "$RESTORED_DB"
t2=$(date +%s)
pg_restore -d "$RESTORED_DB" --no-owner "$DUMP"
t3=$(date +%s)
RESTORE_SECS=$((t3 - t2))
say "restore: ${RESTORE_SECS}s into $RESTORED_DB"

# --- 3. Completeness: per-table row-count parity -----------------------------
TABLES=$(psql_src "SELECT tablename FROM pg_tables WHERE schemaname='public' ORDER BY tablename")
MISMATCH=0; TABLE_ROWS=""
for t in $TABLES; do
  a=$(psql_src "SELECT count(*) FROM \"$t\"")
  b=$(psql_dst "SELECT count(*) FROM \"$t\"")
  status="OK"; [ "$a" != "$b" ] && { status="MISMATCH"; MISMATCH=1; }
  TABLE_ROWS+="| \`$t\` | $a | $b | $status |
"
done

# --- 4. Fidelity: order-independent invoice money checksum -------------------
CK_Q="SELECT md5(string_agg(l, '' ORDER BY l)) FROM (SELECT id::text||'|'||total||'|'||COALESCE(tax_amount,0)||'|'||amount_paid||'|'||status AS l FROM invoices) s"
CK_SRC=$(psql_src "$CK_Q"); CK_DST=$(psql_dst "$CK_Q")
CK_STATUS="OK"; [ "$CK_SRC" != "$CK_DST" ] && { CK_STATUS="MISMATCH"; MISMATCH=1; }

# --- 5. Conservation: double-entry holds in the RESTORED books ---------------
GLOB=$(psql_dst "SELECT COALESCE(SUM(debits_posted),0)||'|'||COALESCE(SUM(credits_posted),0) FROM ledger_accounts")
GD=${GLOB%|*}; GC=${GLOB#*|}
CONS_STATUS="OK"; [ "$GD" != "$GC" ] && { CONS_STATUS="MISMATCH"; MISMATCH=1; }
BAD_TENANTS=$(psql_dst "SELECT count(*) FROM (SELECT tenant_id FROM ledger_accounts GROUP BY tenant_id HAVING SUM(debits_posted) <> SUM(credits_posted)) x")
[ "$BAD_TENANTS" != "0" ] && { CONS_STATUS="MISMATCH"; MISMATCH=1; }
TENANTS=$(psql_dst "SELECT count(*) FROM tenants")
LEDGER_TX=$(psql_dst "SELECT count(*) FROM ledger_transactions")

# --- 6. Report ---------------------------------------------------------------
OVERALL="PASS"; [ "$MISMATCH" != "0" ] && OVERALL="FAIL"
cat > "$REPORT" <<MD
# Restore drill — $(date +%Y-%m-%d)

**Result: ${OVERALL}**

Proves the backup restores completely and the restored books still balance.
Produced by [\`scripts/restore_drill.sh\`](../../scripts/restore_drill.sh); every
number below is regenerable by re-running it.

| Metric | Value |
|---|---|
| Dump size | ${DUMP_SIZE} |
| Dump time | ${DUMP_SECS}s |
| **Restore time (RTO core)** | **${RESTORE_SECS}s** |
| Tenants restored | ${TENANTS} |
| Ledger transactions restored | ${LEDGER_TX} |
| Invoice money checksum (order-independent md5) | ${CK_STATUS} |
| Double-entry conservation, global (Σdebits = Σcredits) | ${CONS_STATUS} (${GD} = ${GC}) |
| Tenants with unbalanced books after restore | ${BAD_TENANTS} |

## Per-table row-count parity

| Table | Source | Restored | Status |
|---|---|---|---|
${TABLE_ROWS}
## Method

1. \`pg_dump -Fc\` of the source database.
2. \`pg_restore\` into a **fresh** database (no pre-existing schema).
3. Row-count parity across every public table.
4. Order-independent md5 over every invoice's money columns
   (\`id|total|tax|paid|status\`).
5. Double-entry conservation on the **restored** ledger, globally and
   per tenant.

Planned v2: drive the in-product reconciler per tenant on the restored
instance for the full leg-completeness / abnormal-balance sweep.
MD

say "report: $REPORT (${OVERALL})"
dropdb "$RESTORED_DB"
rm -f "$DUMP"
[ "$OVERALL" = "PASS" ]
