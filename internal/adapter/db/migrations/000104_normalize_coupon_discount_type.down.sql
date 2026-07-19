-- Best-effort inverse; the aliased values were never canonical.
UPDATE coupons SET discount_type = 'percentage' WHERE discount_type = 'percent';
UPDATE coupons SET discount_type = 'fixed'      WHERE discount_type = 'amount';
