-- ENG-164: make the mandate-debit invoice the durable at-most-once claim for a
-- billing cycle. mandate_cycle_key = "md-<mandate_id>-<cycle>"; a UNIQUE index
-- means a re-attempt of the same cycle can't create a second invoice (and so
-- can't charge again). Only mandate debits set it — other invoices stay NULL, so
-- the partial index leaves them unconstrained.
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS mandate_cycle_key VARCHAR(255);

CREATE UNIQUE INDEX IF NOT EXISTS idx_invoices_mandate_cycle_key
    ON invoices (mandate_cycle_key)
    WHERE mandate_cycle_key IS NOT NULL;
