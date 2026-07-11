-- Replay-protection cache for consumed SAML assertions (ENG-151).
-- Each successfully-validated SAMLResponse assertion is recorded here by its
-- assertion ID; a second ACS POST carrying the same assertion ID is rejected as
-- a replay. Postgres-backed (not in-memory) so protection holds across the
-- multiple Cloud Run instances that can serve /auth/saml/:tenant/acs.
CREATE TABLE IF NOT EXISTS sso_consumed_assertions (
    assertion_id TEXT        NOT NULL PRIMARY KEY,
    tenant_id    UUID        NOT NULL,
    expires_at   TIMESTAMPTZ NOT NULL,
    consumed_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Supports cheap pruning of rows past their assertion validity window.
CREATE INDEX IF NOT EXISTS idx_sso_consumed_assertions_expires_at
    ON sso_consumed_assertions (expires_at);
