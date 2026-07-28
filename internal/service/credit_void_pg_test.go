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

// TestVoidCreditNote_WritesOffAndPostsLedger is the void money-path oracle: an
// operator voiding an issued adjustment credit zeroes its balance, flips it to
// 'void', and posts a balanced reversal (DR Customer Credit 2300 / CR Credits &
// Adjustments 5100) under code 20 at the unspent amount. Refund notes and
// non-issued notes are rejected, and a second void is a no-op.
func TestVoidCreditNote_WritesOffAndPostsLedger(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed credit-void test")
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

	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "CV-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	customerID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1,$2,$3,$4,NOW())`,
		customerID, tenantID, customerID.String()[:8]+"@t.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}

	seed := func(balance int64, status, cnType string) uuid.UUID {
		id := uuid.New()
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO credit_notes (id, tenant_id, customer_id, amount, balance, currency, status, reason, type, refund_status, created_at, updated_at)
			 VALUES ($1,$2,$3,$4,$5,'USD',$6,'test',$7,'none',NOW(),NOW())`,
			id, tenantID, customerID, balance, balance, status, cnType); err != nil {
			t.Fatalf("seed credit note: %v", err)
		}
		return id
	}

	ledger := NewLedgerService(nil, db.NewLedgerRepository(conn))
	svc := NewCreditNoteService(db.NewCreditNoteRepository(dbx), nil, nil, nil)
	svc.SetLedgerService(ledger)

	actor := uuid.New()

	// Happy path: issued adjustment credit → voided, balance written off.
	voidID := seed(5000, "issued", "adjustment")
	cn, err := svc.Void(ctx, tenantID, voidID, actor)
	if err != nil {
		t.Fatalf("Void: %v", err)
	}
	if cn.Status != domain.CreditNoteStatusVoid || cn.Balance != 0 {
		t.Fatalf("returned note = (status %s, balance %d), want (void, 0)", cn.Status, cn.Balance)
	}

	var bal int64
	var status string
	if err := conn.QueryRowContext(ctx, `SELECT balance, status FROM credit_notes WHERE id=$1`, voidID).Scan(&bal, &status); err != nil {
		t.Fatalf("read note: %v", err)
	}
	if bal != 0 || status != "void" {
		t.Fatalf("persisted note = (balance %d, status %s), want (0, void)", bal, status)
	}

	// The reversal leg: code 20, DR 2300 / CR 5100, amount 5000.
	var amount int64
	var drCode, crCode int
	if err := conn.QueryRowContext(ctx,
		`SELECT t.amount, da.code, ca.code
		   FROM ledger_transactions t
		   JOIN ledger_accounts da ON da.id = t.debit_account_id
		   JOIN ledger_accounts ca ON ca.id = t.credit_account_id
		  WHERE t.reference_id = $1 AND t.code = $2`,
		voidID, domain.LedgerCodeCreditVoid).Scan(&amount, &drCode, &crCode); err != nil {
		t.Fatalf("read void ledger leg: %v", err)
	}
	if amount != 5000 {
		t.Fatalf("void amount = %d, want 5000", amount)
	}
	if drCode != domain.AccountCodeCustomerCredit || crCode != domain.AccountCodeCreditsIssued {
		t.Fatalf("void legs DR %d / CR %d, want DR %d / CR %d",
			drCode, crCode, domain.AccountCodeCustomerCredit, domain.AccountCodeCreditsIssued)
	}

	// Idempotent: voiding again is a no-op returning the void note.
	again, err := svc.Void(ctx, tenantID, voidID, actor)
	if err != nil || again.Status != domain.CreditNoteStatusVoid {
		t.Fatalf("second void = (%v, %v), want (void, nil)", again.Status, err)
	}

	// A refund note cannot be voided.
	refundID := seed(4000, "issued", "refund")
	if _, err := svc.Void(ctx, tenantID, refundID, actor); err == nil {
		t.Fatalf("voiding a refund note should error")
	}

	// A pending note cannot be voided (use reject instead).
	pendingID := seed(3000, "pending_approval", "adjustment")
	if _, err := svc.Void(ctx, tenantID, pendingID, actor); err == nil {
		t.Fatalf("voiding a pending note should error")
	}
}
