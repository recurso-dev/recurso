package service

import (
	"testing"
	"time"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// The per-call seller country wins over the service's env-global default, so one
// PDF service can render both an Indian and a US seller's invoices correctly.
func TestBuildInvoiceDataFor_PerCallCountryOverride(t *testing.T) {
	// Service is configured for an India seller (env default).
	svc := NewInvoicePDFService("Bharat Co", "MG Road, Bengaluru", "29ABCDE1234F1Z5", "ABCDE1234F", "KA", "Bank: ...", "IN", "")

	usInv := &domain.Invoice{
		InvoiceNumber: "INV-US-2",
		Currency:      "USD",
		Subtotal:      100000,
		TaxAmount:     8750,
		Total:         108750,
		CreatedAt:     time.Now(),
		DueDate:       time.Now().Add(720 * time.Hour),
	}
	cust := &domain.Customer{
		Name:           pdfStr("Jane Buyer"),
		BillingAddress: domain.BillingAddress{Line1: "5 King St", City: "Austin", State: "TX", Zip: "78701", Country: "US"},
	}

	// Explicit US overrides the IN-configured service.
	if data := svc.BuildInvoiceDataFor(usInv, cust, "US"); data.ShowGST {
		t.Error("per-call US country must render a non-GST invoice even on an IN-configured service")
	}

	// An empty country falls back to the service default (IN) — an INR invoice
	// then reads as GST.
	inrInv := &domain.Invoice{
		InvoiceNumber: "INV-IN-2", Currency: "INR", Subtotal: 100000, CGSTAmount: 9000, SGSTAmount: 9000,
		TaxAmount: 18000, Total: 118000, CreatedAt: time.Now(), DueDate: time.Now().Add(360 * time.Hour),
	}
	if data := svc.BuildInvoiceDataFor(inrInv, cust, ""); !data.ShowGST {
		t.Error("empty per-call country should fall back to the IN service default and show GST for an INR invoice")
	}
}

func TestRegimeForCountry(t *testing.T) {
	cases := map[string]string{
		"":   domain.TaxRegimeGST, // India-focused default (mirrors sellerJurisdiction env fallback)
		"IN": domain.TaxRegimeGST,
		"in": domain.TaxRegimeGST,
		"US": domain.TaxRegimeSalesTax,
		"us": domain.TaxRegimeSalesTax,
		"DE": domain.TaxRegimeVAT,
		"GB": domain.TaxRegimeVAT,
		"JP": domain.TaxRegimePlain,
		"CA": domain.TaxRegimePlain,
		"AU": domain.TaxRegimePlain,
	}
	for country, want := range cases {
		if got := RegimeForCountry(country); got != want {
			t.Errorf("RegimeForCountry(%q) = %q, want %q", country, got, want)
		}
	}
}
