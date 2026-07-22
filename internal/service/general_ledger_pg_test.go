package service

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestGeneralLedger_Postgres proves the GL export flattens each posted
// transaction with both account codes/names and is tenant-scoped: a one-off
// invoice (DR AR / CR Revenue) then its payment (DR Cash / CR AR) produce rows
// carrying the right accounts and amounts, and another tenant's postings never
// appear.
func TestGeneralLedger_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed general-ledger test")
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

	seedTenant := func(tag string) uuid.UUID {
		id := uuid.New()
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`,
			id, tag+"-"+id.String()[:8], id.String()[:8]+"@t.com"); err != nil {
			t.Fatalf("seed tenant: %v", err)
		}
		return id
	}

	tenantID := seedTenant("GL")
	svc := NewLedgerService(nil, db.NewLedgerRepository(conn))

	inv := &domain.Invoice{
		ID: uuid.New(), TenantID: tenantID, CustomerID: uuid.New(),
		InvoiceNumber: "GL-1", Total: 10000, Currency: "USD",
	}
	if err := svc.RecordInvoice(ctx, inv); err != nil {
		t.Fatalf("RecordInvoice: %v", err)
	}
	if err := svc.RecordPayment(ctx, inv); err != nil {
		t.Fatalf("RecordPayment: %v", err)
	}

	// A different tenant with its own invoice — must not appear in tenantID's GL.
	other := seedTenant("GLother")
	otherInv := &domain.Invoice{
		ID: uuid.New(), TenantID: other, CustomerID: uuid.New(),
		InvoiceNumber: "GLO-1", Total: 55555, Currency: "USD",
	}
	if err := svc.RecordInvoice(ctx, otherInv); err != nil {
		t.Fatalf("RecordInvoice(other): %v", err)
	}

	rows, err := svc.GeneralLedger(ctx, tenantID, nil)
	if err != nil {
		t.Fatalf("GeneralLedger: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("expected general-ledger rows, got none")
	}

	var sawInvoicePosting, sawPaymentPosting bool
	for _, r := range rows {
		if r.Amount == 55555 {
			t.Fatalf("cross-tenant leak: found other tenant's 55555 posting in GL")
		}
		// Invoice: DR Accounts Receivable / CR Revenue.
		if r.DebitAccountCode == domain.AccountCodeAR && r.CreditAccountCode == domain.AccountCodeRevenue && r.Amount == 10000 {
			sawInvoicePosting = true
		}
		// Payment: DR Cash / CR Accounts Receivable.
		if r.DebitAccountCode == domain.AccountCodeCash && r.CreditAccountCode == domain.AccountCodeAR && r.Amount == 10000 {
			sawPaymentPosting = true
		}
		if r.DebitAccountName == "" || r.CreditAccountName == "" {
			t.Errorf("row %s missing an account name (debit=%q credit=%q)", r.TransactionID, r.DebitAccountName, r.CreditAccountName)
		}
	}
	if !sawInvoicePosting {
		t.Error("did not find the DR AR / CR Revenue invoice posting")
	}
	if !sawPaymentPosting {
		t.Error("did not find the DR Cash / CR AR payment posting")
	}
}
