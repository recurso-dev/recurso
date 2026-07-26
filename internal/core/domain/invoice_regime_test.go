package domain

import "testing"

func TestInvoiceRegimeFallback(t *testing.T) {
	cases := []struct {
		name string
		inv  Invoice
		want string
	}{
		{"gst split", Invoice{Currency: "USD", CGSTAmount: 900, SGSTAmount: 900}, TaxRegimeGST},
		{"igst only", Invoice{Currency: "USD", IGSTAmount: 1800}, TaxRegimeGST},
		{"inr no split", Invoice{Currency: "INR"}, TaxRegimeGST},
		{"inr lowercase", Invoice{Currency: "inr"}, TaxRegimeGST},
		{"usd no tax", Invoice{Currency: "USD"}, TaxRegimePlain},
		{"eur no split", Invoice{Currency: "EUR"}, TaxRegimePlain},
		{"empty currency", Invoice{}, TaxRegimePlain},
	}
	for _, c := range cases {
		if got := c.inv.RegimeFallback(); got != c.want {
			t.Errorf("%s: RegimeFallback() = %q, want %q", c.name, got, c.want)
		}
	}
}
