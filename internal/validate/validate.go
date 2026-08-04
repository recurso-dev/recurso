// Package validate centralizes the input checks that were previously ad hoc
// across handlers (a bare len(currency)==3, or nothing at all). It exposes
// plain predicates and registers gin binding tags (`currency`, `country`) so a
// request struct can declare `binding:"required,currency"` and reject a
// malformed code at bind time with a consistent message.
//
// Currency and country validity come from golang.org/x/text (the maintained
// CLDR/ISO dataset already in the module graph) rather than a hand-maintained
// table — so the accepted set tracks the upstream standard, not a snapshot.
package validate

import (
	"regexp"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"golang.org/x/text/currency"
	"golang.org/x/text/language"
)

// Currency reports whether code is a valid ISO-4217 alpha currency code.
// Case-insensitive; surrounding space tolerated.
func Currency(code string) bool {
	c := strings.ToUpper(strings.TrimSpace(code))
	if len(c) != 3 {
		return false
	}
	// ISO 4217 reserves XXX ("no currency") and XTS ("for testing"). Both parse
	// but neither is a transactable billing currency, so reject them.
	if c == "XXX" || c == "XTS" {
		return false
	}
	_, err := currency.ParseISO(c)
	return err == nil
}

// Country reports whether code is a valid ISO-3166-1 alpha-2 country code.
func Country(code string) bool {
	c := strings.ToUpper(strings.TrimSpace(code))
	if len(c) != 2 {
		return false
	}
	r, err := language.ParseRegion(c)
	// ParseRegion accepts 3-letter and numeric regions too; constrain to a
	// canonical 2-letter country (IsCountry filters out groupings like "EU").
	return err == nil && r.IsCountry() && r.String() == c
}

// AmountNonNegative reports whether a minor-unit amount is >= 0. Money is int64
// minor units throughout; a negative where a positive is meant is the
// over-refund / negative-charge bug class.
func AmountNonNegative(minor int64) bool { return minor >= 0 }

// AmountPositive reports whether a minor-unit amount is strictly > 0.
func AmountPositive(minor int64) bool { return minor > 0 }

// Percentage reports whether p is a sane percentage in [0, 100].
func Percentage(p float64) bool { return p >= 0 && p <= 100 }

var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// Email reports whether s is a plausibly-valid email address. Intentionally
// permissive (RFC 5322 in full is not worth enforcing at the edge); it rejects
// the obvious garbage while accepting real addresses.
func Email(s string) bool {
	s = strings.TrimSpace(s)
	return len(s) <= 254 && emailRe.MatchString(s)
}

// Register wires the `currency` and `country` binding tags into gin's default
// validator. Call once at startup. Safe to call more than once.
func Register() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		_ = v.RegisterValidation("currency", func(fl validator.FieldLevel) bool {
			return Currency(fl.Field().String())
		})
		_ = v.RegisterValidation("country", func(fl validator.FieldLevel) bool {
			return Country(fl.Field().String())
		})
	}
}
