package tax

import (
	"context"
	"testing"

	"github.com/swapnull-in/recur-so/internal/core/port"
)

// TestGSTFallbackRate locks the Option-C semantics: the tenant's configured
// gst_rate (passed as FallbackRate, a fraction) is used ONLY when the SAC/HSN
// code isn't in the rate map. A recognized code always wins.
func TestGSTFallbackRate(t *testing.T) {
	engine := NewGSTEngine("TN")
	ctx := context.Background()

	tests := []struct {
		name     string
		hsn      string
		fallback float64
		wantRate float64
	}{
		{"unrecognized code uses tenant fallback", "999999", 0.12, 0.12},
		{"recognized code ignores fallback (map wins)", "998314", 0.12, 0.18},
		{"unrecognized code, no fallback -> built-in default", "999999", 0, 0.18},
		{"empty code with fallback uses fallback", "", 0.05, 0.05},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calc, err := engine.CalculateTax(ctx, &port.TaxRequest{
				Amount:       100000,
				Currency:     "INR",
				HSNCode:      tt.hsn,
				FallbackRate: tt.fallback,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if calc.TaxRate != tt.wantRate {
				t.Errorf("TaxRate = %v, want %v", calc.TaxRate, tt.wantRate)
			}
			wantTax := int64(float64(100000) * tt.wantRate)
			if calc.TotalTax != wantTax {
				t.Errorf("TotalTax = %d, want %d", calc.TotalTax, wantTax)
			}
		})
	}
}
