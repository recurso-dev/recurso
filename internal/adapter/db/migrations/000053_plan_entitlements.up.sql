-- Plan-level feature entitlements (Entitlement Engine v1).
--
-- Each row grants a feature to every customer subscribed (active/trialing)
-- to the plan. Two kinds:
--   * 'boolean' -> bool_value holds the grant (true/false)
--   * 'limit'   -> limit_value holds a numeric cap (e.g. 10000 API calls)
--
-- Effective customer entitlements are resolved as the UNION over the plans
-- of the customer's ACTIVE and TRIALING subscriptions:
--   boolean: granted if ANY plan grants true
--   limit:   MAX limit_value across plans (most generous plan wins)

CREATE TABLE IF NOT EXISTS plan_entitlements (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    plan_id UUID NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    feature_key TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('boolean', 'limit')),
    bool_value BOOLEAN,
    limit_value BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (plan_id, feature_key)
);

-- Kind/value consistency: exactly the matching value column is set.
ALTER TABLE plan_entitlements
    ADD CONSTRAINT plan_entitlements_value_check CHECK (
        (kind = 'boolean' AND bool_value IS NOT NULL AND limit_value IS NULL) OR
        (kind = 'limit' AND limit_value IS NOT NULL AND bool_value IS NULL)
    );

-- Tenant-scoped feature lookups (the UNIQUE above already covers
-- plan_id-first lookups and the entitlement-check join by plan_id).
CREATE INDEX IF NOT EXISTS idx_plan_entitlements_tenant_feature
    ON plan_entitlements (tenant_id, feature_key);

-- Hot path for GET /v1/entitlements/check: resolve a customer's
-- active/trialing subscriptions without a table scan.
CREATE INDEX IF NOT EXISTS idx_subscriptions_customer_status
    ON subscriptions (tenant_id, customer_id, status);
