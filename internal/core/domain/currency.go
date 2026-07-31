package domain

import "strings"

// currencyExponent lists ISO-4217 currencies whose minor-unit exponent is NOT
// the default 2. Anything not listed uses exponent 2 (1 major unit == 100 minor
// units). Centralised so rating, invoice formatting, and imports agree on how
// many minor units a major unit has — the importers already special-case the
// zero-decimal set inline, and the usage-rating engine previously hardcoded 100,
// which mispriced JPY/KWD/BHD by 100×/10×.
var currencyExponent = map[string]int{
	// Zero-decimal: no minor unit, so 1 major unit == 1 minor unit.
	"JPY": 0, "KRW": 0, "VND": 0, "CLP": 0, "BIF": 0, "DJF": 0, "GNF": 0,
	"ISK": 0, "KMF": 0, "PYG": 0, "RWF": 0, "UGX": 0, "VUV": 0, "XAF": 0,
	"XOF": 0, "XPF": 0,
	// Three-decimal: 1 major unit == 1000 minor units.
	"BHD": 3, "IQD": 3, "JOD": 3, "KWD": 3, "LYD": 3, "OMR": 3, "TND": 3,
}

// CurrencyExponent returns the number of minor-unit decimal places for an
// ISO-4217 currency code, defaulting to 2 (USD, EUR, INR, …).
func CurrencyExponent(currency string) int {
	if e, ok := currencyExponent[strings.ToUpper(strings.TrimSpace(currency))]; ok {
		return e
	}
	return 2
}

// MinorUnitsPerMajor returns 10^exponent — the factor that converts a major-unit
// amount to minor units for the currency: 100 for USD/EUR, 1 for JPY, 1000 for
// KWD. Use this instead of a hardcoded 100 anywhere a major-unit value (e.g. a
// per-unit price string) is converted to stored int64 minor units.
func MinorUnitsPerMajor(currency string) int64 {
	switch CurrencyExponent(currency) {
	case 0:
		return 1
	case 3:
		return 1000
	default:
		return 100
	}
}
