package accounting

import (
	"math"
	"strings"
)

// The internal ledger stores money as int64 MINOR units and is deliberately
// currency-exponent-agnostic (domain/ledger.go defers formatting). Accounting
// SaaS APIs (Xero/QuickBooks/NetSuite) and the Tally export expect MAJOR units
// (float), so the export boundary — and ONLY the export boundary — must apply
// the currency's exponent. A hardcoded /100 silently corrupts zero- and
// three-decimal currencies in the customer's real books (e.g. ¥10,000 exported
// as 100.00, KWD 10.000 exported as 100.00).

// zeroDecimalCurrencies have no minor unit (exponent 0): 1 major unit == 1
// stored unit. ISO 4217.
var zeroDecimalCurrencies = map[string]bool{
	"BIF": true, "CLP": true, "DJF": true, "GNF": true, "JPY": true,
	"KMF": true, "KRW": true, "MGA": true, "PYG": true, "RWF": true,
	"UGX": true, "VND": true, "VUV": true, "XAF": true, "XOF": true,
	"XPF": true,
}

// threeDecimalCurrencies have a milli minor unit (exponent 3): 1000 stored
// units == 1 major unit. ISO 4217.
var threeDecimalCurrencies = map[string]bool{
	"BHD": true, "IQD": true, "JOD": true, "KWD": true, "LYD": true,
	"OMR": true, "TND": true,
}

// currencyExponent returns the number of minor-unit digits for an ISO-4217
// currency code (0, 2, or 3). Unknown/empty codes default to 2 — the
// overwhelming majority, and the historical assumption this replaces.
func currencyExponent(currency string) int {
	c := strings.ToUpper(strings.TrimSpace(currency))
	switch {
	case zeroDecimalCurrencies[c]:
		return 0
	case threeDecimalCurrencies[c]:
		return 3
	default:
		return 2
	}
}

// minorToMajor converts an int64 minor-unit amount to major units for the
// given currency, honoring its exponent. Use this at the accounting-export
// boundary instead of a literal /100.
func minorToMajor(amount int64, currency string) float64 {
	return float64(amount) / math.Pow10(currencyExponent(currency))
}
