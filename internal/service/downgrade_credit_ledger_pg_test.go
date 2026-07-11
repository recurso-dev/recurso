package service

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/swapnull-in/recur-so/internal/adapter/db"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// TestDowngradeCreditLedgerLifecycle_Postgres proves ENG-154 end-to-end: a
// mid-period downgrade moves the over-deferred revenue into a Customer-Credit
// liability (DR Deferred / CR Customer-Credit) and shrinks the recognition
// schedule by the same amount; applying that credit to a later invoice draws the
// liability back down (DR Customer-Credit / CR AR). Customer-Credit nets to 0.
func TestDowngradeCreditLedgerLifecycle_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed downgrade-ledger test")
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

	// Paid current-plan invoice: post it to the ledger so Deferred = 200000, and
	// seed a matching active recognition schedule (pending = 200000).
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

	if b := acctBalance(t, conn, tenantID, domain.AccountCodeDeferredRevenue); b != 200000 {
		t.Fatalf("pre-downgrade Deferred = %d, want 200000", b)
	}

	// Wire the subscription service with the real ledger + rev-rec.
	subRepo := db.NewSubscriptionRepository(conn)
	invoiceRepo := db.NewInvoiceRepository(conn)
	planRepo := db.NewPlanRepository(conn)
	customerRepo := db.NewCustomerRepository(dbx)
	revrec := NewRevRecService(db.NewRevRecRepository(conn), ledger, subRepo)
	svc := NewSubscriptionService(subRepo, invoiceRepo, planRepo, customerRepo,
		nil, nil, ledger, nil, nil, db.NewTxManager(conn), revrec, nil)
	svc.SetCreditNoteRepo(db.NewCreditNoteRepository(dbx))

	tctx := context.WithValue(ctx, domain.TenantIDKey, tenantID)
	if _, err := svc.UpdateSubscription(tctx, tenantID, subID, targetPlanID); err != nil {
		t.Fatalf("UpdateSubscription (downgrade): %v", err)
	}

	// Read the credit note the downgrade produced.
	var creditNoteID uuid.UUID
	var creditAmount int64
	if err := conn.QueryRowContext(ctx,
		`SELECT id, amount FROM credit_notes WHERE tenant_id = $1 AND customer_id = $2 AND type = 'adjustment'`,
		tenantID, customerID).Scan(&creditNoteID, &creditAmount); err != nil {
		t.Fatalf("read downgrade credit note: %v", err)
	}
	if creditAmount < 40000 || creditAmount > 60000 {
		t.Fatalf("downgrade credit = %d, want ~50000", creditAmount)
	}

	// Ledger: Deferred dropped by the credit; Customer-Credit rose by it.
	if b := acctBalance(t, conn, tenantID, domain.AccountCodeDeferredRevenue); b != 200000-creditAmount {
		t.Errorf("post-downgrade Deferred = %d, want %d", b, 200000-creditAmount)
	}
	if b := acctBalance(t, conn, tenantID, domain.AccountCodeCustomerCredit); b != creditAmount {
		t.Errorf("Customer-Credit = %d, want %d (the downgrade credit)", b, creditAmount)
	}

	// Schedule shrank by the credit: remaining pending = 200000 - credit.
	var pending int64
	if err := conn.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(amount),0) FROM recognition_events WHERE revenue_schedule_id = $1 AND status = 'pending'`,
		schedID).Scan(&pending); err != nil {
		t.Fatalf("sum pending: %v", err)
	}
	if pending != 200000-creditAmount {
		t.Errorf("remaining pending recognition = %d, want %d (reduced by the credit)", pending, 200000-creditAmount)
	}

	// Apply the credit to a later invoice via CreditNoteService (which posts the
	// DR Customer-Credit / CR AR settlement). New-plan invoice total 100000.
	newInvID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO invoices (id, tenant_id, customer_id, subscription_id, currency, subtotal, total, amount_paid, credit_applied, status, invoice_number, created_at, due_date)
		 VALUES ($1,$2,$3,$4,'USD',100000,100000,0,0,'open',$5,NOW(),NOW())`,
		newInvID, tenantID, customerID, subID, "INV2-"+run); err != nil {
		t.Fatalf("seed new invoice: %v", err)
	}
	cnSvc := NewCreditNoteService(db.NewCreditNoteRepository(dbx), nil, nil, nil)
	cnSvc.SetLedgerService(ledger)
	applied, err := cnSvc.ApplyAdjustmentCredits(ctx, tenantID, customerID, "USD", newInvID, 100000)
	if err != nil {
		t.Fatalf("ApplyAdjustmentCredits: %v", err)
	}
	if applied != creditAmount {
		t.Errorf("applied = %d, want %d (full credit balance)", applied, creditAmount)
	}

	// Customer-Credit is drawn back to 0 (downgrade +credit, application -credit).
	if b := acctBalance(t, conn, tenantID, domain.AccountCodeCustomerCredit); b != 0 {
		t.Errorf("Customer-Credit after application = %d, want 0", b)
	}
	// The whole ledger still balances (total debits == total credits).
	var totalD, totalC int64
	if err := conn.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(debits_posted),0), COALESCE(SUM(credits_posted),0) FROM ledger_accounts WHERE tenant_id = $1`,
		tenantID).Scan(&totalD, &totalC); err != nil {
		t.Fatalf("sum posted: %v", err)
	}
	if totalD != totalC {
		t.Errorf("ledger does not balance: debits %d != credits %d", totalD, totalC)
	}
}

// TestManualAdjustmentCreditLedger_Postgres proves a manually-issued adjustment
// credit is booked to the ledger (DR Credits & Adjustments / CR Customer-Credit)
// so that applying it (DR Customer-Credit / CR AR) draws Customer-Credit back to
// zero and the ledger stays balanced — the credit source doesn't matter.
func TestManualAdjustmentCreditLedger_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed manual-credit ledger test")
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
		`INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1, $2, $3, $4, NOW())`,
		customerID, tenantID, "cust-"+run+"@t.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}

	ledger := NewLedgerService(nil, db.NewLedgerRepository(conn))
	cnSvc := NewCreditNoteService(db.NewCreditNoteRepository(dbx), db.NewCustomerRepository(dbx), db.NewInvoiceRepository(conn), nil)
	cnSvc.SetLedgerService(ledger)

	// Issue a manual goodwill adjustment credit of 30000.
	tctx := context.WithValue(ctx, domain.TenantIDKey, tenantID)
	cn, err := cnSvc.Create(tctx, tenantID, domain.CreateCreditNoteRequest{
		CustomerID: customerID, Amount: 30000, Currency: "USD", Reason: "goodwill",
	})
	if err != nil {
		t.Fatalf("Create adjustment credit: %v", err)
	}
	if cn.Type != domain.CreditNoteTypeAdjustment {
		t.Fatalf("credit type = %q, want adjustment", cn.Type)
	}
	if b := acctBalance(t, conn, tenantID, domain.AccountCodeCustomerCredit); b != 30000 {
		t.Errorf("Customer-Credit after issuance = %d, want 30000", b)
	}
	if b := acctBalance(t, conn, tenantID, domain.AccountCodeCreditsIssued); b != 30000 {
		t.Errorf("Credits & Adjustments (expense) = %d, want 30000", b)
	}

	// Apply it to an invoice → Customer-Credit drawn back to 0.
	invID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO invoices (id, tenant_id, customer_id, currency, subtotal, total, amount_paid, credit_applied, status, invoice_number, created_at, due_date)
		 VALUES ($1,$2,$3,'USD',50000,50000,0,0,'open',$4,NOW(),NOW())`,
		invID, tenantID, customerID, "INV-"+run); err != nil {
		t.Fatalf("seed invoice: %v", err)
	}
	applied, err := cnSvc.ApplyAdjustmentCredits(ctx, tenantID, customerID, "USD", invID, 50000)
	if err != nil {
		t.Fatalf("ApplyAdjustmentCredits: %v", err)
	}
	if applied != 30000 {
		t.Errorf("applied = %d, want 30000", applied)
	}
	if b := acctBalance(t, conn, tenantID, domain.AccountCodeCustomerCredit); b != 0 {
		t.Errorf("Customer-Credit after application = %d, want 0", b)
	}
	var totalD, totalC int64
	if err := conn.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(debits_posted),0), COALESCE(SUM(credits_posted),0) FROM ledger_accounts WHERE tenant_id = $1`,
		tenantID).Scan(&totalD, &totalC); err != nil {
		t.Fatalf("sum posted: %v", err)
	}
	if totalD != totalC {
		t.Errorf("ledger does not balance: debits %d != credits %d", totalD, totalC)
	}
}
