-- ENG-191f: revrec_schedule_debt tracks how much of a subscription's
-- UNSCHEDULED deferral a mid-period downgrade credit has already consumed
-- (DR Deferred / CR Customer-Credit posted against deferral that had no
-- recognition schedule yet — e.g. an unpaid upgrade-charge invoice).
-- When such an invoice is later paid, CreateScheduleForInvoice reduces the new
-- schedule by this debt: without it the schedule would recognize revenue whose
-- deferral was already credited back, over-draining Deferred (the
-- deferred_below_scheduled_revenue reconciler finding) and double-counting the
-- credited service as revenue.
ALTER TABLE subscriptions
    ADD COLUMN IF NOT EXISTS revrec_schedule_debt BIGINT NOT NULL DEFAULT 0;
