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
	HSN  []GSTR1HSNSummary `json:"hsn_summary"`

	TotalTaxableValue int64 `json:"total_taxable_value"`
	TotalIGST         int64 `json:"total_igst"`
	TotalCGST         int64 `json:"total_cgst"`
	TotalSGST         int64 `json:"total_sgst"`
	InvoiceCount      int   `json:"invoice_count"`
}
