package validate

import "testing"

func TestCurrency(t *testing.T) {
	valid := []string{"USD", "eur", " inr ", "JPY", "KWD", "GBP"}
	for _, c := range valid {
		if !Currency(c) {
			t.Errorf("Currency(%q) = false, want true", c)
		}
	}
	invalid := []string{"", "US", "USDD", "XXX", "123", "ZZZ", "DOLLARS"}
	for _, c := range invalid {
		if Currency(c) {
			t.Errorf("Currency(%q) = true, want false", c)
		}
	}
}

func TestCountry(t *testing.T) {
	valid := []string{"US", "in", " gb ", "DE", "JP"}
	for _, c := range valid {
		if !Country(c) {
			t.Errorf("Country(%q) = false, want true", c)
		}
	}
	// "EU" is a grouping, not a country; "USA"/"ZZ" are not alpha-2 countries.
	invalid := []string{"", "U", "USA", "EU", "ZZ", "12"}
	for _, c := range invalid {
		if Country(c) {
			t.Errorf("Country(%q) = true, want false", c)
		}
	}
}

func TestAmountAndPercentage(t *testing.T) {
	if !AmountNonNegative(0) || !AmountNonNegative(100) || AmountNonNegative(-1) {
		t.Error("AmountNonNegative boundaries wrong")
	}
	if !AmountPositive(1) || AmountPositive(0) || AmountPositive(-5) {
		t.Error("AmountPositive boundaries wrong")
	}
	if !Percentage(0) || !Percentage(100) || !Percentage(37.5) || Percentage(-0.1) || Percentage(100.1) {
		t.Error("Percentage boundaries wrong")
	}
}

func TestEmail(t *testing.T) {
	for _, e := range []string{"a@b.com", "ada.lovelace@example.co.uk"} {
		if !Email(e) {
			t.Errorf("Email(%q) = false, want true", e)
		}
	}
	for _, e := range []string{"", "no-at", "a@b", "a b@c.com", "@b.com", "a@.com"} {
		if Email(e) {
			t.Errorf("Email(%q) = true, want false", e)
		}
	}
}
