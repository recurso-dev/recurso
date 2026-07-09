ALTER TABLE customers
    DROP COLUMN IF EXISTS default_payment_method,
    DROP COLUMN IF EXISTS stripe_customer_id;
