ALTER TABLE invoices
DROP COLUMN e_invoice_status,
DROP COLUMN tds_amount,
DROP COLUMN signed_qr_code;

DROP TYPE e_invoice_status;
