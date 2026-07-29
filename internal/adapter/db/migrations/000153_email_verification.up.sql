-- Email verification: prove a signup controls the address it registered with.
-- `users.email_verified_at` is NULL until the account confirms; a NULL value
-- means "unverified" and drives the dashboard's verify-your-email banner.
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified_at TIMESTAMPTZ;

-- Single-use, short-lived verification tokens. Only the SHA-256 hash of the
-- token is stored; the raw token travels solely in the emailed verify link —
-- exactly mirroring password_reset_tokens (migration 000065).
CREATE TABLE IF NOT EXISTS email_verification_tokens (
    id         UUID PRIMARY KEY,
    token_hash TEXT NOT NULL,              -- SHA-256 (hex) of the raw token
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,       -- ~24h after creation
    used_at    TIMESTAMPTZ,               -- set once the token is consumed
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS email_verification_tokens_hash_unique
    ON email_verification_tokens (token_hash);
CREATE INDEX IF NOT EXISTS idx_email_verification_tokens_user_id
    ON email_verification_tokens (user_id);
CREATE INDEX IF NOT EXISTS idx_email_verification_tokens_expires_at
    ON email_verification_tokens (expires_at);
