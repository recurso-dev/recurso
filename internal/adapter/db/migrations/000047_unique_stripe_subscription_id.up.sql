-- stripe_subscription_id is used by the Stripe webhook handler to resolve a
-- subscription (and thereby its tenant) without a tenant_id in hand. A unique
-- index guarantees that lookup can never match rows from more than one tenant.
CREATE UNIQUE INDEX IF NOT EXISTS idx_subscriptions_stripe_subscription_id
    ON subscriptions (stripe_subscription_id)
    WHERE stripe_subscription_id IS NOT NULL AND stripe_subscription_id <> '';
