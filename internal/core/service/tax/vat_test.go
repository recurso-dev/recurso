package tax

import (
	"context"
	"testing"

	"github.com/recurso-dev/recurso/internal/core/port"
)

// eu27 is the canonical set of EU member states the rate table must cover.
var eu27 = []string{
	"AT", "BE", "BG", "HR", "CY", "CZ", "DK", "EE", "FI", "FR",
	"DE", "GR", "HU", "IE", "IT", "LV", "LT", "LU", "MT", "NL",
	"PL", "PT", "RO", "SK", "SI", "ES", "SE",
}

func TestEUVATRates_AllMemberStatesPresent(t *testing.T) {
	if len(eu27) != 27 {
		t.Fatalf("test fixture eu27 has %d entries, want 27", len(eu27))
	}
	for _, cc := range eu27 {
		rate, ok := euVATRates[cc]
		if !ok {
			t.Errorf("%s missing from euVATRates table", cc)
			continue
		}
		// Standard EU VAT rates sit between 17% (LU) and 27% (HU); any value
		// outside a sane band signals a typo (e.g. 0.20 written as 20).
		if rate < 0.15 || rate > 0.30 {
			t.Errorf("%s rate = %v, outside the plausible 0.15..0.30 standard-rate band", cc, rate)
		}
	}
}

func TestEUVATRates_KnownSpotChecks(t *testing.T) {
	// A few rates that recently changed or anchor the range, so a careless
	// edit to the table is caught.
	want := map[string]float64{
		"DE": 0.19,  // Germany
		"HU": 0.27,  // highest
		"LU": 0.17,  // lowest
		"EE": 0.24,  // raised Jul 2025
		"FI": 0.255, // raised Sep 2024
		"SK": 0.23,  // raised Jan 2025
		"RO": 0.21,  // raised Aug 2025
	}
	for cc, w := range want {
		if got := euVATRates[cc]; got != w {
			t.Errorf("%s rate = %v, want %v", cc, got, w)
		}
	}
}

func TestIsEUVATCountry_ExcludesGB(t *testing.T) {
	if !IsEUVATCountry("DE") {
		t.Error("IsEUVATCountry(DE) = false, want true")
	}
	if IsEUVATCountry("GB") {
		t.Error("IsEUVATCountry(GB) = true, want false (GB left the EU VAT area)")
	}
	if IsEUVATCountry("US") {
		t.Error("IsEUVATCountry(US) = true, want false")
	}
}

func TestEUVATEngine_Matrix(t *testing.T) {
	ctx := context.Background()
	eng := NewEUVATEngine("DE") // German seller

	tests := []struct {
		name       string
		req        *port.TaxRequest
		wantTax    int64
		wantType   string
		wantRevChg bool
	}{
		{
			name:     "domestic B2C -> DE 19%",
			req:      &port.TaxRequest{Amount: 10000, BuyerCountry: "DE"},
			wantTax:  1900,
			wantType: "vat",
		},
		{
			name:       "cross-border B2B -> reverse charge 0%",
			req:        &port.TaxRequest{Amount: 10000, BuyerCountry: "FR", IsBusiness: true},
			wantTax:    0,
			wantType:   "vat_reverse_charge",
			wantRevChg: true,
		},
		{
			name:     "cross-border B2C -> buyer country FR 20%",
			req:      &port.TaxRequest{Amount: 10000, BuyerCountry: "FR"},
			wantTax:  2000,
			wantType: "vat",
		},
		{
			name:     "export outside EU B2B -> zero-rated",
			req:      &port.TaxRequest{Amount: 10000, BuyerCountry: "US", IsBusiness: true},
			wantTax:  0,
			wantType: "export_exempt",
		},
		{
			// PHASE2 #5: a B2C digital service to a non-EU buyer is out of scope
			// for EU VAT — 0%, NOT the seller's domestic rate (the bug charged DE 19%).
			name:     "export outside EU B2C -> zero-rated (not domestic VAT)",
			req:      &port.TaxRequest{Amount: 10000, BuyerCountry: "US"},
			wantTax:  0,
			wantType: "export_exempt",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := eng.CalculateTax(ctx, tc.req)
			if err != nil {
				t.Fatalf("CalculateTax: %v", err)
			}
			if got.TotalTax != tc.wantTax {
				t.Errorf("TotalTax = %d, want %d", got.TotalTax, tc.wantTax)
			}
			if got.TaxType != tc.wantType {
				t.Errorf("TaxType = %q, want %q", got.TaxType, tc.wantType)
			}
			if got.ReverseCharge != tc.wantRevChg {
				t.Errorf("ReverseCharge = %v, want %v", got.ReverseCharge, tc.wantRevChg)
			}
		})
	}
}
