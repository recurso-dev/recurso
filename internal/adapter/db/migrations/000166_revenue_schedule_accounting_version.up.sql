-- Accounting-model versioning (ADR-008 follow-up): stamp each revenue schedule
-- with the accounting model that produced it, so we can always answer "which
-- accounting rules created this recognition?" — enabling historical
-- reproducibility, per-tenant migration, and rollback.
--   1 = cash recognition (schedule built at payment) — the legacy default
--   2 = accrual recognition (schedule built at issuance)
-- Every existing schedule was produced by the cash model, so DEFAULT 1 backfills
-- them correctly with no data migration.
ALTER TABLE revenue_schedules
    ADD COLUMN IF NOT EXISTS accounting_version SMALLINT NOT NULL DEFAULT 1;
