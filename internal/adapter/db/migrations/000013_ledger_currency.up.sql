CREATE TABLE IF NOT EXISTS ledger_accounts (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    code INT NOT NULL,
    ledger_id INT NOT NULL, /* TigerBeetle Ledger ID */
    user_data_128_high BIGINT DEFAULT 0,
    user_data_128_low BIGINT DEFAULT 0,
    credits_posted BIGINT DEFAULT 0,
    debits_posted BIGINT DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    balance BIGINT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
