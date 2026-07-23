package service

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/recurso-dev/recurso/internal/adapter/db"
)

// TestCreditStatement_BalanceMatchesApplier is the Increment-1 oracle: the
// statement's spendable balance must equal the credit applier's own view
// (SumApplicableAdjustments), grants list every note, and the draw-down history
// surfaces credit_note_applications. Fully-drawn ('used') and refund-type notes
// are excluded from spendable balance but still appear as grants.
func TestCreditStatement_BalanceMatchesApplier(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed credit-statement test")
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
		tenantID, "CS-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	customerID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1,$2,$3,$4,NOW())`,
		customerID, tenantID, customerID.String()[:8]+"@t.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}

	cn := func(amount, balance int64, status, ctype string) uuid.UUID {
		id := uuid.New()
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO credit_notes (id, tenant_id, customer_id, amount, balance, currency, status, reason, type, refund_status, created_at, updated_at)
			 VALUES ($1,$2,$3,$4,$5,'USD',$6,'test',$7,'none',NOW(),NOW())`,
			id, tenantID, customerID, amount, balance, status, ctype); err != nil {
			t.Fatalf("seed credit note: %v", err)
		}
		return id
	}
	cn(5000, 5000, "issued", "adjustment")        // spendable 5000
	cn2 := cn(3000, 1000, "issued", "adjustment") // spendable 1000 (partly drawn)
	cn(2000, 0, "used", "adjustment")             // fully drawn — grant only
	cn(4000, 0, "issued", "refund")               // refund — grant only, not spendable

	// One draw-down: cn2 applied 2000 to an invoice.
	invID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO invoices (id, tenant_id, customer_id, currency, subtotal, total, amount_paid, credit_applied, status, invoice_number, created_at, due_date)
		 VALUES ($1,$2,$3,'USD',3000,3000,0,2000,'open',$4,NOW(),NOW())`,
		invID, tenantID, customerID, "INV-CS-"+invID.String()[:6]); err != nil {
		t.Fatalf("seed invoice: %v", err)
	}
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO credit_note_applications (id, tenant_id, credit_note_id, invoice_id, amount, created_at)
		 VALUES ($1,$2,$3,$4,2000,NOW())`,
		uuid.New(), tenantID, cn2, invID); err != nil {
		t.Fatalf("seed application: %v", err)
	}

	repo := db.NewCreditNoteRepository(dbx)
	svc := NewCreditNoteService(repo, nil, nil, nil)

	stmt, err := svc.GetCreditStatement(ctx, tenantID, customerID)
	if err != nil {
		t.Fatalf("GetCreditStatement: %v", err)
	}

	// Balance: 5000 + 1000 = 6000, and it must equal the applier's own view.
	if len(stmt.Balances) != 1 || stmt.Balances[0].Currency != "USD" || stmt.Balances[0].Balance != 6000 {
		t.Fatalf("balances = %+v, want one USD line of 6000", stmt.Balances)
	}
	applierView, err := repo.SumApplicableAdjustments(ctx, tenantID, customerID, nil, "USD")
	if err != nil {
		t.Fatalf("SumApplicableAdjustments: %v", err)
	}
	if stmt.Balances[0].Balance != applierView {
		t.Fatalf("statement balance %d != applier view %d — the statement can be spent-wrong",
			stmt.Balances[0].Balance, applierView)
	}

	if len(stmt.Grants) != 4 {
		t.Fatalf("grants = %d, want 4 (every note incl used + refund)", len(stmt.Grants))
	}
	if len(stmt.Applications) != 1 || stmt.Applications[0].Amount != 2000 || stmt.Applications[0].InvoiceNumber == "" {
		t.Fatalf("applications = %+v, want one draw-down of 2000 with an invoice number", stmt.Applications)
	}
	if len(stmt.Summary) != 1 {
		t.Fatalf("summary lines = %d, want 1 (USD)", len(stmt.Summary))
	}
	s := stmt.Summary[0]
	// total_issued = adjustment amounts only: 5000+3000+2000 = 10000 (refund excluded).
	if s.TotalIssued != 10000 || s.TotalApplied != 2000 || s.CurrentBalance != 6000 {
		t.Fatalf("summary = %+v, want issued 10000 / applied 2000 / balance 6000", s)
	}
}
