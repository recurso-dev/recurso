package service

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestLedgerInvariants_RandomizedBillingSequences is the invariant property
// harness planned in the rev-rec/ledger audit (archive PR #82 scope): it
// drives RANDOMIZED sequences of real billing operations — new paid
// subscriptions, mid-cycle upgrades and downgrades, one-off invoices,
// recognition runs, and cancels with unwind — through the real services, and
// after EVERY step asserts the reconciliation oracle finds an audit-grade
// ledger: no missing invoice legs, no unbalanced ledger, no abnormal account
// balances.
//
// This is the class of test that would have caught F1 (missing invoice legs
// on upgrade/mandate paths) and F3 (one-off recognition draining unfunded
// Deferred) before production: any future invoice-creating flow that forgets
// its ledger leg, or recognition path that over-drains, fails here on some
// seed instead of surfacing as reconciler drift in prod.
//
// Seeds are fixed for CI determinism; set LEDGER_INVARIANT_SEED to explore a
// specific seed. Failures print seed + step for exact reproduction.
func TestLedgerInvariants_RandomizedBillingSequences(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed invariant harness")
	}
	if err := db.RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	dbx, err := sqlx.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = dbx.Close() }()
	conn := dbx.DB

	seeds := []int64{1, 2, 3, 4, 5, 6, 7, 8}
	if s := os.Getenv("LEDGER_INVARIANT_SEED"); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			t.Fatalf("LEDGER_INVARIANT_SEED %q: %v", s, err)
		}
		seeds = []int64{v}
	}
	const opsPerSeed = 25

	for _, seed := range seeds {
		seed := seed
		t.Run(fmt.Sprintf("seed=%d", seed), func(t *testing.T) {
			h := newInvariantHarness(t, conn, dbx)
			rng := rand.New(rand.NewSource(seed))

			// Every sequence starts from one active, fully-posted subscription.
			h.opNewSubscription(rng)
			h.assertAuditGrade("initial subscription")

			for step := 0; step < opsPerSeed; step++ {
				name := h.randomOp(rng)
				h.assertAuditGrade(fmt.Sprintf("seed=%d step=%d op=%s", seed, step, name))
			}
		})
	}
}

// invariantSub is the harness's view of one subscription under test.
type invariantSub struct {
	id       uuid.UUID
	customer uuid.UUID
	onPricey bool
	active   bool
}

type invariantHarness struct {
	t        *testing.T
	conn     *sql.DB
	ctx      context.Context
	tctx     context.Context
	tenantID uuid.UUID

	cheapPlan  uuid.UUID // 100000 USD/month
	priceyPlan uuid.UUID // 200000 USD/month

	ledger *LedgerService
	revrec *RevRecService
	subSvc *SubscriptionService
	recon  *ReconciliationService

	subs []*invariantSub
	run  string
	n    int // uniqueness counter
}

func newInvariantHarness(t *testing.T, conn *sql.DB, dbx *sqlx.DB) *invariantHarness {
	t.Helper()
	ctx := context.Background()
	tenantID := seedRevRecTenant(t, conn)

	h := &invariantHarness{
		t:        t,
		conn:     conn,
		ctx:      ctx,
		tctx:     context.WithValue(ctx, domain.TenantIDKey, tenantID),
		tenantID: tenantID,
		run:      uuid.New().String()[:8],
	}

	seedPlan := func(name string, amt int64) uuid.UUID {
		id := uuid.New()
		mustExec(t, conn, `INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active) VALUES ($1,$2,$3,$4,'month',1,TRUE)`,
			id, tenantID, name, name+"-"+h.run)
		mustExec(t, conn, `INSERT INTO prices (id, plan_id, currency, amount, type) VALUES ($1,$2,'USD',$3,'recurring')`,
			uuid.New(), id, amt)
		return id
	}
	h.cheapPlan = seedPlan("inv-basic", 100000)
	h.priceyPlan = seedPlan("inv-pro", 200000)

	h.ledger = NewLedgerService(nil, db.NewLedgerRepository(conn))
	subRepo := db.NewSubscriptionRepository(conn)
	h.revrec = NewRevRecService(db.NewRevRecRepository(conn), h.ledger, subRepo)
	h.subSvc = NewSubscriptionService(subRepo, db.NewInvoiceRepository(conn), db.NewPlanRepository(conn),
		db.NewCustomerRepository(dbx), nil, nil, h.ledger, nil, nil, db.NewTxManager(conn), h.revrec, nil)
	h.subSvc.SetCreditNoteRepo(db.NewCreditNoteRepository(dbx))
	h.recon = NewReconciliationService(db.NewLedgerRepository(conn), nil)
	return h
}

// randomOp picks and executes one weighted operation; returns its name.
func (h *invariantHarness) randomOp(rng *rand.Rand) string {
	// Weighted table: plan changes and recognition dominate; structural ops
	// (new sub, one-off, cancel) keep the population evolving.
	switch p := rng.Intn(100); {
	case p < 20:
		h.opNewSubscription(rng)
		return "new_subscription"
	case p < 40:
		return h.opPlanChange(rng, true)
	case p < 55:
		return h.opPlanChange(rng, false)
	case p < 70:
		h.opBackdateOneEvent(rng)
		h.opRecognize()
		return "backdate+recognize"
	case p < 85:
		h.opOneOffInvoice(rng)
		return "one_off_invoice"
	default:
		return h.opCancelWithUnwind(rng)
	}
}

// opNewSubscription seeds a customer + active mid-period subscription on the
// cheap plan with a PAID first invoice, fully posted: invoice leg, cash leg,
// and its recognition schedule — the same baseline every production
// subscription reaches after checkout.
func (h *invariantHarness) opNewSubscription(rng *rand.Rand) {
	t := h.t
	h.n++
	customerID := uuid.New()
	mustExec(t, h.conn, `INSERT INTO customers (id, tenant_id, email, name, country, tax_type, ledger_account_id, created_at, updated_at)
		VALUES ($1,$2,$3,'Inv Cust','United States','individual',$4,NOW(),NOW())`,
		customerID, h.tenantID, fmt.Sprintf("inv-%s-%d@t.com", h.run, h.n), uuid.New())

	subID := uuid.New()
	mustExec(t, h.conn, `INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, status, current_period_start, current_period_end, billing_anchor, created_at, updated_at)
		VALUES ($1,$2,$3,$4,'active', NOW() - INTERVAL '15 days', NOW() + INTERVAL '15 days', NOW() - INTERVAL '15 days', NOW(), NOW())`,
		subID, h.tenantID, customerID, h.cheapPlan)

	invID := uuid.New()
	invNo := fmt.Sprintf("INV-%s-%d", h.run, h.n)
	mustExec(t, h.conn, `INSERT INTO invoices (id, tenant_id, customer_id, subscription_id, currency, subtotal, total, amount_paid, status, invoice_number, created_at, due_date)
		VALUES ($1,$2,$3,$4,'USD',100000,100000,100000,'paid',$5,NOW(),NOW())`,
		invID, h.tenantID, customerID, subID, invNo)

	inv := &domain.Invoice{
		ID: invID, TenantID: h.tenantID, CustomerID: customerID, SubscriptionID: &subID,
		InvoiceNumber: invNo, Total: 100000, Currency: "USD",
	}
	if err := h.ledger.RecordInvoice(h.ctx, inv); err != nil {
		t.Fatalf("RecordInvoice (new sub): %v", err)
	}
	if err := h.ledger.RecordPayment(h.ctx, inv); err != nil {
		t.Fatalf("RecordPayment (new sub): %v", err)
	}
	if err := h.revrec.CreateScheduleForInvoice(h.tctx, inv, nil); err != nil {
		t.Fatalf("CreateScheduleForInvoice (new sub): %v", err)
	}
	h.subs = append(h.subs, &invariantSub{id: subID, customer: customerID, active: true})
}

// opPlanChange upgrades (or downgrades) a random eligible subscription
// through the real UpdateSubscription flow — proration invoice/credit, ledger
// postings, rev-rec adjustments and all.
func (h *invariantHarness) opPlanChange(rng *rand.Rand, up bool) string {
	var candidates []*invariantSub
	for _, s := range h.subs {
		if s.active && s.onPricey != up {
			candidates = append(candidates, s)
		}
	}
	if len(candidates) == 0 {
		return "plan_change_skipped"
	}
	s := candidates[rng.Intn(len(candidates))]
	target := h.priceyPlan
	name := "upgrade"
	if !up {
		target = h.cheapPlan
		name = "downgrade"
	}
	if _, err := h.subSvc.UpdateSubscription(h.tctx, h.tenantID, s.id, target); err != nil {
		h.t.Fatalf("UpdateSubscription (%s, sub %s): %v", name, s.id, err)
	}
	s.onPricey = up
	return name
}

// opBackdateOneEvent simulates the passage of time: one pending recognition
// event (if any) becomes due.
func (h *invariantHarness) opBackdateOneEvent(rng *rand.Rand) {
	rows, err := h.conn.QueryContext(h.ctx,
		`SELECT id FROM recognition_events WHERE tenant_id = $1 AND status = 'pending' LIMIT 20`, h.tenantID)
	if err != nil {
		h.t.Fatalf("list pending events: %v", err)
	}
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			h.t.Fatalf("scan event id: %v", err)
		}
		ids = append(ids, id)
	}
	_ = rows.Close()
	if len(ids) == 0 {
		return
	}
	mustExec(h.t, h.conn, `UPDATE recognition_events SET recognition_date = NOW() - INTERVAL '1 hour' WHERE id = $1`,
		ids[rng.Intn(len(ids))])
}

func (h *invariantHarness) opRecognize() {
	if err := h.revrec.ProcessDueEvents(h.ctx); err != nil {
		h.t.Fatalf("ProcessDueEvents: %v", err)
	}
}

// opOneOffInvoice books a paid one-off (no subscription) invoice with its
// ledger legs and immediate recognition.
func (h *invariantHarness) opOneOffInvoice(rng *rand.Rand) {
	if len(h.subs) == 0 {
		return
	}
	t := h.t
	h.n++
	s := h.subs[rng.Intn(len(h.subs))]
	invID := uuid.New()
	invNo := fmt.Sprintf("INV-%s-OO-%d", h.run, h.n)
	total := int64(5000 + rng.Intn(50000))
	mustExec(t, h.conn, `INSERT INTO invoices (id, tenant_id, customer_id, currency, subtotal, total, amount_paid, status, invoice_number, created_at, due_date)
		VALUES ($1,$2,$3,'USD',$4,$4,$4,'paid',$5,NOW(),NOW())`,
		invID, h.tenantID, s.customer, total, invNo)
	inv := &domain.Invoice{
		ID: invID, TenantID: h.tenantID, CustomerID: s.customer,
		InvoiceNumber: invNo, Total: total, Currency: "USD",
	}
	if err := h.ledger.RecordInvoice(h.ctx, inv); err != nil {
		t.Fatalf("RecordInvoice (one-off): %v", err)
	}
	if err := h.ledger.RecordPayment(h.ctx, inv); err != nil {
		t.Fatalf("RecordPayment (one-off): %v", err)
	}
	if err := h.revrec.CreateScheduleForInvoice(h.ctx, inv, nil); err != nil {
		t.Fatalf("CreateScheduleForInvoice (one-off): %v", err)
	}
}

// opCancelWithUnwind cancels a random active subscription immediately and
// forfeits its still-deferred revenue (breakage), like ENG-147 cancel.
func (h *invariantHarness) opCancelWithUnwind(rng *rand.Rand) string {
	var candidates []*invariantSub
	for _, s := range h.subs {
		if s.active {
			candidates = append(candidates, s)
		}
	}
	// Never cancel the last active sub — keep the population alive.
	if len(candidates) <= 1 {
		return "cancel_skipped"
	}
	s := candidates[rng.Intn(len(candidates))]
	mustExec(h.t, h.conn, `UPDATE subscriptions SET status = 'canceled', updated_at = NOW() WHERE id = $1`, s.id)
	if _, err := h.revrec.UnwindOnCancel(h.ctx, h.tenantID, s.id); err != nil {
		h.t.Fatalf("UnwindOnCancel (sub %s): %v", s.id, err)
	}
	s.active = false
	return "cancel+unwind"
}

// assertAuditGrade runs the reconciliation oracle and fails on any
// completeness or balance discrepancy.
func (h *invariantHarness) assertAuditGrade(label string) {
	h.t.Helper()
	report, err := h.recon.Run(h.ctx, h.tenantID)
	if err != nil {
		h.t.Fatalf("[%s] reconciliation Run: %v", label, err)
	}
	for _, d := range report.Discrepancies {
		switch d.Type {
		case DiscrepancyMissingInvoiceTx, DiscrepancyLedgerUnbalanced, DiscrepancyAbnormalBalance:
			h.t.Fatalf("[%s] ledger not audit-grade: %s %+v", label, d.Type, d)
		}
	}
}
