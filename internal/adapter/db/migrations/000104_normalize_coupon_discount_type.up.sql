-- The demo seed inserted discount_type values ("percentage"/"fixed") that
-- bypass the API enum ("percent"/"amount"). The billing engine compares
-- against the canonical values, so a seeded 20% coupon was silently applied
-- as a 20-minor-unit amount discount. Normalize existing rows.
UPDATE coupons SET discount_type = 'percent' WHERE discount_type = 'percentage';
UPDATE coupons SET discount_type = 'amount'  WHERE discount_type = 'fixed';
