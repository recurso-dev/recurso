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

// TestCreditNoteTaxBreakdown_RoundTrip_Postgres proves B2 (ENG-196) end to
// end against real Postgres: an invoice-linked credit note records the
// invoice's tax sliced proportionally (the same math GSTR-1 CDNR uses), the
// columns ROUND-TRIP through the repository (the #373 lesson — verify
// persistence, not in-memory state), and the document view model renders a
// statutory-grade CDN from the read-back row. A standalone goodwill credit
// records no breakdown and stays gross-only.
func TestCreditNoteTaxBreakdown_RoundTrip_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed credit-note breakdown test")
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
	tenantID := seedRevRecTenant(t, conn)
	run := uuid.New().String()[:8]

	customerID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, name, ledger_account_id, created_at) VALUES ($1,$2,$3,'CDN Cust',$4,NOW())`,
		customerID, tenantID, "cdn-"+run+"@t.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}

	// A paid GST invoice: gross 118000 = net 100000 + IGST 18000 (inter-state).
	invID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO invoices (id, tenant_id, customer_id, currency, subtotal, total, tax_amount, igst_amount, tax_type, hsn_code, amount_paid, status, invoice_number, created_at, due_date)
		 VALUES ($1,$2,$3,'INR',100000,118000,18000,18000,'inter_state','998314',118000,'paid',$4,NOW(),NOW())`,
		invID, tenantID, customerID, "INV-CDN-"+run); err != nil {
		t.Fatalf("seed invoice: %v", err)
	}

	tctx := context.WithValue(ctx, domain.TenantIDKey, tenantID)
	cnSvc := NewCreditNoteService(db.NewCreditNoteRepository(dbx), db.NewCustomerRepository(dbx), db.NewInvoiceRepository(conn), nil)
	cnSvc.SetLedgerService(NewLedgerService(nil, db.NewLedgerRepository(conn)))

	// Half-invoice adjustment credit linked to the invoice: 59000 gross must
	// split into 50000 taxable + 9000 IGST (proportional slice of 18000).
	cn, err := cnSvc.Create(tctx, tenantID, uuid.Nil, "", domain.CreateCreditNoteRequest{
		CustomerID: customerID, InvoiceID: &invID, Amount: 59000, Currency: "INR", Reason: "partial credit",
	})
	if err != nil {
		t.Fatalf("Create invoice-linked credit: %v", err)
	}

	// Read BACK through the repository — the columns must round-trip.
	got, err := db.NewCreditNoteRepository(dbx).GetByID(ctx, cn.ID, tenantID)
	if err != nil || got == nil {
		t.Fatalf("GetByID: %v (nil=%v)", err, got == nil)
	}
	if got.Subtotal != 50000 || got.TaxAmount != 9000 || got.IGSTAmount != 9000 {
		t.Errorf("persisted breakdown = subtotal %d / tax %d / igst %d, want 50000/9000/9000",
			got.Subtotal, got.TaxAmount, got.IGSTAmount)
	}
	if got.CGSTAmount != 0 || got.SGSTAmount != 0 {
		t.Errorf("inter-state credit must carry no CGST/SGST, got %d/%d", got.CGSTAmount, got.SGSTAmount)
	}
	if got.TaxType != "inter_state" || got.HSNCode != "998314" {
		t.Errorf("tax_type/hsn = %q/%q, want inter_state/998314", got.TaxType, got.HSNCode)
	}

	// The document renders a CDN from the read-back row.
	data := cnTestService().BuildCreditNoteData(got, nil, "INV-CDN-"+run)
	if !data.HasTaxBreakdown || data.IGST == "" || data.CGST != "" {
		t.Errorf("document breakdown wrong: has=%v IGST=%q CGST=%q", data.HasTaxBreakdown, data.IGST, data.CGST)
	}

	// A standalone goodwill credit (no invoice) records no breakdown.
	plain, err := cnSvc.Create(tctx, tenantID, uuid.Nil, "", domain.CreateCreditNoteRequest{
		CustomerID: customerID, Amount: 3000, Currency: "INR", Reason: "goodwill",
	})
	if err != nil {
		t.Fatalf("Create standalone credit: %v", err)
	}
	gotPlain, err := db.NewCreditNoteRepository(dbx).GetByID(ctx, plain.ID, tenantID)
	if err != nil || gotPlain == nil {
		t.Fatalf("GetByID standalone: %v", err)
	}
	if gotPlain.Subtotal != 0 || gotPlain.TaxAmount != 0 || gotPlain.TaxType != "" {
		t.Errorf("standalone credit must stay gross-only, got subtotal %d tax %d type %q",
			gotPlain.Subtotal, gotPlain.TaxAmount, gotPlain.TaxType)
	}
}
