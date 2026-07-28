package accounting

import "testing"

func TestMinorToMajorHonorsCurrencyExponent(t *testing.T) {
	cases := []struct {
		currency string
		minor    int64
		want     float64
	}{
		{"USD", 10000, 100.00},  // 2 decimals — the common case
		{"EUR", 2599, 25.99},    // 2 decimals
		{"JPY", 10000, 10000.0}, // 0 decimals — ¥10,000, NOT 100
		{"KRW", 5000, 5000.0},   // 0 decimals
		{"KWD", 10000, 10.000},  // 3 decimals — KWD 10.000, NOT 100
		{"BHD", 1500, 1.500},    // 3 decimals
		{"jpy", 10000, 10000.0}, // case-insensitive
		{"", 10000, 100.00},     // unknown/empty defaults to 2
		{"XYZ", 10000, 100.00},  // unknown code defaults to 2
	}
	for _, c := range cases {
		if got := minorToMajor(c.minor, c.currency); got != c.want {
			t.Errorf("minorToMajor(%d, %q) = %v, want %v", c.minor, c.currency, got, c.want)
		}
	}
}

func TestCurrencyExponent(t *testing.T) {
	for cur, want := range map[string]int{
		"USD": 2, "EUR": 2, "GBP": 2, "INR": 2,
		"JPY": 0, "KRW": 0, "VND": 0,
		"KWD": 3, "BHD": 3, "OMR": 3,
		"": 2, "ZZZ": 2,
	} {
		if got := currencyExponent(cur); got != want {
			t.Errorf("currencyExponent(%q) = %d, want %d", cur, got, want)
		}
	}
}
