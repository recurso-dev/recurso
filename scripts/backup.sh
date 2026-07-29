#!/usr/bin/env bash
#
# backup.sh — take a consistent, compressed logical backup of the Recurso
# Postgres database (the authoritative store, including the ledger).
#
# Usage:
#   DATABASE_URL=postgres://user:pass@host:5432/recurso ./scripts/backup.sh [OUT_DIR]
#
# Output: OUT_DIR/recurso-<UTC-timestamp>.dump  (custom format, gzip-compressed
# by pg_dump -Fc, restorable with scripts/restore.sh).
#
# This is the self-host / portable path. On managed Postgres (Cloud SQL, RDS)
# prefer provider automated backups + PITR as the primary; see
# docs/backups-and-dr.md. This script is the second line and the export path.

set -euo pipefail

: "${DATABASE_URL:?DATABASE_URL is required (postgres connection string)}"

OUT_DIR="${1:-./backups}"
mkdir -p "$OUT_DIR"

# UTC, sortable, filesystem-safe timestamp. No secrets in the filename.
TS="$(date -u +%Y%m%dT%H%M%SZ)"
OUT_FILE="$OUT_DIR/recurso-${TS}.dump"

echo "==> Backing up Recurso database to ${OUT_FILE}"

# -Fc  custom format (compressed, selective restore)
# -Z9  max compression
# --no-owner / --no-privileges  so the dump restores cleanly into a fresh role
pg_dump "$DATABASE_URL" \
  -Fc -Z9 \
  --no-owner \
  --no-privileges \
  -f "$OUT_FILE"

SIZE="$(du -h "$OUT_FILE" | cut -f1)"
echo "==> Done: ${OUT_FILE} (${SIZE})"

# Optional retention: prune local dumps older than RETAIN_DAYS (default 14).
RETAIN_DAYS="${RETAIN_DAYS:-14}"
if [ "$RETAIN_DAYS" -gt 0 ]; then
  echo "==> Pruning local dumps older than ${RETAIN_DAYS} day(s)"
  find "$OUT_DIR" -name 'recurso-*.dump' -type f -mtime "+${RETAIN_DAYS}" -print -delete || true
fi

echo "==> Backup complete. Verify restorability regularly with scripts/verify_backup.sh"
