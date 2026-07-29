-- Managed-cloud buy path (Phase B): give each tenant a billing lifecycle so the
-- product can offer a trial and, later, self-serve plans. Defaults are chosen so
-- EXISTING tenants are unaffected (active/free, no trial banner); new signups
-- explicitly start a trial in code (TenantService.Register).
ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS trial_ends_at  TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS billing_status TEXT NOT NULL DEFAULT 'active',
    ADD COLUMN IF NOT EXISTS plan_tier      TEXT NOT NULL DEFAULT 'free';

-- billing_status: 'trialing' | 'active' | 'past_due' | 'canceled'
-- plan_tier: 'trial' | 'free' | '<paid tier>' (paid tiers defined once pricing lands)
CREATE INDEX IF NOT EXISTS idx_tenants_billing_status ON tenants (billing_status);
CREATE INDEX IF NOT EXISTS idx_tenants_trial_ends_at ON tenants (trial_ends_at);
