-- Recurso Cloud self-billing ("Recurso runs on Recurso"), Increment 1.
--
-- Maps each signup tenant to the Customer that represents it INSIDE the platform
-- (founder) tenant's own Recurso account. This is what makes a signup — e.g.
-- "Spotify" — show up as a customer of Recurso in the founder's dashboard, so
-- the founder bills their own cloud tenants with the same engine everyone else
-- uses.
--
-- Increment 1 records only the customer link. subscription_id is reserved for a
-- later increment (the Recurso Cloud plan + usage metering), when there is
-- something to actually charge. The UNIQUE (platform_tenant_id, tenant_id)
-- makes provisioning idempotent — a retried signup or a re-run backfill can
-- never create a duplicate customer for the same tenant.
CREATE TABLE IF NOT EXISTS cloud_tenant_customer (
    id                 UUID PRIMARY KEY,
    platform_tenant_id UUID NOT NULL,
    tenant_id          UUID NOT NULL,
    customer_id        UUID NOT NULL,
    subscription_id    UUID,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (platform_tenant_id, tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_cloud_tenant_customer_tenant
    ON cloud_tenant_customer (tenant_id);
