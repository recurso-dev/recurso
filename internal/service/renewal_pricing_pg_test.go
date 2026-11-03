package service

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// seedCoupon inserts an active coupon. months is only read for repeating coupons.
func (e *renewalE2E) seedCoupon(discountType string, value int64, duration string, months int) uuid.UUID {
	h := e.h
	h.t.Helper()
	id := uuid.New()
	mustExec(h.t, h.conn, `INSERT INTO coupons (id, tenant_id, code, discount_type, discount_value, duration, duration_months, active, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,NULLIF($7::int, 0),TRUE,NOW(),NOW())`,
		id, h.tenantID, "RENEW-"+id.String()[:8], discountType, value, duration, months)
	return id
}

// seedAddon attaches a new USD or other-currency add-on plan to the subscription.
func (e *renewalE2E) seedAddon(subID uuid.UUID, currency string, amount int64, quantity int) uuid.UUID {
	h := e.h
	h.t.Helper()
	planID := uuid.New()
	mustExec(h.t, h.conn, `INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active) VALUES ($1,$2,'addon',$3,'month',1,TRUE)`,
		planID, h.tenantID, "addon-"+planID.String()[:8])
	mustExec(h.t, h.conn, `INSERT INTO prices (id, plan_id, currency, amount, type) VALUES ($1,$2,$3,$4,'recurring')`,
		uuid.New(), planID, currency, amount)
	addon := &domain.SubscriptionAddon{
		ID: uuid.New(), TenantID: h.tenantID, SubscriptionID: subID, PlanID: planID,
		Quantity: quantity, CreatedAt: time.Now(),
	}
	if err := db.NewSubscriptionAddonRepository(h.conn).Create(h.ctx, addon); err != nil {
		h.t.Fatalf("create add-on: %v", err)
	}
	return addon.ID
}

func (e *renewalE2E) attachCoupon(subID, couponID uuid.UUID, periodsApplied int) {
	e.h.t.Helper()
	mustExec(e.h.t, e.h.conn, `UPDATE subscriptions SET coupon_id = $1, coupon_periods_applied = $2, coupon_applied_current_period = $3 WHERE id = $4`,
		couponID, periodsApplied, periodsApplied > 0, subID)
}

// TestRenewalSweep_CouponsAndAddons_Postgres proves a renewal bills discounts,
// add-ons and one-time charges for exactly the periods they cover, and charges
// the saved card what the invoice says. Each subscription starts two periods
// behind, so consecutive sweeps renew consecutive periods.
func TestRenewalSweep_CouponsAndAddons_Postgres(t *testing.T) {
	conn, dbx := openRenewalE2EDB(t)
	e := newRenewalE2E(t, conn, dbx)
	h := e.h
	const twoBehind = "1 month 1 minute"

	forever := e.seedCoupon("percent", 80, "forever", 0)
	repeating := e.seedCoupon("percent", 50, "repeating", 2)
	once := e.seedCoupon("percent", 50, "once", 0)
	fixed := e.seedCoupon("amount", 30000, "forever", 0)
	inactive := e.seedCoupon("percent", 50, "forever", 0)
	mustExec(t, conn, `UPDATE coupons SET active = FALSE WHERE id = $1`, inactive)

	foreverSub, _ := e.seedDue(twoBehind, false, "cus_forever")
	e.attachCoupon(foreverSub, forever, 1)
	repeatingSub, _ := e.seedDue(twoBehind, false, "cus_repeating")
	e.attachCoupon(repeatingSub, repeating, 1) // one of two periods already used
	onceSub, _ := e.seedDue(twoBehind, false, "cus_once")
	e.attachCoupon(onceSub, once, 1)
	fixedSub, _ := e.seedDue(twoBehind, false, "cus_fixed")
	e.attachCoupon(fixedSub, fixed, 1)
	inactiveSub, _ := e.seedDue(twoBehind, false, "cus_inactive")
	e.attachCoupon(inactiveSub, inactive, 1)

	addonSub, _ := e.seedDue(twoBehind, false, "cus_addon")
	e.seedAddon(addonSub, "USD", 5000, 2)
	removedAddon := e.seedAddon(addonSub, "USD", 700, 1)
	e.seedAddon(addonSub, "EUR", 9900, 1) // cannot sit on a USD invoice

	comboSub, _ := e.seedDue(twoBehind, false, "cus_combo")
	e.attachCoupon(comboSub, forever, 1)
	e.seedAddon(comboSub, "USD", 5000, 2)

	chargeSub, _ := e.seedDue(twoBehind, false, "cus_charge")
	if err := db.NewUnbilledChargeRepository(conn).Create(h.ctx, &domain.UnbilledCharge{
		ID: uuid.New(), SubscriptionID: chargeSub, Amount: 2500, Currency: "USD",
		Description: "Setup", Status: domain.UnbilledChargeStatusPending, CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("create one-time charge: %v", err)
	}

	declinedSub, _ := e.seedDue("1 minute", false, "cus_declined")
	e.attachCoupon(declinedSub, forever, 1)
	e.charger.declined["cus_declined"] = true

	e.sweep()
	h.assertAuditGrade("after first sweep")
	if err := db.NewSubscriptionAddonRepository(conn).Delete(h.ctx, h.tenantID, removedAddon); err != nil {
		t.Fatalf("remove add-on: %v", err)
	}
	e.sweep()
	h.assertAuditGrade("after second sweep")

	billed := func(t *testing.T, subID uuid.UUID, card string, want []int64) {
		t.Helper()
		got := e.invoices(subID)
		if !equalInt64s(invoiceTotals(got), want) {
			t.Fatalf("totals = %v, want %v", invoiceTotals(got), want)
		}
		for _, inv := range got {
			if inv.status != string(domain.InvoiceStatusPaid) || inv.paid != inv.total {
				t.Errorf("invoice %+v is not fully paid", inv)
			}
		}
		if charged := e.charger.chargesOf(card); !equalInt64s(charged, want) {
			t.Errorf("charged %v, want %v", charged, want)
		}
	}
	progress := func(t *testing.T, subID uuid.UUID) (periods int, applied bool) {
		t.Helper()
		if err := conn.QueryRowContext(h.ctx,
			`SELECT coupon_periods_applied, coupon_applied_current_period FROM subscriptions WHERE id = $1`, subID).Scan(&periods, &applied); err != nil {
			t.Fatalf("read coupon progress: %v", err)
		}
		return periods, applied
	}

	t.Run("a forever coupon discounts every renewal", func(t *testing.T) {
		billed(t, foreverSub, "cus_forever", []int64{20000, 20000})
		if periods, applied := progress(t, foreverSub); periods != 3 || !applied {
			t.Errorf("periods = %d applied = %v, want 3 and true", periods, applied)
		}
	})
	t.Run("a repeating coupon stops after its periods", func(t *testing.T) {
		billed(t, repeatingSub, "cus_repeating", []int64{50000, 100000})
		if periods, applied := progress(t, repeatingSub); periods != 2 || applied {
			t.Errorf("periods = %d applied = %v, want 2 and false", periods, applied)
		}
	})
	t.Run("a once coupon never reaches a renewal", func(t *testing.T) {
		billed(t, onceSub, "cus_once", []int64{100000, 100000})
	})
	t.Run("a fixed amount coupon discounts every renewal", func(t *testing.T) {
		billed(t, fixedSub, "cus_fixed", []int64{70000, 70000})
	})
	t.Run("an inactive coupon discounts nothing", func(t *testing.T) {
		billed(t, inactiveSub, "cus_inactive", []int64{100000, 100000})
	})
	t.Run("add-ons bill at quantity until removed, and only in the invoice currency", func(t *testing.T) {
		billed(t, addonSub, "cus_addon", []int64{110700, 110000})
	})
	t.Run("a coupon discounts the plan fee and not its add-ons", func(t *testing.T) {
		billed(t, comboSub, "cus_combo", []int64{30000, 30000})
	})
	t.Run("a one-time charge bills once", func(t *testing.T) {
		billed(t, chargeSub, "cus_charge", []int64{102500, 100000})
	})
	t.Run("a decline on a discounted renewal leaves the discounted amount owing", func(t *testing.T) {
		got := e.invoices(declinedSub)
		if len(got) != 1 || got[0].status != string(domain.InvoiceStatusOpen) || got[0].total != 20000 || got[0].paid != 0 {
			t.Fatalf("invoices = %+v, want one open invoice of 20000", got)
		}
	})

	h.assertClosePackTies("after discounted renewals")
}

// TestRenewalSweep_FullDiscount_Postgres proves a renewal a coupon reduces to
// zero owes nothing: no charge, and no open invoice left for dunning to chase.
func TestRenewalSweep_FullDiscount_Postgres(t *testing.T) {
	conn, dbx := openRenewalE2EDB(t)
	e := newRenewalE2E(t, conn, dbx)
	h := e.h

	free := e.seedCoupon("percent", 100, "forever", 0)
	subID, _ := e.seedDue("1 minute", false, "cus_free")
	e.attachCoupon(subID, free, 1)

	e.sweep()
	h.assertAuditGrade("after free renewal")

	got := e.invoices(subID)
	if len(got) != 1 || got[0].total != 0 || got[0].status != string(domain.InvoiceStatusPaid) {
		t.Fatalf("invoices = %+v, want one paid invoice of 0", got)
	}
	if charged := e.charger.chargesOf("cus_free"); len(charged) != 0 {
		t.Errorf("charged %v, want nothing", charged)
	}
	if status, due := e.subState(subID); status != string(domain.SubscriptionStatusActive) || due {
		t.Errorf("status = %s due = %v, want active and renewed", status, due)
	}

	var code string
	if err := conn.QueryRowContext(h.ctx, `SELECT code FROM coupons WHERE id = $1`, free).Scan(&code); err != nil {
		t.Fatalf("read coupon code: %v", err)
	}
	_, customerID := e.seedDue("-15 days", false, "")

	t.Run("the first invoice of a free subscription is settled", func(t *testing.T) {
		sub, err := h.subSvc.CreateSubscription(h.tctx, CreateSubscriptionInput{
			TenantID: h.tenantID, CustomerID: customerID, PlanID: h.cheapPlan, CouponCode: code,
		})
		if err != nil {
			t.Fatalf("create subscription: %v", err)
		}
		if got := e.invoices(sub.ID); len(got) != 1 || got[0].total != 0 || got[0].status != string(domain.InvoiceStatusPaid) {
			t.Fatalf("invoices = %+v, want one paid invoice of 0", got)
		}
	})

	t.Run("the first invoice after a free trial is settled", func(t *testing.T) {
		sub, err := h.subSvc.CreateSubscription(h.tctx, CreateSubscriptionInput{
			TenantID: h.tenantID, CustomerID: customerID, PlanID: h.cheapPlan, CouponCode: code, TrialDays: 7,
		})
		if err != nil {
			t.Fatalf("create trial: %v", err)
		}
		if _, err := h.subSvc.ConvertTrialToActive(h.tctx, sub); err != nil {
			t.Fatalf("convert trial: %v", err)
		}
		if got := e.invoices(sub.ID); len(got) != 1 || got[0].total != 0 || got[0].status != string(domain.InvoiceStatusPaid) {
			t.Fatalf("invoices = %+v, want one paid invoice of 0", got)
		}
	})

	t.Run("no zero-total invoice is left for dunning", func(t *testing.T) {
		var open int
		if err := conn.QueryRowContext(h.ctx,
			`SELECT COUNT(*) FROM invoices WHERE tenant_id = $1 AND total = 0 AND status IN ('open', 'past_due')`, h.tenantID).Scan(&open); err != nil {
			t.Fatalf("count open invoices: %v", err)
		}
		if open != 0 {
			t.Errorf("%d zero-total invoices are open", open)
		}
	})

	h.assertAuditGrade("after free subscriptions")
	h.assertClosePackTies("after free renewal")
}

// TestRenewalSweep_ProgressiveBilling_Postgres proves interim invoices and the
// period-close invoice together bill each unit of usage exactly once.
func TestRenewalSweep_ProgressiveBilling_Postgres(t *testing.T) {
	conn, dbx := openRenewalE2EDB(t)
	e := newRenewalE2E(t, conn, dbx)
	h := e.h
	e.invoice.SetProgressiveBilling(db.NewProgressiveBillingRepository(conn), h.ledger)
	h.subSvc.SetFinalUsageInvoicer(e.invoice)
	h.subSvc.SetFinalInvoiceCharger(e.svc)

	const dimension = "api_calls"
	planID := e.seedMeteredPlan(dimension)
	usage := func(subID, customerID uuid.UUID, at time.Time, qty int64) {
		t.Helper()
		if err := db.NewUsageRepository(conn).RecordEvent(h.ctx, &domain.UsageEvent{
			ID: uuid.New(), SubscriptionID: subID, CustomerID: customerID,
			Dimension: dimension, Quantity: qty, Timestamp: at,
		}); err != nil {
			t.Fatalf("record usage: %v", err)
		}
	}
	progressive := func(overdue, card string) (uuid.UUID, uuid.UUID) {
		t.Helper()
		subID, customerID := e.seedDueOnPlan(planID, overdue, false, card)
		mustExec(t, conn, `UPDATE subscriptions SET progressive_billing_threshold = 1000 WHERE id = $1`, subID)
		return subID, customerID
	}
	interim := func(subID uuid.UUID) *domain.Invoice {
		t.Helper()
		inv, err := e.invoice.GenerateProgressiveInvoiceForSub(h.tctx, subID)
		if err != nil {
			t.Fatalf("interim billing: %v", err)
		}
		return inv
	}
	endPeriod := func(subID uuid.UUID) {
		t.Helper()
		mustExec(t, conn, `UPDATE subscriptions SET current_period_end = NOW() - INTERVAL '1 minute' WHERE id = $1`, subID)
	}

	t.Run("below the threshold raises no interim invoice", func(t *testing.T) {
		subID, customerID := progressive("-15 days", "")
		usage(subID, customerID, time.Now().Add(-time.Hour), 19) // 950 < 1000
		if inv := interim(subID); inv != nil {
			t.Fatalf("interim = %+v, want none", inv)
		}
		if got := e.invoices(subID); len(got) != 0 {
			t.Errorf("invoices = %+v, want none", got)
		}
	})

	t.Run("interims and the close bill each unit once", func(t *testing.T) {
		subID, customerID := progressive("-15 days", "cus_progressive")
		usage(subID, customerID, time.Now().Add(-2*time.Hour), 100)
		if inv := interim(subID); inv == nil || inv.Total != 5000 {
			t.Fatalf("first interim = %+v, want 5000", inv)
		}
		if inv := interim(subID); inv != nil {
			t.Fatalf("repeat interim = %+v, want none", inv)
		}
		usage(subID, customerID, time.Now().Add(-time.Hour), 60)
		if inv := interim(subID); inv == nil || inv.Total != 3000 {
			t.Fatalf("second interim = %+v, want 3000", inv)
		}
		usage(subID, customerID, time.Now().Add(-30*time.Minute), 10)
		endPeriod(subID)
		e.sweep()
		h.assertAuditGrade("after progressive close")
		if got := invoiceTotals(e.invoices(subID)); !equalInt64s(got, []int64{5000, 3000, 100500}) {
			t.Fatalf("totals = %v, want interims 5000 and 3000, then flat 100000 + final 500", got)
		}
		if _, due := e.subState(subID); due {
			t.Error("period did not advance")
		}
	})

	t.Run("a retry after a failed period advance does not bill the delta again", func(t *testing.T) {
		subID, customerID := progressive("-15 days", "")
		usage(subID, customerID, time.Now().Add(-time.Hour), 100)
		if inv := interim(subID); inv == nil {
			t.Fatal("no interim invoice")
		}
		usage(subID, customerID, time.Now().Add(-30*time.Minute), 10)
		endPeriod(subID)
		e.claims.failUpdate[subID] = true
		e.sweep()
		e.sweep()
		h.assertAuditGrade("after progressive retry")
		if got := invoiceTotals(e.invoices(subID)); !equalInt64s(got, []int64{5000, 100500}) {
			t.Fatalf("totals = %v, want interim 5000, then one close invoice of 100500", got)
		}
	})

	t.Run("an interim run after the period ended leaves next period's usage alone", func(t *testing.T) {
		subID, customerID := progressive("1 hour", "")
		_, end := e.period(subID)
		usage(subID, customerID, end.Add(-time.Hour), 100)
		usage(subID, customerID, end.Add(10*time.Minute), 40) // belongs to the next period

		// The hourly interim sweep reaches a due subscription before its renewal.
		if inv := interim(subID); inv == nil || inv.Total != 5000 {
			t.Fatalf("interim = %+v, want 5000 for the closed period only", inv)
		}
		e.sweep()
		endPeriod(subID)
		e.sweep()
		h.assertAuditGrade("after late interim")

		// 140 units at 50 is 7000 of usage, plus two flat fees.
		var sum int64
		for _, total := range invoiceTotals(e.invoices(subID)) {
			sum += total
		}
		if sum != 207000 {
			t.Fatalf("billed %d across %v, want 207000: usage billed exactly once", sum, invoiceTotals(e.invoices(subID)))
		}
	})

	t.Run("a period-end cancel settles only the unbilled usage", func(t *testing.T) {
		subID, customerID := e.seedDueOnPlan(planID, "-15 days", true, "cus_progressive_cancel")
		mustExec(t, conn, `UPDATE subscriptions SET progressive_billing_threshold = 1000 WHERE id = $1`, subID)
		usage(subID, customerID, time.Now().Add(-2*time.Hour), 100)
		if inv := interim(subID); inv == nil {
			t.Fatal("no interim invoice")
		}
		usage(subID, customerID, time.Now().Add(-time.Hour), 40)
		endPeriod(subID)
		e.sweep()
		h.assertAuditGrade("after progressive period-end cancel")
		if got := invoiceTotals(e.invoices(subID)); !equalInt64s(got, []int64{5000, 2000}) {
			t.Fatalf("totals = %v, want interim 5000 and final 2000", got)
		}
		if status, _ := e.subState(subID); status != string(domain.SubscriptionStatusCanceled) {
			t.Errorf("status = %s, want canceled", status)
		}
	})

	t.Run("an immediate cancel settles only the unbilled usage", func(t *testing.T) {
		subID, customerID := progressive("-15 days", "cus_progressive_now")
		usage(subID, customerID, time.Now().Add(-2*time.Hour), 100)
		if inv := interim(subID); inv == nil {
			t.Fatal("no interim invoice")
		}
		usage(subID, customerID, time.Now().Add(-time.Hour), 40)
		if _, err := h.subSvc.Cancel(h.tctx, h.tenantID, subID, true, "", ""); err != nil {
			t.Fatalf("cancel: %v", err)
		}
		h.assertAuditGrade("after progressive immediate cancel")
		if got := invoiceTotals(e.invoices(subID)); !equalInt64s(got, []int64{5000, 2000}) {
			t.Fatalf("totals = %v, want interim 5000 and final 2000", got)
		}
	})

	t.Run("a declined close invoice stays open without touching paid interims", func(t *testing.T) {
		subID, customerID := progressive("-15 days", "cus_progressive_declines")
		usage(subID, customerID, time.Now().Add(-time.Hour), 100)
		if inv := interim(subID); inv == nil {
			t.Fatal("no interim invoice")
		}
		e.charger.declined["cus_progressive_declines"] = true
		endPeriod(subID)
		e.sweep()
		got := e.invoices(subID)
		if len(got) != 2 || got[1].status != string(domain.InvoiceStatusOpen) || got[1].total != 100000 {
			t.Fatalf("invoices = %+v, want the interim and an open close invoice of 100000", got)
		}
	})

	h.assertAuditGrade("after progressive renewals")
	h.assertClosePackTies("after progressive renewals")
}
