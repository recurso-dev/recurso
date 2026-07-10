-- GenAI analytics hardening (ENG-137): LLM-generated SQL must never be able
-- to cross a tenant boundary or read sensitive tables, as a DATABASE
-- guarantee rather than a prompt instruction.
--
-- Design: a dedicated `genai` schema of views that (a) expose only benign
-- business columns and (b) bake in tenant scoping via the app.tenant_id GUC;
-- plus a NOLOGIN role that can see ONLY this schema. The service runs each
-- generated query inside a transaction with SET LOCAL ROLE genai_readonly,
-- so even a fully adversarial query cannot touch public.* (users, api keys,
-- other tenants' rows) — it gets "permission denied".

CREATE SCHEMA IF NOT EXISTS genai;

-- current_setting(..., true) returns NULL when unset → views yield no rows.
CREATE OR REPLACE VIEW genai.customers AS
    SELECT id, email, name, tax_type, created_at
    FROM public.customers
    WHERE tenant_id = current_setting('app.tenant_id', true)::uuid;

CREATE OR REPLACE VIEW genai.invoices AS
    SELECT id, customer_id, invoice_number, status, currency,
           subtotal, tax_amount, total, amount_paid, due_date, paid_at, created_at
    FROM public.invoices
    WHERE tenant_id = current_setting('app.tenant_id', true)::uuid;

CREATE OR REPLACE VIEW genai.subscriptions AS
    SELECT id, customer_id, plan_id, status,
           current_period_start, current_period_end, trial_end, created_at
    FROM public.subscriptions
    WHERE tenant_id = current_setting('app.tenant_id', true)::uuid;

CREATE OR REPLACE VIEW genai.plans AS
    SELECT id, name, code, interval_unit, interval_count, active, created_at
    FROM public.plans
    WHERE tenant_id = current_setting('app.tenant_id', true)::uuid;

CREATE OR REPLACE VIEW genai.prices AS
    SELECT pr.id, pr.plan_id, pr.currency, pr.amount, pr.type
    FROM public.prices pr
    JOIN public.plans p ON p.id = pr.plan_id
    WHERE p.tenant_id = current_setting('app.tenant_id', true)::uuid;

-- Role creation is cluster-level; guard against races between parallel
-- databases migrating in the same cluster (CI).
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'genai_readonly') THEN
        CREATE ROLE genai_readonly NOLOGIN;
    END IF;
EXCEPTION WHEN duplicate_object THEN
    NULL;
END $$;

GRANT USAGE ON SCHEMA genai TO genai_readonly;
GRANT SELECT ON ALL TABLES IN SCHEMA genai TO genai_readonly;
ALTER DEFAULT PRIVILEGES IN SCHEMA genai GRANT SELECT ON TABLES TO genai_readonly;

-- The app's connecting role must be a member to SET LOCAL ROLE into it.
DO $$
BEGIN
    EXECUTE format('GRANT genai_readonly TO %I', CURRENT_USER);
END $$;
