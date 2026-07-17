package domain

import (
	"github.com/google/uuid"
)

// GSTR3BValues is one value row of GSTR-3B Table 3.1 — taxable value plus the
// IGST/CGST/SGST split, in minor units (paise). Unlike GSTR-1, GSTR-3B is a
// self-declared summary: rows are period aggregates, not per-invoice detail.
type GSTR3BValues struct {
	TaxableValue int64 `json:"taxable_value"`
	IGST         int64 `json:"igst"`
	CGST         int64 `json:"cgst"`
	SGST         int64 `json:"sgst"`
}

// GSTR3BInterStateUnreg is one row of Table 3.2: inter-state outward supplies
// made to unregistered persons, reported per place of supply. A subset of
// Table 3.1(a) — only IGST applies inter-state.
type GSTR3BInterStateUnreg struct {
	PlaceOfSupply string `json:"place_of_supply"`
	TaxableValue  int64  `json:"taxable_value"`
	IGST          int64  `json:"igst"`
}

// GSTR3BReturn is a return-ready GSTR-3B summary for a tenant's tax period.
// Table 3.1(a) is net of the period's credit notes (GSTR-3B reports net
// outward supplies, unlike GSTR-1 which lists credit notes separately).
// Sections this engine has no data for are present but zero, so the export
// shape matches the government schema and the omission is explicit:
//   - ZeroRated (3.1b): zero-rated exports/SEZ — not modelled on invoices yet
//   - NilExempt (3.1c) and NonGST (3.1e): engine bills taxable supplies only
//   - InwardReverseCharge (3.1d) and ITC (Table 4): purchase-side data the
//     billing engine does not hold (ITC is deferred to P2 per the India spec)
type GSTR3BReturn struct {
	TenantID uuid.UUID `json:"tenant_id"`
	Month    int       `json:"month"`
	Year     int       `json:"year"`

	OutwardTaxable      GSTR3BValues `json:"outward_taxable"`       // 3.1(a), net of credit notes
	ZeroRated           GSTR3BValues `json:"zero_rated"`            // 3.1(b)
	NilExempt           GSTR3BValues `json:"nil_exempt"`            // 3.1(c)
	InwardReverseCharge GSTR3BValues `json:"inward_reverse_charge"` // 3.1(d)
	NonGST              GSTR3BValues `json:"non_gst"`               // 3.1(e)

	InterStateUnregistered []GSTR3BInterStateUnreg `json:"inter_state_unregistered"` // 3.2

	// Provenance counts so a reviewer can see what the summary was built from.
	InvoiceCount    int `json:"invoice_count"`
	CreditNoteCount int `json:"credit_note_count"`
}
