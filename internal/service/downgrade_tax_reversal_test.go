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

// TestDowngradeTaxReversal_Postgres proves the ENG-191c ledger split: a
// mid-period downgrade credit (gross) is booked as NET against Deferred and TAX
// against Tax Payable, so Deferred drains only the net it holds (never negative)
// and the output GST is reversed — Customer Credit still receives the full gross.
func TestDowngradeTaxReversal_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed downgrade-tax test")
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
		tenantID, "Dgrd-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	svc := NewLedgerService(nil, db.NewLedgerRepository(conn))

	// GST subscription invoice: gross 118000, tax 18000, net 100000.
	subID := uuid.New()
	inv := &domain.Invoice{
		ID: uuid.New(), TenantID: tenantID, CustomerID: uuid.New(),
		SubscriptionID: &subID, InvoiceNumber: "DG-1",
		Total: 118000, TaxAmount: 18000, Currency: "INR",
	}
	if err := svc.RecordInvoice(ctx, inv); err != nil {
		t.Fatalf("RecordInvoice: %v", err)
	}
	// Deferred holds net 100000, Tax Payable holds 18000.

	// Mid-period downgrade credit of gross 59000 = net 50000 + tax 9000.
	cnID := uuid.New()
	if _, err := svc.RecordDowngradeCredit(ctx, tenantID, nil, cnID, 50000, "downgrade credit (net)"); err != nil {
		t.Fatalf("RecordDowngradeCredit: %v", err)
	}
	if _, err := svc.RecordDowngradeTaxReversal(ctx, tenantID, nil, cnID, 9000, "downgrade GST reversal"); err != nil {
		t.Fatalf("RecordDowngradeTaxReversal: %v", err)
	}

	tb, err := svc.GetTrialBalance(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetTrialBalance: %v", err)
	}

	// Deferred drained by the NET only: 100000 - 50000 = 50000 (never negative).
	if got := balanceForCode(tb, domain.AccountCodeDeferredRevenue); got != 50000 {
		t.Errorf("Deferred after downgrade = %d, want 50000 (net drain, not gross)", got)
	}
	// Tax Payable reversed by the tax portion: 18000 - 9000 = 9000.
	if got := balanceForCode(tb, domain.AccountCodeTaxPayable); got != 9000 {
		t.Errorf("Tax Payable after downgrade = %d, want 9000 (GST on reduced supply reversed)", got)
	}
	// Customer Credit received the full gross: 50000 + 9000 = 59000.
	if got := balanceForCode(tb, domain.AccountCodeCustomerCredit); got != 59000 {
		t.Errorf("Customer Credit after downgrade = %d, want 59000 (gross credit)", got)
	}
	if !tb.Balanced {
		t.Errorf("trial balance not balanced after downgrade: D%d/C%d", tb.TotalDebits, tb.TotalCredits)
	}
	for _, l := range tb.Lines {
		if l.Abnormal {
			t.Errorf("account %d (%s) abnormal after downgrade: balance=%d", l.Code, l.Name, l.Balance)
		}
	}
}
