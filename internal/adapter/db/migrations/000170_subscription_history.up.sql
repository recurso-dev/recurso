-- Subscription history (lifecycle-history increment 3). A subscription moves
-- through trialing → active → past_due → paused / canceled, and switches plans
-- (upgrade/downgrade) — mutated from many code paths. As with invoices, a
-- trigger on the subscriptions row captures every status AND plan transition at
-- the source of truth: append-only, no Go/money-path changes, nothing missed.
-- One unified table: change_type distinguishes a 'status' transition (values are
-- status strings) from a 'plan' switch (values are plan ids, resolved to names
-- in the UI). No changed_by — the DB has no request actor.
CREATE TABLE IF NOT EXISTS subscription_history (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    subscription_id UUID NOT NULL,
    change_type     TEXT NOT NULL,   -- 'status' | 'plan'
    from_value      TEXT,            -- null on the creation row
    to_value        TEXT,
    changed_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_subscription_history_sub
    ON subscription_history (subscription_id, changed_at);

-- Records the subscription's status at creation and every subsequent status or
-- plan change. Non-status/plan updates (period rolls, billing dates) run the
-- check but insert nothing, so the common billing path stays cheap.
CREATE OR REPLACE FUNCTION record_subscription_change() RETURNS TRIGGER AS $$
BEGIN
    IF (TG_OP = 'INSERT') THEN
        IF NEW.status IS NOT NULL THEN
            INSERT INTO subscription_history (tenant_id, subscription_id, change_type, from_value, to_value, changed_at)
            VALUES (NEW.tenant_id, NEW.id, 'status', NULL, NEW.status, COALESCE(NEW.created_at, NOW()));
        END IF;
    ELSIF (TG_OP = 'UPDATE') THEN
        IF NEW.status IS DISTINCT FROM OLD.status THEN
            INSERT INTO subscription_history (tenant_id, subscription_id, change_type, from_value, to_value, changed_at)
            VALUES (NEW.tenant_id, NEW.id, 'status', OLD.status, NEW.status, NOW());
        END IF;
        IF NEW.plan_id IS DISTINCT FROM OLD.plan_id THEN
            INSERT INTO subscription_history (tenant_id, subscription_id, change_type, from_value, to_value, changed_at)
            VALUES (NEW.tenant_id, NEW.id, 'plan', OLD.plan_id::text, NEW.plan_id::text, NOW());
        END IF;
    END IF;
    RETURN NULL; -- AFTER trigger: return value is ignored
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS subscription_history_trg ON subscriptions;
CREATE TRIGGER subscription_history_trg
    AFTER INSERT OR UPDATE ON subscriptions
    FOR EACH ROW EXECUTE FUNCTION record_subscription_change();
