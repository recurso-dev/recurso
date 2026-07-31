package domain

import "testing"

func TestFormatMoneyPlain_Exponents(t *testing.T) {
	cases := []struct {
		amount   int64
		currency string
		want     string
	}{
		{123456, "USD", "1234.56"}, // 2-decimal unchanged
		{5000, "USD", "50.00"},
		{100, "INR", "1.00"},
		{0, "EUR", "0.00"},
		{-500, "USD", "-5.00"},
		{5000, "JPY", "5000"},  // zero-decimal: was wrongly "50.00"
		{1, "KRW", "1"},        // zero-decimal
		{5000, "KWD", "5.000"}, // three-decimal: was wrongly "50.00"
		{1500, "BHD", "1.500"}, // three-decimal
		{123, "OMR", "0.123"},  // three-decimal, sub-major
		{5000, "jpy", "5000"},  // case-insensitive
	}
	for _, c := range cases {
		if got := FormatMoneyPlain(c.amount, c.currency); got != c.want {
			t.Errorf("FormatMoneyPlain(%d, %q) = %q, want %q", c.amount, c.currency, got, c.want)
		}
	}
}

func TestFormatMoney_SymbolsAndExponent(t *testing.T) {
	cases := []struct {
		amount   int64
		currency string
		want     string
	}{
		{123456, "USD", "$1234.56"},
		{100000, "INR", "₹1000.00"},
		{5000, "JPY", "JPY 5000"},  // was wrongly "JPY 50.00"
		{5000, "KWD", "KWD 5.000"}, // was wrongly "KWD 50.00"
		{2500, "EUR", "EUR 25.00"},
	}
	for _, c := range cases {
		if got := FormatMoney(c.amount, c.currency); got != c.want {
			t.Errorf("FormatMoney(%d, %q) = %q, want %q", c.amount, c.currency, got, c.want)
		}
	}
}
