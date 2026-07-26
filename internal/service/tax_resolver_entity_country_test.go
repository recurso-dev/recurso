package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

func constCountry(c string) func(context.Context, uuid.UUID) string {
	return func(context.Context, uuid.UUID) string { return c }
}

// The primary entity's country becomes the seller jurisdiction when there's no
// GST registration, promoting the seller country to a first-class per-tenant
// setting. GST config still wins; an unset entity country falls back to env.
func TestSellerCountry_PrimaryEntityCountry(t *testing.T) {
	ctx := context.Background()
	tenant := uuid.New()

	cases := []struct {
		name    string
		gst     *domain.TenantGSTConfig
		country string // primary entity country_code (nil provider if "<nil>")
		envCC   string
		want    string
	}{
		{"entity US, no GST → US", nil, "US", "IN", "US"},
		{"GST config wins over entity", &domain.TenantGSTConfig{GSTIN: "29ABCDE1234F1Z5", StateCode: "KA"}, "US", "US", "IN"},
		{"empty entity country → env", nil, "", "IN", "IN"},
		{"lowercase/whitespace normalized", nil, " us ", "IN", "US"},
		{"entity EU → that country", nil, "DE", "IN", "DE"},
		{"no provider wired → env", nil, "<nil>", "US", "US"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := NewTaxResolver(&mockGSTConfigProvider{cfg: c.gst}, c.envCC, "TN")
			if c.country != "<nil>" {
				r = r.WithPrimaryEntityCountry(constCountry(c.country))
			}
			if got := r.SellerCountry(ctx, tenant); got != c.want {
				t.Errorf("SellerCountry = %q, want %q", got, c.want)
			}
		})
	}
}

// End-to-end: a US primary-entity country yields the sales_tax presentation
// regime with no GST config in sight.
func TestSellerCountry_DrivesRegime(t *testing.T) {
	r := NewTaxResolver(&mockGSTConfigProvider{}, "IN", "TN").
		WithPrimaryEntityCountry(constCountry("US"))
	if got := RegimeForCountry(r.SellerCountry(context.Background(), uuid.New())); got != domain.TaxRegimeSalesTax {
		t.Errorf("regime = %q, want %q", got, domain.TaxRegimeSalesTax)
	}
}
