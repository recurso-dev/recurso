package domain

import (
	"time"

	"github.com/google/uuid"
)

// EU e-invoicing (Track C). The EU model is structurally different from India's
// IRN clearance: an invoice is expressed in the EN 16931 semantic model, carried
// in a syntax (UBL 2.1 here) and delivered over a network (Peppol) or a national
// platform. Increment 1 builds the document layer — a validatable EN 16931
// UBL 2.1 document — behind a pluggable transport port whose mock stands in for
// the real Access Point until a provider is wired.

// EUInvoiceSyntax identifies the XML syntax an EU e-invoice is expressed in.
// UBL 2.1 (OASIS) is the Peppol BIS Billing 3.0 default and the first target.
type EUInvoiceSyntax string

const (
	EUInvoiceSyntaxUBL EUInvoiceSyntax = "ubl21"
)

// EUInvoiceStatus tracks a structured e-invoice through generation and delivery.
// Distinct from the India EInvoiceStatus (IRN clearance), which this does not
// touch.
type EUInvoiceStatus string

const (
	// EUInvoiceStatusGenerated: the EN 16931 document was built and validated but
	// not yet transmitted.
	EUInvoiceStatusGenerated EUInvoiceStatus = "generated"
	// EUInvoiceStatusSent: handed to the transport (Access Point) for delivery.
	EUInvoiceStatusSent EUInvoiceStatus = "sent"
	// EUInvoiceStatusFailed: generation or transmission failed.
	EUInvoiceStatusFailed EUInvoiceStatus = "failed"
)

// EUParty is a supplier or customer on an EN 16931 invoice. Name and CountryCode
// are the hard-mandatory fields (BT-27/BT-44 name, BT-40/BT-55 country); VATID
// (BT-31/BT-48) is required to invoice under the reverse-charge/VAT rules, and
// the postal-address parts are recommended for a fully-compliant document.
type EUParty struct {
	// Name is the party's registered/legal name (BT-27 seller, BT-44 buyer).
	Name string
	// VATID is the party's VAT identifier including the 2-letter country prefix
	// (e.g. "DE123456789"). Empty for a party without one (some B2C buyers).
	VATID string
	// CountryCode is the ISO 3166-1 alpha-2 code (e.g. "DE"). Mandatory.
	CountryCode string
	// Postal address (recommended). Street/City/PostalZone map to BG-5/BG-8.
	Street     string
	City       string
	PostalZone string
}

// EUInvoiceTransmission is the outcome of handing a document to the transport.
type EUInvoiceTransmission struct {
	// MessageID is the transport's identifier for the delivery (a Peppol message
	// id, or a mock id in increment 1).
	MessageID string
	// Status is the transport-reported status after the hand-off.
	Status EUInvoiceStatus
}

// TenantEUConfig is a tenant's EU e-invoicing configuration — kept isolated from
// the India GST config and the tenant row so regional compliance boundaries stay
// clean. It carries the EN 16931 seller party plus the opt-in flag that gates
// generation (the EU mandate landscape is fragmented, so it is off by default).
type TenantEUConfig struct {
	TenantID    uuid.UUID `json:"tenant_id"`
	Enabled     bool      `json:"enabled"`
	LegalName   string    `json:"legal_name"`
	VATNumber   string    `json:"vat_number"`
	CountryCode string    `json:"country_code"`
	Street      string    `json:"street"`
	City        string    `json:"city"`
	PostalZone  string    `json:"postal_zone"`
}

// SellerParty projects the config into the EN 16931 supplier party.
func (c *TenantEUConfig) SellerParty() EUParty {
	return EUParty{
		Name:        c.LegalName,
		VATID:       c.VATNumber,
		CountryCode: c.CountryCode,
		Street:      c.Street,
		City:        c.City,
		PostalZone:  c.PostalZone,
	}
}

// EUInvoice is the persisted record of an invoice's generated EN 16931 document
// and its delivery status. One per invoice (idempotent upsert).
type EUInvoice struct {
	ID           uuid.UUID       `json:"id"`
	TenantID     uuid.UUID       `json:"tenant_id"`
	InvoiceID    uuid.UUID       `json:"invoice_id"`
	Syntax       EUInvoiceSyntax `json:"syntax"`
	Status       EUInvoiceStatus `json:"status"`
	Document     string          `json:"document"`
	MessageID    string          `json:"message_id"`
	ErrorMessage string          `json:"error_message"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}
