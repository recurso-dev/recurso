-- Subscription add-ons (Multi-product catalog v1, Lane 2).
--
-- An add-on attaches an existing plan to a subscription with a quantity. The
-- subscription keeps its base plan_id unchanged; each add-on becomes an extra
-- line on the subscription's next recurring invoice (base plan amount PLUS
-- each add-on's plan price × quantity, taxed independently). The add-on plan's
-- price currency must match the subscription's currency (enforced in service).
--
-- Timing (v1): add-ons take effect from the NEXT invoice. Mid-cycle proration
-- is a deliberate follow-up, not part of this table's contract.

CREATE TABLE IF NOT EXISTS subscription_addons (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    subscription_id UUID NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES plans(id),
    quantity INTEGER NOT NULL DEFAULT 1 CHECK (quantity > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Hot path: list a subscription's add-ons during invoice generation,
-- tenant-scoped.
CREATE INDEX IF NOT EXISTS idx_subscription_addons_subscription
    ON subscription_addons (tenant_id, subscription_id);
