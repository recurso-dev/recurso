-- Seed Data for Recurso Demo
-- Run with: psql -d recurso -f seed_data.sql

-- First, create a demo tenant
INSERT INTO tenants (id, name, email, api_key, created_at)
VALUES (
    'a1b2c3d4-e5f6-7890-abcd-ef1234567890',
    'Demo Company',
    'admin@demo.recurso.dev',
    'recurso_demo_key_12345',
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- Create demo plans
INSERT INTO plans (id, tenant_id, name, description, price, currency, interval, interval_count, features, created_at) VALUES
('11111111-1111-1111-1111-111111111111', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'Starter', 'Perfect for small teams', 2900, 'USD', 'month', 1, '["5 users", "10GB storage", "Email support"]', NOW() - INTERVAL '90 days'),
('22222222-2222-2222-2222-222222222222', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'Professional', 'For growing businesses', 9900, 'USD', 'month', 1, '["25 users", "100GB storage", "Priority support", "API access"]', NOW() - INTERVAL '90 days'),
('33333333-3333-3333-3333-333333333333', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'Enterprise', 'Unlimited everything', 29900, 'USD', 'month', 1, '["Unlimited users", "1TB storage", "24/7 support", "Custom integrations", "SLA"]', NOW() - INTERVAL '90 days')
ON CONFLICT (id) DO NOTHING;

-- Create demo customers
INSERT INTO customers (id, tenant_id, email, name, phone, tax_id, line1, city, state, zip, country, billing_address, ledger_account_id, gstin, tax_type, place_of_supply, created_at) VALUES
('aaaa1111-1111-1111-1111-111111111111', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'john@acme.com', 'John Smith', '+1-555-0101', 'US123456789', '123 Main St', 'San Francisco', 'CA', '94102', 'US', '{}', 'aaaa1111-1111-1111-1111-111111111111', '', 'business', 'CA', NOW() - INTERVAL '85 days'),
('bbbb2222-2222-2222-2222-222222222222', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'sarah@startup.io', 'Sarah Johnson', '+1-555-0102', 'US987654321', '456 Oak Ave', 'Austin', 'TX', '78701', 'US', '{}', 'bbbb2222-2222-2222-2222-222222222222', '', 'business', 'TX', NOW() - INTERVAL '75 days'),
('cccc3333-3333-3333-3333-333333333333', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'mike@bigcorp.com', 'Mike Williams', '+1-555-0103', 'US456789123', '789 Enterprise Blvd', 'New York', 'NY', '10001', 'US', '{}', 'cccc3333-3333-3333-3333-333333333333', '', 'business', 'NY', NOW() - INTERVAL '60 days'),
('dddd4444-4444-4444-4444-444444444444', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'priya@techfirm.in', 'Priya Sharma', '+91-9876543210', '29AABCU9603R1ZM', '100 MG Road', 'Bangalore', 'KA', '560001', 'IN', '{}', 'dddd4444-4444-4444-4444-444444444444', '29AABCU9603R1ZM', 'business', 'KA', NOW() - INTERVAL '45 days'),
('eeee5555-5555-5555-5555-555555555555', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'alex@devshop.co', 'Alex Chen', '+1-555-0105', 'US111222333', '321 Code Lane', 'Seattle', 'WA', '98101', 'US', '{}', 'eeee5555-5555-5555-5555-555555555555', '', 'business', 'WA', NOW() - INTERVAL '30 days')
ON CONFLICT (id) DO NOTHING;

-- Create demo subscriptions
INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, status, current_period_start, current_period_end, trial_end, canceled_at, created_at) VALUES
('sub11111-1111-1111-1111-111111111111', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'aaaa1111-1111-1111-1111-111111111111', '22222222-2222-2222-2222-222222222222', 'active', NOW() - INTERVAL '30 days', NOW() + INTERVAL '1 day', NULL, NULL, NOW() - INTERVAL '85 days'),
('sub22222-2222-2222-2222-222222222222', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'bbbb2222-2222-2222-2222-222222222222', '11111111-1111-1111-1111-111111111111', 'active', NOW() - INTERVAL '15 days', NOW() + INTERVAL '16 days', NULL, NULL, NOW() - INTERVAL '75 days'),
('sub33333-3333-3333-3333-333333333333', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'cccc3333-3333-3333-3333-333333333333', '33333333-3333-3333-3333-333333333333', 'active', NOW() - INTERVAL '5 days', NOW() + INTERVAL '26 days', NULL, NULL, NOW() - INTERVAL '60 days'),
('sub44444-4444-4444-4444-444444444444', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'dddd4444-4444-4444-4444-444444444444', '22222222-2222-2222-2222-222222222222', 'active', NOW() - INTERVAL '20 days', NOW() + INTERVAL '11 days', NULL, NULL, NOW() - INTERVAL '45 days'),
('sub55555-5555-5555-5555-555555555555', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'eeee5555-5555-5555-5555-555555555555', '11111111-1111-1111-1111-111111111111', 'trialing', NOW(), NOW() + INTERVAL '14 days', NOW() + INTERVAL '14 days', NULL, NOW() - INTERVAL '30 days')
ON CONFLICT (id) DO NOTHING;

-- Create demo invoices (mix of paid and open)
INSERT INTO invoices (id, tenant_id, customer_id, subscription_id, status, amount_due, amount_paid, currency, due_date, paid_at, created_at) VALUES
-- Paid invoices (historical revenue)
('inv11111-1111-1111-1111-111111111111', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'aaaa1111-1111-1111-1111-111111111111', 'sub11111-1111-1111-1111-111111111111', 'paid', 9900, 9900, 'USD', NOW() - INTERVAL '60 days', NOW() - INTERVAL '58 days', NOW() - INTERVAL '65 days'),
('inv22222-2222-2222-2222-222222222222', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'aaaa1111-1111-1111-1111-111111111111', 'sub11111-1111-1111-1111-111111111111', 'paid', 9900, 9900, 'USD', NOW() - INTERVAL '30 days', NOW() - INTERVAL '28 days', NOW() - INTERVAL '35 days'),
('inv33333-3333-3333-3333-333333333333', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'bbbb2222-2222-2222-2222-222222222222', 'sub22222-2222-2222-2222-222222222222', 'paid', 2900, 2900, 'USD', NOW() - INTERVAL '45 days', NOW() - INTERVAL '43 days', NOW() - INTERVAL '50 days'),
('inv44444-4444-4444-4444-444444444444', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'bbbb2222-2222-2222-2222-222222222222', 'sub22222-2222-2222-2222-222222222222', 'paid', 2900, 2900, 'USD', NOW() - INTERVAL '15 days', NOW() - INTERVAL '14 days', NOW() - INTERVAL '20 days'),
('inv55555-5555-5555-5555-555555555555', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'cccc3333-3333-3333-3333-333333333333', 'sub33333-3333-3333-3333-333333333333', 'paid', 29900, 29900, 'USD', NOW() - INTERVAL '35 days', NOW() - INTERVAL '33 days', NOW() - INTERVAL '40 days'),
('inv66666-6666-6666-6666-666666666666', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'cccc3333-3333-3333-3333-333333333333', 'sub33333-3333-3333-3333-333333333333', 'paid', 29900, 29900, 'USD', NOW() - INTERVAL '5 days', NOW() - INTERVAL '4 days', NOW() - INTERVAL '10 days'),
('inv77777-7777-7777-7777-777777777777', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'dddd4444-4444-4444-4444-444444444444', 'sub44444-4444-4444-4444-444444444444', 'paid', 9900, 9900, 'USD', NOW() - INTERVAL '25 days', NOW() - INTERVAL '23 days', NOW() - INTERVAL '30 days'),
-- Open invoices (unpaid)
('inv88888-8888-8888-8888-888888888888', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'aaaa1111-1111-1111-1111-111111111111', 'sub11111-1111-1111-1111-111111111111', 'open', 9900, 0, 'USD', NOW() + INTERVAL '5 days', NULL, NOW() - INTERVAL '2 days'),
('inv99999-9999-9999-9999-999999999999', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'dddd4444-4444-4444-4444-444444444444', 'sub44444-4444-4444-4444-444444444444', 'open', 9900, 0, 'USD', NOW() + INTERVAL '10 days', NULL, NOW() - INTERVAL '1 day')
ON CONFLICT (id) DO NOTHING;

-- Create demo coupons
INSERT INTO coupons (id, tenant_id, code, discount_type, discount_value, max_redemptions, times_redeemed, expires_at, created_at) VALUES
('coup1111-1111-1111-1111-111111111111', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'WELCOME20', 'percent', 20, 100, 5, NOW() + INTERVAL '90 days', NOW() - INTERVAL '30 days'),
('coup2222-2222-2222-2222-222222222222', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'STARTUP50', 'percent', 50, 10, 2, NOW() + INTERVAL '30 days', NOW() - INTERVAL '15 days'),
('coup3333-3333-3333-3333-333333333333', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'FLAT100', 'fixed', 10000, 50, 8, NOW() + INTERVAL '60 days', NOW() - INTERVAL '45 days')
ON CONFLICT (id) DO NOTHING;

-- Summary of seeded data
SELECT 'Seed data loaded!' as status;
SELECT 'Tenants:' as entity, COUNT(*) as count FROM tenants;
SELECT 'Plans:' as entity, COUNT(*) as count FROM plans;
SELECT 'Customers:' as entity, COUNT(*) as count FROM customers;
SELECT 'Subscriptions:' as entity, COUNT(*) as count FROM subscriptions;
SELECT 'Invoices:' as entity, COUNT(*) as count FROM invoices;
SELECT 'Coupons:' as entity, COUNT(*) as count FROM coupons;
