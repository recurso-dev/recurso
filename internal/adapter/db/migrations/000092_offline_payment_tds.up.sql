-- TDS record-on-receipts (docs/spec_india_decisive.md P1): Indian B2B
-- customers deduct tax at source and pay invoices net of it. The deducted
-- portion is recorded on the receipt so AR, the ledger, and the GST return
-- stay consistent.
ALTER TABLE offline_payments ADD COLUMN IF NOT EXISTS tds_amount BIGINT NOT NULL DEFAULT 0;
