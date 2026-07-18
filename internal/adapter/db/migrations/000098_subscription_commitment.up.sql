-- Minimum commitments (Lago-parity B2): a per-period floor in minor units
-- (currency implied by the plan). At period close, if the invoice subtotal
-- (flat + add-ons + metered usage) falls short, a true-up line fills the
-- difference. 0 = no commitment.
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS commitment_amount BIGINT NOT NULL DEFAULT 0;
