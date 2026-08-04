-- Journal-level accounting-model provenance (ADR-008 increment 2): stamp each
-- ledger journal with the accounting model in force when it was posted, so an
-- auditor can always answer "which accounting rules produced THIS journal entry?"
--   1 = cash model      (the legacy default; every existing posting is cash)
--   2 = accrual model   (schedule-at-issuance recognition, opt-in)
-- Every existing journal was posted under the cash model, so DEFAULT 1 backfills
-- them correctly with no data migration.
ALTER TABLE ledger_transactions
    ADD COLUMN IF NOT EXISTS accounting_version SMALLINT NOT NULL DEFAULT 1;
