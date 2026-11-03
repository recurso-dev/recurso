-- Make the renewal invoice the durable claim for a billing period, the way
-- mandate_cycle_key does for mandate debits. renewal_cycle_key =
-- "rn-<subscription_id>-<period_start_unix>"; a retry after a failed period
-- advance finds this invoice and reuses it instead of billing the flat fee
-- again. Only scheduled renewals set it, so the partial index leaves every
-- other invoice unconstrained.
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS renewal_cycle_key VARCHAR(255);

CREATE UNIQUE INDEX IF NOT EXISTS idx_invoices_renewal_cycle_key
    ON invoices (renewal_cycle_key)
    WHERE renewal_cycle_key IS NOT NULL;
