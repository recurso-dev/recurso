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

// TestRefundTaxPortion pins the proportional GST slice of a refund.
func TestRefundTaxPortion(t *testing.T) {
	cases := []struct {
		name                           string
		refund, invoiceTax, invoiceTot int64
		want                           int64
	}{
		{"full refund reverses full tax", 118000, 18000, 118000, 18000},
		{"half refund reverses half tax", 59000, 18000, 118000, 9000},
		{"no tax on invoice", 100000, 0, 100000, 0},
		{"zero refund", 0, 18000, 118000, 0},
		{"rounds to nearest paisa", 100, 18000, 118000, 15}, // 100*18000/118000 = 15.25 -> 15
		{"never over-reverses past invoice tax", 200000, 18000, 118000, 18000},
	}
	for _, c := range cases {
		if got := refundTaxPortion(c.refund, c.invoiceTax, c.invoiceTot); got != c.want {
			t.Errorf("%s: refundTaxPortion(%d,%d,%d) = %d, want %d",
				c.name, c.refund, c.invoiceTax, c.invoiceTot, got, c.want)
		}
	}
}

// TestRefundTaxReversal_Postgres proves the ledger effect end-to-end: a GST
// subscription invoice books Tax Payable, and a full refund reverses it back to
// zero via RecordRefundTaxReversal — leaving the trial balance balanced with no
// abnormal accounts (the ENG-191b fix).
func TestRefundTaxReversal_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed refund-tax test")
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
		tenantID, "RefTax-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	svc := NewLedgerService(nil, db.NewLedgerRepository(conn))

	// GST subscription invoice: gross 118000, tax 18000.
	subID := uuid.New()
	inv := &domain.Invoice{
		ID: uuid.New(), TenantID: tenantID, CustomerID: uuid.New(),
		SubscriptionID: &subID, InvoiceNumber: "REF-1",
		Total: 118000, TaxAmount: 18000, Currency: "INR",
	}
	if err := svc.RecordInvoice(ctx, inv); err != nil {
		t.Fatalf("RecordInvoice: %v", err)
	}
	if err := svc.RecordPayment(ctx, inv); err != nil {
		t.Fatalf("RecordPayment: %v", err)
	}

	// Tax Payable should carry the 18000 GST after invoicing.
	tb, err := svc.GetTrialBalance(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetTrialBalance: %v", err)
	}
	if got := balanceForCode(tb, domain.AccountCodeTaxPayable); got != 18000 {
		t.Fatalf("Tax Payable after invoice = %d, want 18000", got)
	}

	// Full refund: cash refund + deferred reversal + tax reversal.
	cnID := uuid.New()
	if err := svc.RecordRefund(ctx, tenantID, nil, cnID, 118000, "refund"); err != nil {
		t.Fatalf("RecordRefund: %v", err)
	}
	if _, err := svc.RecordDeferredRefundReversal(ctx, tenantID, nil, cnID, 100000, "deferred reversal"); err != nil {
		t.Fatalf("RecordDeferredRefundReversal: %v", err)
	}
	taxPortion := refundTaxPortion(118000, inv.TaxAmount, inv.Total) // 18000
	if _, err := svc.RecordRefundTaxReversal(ctx, tenantID, nil, cnID, taxPortion, "GST reversal"); err != nil {
		t.Fatalf("RecordRefundTaxReversal: %v", err)
	}

	tb, err = svc.GetTrialBalance(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetTrialBalance (post-refund): %v", err)
	}
	if got := balanceForCode(tb, domain.AccountCodeTaxPayable); got != 0 {
		t.Errorf("Tax Payable after full refund = %d, want 0 (GST liability reversed)", got)
	}
	if !tb.Balanced {
		t.Errorf("trial balance not balanced after refund: D%d/C%d", tb.TotalDebits, tb.TotalCredits)
	}
	for _, l := range tb.Lines {
		if l.Abnormal {
			t.Errorf("account %d (%s) abnormal after refund: balance=%d", l.Code, l.Name, l.Balance)
		}
	}
}

func balanceForCode(tb *domain.TrialBalance, code int) int64 {
	for _, l := range tb.Lines {
		if l.Code == code {
			return l.Balance
		}
	}
	return 0
}
