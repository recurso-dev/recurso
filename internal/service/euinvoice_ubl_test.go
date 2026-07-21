package service

import (
	"context"
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/einvoice_eu"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

func sampleEUInvoice() *domain.Invoice {
	return &domain.Invoice{
		ID:            uuid.New(),
		InvoiceNumber: "INV-2026-000042",
		Currency:      "EUR",
		Subtotal:      100000, // €1,000.00
		TaxAmount:     21000,  // €210.00 (21%)
		Total:         121000, // €1,210.00
		CreatedAt:     time.Date(2026, 7, 21, 10, 0, 0, 0, time.UTC),
		DueDate:       time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC),
		LineItems: []domain.InvoiceItem{
			{Description: "Pro plan", Quantity: 1, UnitAmount: 80000, Amount: 80000, TaxRate: 21},
			{Description: "API calls", Quantity: 5000, UnitAmount: 4, Amount: 20000, TaxRate: 21},
		},
	}
}

var euSeller = domain.EUParty{Name: "Acme GmbH", VATID: "DE123456789", CountryCode: "DE", Street: "Hauptstr. 1", City: "Berlin", PostalZone: "10115"}
var euBuyer = domain.EUParty{Name: "Beta Sàrl", VATID: "FR12345678901", CountryCode: "FR", City: "Paris", PostalZone: "75001"}

// parsed mirrors the fields we assert on (namespace-agnostic local names).
type parsedUBL struct {
	XMLName         xml.Name
	CustomizationID string `xml:"CustomizationID"`
	ProfileID       string `xml:"ProfileID"`
	ID              string `xml:"ID"`
	IssueDate       string `xml:"IssueDate"`
	InvoiceTypeCode string `xml:"InvoiceTypeCode"`
	Currency        string `xml:"DocumentCurrencyCode"`
	Supplier        struct {
		Name    string `xml:"Party>PartyName>Name"`
		Country string `xml:"Party>PostalAddress>Country>IdentificationCode"`
		VAT     string `xml:"Party>PartyTaxScheme>CompanyID"`
	} `xml:"AccountingSupplierParty"`
	Customer struct {
		Name    string `xml:"Party>PartyName>Name"`
		Country string `xml:"Party>PostalAddress>Country>IdentificationCode"`
	} `xml:"AccountingCustomerParty"`
	TaxTotal struct {
		TaxAmount string `xml:"TaxAmount"`
		Subtotal  []struct {
			Taxable string `xml:"TaxableAmount"`
			Tax     string `xml:"TaxAmount"`
			Percent string `xml:"TaxCategory>Percent"`
			CatID   string `xml:"TaxCategory>ID"`
		} `xml:"TaxSubtotal"`
	} `xml:"TaxTotal"`
	Monetary struct {
		LineExtension string `xml:"LineExtensionAmount"`
		TaxExclusive  string `xml:"TaxExclusiveAmount"`
		TaxInclusive  string `xml:"TaxInclusiveAmount"`
		Payable       string `xml:"PayableAmount"`
	} `xml:"LegalMonetaryTotal"`
	Lines []struct {
		ID     string `xml:"ID"`
		Qty    string `xml:"InvoicedQuantity"`
		Amount string `xml:"LineExtensionAmount"`
		Name   string `xml:"Item>Name"`
	} `xml:"InvoiceLine"`
}

// TestBuildUBLInvoice_StructureAndTotals proves the generated UBL is well-formed,
// carries the EN 16931 mandatory fields, and that its monetary totals reconcile
// exactly to the source invoice.
func TestBuildUBLInvoice_StructureAndTotals(t *testing.T) {
	inv := sampleEUInvoice()
	out, err := BuildUBLInvoice(inv, euSeller, euBuyer)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	if !strings.HasPrefix(string(out), xml.Header) {
		t.Fatal("missing XML declaration")
	}
	var p parsedUBL
	if err := xml.Unmarshal(out, &p); err != nil {
		t.Fatalf("generated document is not valid XML: %v", err)
	}

	// Mandatory header fields.
	if p.XMLName.Local != "Invoice" {
		t.Fatalf("root element = %q, want Invoice", p.XMLName.Local)
	}
	if !strings.Contains(p.CustomizationID, "en16931") {
		t.Errorf("CustomizationID should reference EN 16931, got %q", p.CustomizationID)
	}
	if p.ID != "INV-2026-000042" {
		t.Errorf("invoice ID = %q", p.ID)
	}
	if p.IssueDate != "2026-07-21" {
		t.Errorf("issue date = %q, want 2026-07-21", p.IssueDate)
	}
	if p.InvoiceTypeCode != "380" {
		t.Errorf("type code = %q, want 380", p.InvoiceTypeCode)
	}
	if p.Currency != "EUR" {
		t.Errorf("currency = %q", p.Currency)
	}

	// Parties (BT-27/BT-31 seller, BT-44/BT-55 buyer).
	if p.Supplier.Name != "Acme GmbH" || p.Supplier.Country != "DE" || p.Supplier.VAT != "DE123456789" {
		t.Errorf("supplier party wrong: %+v", p.Supplier)
	}
	if p.Customer.Name != "Beta Sàrl" || p.Customer.Country != "FR" {
		t.Errorf("customer party wrong: %+v", p.Customer)
	}

	// Monetary totals reconcile to the invoice (major-unit decimals).
	if p.Monetary.LineExtension != "1000.00" || p.Monetary.TaxExclusive != "1000.00" {
		t.Errorf("line/tax-exclusive = %q/%q, want 1000.00", p.Monetary.LineExtension, p.Monetary.TaxExclusive)
	}
	if p.Monetary.TaxInclusive != "1210.00" || p.Monetary.Payable != "1210.00" {
		t.Errorf("tax-inclusive/payable = %q/%q, want 1210.00", p.Monetary.TaxInclusive, p.Monetary.Payable)
	}
	if p.TaxTotal.TaxAmount != "210.00" {
		t.Errorf("tax total = %q, want 210.00", p.TaxTotal.TaxAmount)
	}
	// Single 21% rate -> one subtotal that reconciles.
	if len(p.TaxTotal.Subtotal) != 1 {
		t.Fatalf("want 1 tax subtotal, got %d", len(p.TaxTotal.Subtotal))
	}
	st := p.TaxTotal.Subtotal[0]
	if st.Taxable != "1000.00" || st.Tax != "210.00" || st.Percent != "21" || st.CatID != "S" {
		t.Errorf("tax subtotal wrong: %+v", st)
	}

	// Two lines with the right nets and names.
	if len(p.Lines) != 2 {
		t.Fatalf("want 2 invoice lines, got %d", len(p.Lines))
	}
	if p.Lines[0].Amount != "800.00" || p.Lines[0].Name != "Pro plan" {
		t.Errorf("line 1 wrong: %+v", p.Lines[0])
	}
	if p.Lines[1].Amount != "200.00" || strings.TrimSpace(p.Lines[1].Qty) != "5000" {
		t.Errorf("line 2 wrong: %+v", p.Lines[1])
	}
}

// TestBuildUBLInvoice_MixedRatesReconcile proves the tax subtotals still sum to
// the invoice tax when lines carry different VAT rates.
func TestBuildUBLInvoice_MixedRatesReconcile(t *testing.T) {
	inv := &domain.Invoice{
		InvoiceNumber: "INV-MIX", Currency: "EUR",
		Subtotal: 30000, TaxAmount: 4700, Total: 34700, // 20000@21% (4200) + 10000@5% (500)
		CreatedAt: time.Now().UTC(),
		LineItems: []domain.InvoiceItem{
			{Description: "Standard", Quantity: 1, UnitAmount: 20000, Amount: 20000, TaxRate: 21},
			{Description: "Reduced", Quantity: 1, UnitAmount: 10000, Amount: 10000, TaxRate: 5},
		},
	}
	out, err := BuildUBLInvoice(inv, euSeller, euBuyer)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	var p parsedUBL
	if err := xml.Unmarshal(out, &p); err != nil {
		t.Fatalf("xml: %v", err)
	}
	if len(p.TaxTotal.Subtotal) != 2 {
		t.Fatalf("want 2 subtotals, got %d", len(p.TaxTotal.Subtotal))
	}
	// Σ subtotal tax must equal the header tax total (4700 -> "47.00").
	var sum int
	for _, s := range p.TaxTotal.Subtotal {
		sum += euros(t, s.Tax)
	}
	if sum != 4700 {
		t.Errorf("Σ tax subtotals = %d minor, want 4700 (must equal BT-110)", sum)
	}
	if p.TaxTotal.TaxAmount != "47.00" {
		t.Errorf("header tax = %q, want 47.00", p.TaxTotal.TaxAmount)
	}
}

// TestBuildUBLInvoice_Validation rejects missing mandatory fields.
func TestBuildUBLInvoice_Validation(t *testing.T) {
	base := sampleEUInvoice()
	cases := []struct {
		name          string
		inv           *domain.Invoice
		seller, buyer domain.EUParty
	}{
		{"no invoice number", &domain.Invoice{Currency: "EUR", CreatedAt: time.Now()}, euSeller, euBuyer},
		{"bad currency", &domain.Invoice{InvoiceNumber: "X", Currency: "EU", CreatedAt: time.Now()}, euSeller, euBuyer},
		{"seller no country", base, domain.EUParty{Name: "S"}, euBuyer},
		{"buyer no name", base, euSeller, domain.EUParty{CountryCode: "FR"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := BuildUBLInvoice(c.inv, c.seller, c.buyer); err == nil {
				t.Fatal("expected a validation error")
			}
		})
	}
}

// TestMockTransport_Transmits proves the mock transport accepts a document and
// reports it sent with a deterministic id.
func TestMockTransport_Transmits(t *testing.T) {
	inv := sampleEUInvoice()
	doc, _ := BuildUBLInvoice(inv, euSeller, euBuyer)
	tr := einvoice_eu.NewMockTransport()
	res, err := tr.Transmit(context.Background(), domain.EUInvoiceSyntaxUBL, euBuyer.VATID, doc)
	if err != nil {
		t.Fatalf("transmit: %v", err)
	}
	if res.Status != domain.EUInvoiceStatusSent || !strings.HasPrefix(res.MessageID, "mock-") {
		t.Fatalf("transmission = %+v", res)
	}
	// Deterministic: same document -> same id.
	res2, _ := tr.Transmit(context.Background(), domain.EUInvoiceSyntaxUBL, euBuyer.VATID, doc)
	if res2.MessageID != res.MessageID {
		t.Errorf("message id not deterministic: %q vs %q", res.MessageID, res2.MessageID)
	}
}

func euros(t *testing.T, decimal string) int {
	t.Helper()
	parts := strings.SplitN(decimal, ".", 2)
	whole := 0
	for _, r := range parts[0] {
		whole = whole*10 + int(r-'0')
	}
	frac := 0
	if len(parts) == 2 {
		for _, r := range (parts[1] + "00")[:2] {
			frac = frac*10 + int(r-'0')
		}
	}
	return whole*100 + frac
}
