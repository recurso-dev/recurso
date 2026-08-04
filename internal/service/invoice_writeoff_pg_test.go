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

// TestInvoiceWriteOff_Postgres proves the write-off reversal: marking an unpaid
// subscription invoice uncollectible posts DR Deferred / CR AR at pre-tax plus
// the tax reversal out of Tax Payable — so AR stops carrying money that will
// never arrive and Deferred stops carrying revenue that will never be earned.
// Before this posting existed, a write-off was a bare status flip and both
// balances were overstated forever (found by the close pack's deferred
// tie-out). Also proves idempotency: a second call posts nothing.
func TestInvoiceWriteOff_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed write-off test")
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
		tenantID, "WOff-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	svc := NewLedgerService(nil, db.NewLedgerRepository(conn))

	customerID := uuid.New()
	subID := uuid.New()
	inv := &domain.Invoice{
		ID:             uuid.New(),
		TenantID:       tenantID,
		CustomerID:     customerID,
		SubscriptionID: &subID,
		InvoiceNumber:  "INV-WOFF-1",
		Currency:       "INR",
		Subtotal:       100000,
		TaxAmount:      18000,
		Total:          118000,
		CreatedAt:      time.Now(),
	}
	// Issue: AR → Deferred at gross + tax reclass (the app's Code-1/Code-6).
	if err := svc.RecordInvoice(ctx, inv); err != nil {
		t.Fatalf("record invoice: %v", err)
	}

	balance := func(code int) int64 {
		var n int64
		if err := conn.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(CASE WHEN t.credit_account_id=a.id THEN t.amount ELSE -t.amount END),0)
			FROM ledger_accounts a
			JOIN ledger_transactions t ON t.credit_account_id=a.id OR t.debit_account_id=a.id
			WHERE a.tenant_id=$1 AND a.code=$2`, tenantID, code).Scan(&n); err != nil {
			t.Fatalf("balance(%d): %v", code, err)
		}
		return n
	}
	if got := balance(2100); got != 100000 {
		t.Fatalf("Deferred after issue = %d, want 100000", got)
	}

	// Write off.
	if err := svc.RecordInvoiceWriteOff(ctx, inv); err != nil {
		t.Fatalf("write-off: %v", err)
	}
	if got := balance(2100); got != 0 {
		t.Errorf("Deferred after write-off = %d, want 0", got)
	}
	if got := balance(2200); got != 0 {
		t.Errorf("Tax Payable after write-off = %d, want 0", got)
	}
	// AR is credit-negative of the asset view here: credits(=118000 reversal) −
	// debits(=118000 issue) == 0 in the signed sum.
	var arNet int64
	if err := conn.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(CASE WHEN t.debit_account_id=a.id THEN t.amount ELSE -t.amount END),0)
		FROM ledger_accounts a
		JOIN ledger_transactions t ON t.credit_account_id=a.id OR t.debit_account_id=a.id
		WHERE a.tenant_id=$1 AND a.code=1100`, tenantID).Scan(&arNet); err != nil {
		t.Fatalf("ar balance: %v", err)
	}
	if arNet != 0 {
		t.Errorf("AR after write-off = %d, want 0", arNet)
	}

	// Idempotent: second call is a no-op.
	if err := svc.RecordInvoiceWriteOff(ctx, inv); err != nil {
		t.Fatalf("second write-off: %v", err)
	}
	var count int
	if err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ledger_transactions WHERE reference_id=$1 AND code=$2`,
		inv.ID, int(domain.LedgerCodeInvoiceWriteOff)).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Errorf("write-off legs = %d, want exactly 1", count)
	}

	// Late recovery: the customer pays after all (stale checkout link / late
	// bank transfer — MarkPaid deliberately allows uncollectible → paid). The
	// recovery must re-establish AR, Deferred, and Tax Payable (codes 24/25,
	// exact mirror of 22/23) so the payment's cash leg settles a real
	// receivable instead of driving AR negative.
	if err := svc.RecordWriteOffRecovery(ctx, inv); err != nil {
		t.Fatalf("write-off recovery: %v", err)
	}
	if got := balance(2100); got != 100000 {
		t.Errorf("Deferred after recovery = %d, want 100000 (re-established)", got)
	}
	if got := balance(2200); got != 18000 {
		t.Errorf("Tax Payable after recovery = %d, want 18000 (re-established)", got)
	}
	arBalance := func() int64 {
		var n int64
		if err := conn.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(CASE WHEN t.debit_account_id=a.id THEN t.amount ELSE -t.amount END),0)
			FROM ledger_accounts a
			JOIN ledger_transactions t ON t.credit_account_id=a.id OR t.debit_account_id=a.id
			WHERE a.tenant_id=$1 AND a.code=1100`, tenantID).Scan(&n); err != nil {
			t.Fatalf("ar balance: %v", err)
		}
		return n
	}
	if got := arBalance(); got != 118000 {
		t.Errorf("AR after recovery = %d, want 118000 (receivable re-established)", got)
	}

	// Idempotent: a redelivered webhook can't double-post the recovery.
	if err := svc.RecordWriteOffRecovery(ctx, inv); err != nil {
		t.Fatalf("second recovery: %v", err)
	}
	var recCount int
	if err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ledger_transactions WHERE reference_id=$1 AND code=$2`,
		inv.ID, int(domain.LedgerCodeWriteOffRecovery)).Scan(&recCount); err != nil {
		t.Fatalf("recovery count: %v", err)
	}
	if recCount != 1 {
		t.Errorf("recovery legs = %d, want exactly 1", recCount)
	}

	// The normal payment cash leg then zeroes AR — the end state of the full
	// write-off → pay-after-all arc is identical to a plain paid invoice.
	if err := svc.RecordPayment(ctx, inv); err != nil {
		t.Fatalf("record payment: %v", err)
	}
	if got := arBalance(); got != 0 {
		t.Errorf("AR after payment = %d, want 0", got)
	}
	if got := balance(2100); got != 100000 {
		t.Errorf("Deferred after payment = %d, want 100000 (awaiting recognition)", got)
	}

	// A never-written-off invoice is a no-op for recovery.
	other := &domain.Invoice{
		ID: uuid.New(), TenantID: tenantID, CustomerID: customerID,
		SubscriptionID: &subID, InvoiceNumber: "INV-WOFF-2", Currency: "INR",
		Subtotal: 50000, TaxAmount: 9000, Total: 59000, CreatedAt: time.Now(),
	}
	if err := svc.RecordInvoice(ctx, other); err != nil {
		t.Fatalf("record other invoice: %v", err)
	}
	if err := svc.RecordWriteOffRecovery(ctx, other); err != nil {
		t.Fatalf("recovery on never-written-off invoice: %v", err)
	}
	var otherRec int
	if err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ledger_transactions WHERE reference_id=$1 AND code=$2`,
		other.ID, int(domain.LedgerCodeWriteOffRecovery)).Scan(&otherRec); err != nil {
		t.Fatalf("other recovery count: %v", err)
	}
	if otherRec != 0 {
		t.Errorf("recovery legs on never-written-off invoice = %d, want 0", otherRec)
	}
}
