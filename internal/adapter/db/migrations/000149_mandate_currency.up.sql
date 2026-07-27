-- Mandates route by currency (INR -> Razorpay UPI AutoPay; overridden
-- currencies -> bank-debit gateways like GoCardless). Existing mandates are
-- all UPI, hence the INR default.
ALTER TABLE mandates ADD COLUMN IF NOT EXISTS currency VARCHAR(3) NOT NULL DEFAULT 'INR';
