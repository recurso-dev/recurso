-- TOTP multi-factor authentication for dashboard users.
--
-- mfa_secret holds the base32 TOTP secret. It is populated at "setup" time
-- (before the user has proven possession) and mfa_enabled flips to true only
-- once a valid code is verified. Disabling wipes the secret.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS mfa_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS mfa_secret TEXT;

-- One-time backup codes (10 per user). Only the SHA-256 hash is stored; the
-- plaintext codes are shown to the user exactly once at enable time.
CREATE TABLE IF NOT EXISTS mfa_backup_codes (
    id         UUID PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash  TEXT NOT NULL,              -- SHA-256 (hex) of the raw backup code
    used_at    TIMESTAMPTZ,               -- set once the code is consumed
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_mfa_backup_codes_user_id
    ON mfa_backup_codes (user_id);

-- Short-lived (5 min), single-use challenge tokens issued after a correct
-- password when MFA is on. Exchanged (with a TOTP/backup code) at
-- /auth/login/mfa for a real session. Only the hash is stored.
CREATE TABLE IF NOT EXISTS mfa_login_tokens (
    id         UUID PRIMARY KEY,
    token_hash TEXT NOT NULL,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS mfa_login_tokens_hash_unique
    ON mfa_login_tokens (token_hash);
CREATE INDEX IF NOT EXISTS idx_mfa_login_tokens_expires_at
    ON mfa_login_tokens (expires_at);
