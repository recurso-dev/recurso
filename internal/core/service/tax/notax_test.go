package tax

import (
	"context"
	"testing"

	"github.com/swapnull-in/recur-so/internal/core/port"
)

// TestFactoryDefault_UnsupportedCountry_NoTax proves ENG-152: an unsupported
// country no longer falls through to India GST — it gets a 0% engine.
func TestFactoryDefault_UnsupportedCountry_NoTax(t *testing.T) {
	for _, country := range []string{"BR", "JP", "AU", "ZZ", ""} {
		engine := NewTaxEngine(country, "")
		calc, err := engine.CalculateTax(context.Background(), &port.TaxRequest{
			Amount: 100000, Currency: "USD", BuyerCountry: country,
		})
		if err != nil {
			t.Fatalf("country %q: CalculateTax error: %v", country, err)
		}
		if calc.TotalTax != 0 || calc.TaxRate != 0 {
			t.Errorf("country %q: tax = %d @ %.2f, want 0 (unsupported jurisdiction, not India GST)", country, calc.TotalTax, calc.TaxRate)
		}
		if calc.TaxType != "none" {
			t.Errorf("country %q: TaxType = %q, want none", country, calc.TaxType)
		}
	}
}

// TestFactoryDefault_SupportedCountriesUnchanged guards that the NoTax default
// didn't disturb the real engines.
func TestFactoryDefault_SupportedCountriesUnchanged(t *testing.T) {
	// India still charges GST.
	inCalc, err := NewTaxEngine("IN", "TN").CalculateTax(context.Background(), &port.TaxRequest{
		Amount: 100000, Currency: "INR", BuyerState: "KA", // inter-state → IGST
	})
	if err != nil {
		t.Fatalf("IN: %v", err)
	}
	if inCalc.TotalTax <= 0 {
		t.Errorf("IN GST tax = %d, want > 0 (India engine still applies)", inCalc.TotalTax)
	}
}
