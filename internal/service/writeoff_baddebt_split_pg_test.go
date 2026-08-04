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

// fixedRecognizedReader stubs the recognized-revenue lookup so the accrual
// write-off split can be exercised without standing up a full recognition
// schedule.
type fixedRecognizedReader struct{ recognized int64 }

func (f fixedRecognizedReader) SumRecognizedByInvoice(_ context.Context, _, _ uuid.UUID) (int64, error) {
	return f.recognized, nil
}

// TestWriteOffBadDebtSplit_Postgres proves the accrual-epic write-off split
// (#466/#477): when part of an invoice's revenue is already recognized, the
// write-off expenses that part as Bad Debt (code 26) and reverses only the
// still-deferred part from Deferred (code 22). Recovery then inverts BOTH.
// Books stay balanced throughout.
func TestWriteOffBadDebtSplit_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed bad-debt-split test")
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
		tenantID, "BD-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	// Recognized 40,000 of the 100,000 pre-tax (accrual: schedule built at
	// issuance recognized part of the revenue before the invoice went bad).
	svc := NewLedgerService(nil, db.NewLedgerRepository(conn))
	svc.SetRecognizedReader(fixedRecognizedReader{recognized: 40000})

	customerID := uuid.New()
	subID := uuid.New()
	inv := &domain.Invoice{
		ID: uuid.New(), TenantID: tenantID, CustomerID: customerID, SubscriptionID: &subID,
		InvoiceNumber: "INV-BD-1", Currency: "INR",
		Subtotal: 100000, TaxAmount: 18000, Total: 118000, CreatedAt: time.Now(),
	}
	if err := svc.RecordInvoice(ctx, inv); err != nil {
		t.Fatalf("record invoice: %v", err)
	}

	bal := func(code int) int64 { // signed on the account's normal side via credit-minus-debit
		var n int64
		if err := conn.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(CASE WHEN t.credit_account_id=a.id THEN t.amount ELSE -t.amount END),0)
			FROM ledger_accounts a
			JOIN ledger_transactions t ON t.credit_account_id=a.id OR t.debit_account_id=a.id
			WHERE a.tenant_id=$1 AND a.code=$2`, tenantID, code).Scan(&n); err != nil {
			t.Fatalf("bal(%d): %v", code, err)
		}
		return n
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
	legs := func(code int) int {
		var n int
		if err := conn.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM ledger_transactions WHERE reference_id=$1 AND code=$2`, inv.ID, code).Scan(&n); err != nil {
			t.Fatalf("legs(%d): %v", code, err)
		}
		return n
	}

	// After issue: Deferred (2100) = 100000, AR (1100) debit = 118000.
	if got := bal(2100); got != 100000 {
		t.Fatalf("Deferred after issue = %d, want 100000", got)
	}

	// Write off — split: deferred 60000 (code 22), bad debt 40000 (code 26).
	if err := svc.RecordInvoiceWriteOff(ctx, inv); err != nil {
		t.Fatalf("write-off: %v", err)
	}
	if legs(int(domain.LedgerCodeInvoiceWriteOff)) != 1 || legs(int(domain.LedgerCodeBadDebtWriteOff)) != 1 {
		t.Fatalf("want one code-22 and one code-26 leg; got 22=%d 26=%d",
			legs(int(domain.LedgerCodeInvoiceWriteOff)), legs(int(domain.LedgerCodeBadDebtWriteOff)))
	}
	if got := bal(2100); got != 40000 {
		// Deferred reduced by the 60000 deferred portion only (100000-60000).
		t.Errorf("Deferred after write-off = %d, want 40000 (only the deferred portion reversed)", got)
	}
	if got := debitBal(domain.AccountCodeBadDebtExpense); got != 40000 {
		t.Errorf("Bad Debt Expense after write-off = %d, want 40000 (the recognized portion)", got)
	}
	if got := debitBal(1100); got != 0 {
		t.Errorf("AR after write-off = %d, want 0 (fully relieved)", got)
	}
	if got := bal(2200); got != 0 {
		t.Errorf("Tax Payable after write-off = %d, want 0 (reversed)", got)
	}

	// Recovery inverts all three: Deferred back to 100000, Bad Debt back to 0,
	// AR re-established to 118000.
	if err := svc.RecordWriteOffRecovery(ctx, inv); err != nil {
		t.Fatalf("recovery: %v", err)
	}
	if legs(int(domain.LedgerCodeWriteOffRecovery)) != 1 || legs(int(domain.LedgerCodeBadDebtRecovery)) != 1 {
		t.Fatalf("want one code-24 and one code-27 recovery leg; got 24=%d 27=%d",
			legs(int(domain.LedgerCodeWriteOffRecovery)), legs(int(domain.LedgerCodeBadDebtRecovery)))
	}
	if got := bal(2100); got != 100000 {
		t.Errorf("Deferred after recovery = %d, want 100000 (re-established)", got)
	}
	if got := debitBal(domain.AccountCodeBadDebtExpense); got != 0 {
		t.Errorf("Bad Debt Expense after recovery = %d, want 0 (reversed)", got)
	}
	if got := debitBal(1100); got != 118000 {
		t.Errorf("AR after recovery = %d, want 118000 (receivable re-established)", got)
	}

	// Books balanced: total debits == total credits across all postings.
	var dr, cr int64
	if err := conn.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount),0) FROM ledger_transactions t
		JOIN ledger_accounts a ON a.id=t.debit_account_id WHERE a.tenant_id=$1`, tenantID).Scan(&dr); err != nil {
		t.Fatalf("dr: %v", err)
	}
	if err := conn.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount),0) FROM ledger_transactions t
		JOIN ledger_accounts a ON a.id=t.credit_account_id WHERE a.tenant_id=$1`, tenantID).Scan(&cr); err != nil {
		t.Fatalf("cr: %v", err)
	}
	if dr != cr {
		t.Errorf("books unbalanced: debits %d != credits %d", dr, cr)
	}
}
