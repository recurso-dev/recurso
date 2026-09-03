ALTER TABLE invoices DROP COLUMN IF EXISTS retry_count;
ALTER TABLE invoices DROP COLUMN IF EXISTS next_retry_at;
