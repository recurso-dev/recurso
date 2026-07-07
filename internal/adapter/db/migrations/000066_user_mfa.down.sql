DROP TABLE IF EXISTS mfa_login_tokens;
DROP TABLE IF EXISTS mfa_backup_codes;
ALTER TABLE users DROP COLUMN IF EXISTS mfa_secret;
ALTER TABLE users DROP COLUMN IF EXISTS mfa_enabled;
