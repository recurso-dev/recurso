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

// TestAccrualWriteOff_Postgres is the increment-3 end-to-end proof against a
// REAL recognition schedule: an accrual invoice recognizes part of its revenue,
// then is written off — the recognized part expenses Bad Debt (26), the still-
// deferred part reverses from Deferred (22), and the schedule's PENDING events
// are cancelled so the reversed-out Deferred can't be re-recognized. Books stay
// balanced. This wires the real revrec repo (recognized lookup) and the real
// RevRecService (pending-event canceller), not stubs.
func TestAccrualWriteOff_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed accrual write-off test")
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
		tenantID, "AW-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	revrecRepo := db.NewRevRecRepository(conn)
	ledger := NewLedgerService(nil, db.NewLedgerRepository(conn))
	revrec := NewRevRecService(revrecRepo, ledger, nil)
	ledger.SetRecognizedReader(revrecRepo) // real recognized-amount lookup
	ledger.SetScheduleCanceller(revrec)    // real pending-event canceller

	customerID := uuid.New()
	subID := uuid.New()
	inv := &domain.Invoice{
		ID: uuid.New(), TenantID: tenantID, CustomerID: customerID, SubscriptionID: &subID,
		InvoiceNumber: "INV-AW-1", Currency: "USD",
		Subtotal: 100000, TaxAmount: 0, Total: 100000, CreatedAt: time.Now(),
	}
	if err := ledger.RecordInvoice(ctx, inv); err != nil {
		t.Fatalf("record invoice: %v", err)
	}

	// A schedule FKs invoices — seed the invoice row.
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO invoices (id, tenant_id, currency, subtotal, total, invoice_number, tax_type, status, created_at, updated_at)
		 VALUES ($1, $2, 'USD', 100000, 100000, $3, 'none', 'open', NOW(), NOW())`,
		inv.ID, tenantID, inv.InvoiceNumber); err != nil {
		t.Fatalf("seed invoice: %v", err)
	}

	// Accrual: schedule built at issuance with two events — one recognized
	// (30,000), one pending (70,000).
	sched := &domain.RevenueSchedule{
		ID: uuid.New(), TenantID: tenantID, InvoiceID: inv.ID,
		TotalAmount: 100000, Currency: "USD", StartDate: time.Now(), EndDate: time.Now().AddDate(0, 1, 0),
		Status: "active",
	}
	if err := revrecRepo.CreateSchedule(ctx, sched); err != nil {
		t.Fatalf("create schedule: %v", err)
	}
	recEvent := uuid.New()
	if err := revrecRepo.CreateEvents(ctx, []*domain.RecognitionEvent{
		{ID: recEvent, RevenueScheduleID: sched.ID, TenantID: tenantID, Amount: 30000, RecognitionDate: time.Now(), Status: "recognized"},
		{ID: uuid.New(), RevenueScheduleID: sched.ID, TenantID: tenantID, Amount: 70000, RecognitionDate: time.Now().AddDate(0, 0, 15), Status: "pending"},
	}); err != nil {
		t.Fatalf("create events: %v", err)
	}

	debitBal := func(code int) int64 {
		var n int64
		if err := conn.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(CASE WHEN t.debit_account_id=a.id THEN t.amount ELSE -t.amount END),0)
			FROM ledger_accounts a
			JOIN ledger_transactions t ON t.credit_account_id=a.id OR t.debit_account_id=a.id
			WHERE a.tenant_id=$1 AND a.code=$2`, tenantID, code).Scan(&n); err != nil {
			t.Fatalf("debitBal(%d): %v", code, err)
		}
		return n
	}
	creditBal := func(code int) int64 {
		var n int64
		if err := conn.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(CASE WHEN t.credit_account_id=a.id THEN t.amount ELSE -t.amount END),0)
			FROM ledger_accounts a
			JOIN ledger_transactions t ON t.credit_account_id=a.id OR t.debit_account_id=a.id
			WHERE a.tenant_id=$1 AND a.code=$2`, tenantID, code).Scan(&n); err != nil {
			t.Fatalf("creditBal(%d): %v", code, err)
		}
		return n
	}

	// Write off. Recognized 30k → Bad Debt; deferred 70k → Deferred reversal.
	if err := ledger.RecordInvoiceWriteOff(ctx, inv); err != nil {
		t.Fatalf("write-off: %v", err)
	}
	if got := creditBal(2100); got != 30000 {
		// Deferred started 100k, minus the 70k deferred-portion reversal = 30k.
		t.Errorf("Deferred after write-off = %d, want 30000 (100k − 70k reversed)", got)
	}
	if got := debitBal(domain.AccountCodeBadDebtExpense); got != 30000 {
		t.Errorf("Bad Debt Expense = %d, want 30000 (the recognized portion)", got)
	}
	if got := debitBal(1100); got != 0 {
		t.Errorf("AR after write-off = %d, want 0 (fully relieved)", got)
	}

	// The schedule's PENDING events are cancelled and the schedule is no longer
	// active — so the recognition worker can't drain the reversed-out Deferred.
	var pending int
	if err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM recognition_events e JOIN revenue_schedules s ON s.id=e.revenue_schedule_id
		 WHERE s.invoice_id=$1 AND e.status='pending'`, inv.ID).Scan(&pending); err != nil {
		t.Fatalf("pending count: %v", err)
	}
	if pending != 0 {
		t.Errorf("pending events after write-off = %d, want 0 (cancelled)", pending)
	}
	if got, _ := revrecRepo.GetActiveScheduleByInvoice(ctx, tenantID, inv.ID); got != nil {
		t.Error("schedule should be marked canceled (not active) after write-off")
	}

	// Books balanced.
	var dr, cr int64
	_ = conn.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount),0) FROM ledger_transactions t JOIN ledger_accounts a ON a.id=t.debit_account_id WHERE a.tenant_id=$1`, tenantID).Scan(&dr)
	_ = conn.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount),0) FROM ledger_transactions t JOIN ledger_accounts a ON a.id=t.credit_account_id WHERE a.tenant_id=$1`, tenantID).Scan(&cr)
	if dr != cr {
		t.Errorf("books unbalanced: debits %d != credits %d", dr, cr)
	}
}
