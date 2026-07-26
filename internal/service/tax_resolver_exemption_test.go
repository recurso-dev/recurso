package service

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// A US sales-tax exemption is honored only through its certificate's expiry
// date; a nil expiry never expires, and an expired certificate collects tax.
func TestExemptForCustomer(t *testing.T) {
	day := 24 * time.Hour
	future := time.Now().Add(30 * day)
	past := time.Now().Add(-2 * day)
	today := time.Now().Truncate(day)

	cases := []struct {
		name   string
		exempt bool
		expiry *time.Time
		want   bool
	}{
		{"not exempt", false, nil, false},
		{"exempt, no expiry on file", true, nil, true},
		{"exempt, future expiry", true, &future, true},
		{"exempt, expires today (valid through expiry)", true, &today, true},
		{"exempt, expired", true, &past, false},
		{"not exempt ignores a future expiry", false, &future, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cust := &domain.Customer{ID: uuid.New(), TaxExempt: c.exempt, TaxExemptionExpiresAt: c.expiry}
			if got := exemptForCustomer(nil, cust); got != c.want {
				t.Errorf("exemptForCustomer = %v, want %v", got, c.want)
			}
		})
	}
}
