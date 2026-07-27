-- Occurrence-aware ledger idempotency (docs/design-ledger-occurrence.md).
-- The (reference_id, code) unique index assumed each code posts at most once
-- per reference for the invoice's lifetime. ACH late returns (PR #199) broke
-- that: a returned invoice is re-collected, and its second code-3 cash leg was
-- silently swallowed by ON CONFLICT DO NOTHING — Cash understated, AR
-- overstated, invisibly. Widening the key with an occurrence (cycle) counter
-- keeps same-cycle dedup exact while letting each new settle→reverse cycle
-- post its legs. Every existing row and non-cycle posting site stays at
-- occurrence 0 — byte-identical behavior outside the cycle.
ALTER TABLE ledger_transactions ADD COLUMN IF NOT EXISTS occurrence SMALLINT NOT NULL DEFAULT 0;

DROP INDEX IF EXISTS uq_ledger_tx_reference_code;

CREATE UNIQUE INDEX IF NOT EXISTS uq_ledger_tx_reference_code_occ
    ON ledger_transactions (reference_id, code, occurrence)
    WHERE reference_id <> '00000000-0000-0000-0000-000000000000';
