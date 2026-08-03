-- R3 (ENG-195): records whether the subscription's CURRENT billing period was
-- invoiced with its coupon's discount. Plan-change proration must credit/charge
-- at the prices the customer actually pays this period: a coupon-blind
-- proration credited unused time at LIST price, so a heavily-discounted
-- subscription could downgrade into more account credit than it ever paid
-- (money-out over-credit), and an upgrading discounted customer was charged the
-- full list-price difference. The coupon counter alone cannot derive this (a
-- repeating coupon past its window keeps its final count), so the fact is
-- recorded at each invoice-generation site.
ALTER TABLE subscriptions
    ADD COLUMN IF NOT EXISTS coupon_applied_current_period BOOLEAN NOT NULL DEFAULT FALSE;
