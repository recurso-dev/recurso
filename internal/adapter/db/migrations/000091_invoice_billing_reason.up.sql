-- Persist the invoice billing_reason that the API/SDK already document but the
-- code never stored. Nullable: rows created before this migration have no
-- recorded reason and read back as '' (COALESCE). New invoices set it at
-- creation (subscription_create/cycle/update, mandate_debit, gift_purchase, manual).
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS billing_reason VARCHAR(50);
