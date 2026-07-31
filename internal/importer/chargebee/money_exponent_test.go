package chargebee

import "testing"

// Test money() renders the currency exponent correctly. The old code special-
// cased only JPY/KRW/VND/CLP and otherwise divided by 100, misrendering
// three-decimal currencies (KWD) and most zero-decimal ones.
func TestMoneyExponent(t *testing.T) {
	cases := []struct {
		amount   int64
		currency string
		want     string
	}{
		{123456, "USD", "1234.56 USD"}, // 2-decimal unchanged
		{5000, "JPY", "5000 JPY"},      // zero-decimal
		{5000, "KWD", "5.000 KWD"},     // three-decimal: was "50.00 KWD"
		{500000, "ISK", "500000 ISK"},  // zero-decimal beyond the old hardcoded set
	}
	for _, c := range cases {
		if got := money(c.amount, c.currency); got != c.want {
			t.Errorf("money(%d, %q) = %q, want %q", c.amount, c.currency, got, c.want)
		}
	}
}
