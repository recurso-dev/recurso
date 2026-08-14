DROP TRIGGER IF EXISTS invoice_status_history_trg ON invoices;
DROP FUNCTION IF EXISTS record_invoice_status_change();
DROP TABLE IF EXISTS invoice_status_history;
