CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    api_key_hash VARCHAR(64) NOT NULL,
    default_currency CHAR(3) DEFAULT 'USD',
    timezone VARCHAR(50) DEFAULT 'UTC',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS customers (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    billing_address JSONB,
    tax_id VARCHAR(50),
    ledger_account_id UUID NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(tenant_id, email)
);

CREATE TABLE IF NOT EXISTS plans (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(100) NOT NULL,
    code VARCHAR(50) NOT NULL,
    interval_unit VARCHAR(10) CHECK (interval_unit IN ('day', 'week', 'month', 'year')),
    interval_count INT DEFAULT 1,
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, code)
);

CREATE TABLE IF NOT EXISTS prices (
    id UUID PRIMARY KEY,
    plan_id UUID NOT NULL REFERENCES plans(id),
    currency CHAR(3) NOT NULL,
    amount BIGINT NOT NULL,
    type VARCHAR(20) DEFAULT 'recurring',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(plan_id, currency)
);

CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    customer_id UUID NOT NULL REFERENCES customers(id),
    plan_id UUID NOT NULL REFERENCES plans(id),
    status VARCHAR(20) CHECK (status IN ('trialing', 'active', 'past_due', 'paused', 'canceled', 'unpaid')),
    current_period_start TIMESTAMPTZ NOT NULL,
    current_period_end TIMESTAMPTZ NOT NULL,
    billing_anchor TIMESTAMPTZ NOT NULL,
    payment_method_id VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    canceled_at TIMESTAMPTZ,
    metadata JSONB
);

CREATE TABLE IF NOT EXISTS invoices (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    subscription_id UUID REFERENCES subscriptions(id),
    customer_id UUID REFERENCES customers(id),
    currency CHAR(3) NOT NULL,
    subtotal BIGINT NOT NULL,
    tax_amount BIGINT DEFAULT 0,
    total BIGINT NOT NULL,
    amount_paid BIGINT DEFAULT 0,
    amount_remaining BIGINT GENERATED ALWAYS AS (total - amount_paid) STORED,
    status VARCHAR(20) CHECK (status IN ('draft', 'open', 'paid', 'void', 'uncollectible')),
    due_date TIMESTAMPTZ,
    paid_at TIMESTAMPTZ,
    invoice_number VARCHAR(50) NOT NULL,
    pdf_url TEXT,
    irn VARCHAR(100),
    qr_code TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
