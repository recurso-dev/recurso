-- Payment attempts (US Market Readiness, Inc 3 / ACH). Cards settle synchronously,
-- but ACH (us_bank_account) is asynchronous: a debit sits in `processing` for
-- 1–5 business days, then settles or fails, and can even be RETURNED days after
-- it settled. The invoice status enum has no state for "authorized but not
-- cleared", so this row carries the async settlement lifecycle out-of-band; the
-- invoice stays `open` (with a derived "payment processing" indicator) until an
-- attempt reaches `succeeded`.
CREATE TABLE IF NOT EXISTS payment_attempts (
    id                        UUID PRIMARY KEY,
    tenant_id                 UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    invoice_id                UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    gateway                   TEXT   NOT NULL,               -- e.g. 'stripe'
    method                    TEXT   NOT NULL,               -- e.g. 'us_bank_account'
    gateway_payment_intent_id TEXT   NOT NULL DEFAULT '',    -- pi_* the webhook keys on
    status                    TEXT   NOT NULL DEFAULT 'initiated'
        CHECK (status IN ('initiated', 'processing', 'succeeded', 'failed', 'returned')),
    failure_code              TEXT   NOT NULL DEFAULT '',     -- ACH return/failure code (R01, …)
    amount                    BIGINT NOT NULL,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    settled_at                TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_payment_attempts_invoice ON payment_attempts (invoice_id);

-- A webhook advances the attempt by its PaymentIntent id; the partial-unique
-- index makes that lookup exact and idempotent (one attempt per intent).
CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_attempts_pi
    ON payment_attempts (gateway_payment_intent_id)
    WHERE gateway_payment_intent_id <> '';
