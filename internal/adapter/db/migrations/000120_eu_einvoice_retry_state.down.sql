DROP INDEX IF EXISTS idx_eu_einvoices_retry_due;

ALTER TABLE eu_einvoices
    DROP COLUMN IF EXISTS next_retry_at,
    DROP COLUMN IF EXISTS retry_count,
    DROP COLUMN IF EXISTS recipient_vat_id;
