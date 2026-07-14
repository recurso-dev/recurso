package domain

import (
	"time"

	"github.com/google/uuid"
)

// GSTR1Invoice is one finalized invoice flattened with its buyer's GST identity
// — the input the GSTR-1 builder consumes. TaxableValue and the tax amounts are
// the values already computed on the invoice (net-of-tax + the CGST/SGST/IGST
// split), so the return reconciles to what was billed and to the ledger.
type GSTR1Invoice struct {
	InvoiceNumber string
	Date          time.Time
	BuyerGSTIN    string // empty => treated as B2C (unregistered buyer)
	PlaceOfSupply string // buyer's state code, e.g. "27" (Maharashtra)
	TaxableValue  int64  // net-of-tax value, minor units
	IGST          int64
	CGST          int64
	SGST          int64
	HSNCode       string
}

// GSTR1B2BInvoice is one invoice line under a registered (B2B) counterparty.
type GSTR1B2BInvoice struct {
	InvoiceNumber string    `json:"invoice_number"`
	Date          time.Time `json:"date"`
	PlaceOfSupply string    `json:"place_of_supply"`
	TaxableValue  int64     `json:"taxable_value"`
	IGST          int64     `json:"igst"`
	CGST          int64     `json:"cgst"`
	SGST          int64     `json:"sgst"`
	Rate          float64   `json:"rate"` // combined GST rate %, e.g. 18
}

// GSTR1B2B groups a registered counterparty's invoices under its GSTIN.
type GSTR1B2B struct {
	GSTIN    string            `json:"gstin"`
	Invoices []GSTR1B2BInvoice `json:"invoices"`
}

// GSTR1B2CS is a rate-wise summary of B2C (unregistered-buyer) supplies for a
// place of supply — GSTR-1 reports B2C small supplies in summary, not per-invoice.
type GSTR1B2CS struct {
	PlaceOfSupply string  `json:"place_of_supply"`
	Rate          float64 `json:"rate"`
	TaxableValue  int64   `json:"taxable_value"`
	IGST          int64   `json:"igst"`
	CGST          int64   `json:"cgst"`
	SGST          int64   `json:"sgst"`
}

// GSTR1CreditNote is one credit note flattened with its buyer's GST identity —
// the input for the CDNR (Credit/Debit Notes, Registered) section. The tax
// amounts are the GST reversed by the note; TaxableValue is its net-of-tax value.
type GSTR1CreditNote struct {
	NoteNumber            string
	Date                  time.Time
	BuyerGSTIN            string // registered => CDNR (this v1 handles registered only)
	PlaceOfSupply         string
	OriginalInvoiceNumber string
	TaxableValue          int64
	IGST                  int64
	CGST                  int64
	SGST                  int64
}

// GSTR1CDNRNote is one credit note under a registered counterparty.
type GSTR1CDNRNote struct {
	NoteNumber            string    `json:"note_number"`
	Date                  time.Time `json:"date"`
	OriginalInvoiceNumber string    `json:"original_invoice_number"`
	PlaceOfSupply         string    `json:"place_of_supply"`
	TaxableValue          int64     `json:"taxable_value"`
	IGST                  int64     `json:"igst"`
	CGST                  int64     `json:"cgst"`
	SGST                  int64     `json:"sgst"`
	Rate                  float64   `json:"rate"`
}

// GSTR1CDNR groups a registered counterparty's credit notes under its GSTIN.
type GSTR1CDNR struct {
	GSTIN string          `json:"gstin"`
	Notes []GSTR1CDNRNote `json:"notes"`
}

// GSTR1HSNSummary is the per-HSN rollup GSTR-1 requires.
type GSTR1HSNSummary struct {
	HSNCode      string `json:"hsn_code"`
	TaxableValue int64  `json:"taxable_value"`
	IGST         int64  `json:"igst"`
	CGST         int64  `json:"cgst"`
	SGST         int64  `json:"sgst"`
	InvoiceCount int    `json:"invoice_count"`
}

// GSTR1Return is a return-ready GSTR-1 for a tenant's tax period: registered
// (B2B) supplies invoice-by-invoice, unregistered (B2CS) supplies summarized
// rate-wise, an HSN rollup, and control totals. Read-only; assembled from
// finalized invoices, never recomputed from floats.
type GSTR1Return struct {
	TenantID uuid.UUID `json:"tenant_id"`
	Month    int       `json:"month"`
	Year     int       `json:"year"`

	B2B  []GSTR1B2B        `json:"b2b"`
	B2CS []GSTR1B2CS       `json:"b2cs"`
	CDNR []GSTR1CDNR       `json:"cdnr"` // credit notes to registered buyers
	HSN  []GSTR1HSNSummary `json:"hsn_summary"`

	// Outward-supply totals (from invoices). GSTR-1 reports credit notes in a
	// separate section, so these are the gross supply figures, not net of notes.
	TotalTaxableValue int64 `json:"total_taxable_value"`
	TotalIGST         int64 `json:"total_igst"`
	TotalCGST         int64 `json:"total_cgst"`
	TotalSGST         int64 `json:"total_sgst"`
	InvoiceCount      int   `json:"invoice_count"`

	// Credit-note (CDNR) totals — the tax reduced by notes in the period.
	TotalCreditTaxableValue int64 `json:"total_credit_taxable_value"`
	TotalCreditIGST         int64 `json:"total_credit_igst"`
	TotalCreditCGST         int64 `json:"total_credit_cgst"`
	TotalCreditSGST         int64 `json:"total_credit_sgst"`
	CreditNoteCount         int   `json:"credit_note_count"`
}
