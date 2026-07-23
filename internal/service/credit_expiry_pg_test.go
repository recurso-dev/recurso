package service

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestExpireDueCredits_WritesOffAndPostsLedger is the Increment-2 money-path
// oracle: the sweep expires ONLY dated credits whose expiry has passed, zeroes
// their balance, flips them to 'expired', and posts a balanced GL write-off
// (DR Customer Credit 2300 / CR Credits & Adjustments 5100) at the written-off
// amount. Future-dated and never-expiring credits are untouched.
func TestExpireDueCredits_WritesOffAndPostsLedger(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed credit-expiry test")
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
		tenantID, "CE-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	customerID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1,$2,$3,$4,NOW())`,
		customerID, tenantID, customerID.String()[:8]+"@t.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}

	cn := func(balance int64, expires *time.Time) uuid.UUID {
		id := uuid.New()
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO credit_notes (id, tenant_id, customer_id, amount, balance, currency, status, reason, type, refund_status, expires_at, created_at, updated_at)
			 VALUES ($1,$2,$3,$4,$4,'USD','issued','test','adjustment','none',$5,NOW(),NOW())`,
			id, tenantID, customerID, balance, expires); err != nil {
			t.Fatalf("seed credit note: %v", err)
		}
		return id
	}
	past := time.Now().Add(-24 * time.Hour)
	future := time.Now().Add(24 * time.Hour)
	dueID := cn(5000, &past)      // expired → written off
	futureID := cn(3000, &future) // not yet due → untouched
	foreverID := cn(2000, nil)    // never expires → untouched

	ledger := NewLedgerService(nil, db.NewLedgerRepository(conn))
	svc := NewCreditNoteService(db.NewCreditNoteRepository(dbx), nil, nil, nil)
	svc.SetLedgerService(ledger)

	n, err := svc.ExpireDueCredits(ctx)
	if err != nil {
		t.Fatalf("ExpireDueCredits: %v", err)
	}
	if n != 1 {
		t.Fatalf("expired %d credits, want 1", n)
	}

	// The due note is written off; the others are untouched.
	check := func(id uuid.UUID, wantBalance int64, wantStatus string) {
		var bal int64
		var status string
		if err := conn.QueryRowContext(ctx, `SELECT balance, status FROM credit_notes WHERE id=$1`, id).Scan(&bal, &status); err != nil {
			t.Fatalf("read note: %v", err)
		}
		if bal != wantBalance || status != wantStatus {
			t.Fatalf("note %s = (balance %d, status %s), want (%d, %s)", id, bal, status, wantBalance, wantStatus)
		}
	}
	check(dueID, 0, "expired")
	check(futureID, 3000, "issued")
	check(foreverID, 2000, "issued")

	// The write-off leg: code 18, referencing the expired note, DR 2300 / CR 5100, amount 5000.
	var amount int64
	var drCode, crCode int
	if err := conn.QueryRowContext(ctx,
		`SELECT t.amount, da.code, ca.code
		   FROM ledger_transactions t
		   JOIN ledger_accounts da ON da.id = t.debit_account_id
		   JOIN ledger_accounts ca ON ca.id = t.credit_account_id
		  WHERE t.reference_id = $1 AND t.code = $2`,
		dueID, domain.LedgerCodeCreditExpiry).Scan(&amount, &drCode, &crCode); err != nil {
		t.Fatalf("read expiry ledger leg: %v", err)
	}
	if amount != 5000 {
		t.Fatalf("write-off amount = %d, want 5000", amount)
	}
	if drCode != domain.AccountCodeCustomerCredit || crCode != domain.AccountCodeCreditsIssued {
		t.Fatalf("write-off legs DR %d / CR %d, want DR %d (Customer Credit) / CR %d (Credits & Adjustments)",
			drCode, crCode, domain.AccountCodeCustomerCredit, domain.AccountCodeCreditsIssued)
	}

	// Idempotent: a second sweep finds nothing (the note is already 'expired').
	n2, err := svc.ExpireDueCredits(ctx)
	if err != nil || n2 != 0 {
		t.Fatalf("second sweep = %d (err %v), want 0", n2, err)
	}
}
