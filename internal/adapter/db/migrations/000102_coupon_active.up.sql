-- Coupons gain a soft-deactivate flag. Existing coupons stay redeemable
-- (DEFAULT TRUE); deactivated ones are rejected at subscription creation.
ALTER TABLE coupons ADD COLUMN active BOOLEAN NOT NULL DEFAULT TRUE;
