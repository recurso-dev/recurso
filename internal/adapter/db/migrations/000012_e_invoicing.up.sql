CREATE TYPE e_invoice_status AS ENUM ('PENDING', 'GENERATED', 'FAILED');

ALTER TABLE invoices
ADD COLUMN signed_qr_code TEXT,
ADD COLUMN tds_amount BIGINT DEFAULT 0,
ADD COLUMN e_invoice_status e_invoice_status DEFAULT 'PENDING';
