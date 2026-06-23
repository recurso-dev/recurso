CREATE TABLE IF NOT EXISTS ledger_transactions (
    id UUID PRIMARY KEY,
    debit_account_id UUID NOT NULL REFERENCES ledger_accounts(id),
    credit_account_id UUID NOT NULL REFERENCES ledger_accounts(id),
    amount BIGINT NOT NULL,
    ledger_id INTEGER NOT NULL DEFAULT 1,
    code SMALLINT NOT NULL,
    reference_id UUID,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ledger_transactions_debit ON ledger_transactions(debit_account_id);
CREATE INDEX idx_ledger_transactions_credit ON ledger_transactions(credit_account_id);
CREATE INDEX idx_ledger_transactions_reference ON ledger_transactions(reference_id);
