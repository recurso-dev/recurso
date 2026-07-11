-- ENG-151: single-use TOTP. Record the last consumed timestep so a captured
-- code can't be replayed within the ~30-90s validation window. Any code whose
-- timestep is <= this value is rejected.
ALTER TABLE users ADD COLUMN IF NOT EXISTS mfa_last_timestep BIGINT NOT NULL DEFAULT 0;
