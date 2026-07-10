-- Persist a reusable payment method per customer for the self-service portal
-- (ENG-5 Phase 1). stripe_customer_id is the gateway-side Customer that saved
-- payment methods attach to; default_payment_method is the PaymentMethod id
-- (pm_*) to charge for future invoices. Card display fields (card_brand,
-- card_last4, card_exp_*) already exist and are refreshed alongside these.
ALTER TABLE customers
    ADD COLUMN IF NOT EXISTS stripe_customer_id TEXT,
    ADD COLUMN IF NOT EXISTS default_payment_method TEXT;
