-- Store the Razorpay customer id so mandate tokens can be revoked via
-- DELETE /v1/customers/{customer_id}/tokens/{token_id}
ALTER TABLE mandates ADD COLUMN IF NOT EXISTS razorpay_customer_id VARCHAR(100) NOT NULL DEFAULT '';
