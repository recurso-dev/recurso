package db

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

func openCreditAppTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed credit-application test")
	}
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	dbx, err := sqlx.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	return dbx
}

func seedCreditAppTenantCustomer(t *testing.T, conn *sql.DB) (uuid.UUID, uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`,
		tenantID, "CreditApp-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	customerID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1, $2, $3, $4, NOW())`,
		customerID, tenantID, customerID.String()[:8]+"@t.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}
	return tenantID, customerID
}

func seedInvoiceRow(t *testing.T, conn *sql.DB, tenantID, customerID uuid.UUID, total int64) uuid.UUID {
	t.Helper()
	invID := uuid.New()
	if _, err := conn.ExecContext(context.Background(),
		`INSERT INTO invoices (id, tenant_id, customer_id, currency, subtotal, total, amount_paid, credit_applied, status, invoice_number, created_at, due_date)
		 VALUES ($1, $2, $3, 'USD', $4, $4, 0, 0, 'open', $5, NOW(), NOW())`,
		invID, tenantID, customerID, total, "INV-"+invID.String()[:8]); err != nil {
		t.Fatalf("seed invoice: %v", err)
	}
	return invID
}

// seedAdjustmentCredit inserts an issued adjustment credit note with the given
// balance. createdOffset shifts created_at so FIFO order is deterministic.
func seedAdjustmentCredit(t *testing.T, conn *sql.DB, tenantID, customerID uuid.UUID, balance int64, createdOffset time.Duration) uuid.UUID {
	t.Helper()
	id := uuid.New()
	createdAt := time.Now().Add(createdOffset)
	if _, err := conn.ExecContext(context.Background(),
		`INSERT INTO credit_notes (id, tenant_id, customer_id, amount, balance, currency, status, reason, type, refund_status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $4, 'USD', 'issued', 'test credit', 'adjustment', 'none', $5, NOW())`,
		id, tenantID, customerID, balance, createdAt); err != nil {
		t.Fatalf("seed credit note: %v", err)
	}
	return id
}

func creditNoteState(t *testing.T, conn *sql.DB, id uuid.UUID) (int64, string) {
	t.Helper()
	var bal int64
	var status string
	if err := conn.QueryRowContext(context.Background(),
		`SELECT balance, status FROM credit_notes WHERE id = $1`, id).Scan(&bal, &status); err != nil {
		t.Fatalf("read credit note: %v", err)
	}
	return bal, status
}

// TestApplyAdjustmentCredits_FIFO_Postgres proves ENG-153 FIFO + boundary split
// + full-cover: an invoice of 4000 with an older 3000 credit and a newer 2000
// credit draws the older one fully (used) and the newer one partially (1000
// left), fully covers the invoice (marked paid), and records two audit rows.
func TestApplyAdjustmentCredits_FIFO_Postgres(t *testing.T) {
	dbx := openCreditAppTestDB(t)
	defer func() { _ = dbx.Close() }()
	conn := dbx.DB
	ctx := context.Background()
	tenantID, customerID := seedCreditAppTenantCustomer(t, conn)

	invID := seedInvoiceRow(t, conn, tenantID, customerID, 4000)
	older := seedAdjustmentCredit(t, conn, tenantID, customerID, 3000, -2*time.Hour)
	newer := seedAdjustmentCredit(t, conn, tenantID, customerID, 2000, -1*time.Hour)

	repo := NewCreditNoteRepository(dbx)
	applied, err := repo.ApplyAdjustmentCredits(ctx, tenantID, customerID, "USD", invID, 4000)
	if err != nil {
		t.Fatalf("ApplyAdjustmentCredits: %v", err)
	}
	if applied != 4000 {
		t.Fatalf("applied = %d, want 4000", applied)
	}

	// FIFO: older fully drawn (0, used), newer partially (1000 left, issued).
	if bal, status := creditNoteState(t, conn, older); bal != 0 || status != "used" {
		t.Errorf("older credit = (%d, %s), want (0, used)", bal, status)
	}
	if bal, status := creditNoteState(t, conn, newer); bal != 1000 || status != "issued" {
		t.Errorf("newer credit = (%d, %s), want (1000, issued)", bal, status)
	}

	// Invoice fully covered → credit_applied=4000, status paid, amount_due 0.
	var creditApplied, total int64
	var status string
	var paidAt sql.NullTime
	if err := conn.QueryRowContext(ctx,
		`SELECT credit_applied, total, status, paid_at FROM invoices WHERE id = $1`, invID).Scan(&creditApplied, &total, &status, &paidAt); err != nil {
		t.Fatalf("read invoice: %v", err)
	}
	if creditApplied != 4000 || status != "paid" || !paidAt.Valid {
		t.Errorf("invoice = (credit_applied=%d, status=%s, paid_at.valid=%v), want (4000, paid, true)", creditApplied, status, paidAt.Valid)
	}

	// Two audit rows summing to 4000.
	var appCount int
	var appSum int64
	if err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(SUM(amount),0) FROM credit_note_applications WHERE invoice_id = $1`, invID).Scan(&appCount, &appSum); err != nil {
		t.Fatalf("read applications: %v", err)
	}
	if appCount != 2 || appSum != 4000 {
		t.Errorf("applications = (count=%d, sum=%d), want (2, 4000)", appCount, appSum)
	}
}

// TestApplyAdjustmentCredits_PartialInvoice_Postgres proves that when available
// credit is less than the invoice, all credit is consumed but the invoice stays
// open with a reduced amount due.
func TestApplyAdjustmentCredits_PartialInvoice_Postgres(t *testing.T) {
	dbx := openCreditAppTestDB(t)
	defer func() { _ = dbx.Close() }()
	conn := dbx.DB
	ctx := context.Background()
	tenantID, customerID := seedCreditAppTenantCustomer(t, conn)

	invID := seedInvoiceRow(t, conn, tenantID, customerID, 10000)
	seedAdjustmentCredit(t, conn, tenantID, customerID, 3000, -2*time.Hour)
	seedAdjustmentCredit(t, conn, tenantID, customerID, 2000, -1*time.Hour)

	repo := NewCreditNoteRepository(dbx)
	applied, err := repo.ApplyAdjustmentCredits(ctx, tenantID, customerID, "USD", invID, 10000)
	if err != nil {
		t.Fatalf("ApplyAdjustmentCredits: %v", err)
	}
	if applied != 5000 {
		t.Fatalf("applied = %d, want 5000 (all available credit)", applied)
	}

	inv, err := NewInvoiceRepository(conn).(*InvoiceRepository).GetByIDPublic(ctx, invID)
	if err != nil {
		t.Fatalf("GetByIDPublic: %v", err)
	}
	if inv.Status != domain.InvoiceStatusOpen {
		t.Errorf("invoice status = %q, want open (partially covered)", inv.Status)
	}
	if inv.CreditApplied != 5000 {
		t.Errorf("credit_applied = %d, want 5000", inv.CreditApplied)
	}
	if inv.AmountDue != 5000 {
		t.Errorf("amount_due = %d, want 5000 (10000 - 5000 credit)", inv.AmountDue)
	}

	// A currency mismatch must not touch these credits.
	if sum, err := repo.SumApplicableAdjustments(ctx, tenantID, customerID, "INR"); err != nil || sum != 0 {
		t.Errorf("SumApplicableAdjustments(INR) = (%d, %v), want (0, nil)", sum, err)
	}
	if sum, err := repo.SumApplicableAdjustments(ctx, tenantID, customerID, "USD"); err != nil || sum != 0 {
		t.Errorf("SumApplicableAdjustments(USD) after full draw = (%d, %v), want (0, nil)", sum, err)
	}
}

// TestMarkPaid_WithCreditApplied_Postgres proves MarkPaid records only the cash
// portion (total - credit_applied) so amount_paid + credit_applied = total.
func TestMarkPaid_WithCreditApplied_Postgres(t *testing.T) {
	dbx := openCreditAppTestDB(t)
	defer func() { _ = dbx.Close() }()
	conn := dbx.DB
	ctx := context.Background()
	tenantID, customerID := seedCreditAppTenantCustomer(t, conn)

	invID := seedInvoiceRow(t, conn, tenantID, customerID, 10000)
	if _, err := conn.ExecContext(ctx, `UPDATE invoices SET credit_applied = 4000 WHERE id = $1`, invID); err != nil {
		t.Fatalf("set credit_applied: %v", err)
	}

	repo := NewInvoiceRepository(conn).(*InvoiceRepository)
	ok, err := repo.MarkPaid(ctx, invID, time.Now())
	if err != nil || !ok {
		t.Fatalf("MarkPaid = (%v, %v), want (true, nil)", ok, err)
	}

	inv, err := repo.GetByIDPublic(ctx, invID)
	if err != nil {
		t.Fatalf("GetByIDPublic: %v", err)
	}
	if inv.AmountPaid != 6000 {
		t.Errorf("amount_paid = %d, want 6000 (cash = total - credit)", inv.AmountPaid)
	}
	if inv.CreditApplied != 4000 {
		t.Errorf("credit_applied = %d, want 4000", inv.CreditApplied)
	}
	if inv.AmountDue != 0 {
		t.Errorf("amount_due = %d, want 0 (fully settled by cash + credit)", inv.AmountDue)
	}
}
