package service

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestBuildGSTR3B_NetsAndBuckets exercises the summary assembler: Table 3.1(a)
// nets ALL credit notes (registered and unregistered) from the invoice totals,
// and Table 3.2 collects only the inter-state (IGST) share supplied to
// unregistered buyers, per place of supply, net of that bucket's notes.
func TestBuildGSTR3B_NetsAndBuckets(t *testing.T) {
	tenant := uuid.New()
	invs := []domain.GSTR1Invoice{
		// B2B intra-state: counts in 3.1(a), NOT in 3.2 (registered buyer).
		{InvoiceNumber: "INV-1", Date: d(1), BuyerGSTIN: "27AAAAA0000A1Z5", PlaceOfSupply: "27", TaxableValue: 100000, CGST: 9000, SGST: 9000},
		// B2B inter-state: 3.1(a) only — 3.2 is unregistered-only.
		{InvoiceNumber: "INV-2", Date: d(2), BuyerGSTIN: "29BBBBB1111B1Z4", PlaceOfSupply: "29", TaxableValue: 200000, IGST: 36000},
		// B2C inter-state, place 29: 3.1(a) AND 3.2 row for POS 29.
		{InvoiceNumber: "INV-3", Date: d(3), PlaceOfSupply: "29", TaxableValue: 50000, IGST: 9000},
		// B2C inter-state, place 24: its own 3.2 row.
		{InvoiceNumber: "INV-4", Date: d(4), PlaceOfSupply: "24", TaxableValue: 30000, IGST: 5400},
		// B2C intra-state: 3.1(a) only — no IGST, so not inter-state.
		{InvoiceNumber: "INV-5", Date: d(5), PlaceOfSupply: "27", TaxableValue: 10000, CGST: 900, SGST: 900},
	}
	notes := []domain.GSTR1CreditNote{
		// Registered note: netted from 3.1(a) only.
		{NoteNumber: "CN-1", Date: d(10), BuyerGSTIN: "27AAAAA0000A1Z5", PlaceOfSupply: "27", TaxableValue: 20000, CGST: 1800, SGST: 1800},
		// Unregistered inter-state note, place 29: netted from 3.1(a) AND its 3.2 row.
		{NoteNumber: "CN-2", Date: d(11), PlaceOfSupply: "29", TaxableValue: 10000, IGST: 1800},
	}

	r := BuildGSTR3B(tenant, 1, 2026, invs, notes)

	// 3.1(a): invoices (390000 taxable) minus notes (30000) = 360000.
	if r.OutwardTaxable.TaxableValue != 360000 {
		t.Errorf("3.1(a) taxable = %d, want 360000", r.OutwardTaxable.TaxableValue)
	}
	// IGST: 36000+9000+5400 - 1800 = 48600; CGST/SGST: 9900 - 1800 = 8100 each.
	if r.OutwardTaxable.IGST != 48600 || r.OutwardTaxable.CGST != 8100 || r.OutwardTaxable.SGST != 8100 {
		t.Errorf("3.1(a) igst/cgst/sgst = %d/%d/%d, want 48600/8100/8100",
			r.OutwardTaxable.IGST, r.OutwardTaxable.CGST, r.OutwardTaxable.SGST)
	}
	if r.InvoiceCount != 5 || r.CreditNoteCount != 2 {
		t.Errorf("counts = %d invoices / %d notes, want 5/2", r.InvoiceCount, r.CreditNoteCount)
	}

	// Sections we have no data for stay zero.
	if r.ZeroRated != (domain.GSTR3BValues{}) || r.NilExempt != (domain.GSTR3BValues{}) ||
		r.InwardReverseCharge != (domain.GSTR3BValues{}) || r.NonGST != (domain.GSTR3BValues{}) {
		t.Error("zero-data sections must remain zero")
	}

	// 3.2: two rows, sorted by place of supply (24 before 29); POS 29 netted.
	if len(r.InterStateUnregistered) != 2 {
		t.Fatalf("3.2 rows = %d, want 2", len(r.InterStateUnregistered))
	}
	if r.InterStateUnregistered[0].PlaceOfSupply != "24" ||
		r.InterStateUnregistered[0].TaxableValue != 30000 || r.InterStateUnregistered[0].IGST != 5400 {
		t.Errorf("3.2[0] = %+v, want POS 24 / 30000 / 5400", r.InterStateUnregistered[0])
	}
	if r.InterStateUnregistered[1].PlaceOfSupply != "29" ||
		r.InterStateUnregistered[1].TaxableValue != 40000 || r.InterStateUnregistered[1].IGST != 7200 {
		t.Errorf("3.2[1] = %+v, want POS 29 / 40000 / 7200", r.InterStateUnregistered[1])
	}
}

// TestBuildGSTR3B_Empty produces a valid all-zero return for a quiet month.
func TestBuildGSTR3B_Empty(t *testing.T) {
	r := BuildGSTR3B(uuid.New(), 2, 2026, nil, nil)
	if r.OutwardTaxable != (domain.GSTR3BValues{}) || len(r.InterStateUnregistered) != 0 {
		t.Errorf("empty period must be all zeros, got %+v", r)
	}
}

// TestBuildGSTR3BGovDocument_Schema locks the government JSON shape: official
// field names, rupee amounts, the fixed MMYYYY period, and empty (not null)
// Table 3.2 arrays. Marshaled and compared as JSON so a field rename or type
// change fails loudly.
func TestBuildGSTR3BGovDocument_Schema(t *testing.T) {
	r := &domain.GSTR3BReturn{
		Month: 1, Year: 2026,
		OutwardTaxable: domain.GSTR3BValues{TaxableValue: 360000, IGST: 48600, CGST: 8100, SGST: 8100},
		InterStateUnregistered: []domain.GSTR3BInterStateUnreg{
			{PlaceOfSupply: "24", TaxableValue: 30000, IGST: 5400},
		},
	}

	got, err := json.Marshal(BuildGSTR3BGovDocument("27AAAAA0000A1Z5", r))
	if err != nil {
		t.Fatal(err)
	}

	want := `{"gstin":"27AAAAA0000A1Z5","ret_period":"012026",` +
		`"sup_details":{` +
		`"osup_det":{"txval":3600,"iamt":486,"camt":81,"samt":81,"csamt":0},` +
		`"osup_zero":{"txval":0,"iamt":0,"camt":0,"samt":0,"csamt":0},` +
		`"osup_nil_exmp":{"txval":0,"iamt":0,"camt":0,"samt":0,"csamt":0},` +
		`"isup_rev":{"txval":0,"iamt":0,"camt":0,"samt":0,"csamt":0},` +
		`"osup_nongst":{"txval":0,"iamt":0,"camt":0,"samt":0,"csamt":0}},` +
		`"inter_sup":{"unreg_details":[{"pos":"24","txval":300,"iamt":54}],"comp_details":[],"uin_details":[]},` +
		`"itc_elg":{"itc_net":{"iamt":0,"camt":0,"samt":0,"csamt":0}}}`

	if string(got) != want {
		t.Errorf("gov JSON mismatch:\n got: %s\nwant: %s", got, want)
	}
}
