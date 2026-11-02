package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// tenantClaims runs the real claim query but hands back only this test's
// rows, so a sweep on a shared database never renews another test's
// subscriptions. failUpdate fails the listed subscription's next period
// advance, the crash window between invoicing and advancing.
type tenantClaims struct {
	*db.SubscriptionRepository
	tenantID   uuid.UUID
	failUpdate map[uuid.UUID]bool
}

func (c tenantClaims) Update(ctx context.Context, sub *domain.Subscription) error {
	if c.failUpdate[sub.ID] {
		delete(c.failUpdate, sub.ID)
		return errors.New("injected period advance failure")
	}
	return c.SubscriptionRepository.Update(ctx, sub)
}

func (c tenantClaims) ClaimDueForRenewal(ctx context.Context, lease time.Duration, limit int) ([]*domain.Subscription, error) {
	subs, err := c.SubscriptionRepository.ClaimDueForRenewal(ctx, lease, limit)
	if err != nil {
		return nil, err
	}
	var own []*domain.Subscription
	for _, s := range subs {
		if s.TenantID == c.tenantID {
			own = append(own, s)
		}
	}
	return own, nil
}

// scriptedCharger records saved-card charges and declines the listed customers.
type scriptedCharger struct {
	declined map[string]bool
	charges  []string // idempotency keys, in order
	amounts  map[string][]int64
}

func (c *scriptedCharger) chargesOf(stripeCustomerID string) []int64 {
	return c.amounts[stripeCustomerID]
}

func (c *scriptedCharger) ChargeSavedPaymentMethod(_ context.Context, stripeCustomerID, _ string, amount int64, _, _, idempotencyKey string) (*port.PaymentResult, error) {
	c.charges = append(c.charges, idempotencyKey)
	c.amounts[stripeCustomerID] = append(c.amounts[stripeCustomerID], amount)
	return &port.PaymentResult{Success: !c.declined[stripeCustomerID]}, nil
}

// renewalE2E wires the renewal engine as cmd/api does (real repositories,
// invoice service, ledger, rev-rec and settlement) with only the gateway faked.
type renewalE2E struct {
	h       *invariantHarness
	svc     *RenewalService
	invoice *InvoiceService
	charger *scriptedCharger
	claims  tenantClaims
}

func newRenewalE2E(t *testing.T, conn *sql.DB, dbx *sqlx.DB) *renewalE2E {
	t.Helper()
	h := newInvariantHarness(t, conn, dbx)

	subRepo := db.NewSubscriptionRepository(conn)
	planRepo := db.NewPlanRepository(conn)
	customerRepo := db.NewCustomerRepository(dbx)
	invoices := NewInvoiceService(db.NewInvoiceRepository(conn), planRepo, customerRepo,
		db.NewUnbilledChargeRepository(conn), subRepo, nil, NewTaxResolver(harnessGSTConfigs{}, "United States", ""))
	invoices.CouponRepo = db.NewCouponRepository(conn)
	invoices.AddonRepo = db.NewSubscriptionAddonRepository(conn)
	invoices.ChargeRepo = db.NewChargeRepository(conn)
	invoices.UsageRepo = db.NewUsageRepository(conn)
	invoices.RatingRepo = db.NewUsageRatingRepository(conn)
	invoices.LedgerPoster = h.ledger
	invoices.RevRecScheduler = h.revrec

	charger := &scriptedCharger{declined: map[string]bool{}, amounts: map[string][]int64{}}
	claims := tenantClaims{subRepo.(*db.SubscriptionRepository), h.tenantID, map[uuid.UUID]bool{}}
	svc := NewRenewalService(claims, planRepo, invoices)
	svc.SetSavedMethodCharging(charger, customerRepo, h.subSvc)
	return &renewalE2E{h: h, svc: svc, invoice: invoices, charger: charger, claims: claims}
}

// seedDue inserts a customer and an active subscription whose period ended
// `overdue` ago. A non-empty card saves a payment method on the customer.
func (e *renewalE2E) seedDue(overdue string, cancelAtPeriodEnd bool, card string) (subID, customerID uuid.UUID) {
	return e.seedDueOnPlan(e.h.cheapPlan, overdue, cancelAtPeriodEnd, card)
}

func (e *renewalE2E) seedDueOnPlan(planID uuid.UUID, overdue string, cancelAtPeriodEnd bool, card string) (subID, customerID uuid.UUID) {
	h := e.h
	h.t.Helper()
	h.n++
	customerID, subID = uuid.New(), uuid.New()
	var stripeID, method any
	if card != "" {
		stripeID, method = card, "pm_"+card
	}
	mustExec(h.t, h.conn, `INSERT INTO customers (id, tenant_id, email, name, country, tax_type, ledger_account_id, stripe_customer_id, default_payment_method, created_at, updated_at)
		VALUES ($1,$2,$3,'Renewal Cust','United States','individual',$4,$5,$6,NOW(),NOW())`,
		customerID, h.tenantID, fmt.Sprintf("renew-%s-%d@t.com", h.run, h.n), uuid.New(), stripeID, method)
	mustExec(h.t, h.conn, `INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, status, current_period_start, current_period_end, billing_anchor, cancel_at_period_end, created_at, updated_at)
		VALUES ($1,$2,$3,$4,'active', NOW() - INTERVAL '1 month' - $5::interval, NOW() - $5::interval, NOW() - INTERVAL '1 month' - $5::interval, $6, NOW(), NOW())`,
		subID, h.tenantID, customerID, planID, overdue, cancelAtPeriodEnd)
	return subID, customerID
}

// sweep calls ProcessDueRenewals exactly as BillingCycleScheduler does: on
// context.Background(), with no tenant. Leases are cleared first so a due row
// is re-claimed the way it would be once its lease lapses.
func (e *renewalE2E) sweep() {
	h := e.h
	h.t.Helper()
	mustExec(h.t, h.conn, `UPDATE subscriptions SET renewal_claimed_at = NULL WHERE tenant_id = $1`, h.tenantID)
	if _, err := e.svc.ProcessDueRenewals(context.Background()); err != nil {
		h.t.Fatalf("ProcessDueRenewals: %v", err)
	}
}

type renewalInvoiceRow struct {
	status string
	total  int64
	paid   int64
}

func (e *renewalE2E) invoices(subID uuid.UUID) []renewalInvoiceRow {
	h := e.h
	h.t.Helper()
	rows, err := h.conn.QueryContext(h.ctx,
		`SELECT status, total, amount_paid FROM invoices WHERE subscription_id = $1 ORDER BY created_at`, subID)
	if err != nil {
		h.t.Fatalf("list invoices: %v", err)
	}
	defer func() { _ = rows.Close() }()
	var out []renewalInvoiceRow
	for rows.Next() {
		var r renewalInvoiceRow
		if err := rows.Scan(&r.status, &r.total, &r.paid); err != nil {
			h.t.Fatalf("scan invoice: %v", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		h.t.Fatalf("iterate invoices: %v", err)
	}
	return out
}

func (e *renewalE2E) subState(subID uuid.UUID) (status string, due bool) {
	h := e.h
	h.t.Helper()
	if err := h.conn.QueryRowContext(h.ctx,
		`SELECT status, current_period_end <= NOW() FROM subscriptions WHERE id = $1`, subID).Scan(&status, &due); err != nil {
		h.t.Fatalf("read subscription: %v", err)
	}
	return status, due
}

func openRenewalE2EDB(t *testing.T) (*sql.DB, *sqlx.DB) {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed renewal e2e test")
	}
	if err := db.RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn, sqlx.NewDb(conn, "postgres")
}

// TestRenewalSweep_EndToEnd_Postgres drives scheduled renewals through the real
// repositories, invoice service, ledger and settlement. It exists because the
// sweep's tenant-less context once failed every scheduled renewal, which unit
// tests on fakes could not see: the failure lives in the tenant-scoped plan and
// invoice lookups. Every outcome must leave the ledger audit-grade.
func TestRenewalSweep_EndToEnd_Postgres(t *testing.T) {
	conn, dbx := openRenewalE2EDB(t)

	const planAmount = int64(100000)
	e := newRenewalE2E(t, conn, dbx)

	paidSub, _ := e.seedDue("1 minute", false, "cus_pays")
	openSub, _ := e.seedDue("1 minute", false, "")
	declinedSub, _ := e.seedDue("1 minute", false, "cus_declines")
	e.charger.declined["cus_declines"] = true
	cancelSub, _ := e.seedDue("1 minute", true, "cus_cancels")
	behindSub, _ := e.seedDue("1 month 1 minute", false, "")

	e.sweep()
	e.h.assertAuditGrade("after first sweep")

	t.Run("saved card is charged and the invoice settles", func(t *testing.T) {
		got := e.invoices(paidSub)
		if len(got) != 1 || got[0].status != string(domain.InvoiceStatusPaid) || got[0].total != planAmount || got[0].paid != planAmount {
			t.Fatalf("invoices = %+v, want one paid invoice of %d", got, planAmount)
		}
		if status, due := e.subState(paidSub); status != "active" || due {
			t.Errorf("subscription status=%s due=%v, want active and no longer due", status, due)
		}
	})

	t.Run("no saved card leaves the invoice open", func(t *testing.T) {
		got := e.invoices(openSub)
		if len(got) != 1 || got[0].status != string(domain.InvoiceStatusOpen) || got[0].paid != 0 {
			t.Fatalf("invoices = %+v, want one open unpaid invoice", got)
		}
		if _, due := e.subState(openSub); due {
			t.Error("period did not advance")
		}
	})

	t.Run("declined card leaves the invoice open for dunning", func(t *testing.T) {
		got := e.invoices(declinedSub)
		if len(got) != 1 || got[0].status != string(domain.InvoiceStatusOpen) || got[0].paid != 0 {
			t.Fatalf("invoices = %+v, want one open unpaid invoice", got)
		}
		if _, due := e.subState(declinedSub); due {
			t.Error("period did not advance")
		}
	})

	t.Run("cancel at period end cancels without a flat fee or charge", func(t *testing.T) {
		for _, inv := range e.invoices(cancelSub) {
			if inv.total >= planAmount {
				t.Errorf("final invoice %+v carries the flat fee for a period that will not run", inv)
			}
		}
		if status, _ := e.subState(cancelSub); status != string(domain.SubscriptionStatusCanceled) {
			t.Errorf("status = %s, want canceled", status)
		}
	})

	t.Run("a subscription behind by a cycle catches up one invoice per sweep", func(t *testing.T) {
		if got := e.invoices(behindSub); len(got) != 1 {
			t.Fatalf("after first sweep: %d invoices, want 1", len(got))
		}
		if _, due := e.subState(behindSub); !due {
			t.Fatal("after first sweep: want still due")
		}
	})

	if len(e.charger.charges) != 2 {
		t.Errorf("charges = %v, want exactly the paying and declining customers", e.charger.charges)
	}

	// The next sweep may only bill the subscription still behind; everything
	// else is undue, so a repeat must neither re-invoice nor re-charge.
	e.sweep()
	e.h.assertAuditGrade("after second sweep")

	if got := e.invoices(behindSub); len(got) != 2 {
		t.Errorf("behind subscription: %d invoices after catch-up, want 2", len(got))
	}
	if _, due := e.subState(behindSub); due {
		t.Error("behind subscription: still due after catch-up")
	}
	for name, id := range map[string]uuid.UUID{"paid": paidSub, "open": openSub, "declined": declinedSub} {
		if got := e.invoices(id); len(got) != 1 {
			t.Errorf("%s subscription re-invoiced on repeat sweep: %+v", name, got)
		}
	}
	if len(e.charger.charges) != 2 {
		t.Errorf("repeat sweep charged again: %v", e.charger.charges)
	}

	e.h.assertClosePackTies("after renewals")
}

// seedMeteredPlan creates a hybrid plan: a 100000 USD flat fee plus a per-unit
// charge of 50 minor units on a summed metric.
func (e *renewalE2E) seedMeteredPlan(dimension string) uuid.UUID {
	h := e.h
	h.t.Helper()
	now := time.Now().UTC()
	planID := uuid.New()
	mustExec(h.t, h.conn, `INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active) VALUES ($1,$2,'metered',$3,'month',1,TRUE)`,
		planID, h.tenantID, "metered-"+h.run)
	mustExec(h.t, h.conn, `INSERT INTO prices (id, plan_id, currency, amount, type) VALUES ($1,$2,'USD',100000,'recurring')`,
		uuid.New(), planID)
	metric := &domain.BillableMetric{
		ID: uuid.New(), TenantID: h.tenantID, Name: "API calls", Code: dimension,
		AggregationType: domain.AggregationSum, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.NewBillableMetricRepository(h.conn).Create(h.ctx, metric); err != nil {
		h.t.Fatalf("create metric: %v", err)
	}
	charge := domain.Charge{
		ID: uuid.New(), TenantID: h.tenantID, PlanID: planID, MetricID: metric.ID,
		ChargeModel: domain.ChargePerUnit,
		Amounts:     map[string]domain.ChargeAmounts{"USD": {UnitAmount: "0.50"}},
		CreatedAt:   now, UpdatedAt: now,
	}
	if err := db.NewChargeRepository(h.conn).ReplaceForPlan(h.ctx, h.tenantID, planID, []domain.Charge{charge}); err != nil {
		h.t.Fatalf("create charge: %v", err)
	}
	return planID
}

func (e *renewalE2E) period(subID uuid.UUID) (start, end time.Time) {
	h := e.h
	h.t.Helper()
	if err := h.conn.QueryRowContext(h.ctx,
		`SELECT current_period_start, current_period_end FROM subscriptions WHERE id = $1`, subID).Scan(&start, &end); err != nil {
		h.t.Fatalf("read period: %v", err)
	}
	return start, end
}

func (e *renewalE2E) ratings(subID uuid.UUID) (out [][2]int64) {
	h := e.h
	h.t.Helper()
	rows, err := h.conn.QueryContext(h.ctx,
		`SELECT quantity, amount FROM usage_ratings WHERE subscription_id = $1 ORDER BY period_start`, subID)
	if err != nil {
		h.t.Fatalf("list ratings: %v", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var qty, amount int64
		if err := rows.Scan(&qty, &amount); err != nil {
			h.t.Fatalf("scan rating: %v", err)
		}
		out = append(out, [2]int64{qty, amount})
	}
	if err := rows.Err(); err != nil {
		h.t.Fatalf("iterate ratings: %v", err)
	}
	return out
}

func invoiceTotals(rows []renewalInvoiceRow) []int64 {
	out := make([]int64, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.total)
	}
	return out
}

func equalInt64s(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestRenewalSweep_MeteredUsage_Postgres proves usage reaches renewal invoices
// through the real aggregation and rating: the billed window is the closed
// period [start, end), each window is rated once, and every outcome leaves the
// ledger audit-grade.
func TestRenewalSweep_MeteredUsage_Postgres(t *testing.T) {
	conn, dbx := openRenewalE2EDB(t)
	e := newRenewalE2E(t, conn, dbx)
	h := e.h

	const dimension = "api_calls"
	planID := e.seedMeteredPlan(dimension)
	usage := func(subID, customerID uuid.UUID, at time.Time, qty int64) {
		t.Helper()
		err := db.NewUsageRepository(conn).RecordEvent(h.ctx, &domain.UsageEvent{
			ID: uuid.New(), SubscriptionID: subID, CustomerID: customerID,
			Dimension: dimension, Quantity: qty, Timestamp: at,
		})
		if err != nil {
			t.Fatalf("record usage: %v", err)
		}
	}

	windowSub, windowCust := e.seedDueOnPlan(planID, "1 minute", false, "")
	start, end := e.period(windowSub)
	usage(windowSub, windowCust, start, 60)                  // first instant of the window: billed
	usage(windowSub, windowCust, end.Add(-time.Second), 40)  // inside: billed
	usage(windowSub, windowCust, start.Add(-time.Second), 7) // previous period: excluded
	usage(windowSub, windowCust, end, 9)                     // next period: excluded

	paidSub, paidCust := e.seedDueOnPlan(planID, "1 minute", false, "cus_metered_pays")
	_, end = e.period(paidSub)
	usage(paidSub, paidCust, end.Add(-time.Hour), 10)

	idleSub, _ := e.seedDueOnPlan(planID, "1 minute", false, "")

	behindSub, behindCust := e.seedDueOnPlan(planID, "1 month 1 minute", false, "")
	_, end = e.period(behindSub)
	usage(behindSub, behindCust, end.Add(-24*time.Hour), 20)
	usage(behindSub, behindCust, end.Add(24*time.Hour), 30)

	cancelSub, cancelCust := e.seedDueOnPlan(planID, "1 minute", true, "cus_metered_cancels")
	_, end = e.period(cancelSub)
	usage(cancelSub, cancelCust, end.Add(-time.Hour), 40)

	declinedCancelSub, declinedCancelCust := e.seedDueOnPlan(planID, "1 minute", true, "cus_metered_cancel_declines")
	_, end = e.period(declinedCancelSub)
	usage(declinedCancelSub, declinedCancelCust, end.Add(-time.Hour), 40)
	e.charger.declined["cus_metered_cancel_declines"] = true

	idleCancelSub, _ := e.seedDueOnPlan(planID, "1 minute", true, "cus_metered_cancel_idle")

	crashSub, crashCust := e.seedDueOnPlan(planID, "1 minute", false, "")
	_, end = e.period(crashSub)
	usage(crashSub, crashCust, end.Add(-time.Hour), 100)
	e.claims.failUpdate[crashSub] = true

	e.sweep()
	h.assertAuditGrade("after first metered sweep")

	t.Run("only usage inside the closed period is billed", func(t *testing.T) {
		if got := invoiceTotals(e.invoices(windowSub)); !equalInt64s(got, []int64{105000}) {
			t.Fatalf("totals = %v, want flat 100000 + 100 units x 50 = 105000", got)
		}
		if got := e.ratings(windowSub); len(got) != 1 || got[0] != [2]int64{100, 5000} {
			t.Errorf("ratings = %v, want one of 100 units / 5000", got)
		}
	})

	t.Run("the saved card is charged for flat fee plus usage", func(t *testing.T) {
		got := e.invoices(paidSub)
		if len(got) != 1 || got[0].status != string(domain.InvoiceStatusPaid) || got[0].total != 100500 || got[0].paid != 100500 {
			t.Fatalf("invoices = %+v, want one paid invoice of 100500", got)
		}
		if got := e.charger.chargesOf("cus_metered_pays"); !equalInt64s(got, []int64{100500}) {
			t.Errorf("charged %v, want exactly 100500", got)
		}
	})

	t.Run("no usage bills the flat fee alone", func(t *testing.T) {
		if got := invoiceTotals(e.invoices(idleSub)); !equalInt64s(got, []int64{100000}) {
			t.Fatalf("totals = %v, want 100000", got)
		}
		if got := e.ratings(idleSub); len(got) != 0 {
			t.Errorf("ratings = %v, want none", got)
		}
	})

	t.Run("cancel at period end charges the saved card for usage without the flat fee", func(t *testing.T) {
		got := e.invoices(cancelSub)
		if len(got) != 1 || got[0].status != string(domain.InvoiceStatusPaid) || got[0].total != 2000 || got[0].paid != 2000 {
			t.Fatalf("invoices = %+v, want one paid final usage invoice: 40 units x 50 = 2000", got)
		}
		if got := e.charger.chargesOf("cus_metered_cancels"); !equalInt64s(got, []int64{2000}) {
			t.Errorf("charged %v, want exactly 2000", got)
		}
		if status, _ := e.subState(cancelSub); status != string(domain.SubscriptionStatusCanceled) {
			t.Errorf("status = %s, want canceled", status)
		}
		if got := e.ratings(cancelSub); len(got) != 1 || got[0] != [2]int64{40, 2000} {
			t.Errorf("ratings = %v, want one of 40 units / 2000", got)
		}
	})

	t.Run("a declined final charge leaves the invoice open and the subscription canceled", func(t *testing.T) {
		got := e.invoices(declinedCancelSub)
		if len(got) != 1 || got[0].status != string(domain.InvoiceStatusOpen) || got[0].paid != 0 {
			t.Fatalf("invoices = %+v, want one open unpaid invoice", got)
		}
		if status, _ := e.subState(declinedCancelSub); status != string(domain.SubscriptionStatusCanceled) {
			t.Errorf("status = %s, want canceled", status)
		}
	})

	t.Run("cancel at period end with no usage raises no invoice or charge", func(t *testing.T) {
		if got := e.invoices(idleCancelSub); len(got) != 0 {
			t.Fatalf("invoices = %+v, want none", got)
		}
		if got := e.charger.chargesOf("cus_metered_cancel_idle"); len(got) != 0 {
			t.Errorf("charged %v, want nothing", got)
		}
		if status, _ := e.subState(idleCancelSub); status != string(domain.SubscriptionStatusCanceled) {
			t.Errorf("status = %s, want canceled", status)
		}
	})

	e.sweep()
	h.assertAuditGrade("after second metered sweep")

	t.Run("a catch-up bills each window its own usage", func(t *testing.T) {
		if got := invoiceTotals(e.invoices(behindSub)); !equalInt64s(got, []int64{101000, 101500}) {
			t.Fatalf("totals = %v, want 20 units then 30 units on separate invoices", got)
		}
		if got := e.ratings(behindSub); len(got) != 2 || got[0] != [2]int64{20, 1000} || got[1] != [2]int64{30, 1500} {
			t.Errorf("ratings = %v", got)
		}
	})

	t.Run("a retry after a failed period advance bills neither flat fee nor usage again", func(t *testing.T) {
		if got := invoiceTotals(e.invoices(crashSub)); !equalInt64s(got, []int64{105000}) {
			t.Fatalf("totals = %v, want one invoice of 105000", got)
		}
		if got := e.ratings(crashSub); len(got) != 1 || got[0] != [2]int64{100, 5000} {
			t.Errorf("ratings = %v, want one of 100 units / 5000", got)
		}
		if _, due := e.subState(crashSub); due {
			t.Error("period did not advance on retry")
		}
	})

	h.assertClosePackTies("after metered renewals")
}

// TestRenewalSweep_RetryReusesPeriodInvoice_Postgres proves a billing period gets
// one renewal invoice however many times its renewal is attempted.
func TestRenewalSweep_RetryReusesPeriodInvoice_Postgres(t *testing.T) {
	conn, dbx := openRenewalE2EDB(t)
	e := newRenewalE2E(t, conn, dbx)
	h := e.h

	cardSub, _ := e.seedDue("1 minute", false, "cus_retry_pays")
	e.claims.failUpdate[cardSub] = true

	couponSub, _ := e.seedDue("1 minute", false, "")
	mustExec(t, conn, `UPDATE subscriptions SET coupon_id = $1, coupon_periods_applied = 1, coupon_applied_current_period = TRUE WHERE id = $2`,
		h.couponID, couponSub)
	e.claims.failUpdate[couponSub] = true

	couponProgress := func() (periods int, applied bool) {
		t.Helper()
		if err := conn.QueryRowContext(h.ctx,
			`SELECT coupon_periods_applied, coupon_applied_current_period FROM subscriptions WHERE id = $1`, couponSub).Scan(&periods, &applied); err != nil {
			t.Fatalf("read coupon progress: %v", err)
		}
		return periods, applied
	}

	e.sweep()
	h.assertAuditGrade("after failed period advance")

	if got := e.charger.chargesOf("cus_retry_pays"); len(got) != 0 {
		t.Fatalf("charged %v before the period advanced, want nothing", got)
	}
	if _, due := e.subState(cardSub); !due {
		t.Fatal("period advanced despite the injected failure")
	}
	if periods, _ := couponProgress(); periods != 1 {
		t.Fatalf("coupon periods = %d before the retry commits, want 1", periods)
	}

	e.sweep()
	h.assertAuditGrade("after retry")

	t.Run("the retry charges the original invoice once", func(t *testing.T) {
		got := e.invoices(cardSub)
		if len(got) != 1 || got[0].status != string(domain.InvoiceStatusPaid) || got[0].total != 100000 {
			t.Fatalf("invoices = %+v, want one paid invoice of 100000", got)
		}
		if got := e.charger.chargesOf("cus_retry_pays"); !equalInt64s(got, []int64{100000}) {
			t.Errorf("charged %v, want exactly 100000", got)
		}
		if _, due := e.subState(cardSub); due {
			t.Error("period did not advance on retry")
		}
	})

	t.Run("the retry restores coupon progress exactly once", func(t *testing.T) {
		if got := invoiceTotals(e.invoices(couponSub)); !equalInt64s(got, []int64{20000}) {
			t.Fatalf("totals = %v, want one invoice of 20000 at 80%% off", got)
		}
		if periods, applied := couponProgress(); periods != 2 || !applied {
			t.Errorf("coupon periods = %d applied = %v, want 2 and true", periods, applied)
		}
	})

	t.Run("the unique key refuses a second invoice for the same period", func(t *testing.T) {
		raceSub, _ := e.seedDue("1 minute", false, "")
		sub, err := db.NewSubscriptionRepository(conn).GetByID(h.tctx, raceSub)
		if err != nil {
			t.Fatalf("load subscription: %v", err)
		}
		key := renewalCycleKey(sub)
		if _, err := e.invoice.generateInvoice(h.tctx, sub, key); err != nil {
			t.Fatalf("first generation: %v", err)
		}
		if _, err := e.invoice.generateInvoice(h.tctx, sub, key); err == nil {
			t.Fatal("second generation for the same period succeeded, want a unique-key error")
		}
		if got := e.invoices(raceSub); len(got) != 1 {
			t.Errorf("invoices = %+v, want one", got)
		}
	})

	// No close-pack assertion here: SumUnscheduledDeferral counts an unpaid
	// invoice's gross subtotal, so the open 80%-off invoice would not tie out.
	h.assertAuditGrade("after refused duplicate")
}
