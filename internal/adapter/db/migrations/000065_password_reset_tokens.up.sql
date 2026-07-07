-- Password reset: single-use, short-lived tokens. Only the SHA-256 hash of the
-- token is stored; the raw token travels solely in the emailed reset link.
CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id         UUID PRIMARY KEY,
    token_hash TEXT NOT NULL,              -- SHA-256 (hex) of the raw token
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,       -- ~1h after creation
    used_at    TIMESTAMPTZ,               -- set once the token is consumed
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS password_reset_tokens_hash_unique
    ON password_reset_tokens (token_hash);
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user_id
    ON password_reset_tokens (user_id);
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_expires_at
    ON password_reset_tokens (expires_at);
