-- Best-effort down: recreating the narrower index fails if any reference has
-- accumulated multi-occurrence legs (a completed settle→reverse→re-settle
-- cycle) — acceptable for a down migration.
DROP INDEX IF EXISTS uq_ledger_tx_reference_code_occ;
CREATE UNIQUE INDEX IF NOT EXISTS uq_ledger_tx_reference_code
    ON ledger_transactions (reference_id, code)
    WHERE reference_id <> '00000000-0000-0000-0000-000000000000';
ALTER TABLE ledger_transactions DROP COLUMN IF EXISTS occurrence;
