package db

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// TestInvoiceUpdate_PreservesAmountPaid_Postgres proves the ENG-144 fix: Update
// writes the invoice's actual amount_paid, not its total. An unpaid invoice that
// is Updated (retry reschedule, e-invoice status, dunning) must keep
// amount_paid = 0, otherwise AR is silently corrupted. It also checks the new
// tenant_id guard in the WHERE clause.
func TestInvoiceUpdate_PreservesAmountPaid_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed invoice-update test")
	}
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = conn.Close() }()
	ctx := context.Background()
	run := uuid.New().String()[:8]

	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`,
		tenantID, "AR-"+run, "ar-"+run+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	customerID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1, $2, $3, $4, NOW())`,
		customerID, tenantID, "c-"+run+"@t.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}
	invoiceID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO invoices (id, tenant_id, customer_id, currency, subtotal, total, amount_paid, status, invoice_number, due_date, created_at)
		 VALUES ($1, $2, $3, 'INR', 100000, 118000, 0, 'open', $4, NOW(), NOW())`,
		invoiceID, tenantID, customerID, "INV-"+run); err != nil {
		t.Fatalf("seed invoice: %v", err)
	}

	repo := NewInvoiceRepository(conn)

	// Load the invoice the way real callers do (GetByID needs the tenant in
	// context), then reschedule it for retry — the exact unpaid-invoice Update
	// the hardcode corrupted.
	tctx := context.WithValue(ctx, domain.TenantIDKey, tenantID)
	inv, err := repo.GetByID(tctx, invoiceID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if inv.AmountPaid != 0 {
		t.Fatalf("seeded amount_paid = %d, want 0", inv.AmountPaid)
	}
	next := time.Now().Add(24 * time.Hour)
	inv.NextRetryAt = &next
	inv.RetryCount = 1
	if err := repo.Update(ctx, inv); err != nil {
		t.Fatalf("Update: %v", err)
	}

	var amountPaid int64
	var status string
	if err := conn.QueryRowContext(ctx,
		`SELECT amount_paid, status FROM invoices WHERE id = $1`, invoiceID).Scan(&amountPaid, &status); err != nil {
		t.Fatalf("read invoice: %v", err)
	}
	if amountPaid != 0 {
		t.Fatalf("amount_paid = %d after updating an UNPAID invoice, want 0 (total must not be hardcoded)", amountPaid)
	}

	// tenant_id guard: an Update carrying the wrong tenant must not touch the row.
	inv.TenantID = uuid.New()
	inv.RetryCount = 99
	if err := repo.Update(ctx, inv); err != nil {
		t.Fatalf("Update (wrong tenant): %v", err)
	}
	var retryCount int
	if err := conn.QueryRowContext(ctx,
		`SELECT retry_count FROM invoices WHERE id = $1`, invoiceID).Scan(&retryCount); err != nil {
		t.Fatalf("read retry_count: %v", err)
	}
	if retryCount != 1 {
		t.Fatalf("retry_count = %d after a wrong-tenant Update, want 1 (tenant_id guard must scope the write)", retryCount)
	}
}
