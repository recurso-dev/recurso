-- SAML SSO foundation: one IdP connection per tenant.
--
-- A connection is created disabled. An owner/admin configures the tenant's IdP
-- (via raw metadata XML or the discrete entity-id / SSO-url / signing cert),
-- then flips enabled=true. SP-initiated login and the ACS endpoint are 404 for
-- any tenant without an enabled connection (feature-flagged per tenant).

CREATE TABLE IF NOT EXISTS sso_connections (
    id                UUID PRIMARY KEY,
    tenant_id         UUID NOT NULL UNIQUE REFERENCES tenants(id) ON DELETE CASCADE,
    idp_metadata_xml  TEXT,                    -- optional: full IdP metadata (parsed if present)
    idp_entity_id     TEXT NOT NULL DEFAULT '',
    idp_sso_url       TEXT NOT NULL DEFAULT '', -- HTTP-Redirect SingleSignOnService location
    idp_certificate   TEXT NOT NULL DEFAULT '', -- base64/PEM X.509 signing cert
    enabled           BOOLEAN NOT NULL DEFAULT FALSE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
