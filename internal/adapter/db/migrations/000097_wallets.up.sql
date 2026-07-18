-- Prepaid wallets (Lago-parity B1): money-denominated balance per
-- customer+currency, drained oldest-expiring-first at invoice time.
-- wallet_transactions is append-only; top_up rows carry `remaining`, the
-- undrained residue that drains decrement and expiry writes off. The
-- wallet's balance is the denormalized SUM of open residues, updated in the
-- same transaction as every movement (CHECK keeps it non-negative).

CREATE TABLE IF NOT EXISTS wallets (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    customer_id UUID NOT NULL REFERENCES customers(id),
    currency VARCHAR(3) NOT NULL,
    balance BIGINT NOT NULL DEFAULT 0 CHECK (balance >= 0),
    auto_recharge_threshold BIGINT,
    auto_recharge_amount BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, customer_id, currency)
);

CREATE TABLE IF NOT EXISTS wallet_transactions (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    wallet_id UUID NOT NULL REFERENCES wallets(id),
    type TEXT NOT NULL CHECK (type IN ('top_up', 'drain', 'expiry')),
    source TEXT NOT NULL DEFAULT '',
    amount BIGINT NOT NULL CHECK (amount > 0),
    remaining BIGINT CHECK (remaining IS NULL OR remaining >= 0),
    balance_after BIGINT NOT NULL,
    invoice_id UUID,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wallet_tx_wallet
    ON wallet_transactions (wallet_id, created_at DESC);

-- Drain hot path: open residues of a wallet, oldest expiry first.
CREATE INDEX IF NOT EXISTS idx_wallet_tx_open_topups
    ON wallet_transactions (wallet_id, expires_at)
    WHERE type = 'top_up' AND remaining > 0;
