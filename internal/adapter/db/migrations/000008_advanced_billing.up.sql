-- Add advanced billing fields to subscriptions
ALTER TABLE subscriptions ADD COLUMN billing_anchor_type VARCHAR(50) DEFAULT 'acquisition'; -- 'acquisition', 'first_of_month', 'specific_day'
ALTER TABLE subscriptions ADD COLUMN billing_anchor_day INT DEFAULT 0; -- 0-31 (0 means use start_date day, else specific day)
ALTER TABLE subscriptions ADD COLUMN payment_terms VARCHAR(50) DEFAULT 'net0'; -- 'net0', 'net15', 'net30', 'net60'

-- Add payment terms to invoices to track due dates properly (due_date already exists)
-- ALTER TABLE invoices ADD COLUMN due_date TIMESTAMPTZ;
ALTER TABLE invoices ADD COLUMN payment_terms VARCHAR(50) DEFAULT 'net0';

-- Create table for unbilled charges (one-off items to be added to next invoice)
CREATE TABLE unbilled_charges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID NOT NULL REFERENCES subscriptions(id),
    amount BIGINT NOT NULL, -- in cents
    currency VARCHAR(3) NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    period_start TIMESTAMPTZ, -- optional: if charge relates to a period
    period_end TIMESTAMPTZ,
    status VARCHAR(50) DEFAULT 'pending' -- 'pending', 'invoiced', 'canceled'
);

CREATE INDEX idx_unbilled_charges_subscription_id ON unbilled_charges(subscription_id);
CREATE INDEX idx_unbilled_charges_status ON unbilled_charges(status);
