package db

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func openGSTR1TestDB(t *testing.T) *sql.DB {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed GSTR-1 export test")
	}
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// seedGSTCustomer inserts a customer with an optional GST identity. An empty
// gstin models an unregistered (B2C) buyer.
func seedGSTCustomer(t *testing.T, conn *sql.DB, tenantID uuid.UUID, gstin, placeOfSupply string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	if _, err := conn.ExecContext(context.Background(),
		`INSERT INTO customers (id, tenant_id, email, ledger_account_id, gstin, place_of_supply, created_at)
		 VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, NOW())`,
		id, tenantID, id.String()[:8]+"@t.com", uuid.New(), gstin, placeOfSupply); err != nil {
		t.Fatalf("seed gst customer: %v", err)
	}
	return id
}

// seedGSTInvoice inserts an invoice with an explicit GST split, status, HSN and
// issue date so the export query's filters (status, period) can be exercised.
func seedGSTInvoice(t *testing.T, conn *sql.DB, tenantID, customerID uuid.UUID, number, status, hsn string, subtotal, igst, cgst, sgst int64, createdAt time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	total := subtotal + igst + cgst + sgst
	if _, err := conn.ExecContext(context.Background(),
		`INSERT INTO invoices (id, tenant_id, customer_id, currency, subtotal, tax_amount, total, amount_paid, credit_applied,
		        igst_amount, cgst_amount, sgst_amount, hsn_code, status, invoice_number, created_at, due_date)
		 VALUES ($1, $2, $3, 'INR', $4, $5, $6, 0, 0, $7, $8, $9, $10, $11, $12, $13, $13)`,
		id, tenantID, customerID, subtotal, igst+cgst+sgst, total, igst, cgst, sgst, hsn, status, number, createdAt); err != nil {
		t.Fatalf("seed gst invoice: %v", err)
	}
	return id
}

func seedRefundCreditNote(t *testing.T, conn *sql.DB, tenantID, customerID, invoiceID uuid.UUID, reference string, amount int64, createdAt time.Time) {
	t.Helper()
	if _, err := conn.ExecContext(context.Background(),
		`INSERT INTO credit_notes (id, tenant_id, customer_id, invoice_id, reference, amount, balance, currency, status, type, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, 0, 'INR', 'issued', 'refund', $7)`,
		uuid.New(), tenantID, customerID, invoiceID, reference, amount, createdAt); err != nil {
		t.Fatalf("seed refund credit note: %v", err)
	}
}

// TestGSTR1Export_QueriesAndReconciliation is the objective (no-CA) gate for the
// GSTR-1 read layer: the queries pull exactly the finalized invoices of the
// requested month, split registered vs unregistered buyers, derive a credit
// note's reversed tax proportionally, and — the reconciliation — the return's
// taxable value + tax equals the gross the same invoices book to the ledger
// (Revenue + Tax Payable = AR).
func TestGSTR1Export_QueriesAndReconciliation(t *testing.T) {
	conn := openGSTR1TestDB(t)
	repo := NewInvoiceRepository(conn).(*InvoiceRepository)
	ctx := context.Background()

	tenantID, _ := seedCreditAppTenantCustomer(t, conn)
	registered := seedGSTCustomer(t, conn, tenantID, "27AAAAA0000A1Z5", "27")
	unregistered := seedGSTCustomer(t, conn, tenantID, "", "27")

	july := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	june := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)

	// Included: an open B2B invoice and a paid B2C invoice, both in July.
	b2bInv := seedGSTInvoice(t, conn, tenantID, registered, "GST-B2B-1", "open", "9983", 100000, 0, 9000, 9000, july)
	seedGSTInvoice(t, conn, tenantID, unregistered, "GST-B2C-1", "paid", "9983", 10000, 0, 900, 900, july)
	// Excluded: a draft, a void (both July) and a finalized invoice from June.
	seedGSTInvoice(t, conn, tenantID, registered, "GST-DRAFT-1", "draft", "9983", 5000, 0, 450, 450, july)
	seedGSTInvoice(t, conn, tenantID, registered, "GST-VOID-1", "void", "9983", 7000, 0, 630, 630, july)
	seedGSTInvoice(t, conn, tenantID, registered, "GST-JUN-1", "open", "9983", 8000, 0, 720, 720, june)

	// A refund of half the B2B invoice (total 118000 → 59000) issued in July: its
	// reversed tax is derived proportionally (9000 → 4500 each; taxable 50000).
	seedRefundCreditNote(t, conn, tenantID, registered, b2bInv, "CN-1", 59000, july)

	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	invoices, err := repo.GetGSTR1Invoices(ctx, tenantID, start, end)
	if err != nil {
		t.Fatalf("GetGSTR1Invoices: %v", err)
	}
	if len(invoices) != 2 {
		t.Fatalf("GetGSTR1Invoices returned %d rows, want 2 (draft/void/June excluded): %+v", len(invoices), invoices)
	}

	var totalTaxable, totalTax, totalGross int64
	var sawRegistered, sawUnregistered bool
	for _, inv := range invoices {
		tax := inv.IGST + inv.CGST + inv.SGST
		totalTaxable += inv.TaxableValue
		totalTax += tax
		totalGross += inv.TaxableValue + tax
		switch inv.BuyerGSTIN {
		case "27AAAAA0000A1Z5":
			sawRegistered = true
			if inv.PlaceOfSupply != "27" || inv.TaxableValue != 100000 || inv.CGST != 9000 || inv.SGST != 9000 {
				t.Errorf("B2B row = %+v, want pos 27 / taxable 100000 / cgst,sgst 9000", inv)
			}
		case "":
			sawUnregistered = true
			if inv.TaxableValue != 10000 || inv.CGST != 900 || inv.SGST != 900 {
				t.Errorf("B2C row = %+v, want taxable 10000 / cgst,sgst 900", inv)
			}
		default:
			t.Errorf("unexpected buyer GSTIN %q", inv.BuyerGSTIN)
		}
	}
	if !sawRegistered || !sawUnregistered {
		t.Errorf("expected one registered and one unregistered invoice; registered=%v unregistered=%v", sawRegistered, sawUnregistered)
	}

	// Reconciliation: what GSTR-1 reports as outward supply must equal what the
	// invoices book to the ledger. Revenue = Σ taxable, Tax Payable = Σ tax, and
	// their sum = Σ AR (invoice totals) = 118000 + 11800.
	if totalTaxable != 110000 {
		t.Errorf("Σ taxable = %d, want 110000 (Revenue booked)", totalTaxable)
	}
	if totalTax != 19800 {
		t.Errorf("Σ tax = %d, want 19800 (Tax Payable booked)", totalTax)
	}
	if totalGross != 129800 {
		t.Errorf("Σ (taxable+tax) = %d, want 129800 (AR booked)", totalGross)
	}

	// Credit notes: the proportional split reverses exactly half the B2B tax.
	notes, err := repo.GetGSTR1CreditNotes(ctx, tenantID, start, end)
	if err != nil {
		t.Fatalf("GetGSTR1CreditNotes: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("GetGSTR1CreditNotes returned %d rows, want 1: %+v", len(notes), notes)
	}
	cn := notes[0]
	if cn.BuyerGSTIN != "27AAAAA0000A1Z5" || cn.OriginalInvoiceNumber != "GST-B2B-1" {
		t.Errorf("credit note buyer/origin = %q/%q, want the B2B GSTIN / GST-B2B-1", cn.BuyerGSTIN, cn.OriginalInvoiceNumber)
	}
	if cn.TaxableValue != 50000 || cn.CGST != 4500 || cn.SGST != 4500 || cn.IGST != 0 {
		t.Errorf("credit note split = taxable %d / igst %d / cgst %d / sgst %d, want 50000/0/4500/4500", cn.TaxableValue, cn.IGST, cn.CGST, cn.SGST)
	}
}
