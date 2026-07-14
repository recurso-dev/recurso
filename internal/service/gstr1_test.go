package service

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

func d(day int) time.Time { return time.Date(2026, 1, day, 0, 0, 0, 0, time.UTC) }

// TestBuildGSTR1_BucketsAndTotals exercises the whole assembler: registered
// buyers go to B2B invoice-level (grouped by GSTIN), unregistered to B2CS
// (rate-wise per place of supply), the HSN rollup covers all, and the control
// totals sum every invoice.
func TestBuildGSTR1_BucketsAndTotals(t *testing.T) {
	tenant := uuid.New()
	invs := []domain.GSTR1Invoice{
		// B2B, buyer GST27, intra-state (CGST+SGST), 18% on 100000, HSN 9983.
		{InvoiceNumber: "INV-2", Date: d(2), BuyerGSTIN: "27AAAAA0000A1Z5", PlaceOfSupply: "27", TaxableValue: 100000, CGST: 9000, SGST: 9000, HSNCode: "9983"},
		// Same B2B buyer, another invoice — must group under the same GSTIN.
		{InvoiceNumber: "INV-1", Date: d(1), BuyerGSTIN: "27AAAAA0000A1Z5", PlaceOfSupply: "27", TaxableValue: 50000, CGST: 4500, SGST: 4500, HSNCode: "9983"},
		// B2B, different buyer, inter-state (IGST) 18% on 200000, HSN 9984.
		{InvoiceNumber: "INV-3", Date: d(3), BuyerGSTIN: "29BBBBB1111B1Z4", PlaceOfSupply: "29", TaxableValue: 200000, IGST: 36000, HSNCode: "9984"},
		// B2C (no GSTIN), 18% intra-state, place 27, HSN 9983.
		{InvoiceNumber: "INV-4", Date: d(4), PlaceOfSupply: "27", TaxableValue: 10000, CGST: 900, SGST: 900, HSNCode: "9983"},
		// Another B2C, same place + rate — must merge into the same B2CS row.
		{InvoiceNumber: "INV-5", Date: d(5), PlaceOfSupply: "27", TaxableValue: 20000, CGST: 1800, SGST: 1800, HSNCode: "9983"},
	}

	r := BuildGSTR1(tenant, 1, 2026, invs)

	// --- Totals tie to every invoice ---
	if r.TotalTaxableValue != 380000 {
		t.Errorf("total taxable = %d, want 380000", r.TotalTaxableValue)
	}
	if r.TotalIGST != 36000 || r.TotalCGST != 16200 || r.TotalSGST != 16200 {
		t.Errorf("totals igst/cgst/sgst = %d/%d/%d, want 36000/16200/16200", r.TotalIGST, r.TotalCGST, r.TotalSGST)
	}
	if r.InvoiceCount != 5 {
		t.Errorf("invoice count = %d, want 5", r.InvoiceCount)
	}

	// --- B2B: two counterparties, sorted by GSTIN; first buyer has 2 invoices ---
	if len(r.B2B) != 2 {
		t.Fatalf("B2B counterparties = %d, want 2", len(r.B2B))
	}
	if r.B2B[0].GSTIN != "27AAAAA0000A1Z5" {
		t.Errorf("B2B[0] GSTIN = %q, want the 27… buyer (sorted first)", r.B2B[0].GSTIN)
	}
	if len(r.B2B[0].Invoices) != 2 {
		t.Fatalf("first B2B buyer invoices = %d, want 2 (grouped)", len(r.B2B[0].Invoices))
	}
	// Grouped invoices sorted by number: INV-1 then INV-2.
	if r.B2B[0].Invoices[0].InvoiceNumber != "INV-1" || r.B2B[0].Invoices[1].InvoiceNumber != "INV-2" {
		t.Errorf("B2B invoices not sorted: %q, %q", r.B2B[0].Invoices[0].InvoiceNumber, r.B2B[0].Invoices[1].InvoiceNumber)
	}
	if got := r.B2B[0].Invoices[0].Rate; got != 18 {
		t.Errorf("rate = %v, want 18 (9000/50000*... = 18%%)", got)
	}

	// --- B2CS: two B2C invoices merged into one (place 27, rate 18) ---
	if len(r.B2CS) != 1 {
		t.Fatalf("B2CS rows = %d, want 1 (same place+rate merged)", len(r.B2CS))
	}
	if r.B2CS[0].PlaceOfSupply != "27" || r.B2CS[0].Rate != 18 {
		t.Errorf("B2CS row = place %q rate %v, want 27/18", r.B2CS[0].PlaceOfSupply, r.B2CS[0].Rate)
	}
	if r.B2CS[0].TaxableValue != 30000 || r.B2CS[0].CGST != 2700 || r.B2CS[0].SGST != 2700 {
		t.Errorf("B2CS totals = %d/%d/%d, want taxable 30000, cgst 2700, sgst 2700", r.B2CS[0].TaxableValue, r.B2CS[0].CGST, r.B2CS[0].SGST)
	}

	// --- HSN rollup covers B2B + B2C; sums reconcile to the control totals ---
	var hsnTaxable, hsnTax int64
	var hsnInvoices int
	for _, h := range r.HSN {
		hsnTaxable += h.TaxableValue
		hsnTax += h.IGST + h.CGST + h.SGST
		hsnInvoices += h.InvoiceCount
	}
	if hsnTaxable != r.TotalTaxableValue {
		t.Errorf("HSN taxable sum = %d, want %d (must reconcile to totals)", hsnTaxable, r.TotalTaxableValue)
	}
	if hsnTax != r.TotalIGST+r.TotalCGST+r.TotalSGST {
		t.Errorf("HSN tax sum = %d, want %d", hsnTax, r.TotalIGST+r.TotalCGST+r.TotalSGST)
	}
	if hsnInvoices != r.InvoiceCount {
		t.Errorf("HSN invoice count sum = %d, want %d", hsnInvoices, r.InvoiceCount)
	}
}

// TestBuildGSTR1_Empty: no invoices -> a well-formed empty return.
func TestBuildGSTR1_Empty(t *testing.T) {
	r := BuildGSTR1(uuid.New(), 3, 2026, nil)
	if r.InvoiceCount != 0 || len(r.B2B) != 0 || len(r.B2CS) != 0 || len(r.HSN) != 0 {
		t.Errorf("empty return not empty: %+v", r)
	}
	if r.Month != 3 || r.Year != 2026 {
		t.Errorf("period = %d/%d, want 3/2026", r.Month, r.Year)
	}
}

// TestBuildGSTR1_ZeroRatedNoDivideByZero: a zero-taxable-value invoice must not
// divide by zero when deriving the rate.
func TestBuildGSTR1_ZeroRatedNoDivideByZero(t *testing.T) {
	r := BuildGSTR1(uuid.New(), 1, 2026, []domain.GSTR1Invoice{
		{InvoiceNumber: "Z-1", Date: d(1), PlaceOfSupply: "27", TaxableValue: 0, HSNCode: "0000"},
	})
	if len(r.B2CS) != 1 || r.B2CS[0].Rate != 0 {
		t.Errorf("zero-rated B2CS = %+v, want one row at rate 0", r.B2CS)
	}
}
