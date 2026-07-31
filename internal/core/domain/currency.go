package domain

import (
	"fmt"
	"strconv"
	"strings"
)

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

// MinorToMajor converts an int64 minor-unit amount to a major-unit float for the
// currency's exponent: 5000 JPY → 5000.0, 5000 KWD → 5.0, 12345 USD → 123.45.
// Use for external systems (accounting, tax providers) that expect a decimal
// major-unit value; never hardcode /100, which is wrong for JPY/KWD/BHD/…
func MinorToMajor(amount int64, currency string) float64 {
	return float64(amount) / float64(MinorUnitsPerMajor(currency))
}

// FormatMoneyPlain renders an int64 minor-unit amount as a decimal string in the
// currency's own exponent, with NO symbol: 5000 JPY → "5000", 5000 KWD →
// "5.000", 123456 USD → "1234.56". Negative amounts keep a leading "-". Use this
// instead of a hardcoded /100 wherever a stored amount is shown to a user.
func FormatMoneyPlain(amount int64, currency string) string {
	exp := CurrencyExponent(currency)
	neg := amount < 0
	if neg {
		amount = -amount
	}
	per := MinorUnitsPerMajor(currency)
	out := strconv.FormatInt(amount/per, 10)
	if exp > 0 {
		out = fmt.Sprintf("%s.%0*d", out, exp, amount%per)
	}
	if neg {
		out = "-" + out
	}
	return out
}

// FormatMoney renders a minor-unit amount with a leading currency symbol (₹ for
// INR, $ for USD) or, for anything else, the ISO code followed by the amount —
// all exponent-aware. Prefer this over `fmt.Sprintf("%.2f", x/100)` in any
// customer-facing string, which is wrong for non-2-decimal currencies.
func FormatMoney(amount int64, currency string) string {
	n := FormatMoneyPlain(amount, currency)
	switch strings.ToUpper(strings.TrimSpace(currency)) {
	case "INR":
		return "₹" + n
	case "USD":
		return "$" + n
	default:
		return strings.ToUpper(strings.TrimSpace(currency)) + " " + n
	}
}
