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

// TestAccrualFoundation_Postgres proves the accrual epic's increment-1
// foundation against real Postgres, without changing any money behavior:
//  1. SetupTenantAccounts seeds the new Bad Debt Expense account (5200).
//  2. SumRecognizedByInvoice returns 0 for an invoice with no schedule (the
//     current cash-model case) and the recognized total once events exist —
//     the query the write-off split will consult in increment 2.
func TestAccrualFoundation_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed accrual-foundation test")
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
		tenantID, "Accr-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	ledger := NewLedgerService(nil, db.NewLedgerRepository(conn))
	if err := ledger.SetupTenantAccounts(ctx, tenantID); err != nil {
		t.Fatalf("setup accounts: %v", err)
	}

	// (1) Bad Debt Expense (5200) is now in the tenant's chart, typed Expense.
	var name string
	var typ int
	if err := conn.QueryRowContext(ctx,
		`SELECT name, type FROM ledger_accounts WHERE tenant_id=$1 AND code=$2`,
		tenantID, domain.AccountCodeBadDebtExpense).Scan(&name, &typ); err != nil {
		t.Fatalf("bad debt account not seeded: %v", err)
	}
	if typ != int(domain.AccountTypeExpense) {
		t.Errorf("Bad Debt Expense type = %d, want %d (expense)", typ, domain.AccountTypeExpense)
	}

	// (2) SumRecognizedByInvoice.
	revrec := db.NewRevRecRepository(conn)
	invoiceID := uuid.New()

	// No schedule yet → 0 (the current cash-model write-off stays byte-identical).
	if got, err := revrec.SumRecognizedByInvoice(ctx, tenantID, invoiceID); err != nil || got != 0 {
		t.Fatalf("SumRecognizedByInvoice (no schedule) = %d, err %v; want 0, nil", got, err)
	}

	// A schedule FKs invoices — seed a minimal invoice row (defaults cover the
	// rest of the NOT NULL columns).
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO invoices (id, tenant_id, currency, subtotal, total, invoice_number, tax_type, status, created_at, updated_at)
		 VALUES ($1, $2, 'USD', 10000, 10000, $3, 'none', 'open', NOW(), NOW())`,
		invoiceID, tenantID, "INV-ACCR-1"); err != nil {
		t.Fatalf("seed invoice: %v", err)
	}

	// Create a schedule + two events (one recognized, one pending); the
	// recognized one should sum, the pending one should not. subscription_id
	// left nil (one-off) to avoid seeding a subscription for this query test.
	sched := &domain.RevenueSchedule{
		ID: uuid.New(), TenantID: tenantID, InvoiceID: invoiceID,
		TotalAmount: 10000, Currency: "USD", StartDate: time.Now(), EndDate: time.Now().AddDate(0, 1, 0),
		Status: "active",
	}
	if err := revrec.CreateSchedule(ctx, sched); err != nil {
		t.Fatalf("create schedule: %v", err)
	}
	events := []*domain.RecognitionEvent{
		{ID: uuid.New(), RevenueScheduleID: sched.ID, TenantID: tenantID, Amount: 4000, RecognitionDate: time.Now(), Status: "recognized"},
		{ID: uuid.New(), RevenueScheduleID: sched.ID, TenantID: tenantID, Amount: 6000, RecognitionDate: time.Now().AddDate(0, 0, 15), Status: "pending"},
	}
	if err := revrec.CreateEvents(ctx, events); err != nil {
		t.Fatalf("create events: %v", err)
	}

	got, err := revrec.SumRecognizedByInvoice(ctx, tenantID, invoiceID)
	if err != nil {
		t.Fatalf("SumRecognizedByInvoice: %v", err)
	}
	if got != 4000 {
		t.Errorf("recognized sum = %d, want 4000 (only the recognized event, not the pending)", got)
	}
}
