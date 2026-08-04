package service

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestClosePackTieOut_AccrualZeroes_Postgres is the verification the founder
// asked for: under ACCRUAL (recognition schedule built at issuance), an unpaid
// subscription invoice's deferred lives in the SCHEDULED bucket, not the
// awaiting-payment bucket, so the month-end deferred tie-out is exactly zero.
//
// It also guards the load-bearing fix: SumUnscheduledDeferral must EXCLUDE
// invoices that have an active schedule, or an accrual invoice would be counted
// in both scheduled and awaiting-payment and the tie-out would go negative.
func TestClosePackTieOut_AccrualZeroes_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed tie-out test")
	}
	if err := db.RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = conn.Close() }()
	ctx := context.Background()

	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`,
		tenantID, "TO-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	ledgerRepo := db.NewLedgerRepository(conn)
	ledger := NewLedgerService(nil, ledgerRepo)
	if err := ledger.SetupTenantAccounts(ctx, tenantID); err != nil {
		t.Fatalf("setup accounts: %v", err)
	}
	revrecRepo := db.NewRevRecRepository(conn)
	revrec := NewRevRecService(revrecRepo, ledger, nil)
	invoiceRepo := db.NewInvoiceRepository(conn)
	recon := NewReconciliationService(ledgerRepo, nil)

	closePack := NewClosePackService(ledger, recon)
	closePack.SetRevRecService(revrec)
	closePack.SetUnscheduledDeferralReader(invoiceRepo)

	now := time.Now().UTC()

	// Seed the FK graph: customer (points its ledger_account_id at the tenant's
	// AR account) → plan → subscription → invoice.
	var arAccountID uuid.UUID
	if err := conn.QueryRowContext(ctx,
		`SELECT id FROM ledger_accounts WHERE tenant_id=$1 AND code=1100 LIMIT 1`, tenantID).Scan(&arAccountID); err != nil {
		t.Fatalf("find AR account: %v", err)
	}
	customerID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, ledger_account_id, phone, line1, city, state, zip, country,
		    active, tax_exempt, tax_exemption_number, tax_exemption_code, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,'','','','','','',true,false,'','',$5,$5)`,
		customerID, tenantID, "c-"+customerID.String()[:8]+"@t.com", arAccountID, now); err != nil {
		t.Fatalf("seed customer: %v", err)
	}
	planID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO plans (id, tenant_id, name, code, hsn_code, created_at, updated_at)
		 VALUES ($1,$2,'Test Plan',$3,'',$4,$4)`,
		planID, tenantID, "plan-"+planID.String()[:8], now); err != nil {
		t.Fatalf("seed plan: %v", err)
	}
	subID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, status, current_period_start, current_period_end, billing_anchor, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,'active',$5,$6,$5,$5,$5)`,
		subID, tenantID, customerID, planID, now, now.AddDate(0, 1, 0)); err != nil {
		t.Fatalf("seed subscription: %v", err)
	}

	// One UNPAID subscription invoice: pre-tax 100,000 deferred at issuance.
	inv := &domain.Invoice{
		ID: uuid.New(), TenantID: tenantID, CustomerID: customerID, SubscriptionID: &subID,
		InvoiceNumber: "INV-TO-1", Currency: "USD",
		Subtotal: 100000, TaxAmount: 0, Total: 100000, CreatedAt: now,
	}
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO invoices (id, tenant_id, subscription_id, customer_id, currency, subtotal, total, invoice_number, tax_type, status, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,'USD',100000,100000,$5,'none','open',$6,$6)`,
		inv.ID, tenantID, subID, customerID, inv.InvoiceNumber, now); err != nil {
		t.Fatalf("seed invoice: %v", err)
	}
	if err := ledger.RecordInvoice(ctx, inv); err != nil { // Code-1: AR → Deferred 100000
		t.Fatalf("record invoice: %v", err)
	}

	month, year := int(now.Month()), now.Year()

	// --- Case A: CASH model (no schedule). The invoice sits in awaiting-payment;
	// the tie-out holds via that term. ---
	packCash, err := closePack.Generate(ctx, tenantID, month, year)
	if err != nil {
		t.Fatalf("close pack (cash): %v", err)
	}
	if packCash.Deferred.AwaitingPayment != 100000 {
		t.Errorf("cash awaiting-payment = %d, want 100000 (no schedule yet)", packCash.Deferred.AwaitingPayment)
	}
	if packCash.Deferred.UnexplainedDelta != 0 {
		t.Errorf("cash tie-out delta = %d, want 0", packCash.Deferred.UnexplainedDelta)
	}

	// --- Case B: ACCRUAL. Build the schedule at issuance: the full 100,000
	// pending. The invoice now has an active schedule. ---
	sched := &domain.RevenueSchedule{
		ID: uuid.New(), TenantID: tenantID, InvoiceID: inv.ID,
		TotalAmount: 100000, Currency: "USD", StartDate: now, EndDate: now.AddDate(0, 1, 0),
		Status: "active",
	}
	if err := revrecRepo.CreateSchedule(ctx, sched); err != nil {
		t.Fatalf("create schedule: %v", err)
	}
	if err := revrecRepo.CreateEvents(ctx, []*domain.RecognitionEvent{
		{ID: uuid.New(), RevenueScheduleID: sched.ID, TenantID: tenantID, Amount: 100000, RecognitionDate: now.AddDate(0, 0, 10), Status: "pending"},
	}); err != nil {
		t.Fatalf("create events: %v", err)
	}

	packAccrual, err := closePack.Generate(ctx, tenantID, month, year)
	if err != nil {
		t.Fatalf("close pack (accrual): %v", err)
	}
	// The invoice moved OUT of awaiting-payment (it now has a schedule) and INTO
	// scheduled — SumUnscheduledDeferral must exclude it, else double-count.
	if packAccrual.Deferred.AwaitingPayment != 0 {
		t.Errorf("accrual awaiting-payment = %d, want 0 (invoice now scheduled — the exclusion fix)", packAccrual.Deferred.AwaitingPayment)
	}
	if packAccrual.Deferred.Recognition == nil || packAccrual.Deferred.Recognition.DeferredBalance != 100000 {
		t.Errorf("accrual scheduled deferred = %v, want 100000", packAccrual.Deferred.Recognition)
	}
	// THE VERIFICATION: the deferred tie-out is exactly zero under accrual.
	if packAccrual.Deferred.UnexplainedDelta != 0 || !packAccrual.Deferred.Ties {
		t.Errorf("accrual tie-out delta = %d, ties = %v; want 0 / true",
			packAccrual.Deferred.UnexplainedDelta, packAccrual.Deferred.Ties)
	}
	t.Logf("VERIFIED: accrual tie-out — ledger closing %d = scheduled %d + awaiting %d, delta %d (ties=%v)",
		packAccrual.Deferred.Rollforward.Closing, packAccrual.Deferred.Recognition.DeferredBalance,
		packAccrual.Deferred.AwaitingPayment, packAccrual.Deferred.UnexplainedDelta, packAccrual.Deferred.Ties)
}
