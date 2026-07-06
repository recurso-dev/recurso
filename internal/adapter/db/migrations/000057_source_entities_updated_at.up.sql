-- Dirty tracking for accounting sync: SyncAllForTenant compares each source
-- entity's updated_at against its accounting_entity_mappings row's updated_at
-- (refreshed on every successful sync) and skips entities that have not
-- changed since they were last pushed. Subscriptions already carry
-- updated_at (000001); customers, invoices and plans did not.
--
-- Defaulting to NOW() stamps existing rows with the migration time, which is
-- >= every existing mapping's updated_at, so each mapped entity is pushed at
-- most once more after this migration and then settles.
ALTER TABLE customers ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE plans ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
