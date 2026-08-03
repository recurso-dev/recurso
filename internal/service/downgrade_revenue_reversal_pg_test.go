package service

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestDowngradeCreditRevenueReversal_Postgres proves the fix for the latent
// revrec bug where a downgrade credit unconditionally debited the FULL net
// credit out of Deferred Revenue — even when recognition had already run ahead
// of the proration boundary and Deferred no longer held that much.
//
// Setup: a paid 200000 current-plan invoice funds Deferred = 200000 with a
// matching recognition schedule, and then the WHOLE schedule is recognized
// (Deferred -> 0, Recognized Revenue -> 200000, no pending events left). Now a
// mid-period downgrade issues a ~50000 credit. There is nothing left on the
// schedule to give back, so the credit's net portion must be clawed back out of
// Recognized Revenue, NOT out of Deferred.
//
// Old behavior: DR Deferred 50000 / CR Customer-Credit 50000 drove Deferred to
// -50000 (a wrong-sign liability the reconciler flags as abnormal) and left
// Recognized Revenue overstated by 50000. Trial balance still netted to zero, so
// aggregated-across-subscriptions the harness never caught it.
//
// Fixed behavior: the Deferred debit is clamped to what the schedule actually
// gave up (0 here) and the remainder is reversed out of Recognized Revenue:
// DR Recognized Revenue 50000 / CR Customer-Credit 50000. Deferred stays 0,
// Recognized Revenue drops to 150000, Customer-Credit is the full 50000.
func TestDowngradeCreditRevenueReversal_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed downgrade revenue-reversal test")
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
	ctx := context.Background()
	tenantID := seedRevRecTenant(t, conn)
	run := uuid.New().String()[:8]

	// US customer (no tax), current plan 200000, cheaper plan 100000.
	customerID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, name, country, tax_type, ledger_account_id, created_at, updated_at)
		 VALUES ($1, $2, $3, 'Acme US', 'United States', 'individual', $4, NOW(), NOW())`,
		customerID, tenantID, "cust-"+run+"@t.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}
	seedPlan := func(name string, amt int64) uuid.UUID {
		id := uuid.New()
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active) VALUES ($1,$2,$3,$4,'month',1,TRUE)`,
			id, tenantID, name, name+"-"+run); err != nil {
			t.Fatalf("seed plan %s: %v", name, err)
		}
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO prices (id, plan_id, currency, amount, type) VALUES ($1,$2,'USD',$3,'recurring')`,
			uuid.New(), id, amt); err != nil {
			t.Fatalf("seed price %s: %v", name, err)
		}
		return id
	}
	currentPlanID := seedPlan("Pro", 200000)
	targetPlanID := seedPlan("Basic", 100000)

	subID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, status, current_period_start, current_period_end, billing_anchor, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,'active', NOW() - INTERVAL '15 days', NOW() + INTERVAL '15 days', NOW() - INTERVAL '15 days', NOW(), NOW())`,
		subID, tenantID, customerID, currentPlanID); err != nil {
		t.Fatalf("seed subscription: %v", err)
	}

	// Paid current-plan invoice: post it so Deferred = 200000, plus a matching
	// active recognition schedule (pending = 200000 over 10 events).
	curInvID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO invoices (id, tenant_id, customer_id, subscription_id, currency, subtotal, total, amount_paid, status, invoice_number, created_at, due_date)
		 VALUES ($1,$2,$3,$4,'USD',200000,200000,200000,'paid',$5,NOW(),NOW())`,
		curInvID, tenantID, customerID, subID, "INV-"+run); err != nil {
		t.Fatalf("seed current invoice: %v", err)
	}
	ledger := NewLedgerService(nil, db.NewLedgerRepository(conn))
	if err := ledger.RecordInvoice(ctx, &domain.Invoice{
		ID: curInvID, TenantID: tenantID, CustomerID: customerID, SubscriptionID: &subID,
		InvoiceNumber: "INV-" + run, Total: 200000, Currency: "USD",
	}); err != nil {
		t.Fatalf("RecordInvoice: %v", err)
	}
	schedID := seedRevRecSchedule(t, conn, tenantID, curInvID, subID, 20000, 10) // pending 200000

	subRepo := db.NewSubscriptionRepository(conn)
	revrec := NewRevRecService(db.NewRevRecRepository(conn), ledger, subRepo)

	// Recognize the WHOLE schedule: backdate every event to due, then process.
	// Deferred drains to 0 and Recognized Revenue rises to 200000.
	if _, err := conn.ExecContext(ctx,
		`UPDATE recognition_events SET recognition_date = NOW() - INTERVAL '1 day' WHERE revenue_schedule_id = $1`,
		schedID); err != nil {
		t.Fatalf("backdate events: %v", err)
	}
	if err := revrec.ProcessDueEvents(ctx); err != nil {
		t.Fatalf("ProcessDueEvents: %v", err)
	}
	if b := acctBalance(t, conn, tenantID, domain.AccountCodeDeferredRevenue); b != 0 {
		t.Fatalf("post-recognition Deferred = %d, want 0 (fully recognized)", b)
	}
	if b := acctBalance(t, conn, tenantID, domain.AccountCodeRecognizedRevenue); b != 200000 {
		t.Fatalf("post-recognition Recognized Revenue = %d, want 200000", b)
	}
	if p := countEventsByStatus(t, conn, schedID, "pending"); p != 0 {
		t.Fatalf("pending events after recognition = %d, want 0", p)
	}

	// Wire the subscription service and downgrade mid-period.
	invoiceRepo := db.NewInvoiceRepository(conn)
	planRepo := db.NewPlanRepository(conn)
	customerRepo := db.NewCustomerRepository(dbx)
	svc := NewSubscriptionService(subRepo, invoiceRepo, planRepo, customerRepo,
		nil, nil, ledger, nil, nil, db.NewTxManager(conn), revrec, nil)
	svc.SetCreditNoteRepo(db.NewCreditNoteRepository(dbx))

	tctx := context.WithValue(ctx, domain.TenantIDKey, tenantID)
	if _, err := svc.UpdateSubscription(tctx, tenantID, subID, targetPlanID); err != nil {
		t.Fatalf("UpdateSubscription (downgrade): %v", err)
	}

	var creditAmount int64
	if err := conn.QueryRowContext(ctx,
		`SELECT amount FROM credit_notes WHERE tenant_id = $1 AND customer_id = $2 AND type = 'adjustment'`,
		tenantID, customerID).Scan(&creditAmount); err != nil {
		t.Fatalf("read downgrade credit note: %v", err)
	}
	if creditAmount < 40000 || creditAmount > 60000 {
		t.Fatalf("downgrade credit = %d, want ~50000", creditAmount)
	}

	// Deferred must NOT go negative — there was nothing left to give back, so the
	// credit is clawed out of Recognized Revenue instead. (Old code: -creditAmount.)
	if b := acctBalance(t, conn, tenantID, domain.AccountCodeDeferredRevenue); b != 0 {
		t.Errorf("post-downgrade Deferred = %d, want 0 (never driven negative)", b)
	}
	// Recognized Revenue is clawed back by the net credit. (Old code: 200000.)
	if b := acctBalance(t, conn, tenantID, domain.AccountCodeRecognizedRevenue); b != 200000-creditAmount {
		t.Errorf("post-downgrade Recognized Revenue = %d, want %d (reversed by the credit)", b, 200000-creditAmount)
	}
	// Customer-Credit liability is the full credit regardless of where it was sourced.
	if b := acctBalance(t, conn, tenantID, domain.AccountCodeCustomerCredit); b != creditAmount {
		t.Errorf("Customer-Credit = %d, want %d (the downgrade credit)", b, creditAmount)
	}

	// The reconciler must report no wrong-sign balance.
	tb, err := ledger.GetTrialBalance(ctx, tenantID, nil)
	if err != nil {
		t.Fatalf("GetTrialBalance: %v", err)
	}
	for _, l := range tb.Lines {
		if l.Abnormal {
			t.Errorf("account %d (%s) carries an abnormal (wrong-sign) balance %d", l.Code, l.Name, l.Balance)
		}
	}
	if !tb.Balanced {
		t.Errorf("ledger does not balance: debits %d != credits %d", tb.TotalDebits, tb.TotalCredits)
	}
}

// TestDowngradeRevenueReversalCappedAtRecognized_Postgres proves ENG-191e: when
// a downgrade credit exceeds BOTH the schedule's pending events AND what the
// subscription genuinely recognized, only the recognized portion may be clawed
// out of Recognized Revenue — the residual is deferred-but-unscheduled value
// (e.g. an unpaid upgrade-charge invoice funds Deferred with no schedule) and
// must come out of Deferred, where that funding still sits.
//
// Setup: Deferred is funded 200000 by a paid invoice, but its schedule holds
// only ONE 20000 event, which is fully recognized (pool 20000, pending 0,
// Deferred 180000). A downgrade then issues a ~50000 credit. Old behavior
// attributed the whole 50000 shortfall to "already recognized" and posted
// DR Recognized Revenue 50000 — driving account 4100 to −30000 (wrong-sign).
// Fixed: the reversal is capped at the 20000 genuinely recognized (the event
// flips to 'reversed' so a repeat downgrade can't reverse it again), and the
// remaining 30000 debits Deferred.
func TestDowngradeRevenueReversalCappedAtRecognized_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed capped-reversal test")
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
	ctx := context.Background()
	tenantID := seedRevRecTenant(t, conn)
	run := uuid.New().String()[:8]

	customerID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, name, country, tax_type, ledger_account_id, created_at, updated_at)
		 VALUES ($1, $2, $3, 'Acme US', 'United States', 'individual', $4, NOW(), NOW())`,
		customerID, tenantID, "cust-"+run+"@t.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}
	seedPlan := func(name string, amt int64) uuid.UUID {
		id := uuid.New()
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active) VALUES ($1,$2,$3,$4,'month',1,TRUE)`,
			id, tenantID, name, name+"-"+run); err != nil {
			t.Fatalf("seed plan %s: %v", name, err)
		}
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO prices (id, plan_id, currency, amount, type) VALUES ($1,$2,'USD',$3,'recurring')`,
			uuid.New(), id, amt); err != nil {
			t.Fatalf("seed price %s: %v", name, err)
		}
		return id
	}
	currentPlanID := seedPlan("Pro", 200000)
	targetPlanID := seedPlan("Basic", 100000)

	subID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, status, current_period_start, current_period_end, billing_anchor, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,'active', NOW() - INTERVAL '15 days', NOW() + INTERVAL '15 days', NOW() - INTERVAL '15 days', NOW(), NOW())`,
		subID, tenantID, customerID, currentPlanID); err != nil {
		t.Fatalf("seed subscription: %v", err)
	}

	// Deferred funded 200000; schedule holds ONLY one 20000 event (the other
	// 180000 of deferral is unscheduled — the unpaid-upgrade-charge analogue).
	curInvID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO invoices (id, tenant_id, customer_id, subscription_id, currency, subtotal, total, amount_paid, status, invoice_number, created_at, due_date)
		 VALUES ($1,$2,$3,$4,'USD',200000,200000,200000,'paid',$5,NOW(),NOW())`,
		curInvID, tenantID, customerID, subID, "INV-"+run); err != nil {
		t.Fatalf("seed current invoice: %v", err)
	}
	ledger := NewLedgerService(nil, db.NewLedgerRepository(conn))
	if err := ledger.RecordInvoice(ctx, &domain.Invoice{
		ID: curInvID, TenantID: tenantID, CustomerID: customerID, SubscriptionID: &subID,
		InvoiceNumber: "INV-" + run, Total: 200000, Currency: "USD",
	}); err != nil {
		t.Fatalf("RecordInvoice: %v", err)
	}
	schedID := seedRevRecSchedule(t, conn, tenantID, curInvID, subID, 20000, 1) // one 20000 event

	subRepo := db.NewSubscriptionRepository(conn)
	revrec := NewRevRecService(db.NewRevRecRepository(conn), ledger, subRepo)

	// Recognize the lone event: pool 20000, pending 0, Deferred 180000.
	if _, err := conn.ExecContext(ctx,
		`UPDATE recognition_events SET recognition_date = NOW() - INTERVAL '1 day' WHERE revenue_schedule_id = $1`,
		schedID); err != nil {
		t.Fatalf("backdate event: %v", err)
	}
	if err := revrec.ProcessDueEvents(ctx); err != nil {
		t.Fatalf("ProcessDueEvents: %v", err)
	}
	if b := acctBalance(t, conn, tenantID, domain.AccountCodeRecognizedRevenue); b != 20000 {
		t.Fatalf("pre-downgrade Recognized Revenue = %d, want 20000", b)
	}

	invoiceRepo := db.NewInvoiceRepository(conn)
	planRepo := db.NewPlanRepository(conn)
	customerRepo := db.NewCustomerRepository(dbx)
	svc := NewSubscriptionService(subRepo, invoiceRepo, planRepo, customerRepo,
		nil, nil, ledger, nil, nil, db.NewTxManager(conn), revrec, nil)
	svc.SetCreditNoteRepo(db.NewCreditNoteRepository(dbx))

	tctx := context.WithValue(ctx, domain.TenantIDKey, tenantID)
	if _, err := svc.UpdateSubscription(tctx, tenantID, subID, targetPlanID); err != nil {
		t.Fatalf("UpdateSubscription (downgrade): %v", err)
	}

	var creditAmount int64
	if err := conn.QueryRowContext(ctx,
		`SELECT amount FROM credit_notes WHERE tenant_id = $1 AND customer_id = $2 AND type = 'adjustment'`,
		tenantID, customerID).Scan(&creditAmount); err != nil {
		t.Fatalf("read downgrade credit note: %v", err)
	}
	if creditAmount < 40000 || creditAmount > 60000 {
		t.Fatalf("downgrade credit = %d, want ~50000", creditAmount)
	}

	// Recognized Revenue is clawed back ONLY by the 20000 genuinely recognized —
	// never below zero. (Old code: 20000 − creditAmount ≈ −30000, wrong-sign.)
	if b := acctBalance(t, conn, tenantID, domain.AccountCodeRecognizedRevenue); b != 0 {
		t.Errorf("post-downgrade Recognized Revenue = %d, want 0 (capped at what was recognized)", b)
	}
	// The residual (credit − 20000) comes out of Deferred, where the unscheduled
	// funding sits: 180000 − (creditAmount − 20000).
	wantDeferred := 180000 - (creditAmount - 20000)
	if b := acctBalance(t, conn, tenantID, domain.AccountCodeDeferredRevenue); b != wantDeferred {
		t.Errorf("post-downgrade Deferred = %d, want %d (residual funded from unscheduled deferral)", b, wantDeferred)
	}
	// Customer is credited the full net either way.
	if b := acctBalance(t, conn, tenantID, domain.AccountCodeCustomerCredit); b != creditAmount {
		t.Errorf("Customer-Credit = %d, want %d", b, creditAmount)
	}
	// The recognized event is marked reversed — a repeat downgrade can never
	// claw the same revenue back twice.
	var reversedCount int
	if err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM recognition_events WHERE revenue_schedule_id = $1 AND status = 'reversed'`,
		schedID).Scan(&reversedCount); err != nil {
		t.Fatalf("count reversed events: %v", err)
	}
	if reversedCount != 1 {
		t.Errorf("reversed events = %d, want 1 (the recognized event was consumed by the reversal)", reversedCount)
	}

	// No wrong-sign balance anywhere; books balance.
	tb, err := ledger.GetTrialBalance(ctx, tenantID, nil)
	if err != nil {
		t.Fatalf("GetTrialBalance: %v", err)
	}
	for _, l := range tb.Lines {
		if l.Abnormal {
			t.Errorf("account %d (%s) carries an abnormal (wrong-sign) balance %d", l.Code, l.Name, l.Balance)
		}
	}
	if !tb.Balanced {
		t.Errorf("ledger does not balance: debits %d != credits %d", tb.TotalDebits, tb.TotalCredits)
	}
}
