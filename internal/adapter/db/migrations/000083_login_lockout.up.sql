-- ENG-151: per-account lockout. Track consecutive failed login/MFA attempts and
-- a temporary lock window, so credential-stuffing spread across many IPs (which
-- slips past the per-IP rate limit) is bounded per account.
ALTER TABLE users ADD COLUMN IF NOT EXISTS failed_login_attempts INT NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS locked_until TIMESTAMPTZ;
