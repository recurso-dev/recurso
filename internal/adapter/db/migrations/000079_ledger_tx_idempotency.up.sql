-- Make ledger postings idempotent per (reference_id, code): an invoice (code 1),
-- payment (code 3), or refund (code 4) is posted at most once for its reference,
-- so a replayed or concurrently-lost settle can never double-post. Revenue-
-- recognition rows (code 2) carry no reference (the zero UUID) and legitimately
-- post many times per invoice — they are excluded from the index.

-- First remove any pre-existing duplicates (from the pre-fix double-post race)
-- so the unique index can be created; keep the earliest row per (reference, code).
-- NOTE: this de-duplicates transaction rows only; it does not reverse balances a
-- duplicate may already have moved — run ledger reconciliation to detect those.
DELETE FROM ledger_transactions
WHERE reference_id <> '00000000-0000-0000-0000-000000000000'
  AND id NOT IN (
    SELECT DISTINCT ON (reference_id, code) id
    FROM ledger_transactions
    WHERE reference_id <> '00000000-0000-0000-0000-000000000000'
    ORDER BY reference_id, code, created_at, id
  );

CREATE UNIQUE INDEX IF NOT EXISTS uq_ledger_tx_reference_code
    ON ledger_transactions (reference_id, code)
    WHERE reference_id <> '00000000-0000-0000-0000-000000000000';
