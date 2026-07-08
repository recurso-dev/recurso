-- Native OAuth social login (Google + GitHub).
--
-- Each row links a dashboard user to an external identity provider account.
-- A user may have several identities (one per provider). No provider access or
-- refresh tokens are ever stored — only the stable provider user id + the email
-- seen at link time (for audit).

CREATE TABLE IF NOT EXISTS user_oauth_identities (
    id               UUID PRIMARY KEY,
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider         TEXT NOT NULL,           -- 'google' | 'github'
    provider_user_id TEXT NOT NULL,           -- stable, opaque id from the provider
    email            TEXT NOT NULL,           -- email seen at link time (lower-cased by the app)
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- One external account maps to exactly one identity row: this is the primary
-- lookup key used by the callback (find-or-create step 1) and prevents two
-- users from claiming the same provider account.
CREATE UNIQUE INDEX IF NOT EXISTS user_oauth_identities_provider_uid_unique
    ON user_oauth_identities (provider, provider_user_id);

-- List a user's linked identities.
CREATE INDEX IF NOT EXISTS idx_user_oauth_identities_user
    ON user_oauth_identities (user_id);
