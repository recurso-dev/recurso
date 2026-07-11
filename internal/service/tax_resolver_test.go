package service

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/service/tax"
)

// --- Mocks for tax resolver tests ---

type mockGSTConfigProvider struct {
	cfg   *domain.TenantGSTConfig
	err   error
	calls int
}

func (m *mockGSTConfigProvider) GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.TenantGSTConfig, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return m.cfg, nil
}

func inCustomer(pos string) *domain.Customer {
	c := &domain.Customer{
		ID:             uuid.New(),
		BillingAddress: domain.BillingAddress{Country: "India"},
	}
	if pos != "" {
		c.PlaceOfSupply = domain.StringPtr(pos)
	}
	return c
}

// --- India GST ---

func TestResolveInvoiceTax_INR_IntraState_TenantState(t *testing.T) {
	// Tenant registered in Karnataka (numeric GSTIN state code "29").
	// A KA place of supply must be intra-state (CGST+SGST), which proves the
	// tenant's state is used instead of the historical hardcoded "TN".
	provider := &mockGSTConfigProvider{cfg: &domain.TenantGSTConfig{
		GSTIN:     "29ABCDE1234F1Z5",
		StateCode: "29",
		SACCode:   "998314",
	}}
	r := NewTaxResolver(provider, "IN", "TN")

	res := r.ResolveInvoiceTax(context.Background(), uuid.New(), inCustomer("KA"), "INR", 100000, "")

	if res.Total != 18000 {
		t.Errorf("Total = %d, want 18000", res.Total)
	}
	if res.CGST != 9000 || res.SGST != 9000 {
		t.Errorf("CGST/SGST = %d/%d, want 9000/9000 (intra-state in tenant's KA)", res.CGST, res.SGST)
	}
	if res.IGST != 0 {
		t.Errorf("IGST = %d, want 0 for intra-state", res.IGST)
	}
	if provider.calls != 1 {
		t.Errorf("GST config lookups = %d, want 1", provider.calls)
	}
}

func TestResolveInvoiceTax_INR_IntraState_NumericVsAlphaStateCodes(t *testing.T) {
	// Tenant config stores the numeric GSTIN code "33" (Tamil Nadu) while the
	// customer's place of supply is the abbreviation "TN". They must compare
	// equal after normalization.
	provider := &mockGSTConfigProvider{cfg: &domain.TenantGSTConfig{
		GSTIN:     "33ABCDE1234F1Z5",
		StateCode: "33",
	}}
	r := NewTaxResolver(provider, "IN", "KA")

	res := r.ResolveInvoiceTax(context.Background(), uuid.New(), inCustomer("TN"), "INR", 100000, "")

	if res.CGST != 9000 || res.SGST != 9000 {
		t.Errorf("CGST/SGST = %d/%d, want 9000/9000 (numeric '33' == alpha 'TN')", res.CGST, res.SGST)
	}
	if res.IGST != 0 {
		t.Errorf("IGST = %d, want 0", res.IGST)
	}
}

func TestResolveInvoiceTax_INR_InterState_IGST(t *testing.T) {
	provider := &mockGSTConfigProvider{cfg: &domain.TenantGSTConfig{
		GSTIN:     "33ABCDE1234F1Z5",
		StateCode: "33", // TN
	}}
	r := NewTaxResolver(provider, "IN", "TN")

	res := r.ResolveInvoiceTax(context.Background(), uuid.New(), inCustomer("KA"), "INR", 100000, "")

	if res.IGST != 18000 {
		t.Errorf("IGST = %d, want 18000 for inter-state", res.IGST)
	}
	if res.CGST != 0 || res.SGST != 0 {
		t.Errorf("CGST/SGST = %d/%d, want 0/0 for inter-state", res.CGST, res.SGST)
	}
	if res.Total != 18000 {
		t.Errorf("Total = %d, want 18000", res.Total)
	}
}

func TestResolveInvoiceTax_INR_BillingStateFallback(t *testing.T) {
	// No PlaceOfSupply, but the Indian billing address carries the state.
	provider := &mockGSTConfigProvider{cfg: &domain.TenantGSTConfig{StateCode: "33"}}
	r := NewTaxResolver(provider, "IN", "TN")

	customer := &domain.Customer{
		ID:             uuid.New(),
		BillingAddress: domain.BillingAddress{Country: "India", State: "Tamil Nadu"},
	}

	res := r.ResolveInvoiceTax(context.Background(), uuid.New(), customer, "INR", 100000, "")

	if res.CGST != 9000 || res.SGST != 9000 {
		t.Errorf("CGST/SGST = %d/%d, want 9000/9000 (billing state fallback)", res.CGST, res.SGST)
	}
}

func TestResolveInvoiceTax_IndianSeller_USD_Export_ZeroTax(t *testing.T) {
	for _, tc := range []struct {
		name   string
		hasLUT bool
	}{
		{"with LUT", true},
		{"without LUT info", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			provider := &mockGSTConfigProvider{cfg: &domain.TenantGSTConfig{
				GSTIN:     "33ABCDE1234F1Z5",
				StateCode: "33",
				HasLUT:    tc.hasLUT,
			}}
			r := NewTaxResolver(provider, "IN", "TN")

			customer := &domain.Customer{
				ID:             uuid.New(),
				BillingAddress: domain.BillingAddress{Country: "US"},
			}

			res := r.ResolveInvoiceTax(context.Background(), uuid.New(), customer, "USD", 9900, "")

			if res.Total != 0 || res.IGST != 0 || res.CGST != 0 || res.SGST != 0 {
				t.Errorf("tax = %+v, want all zero for export", res)
			}
			if res.TaxType != "export" {
				t.Errorf("TaxType = %q, want 'export'", res.TaxType)
			}
			if res.Note == "" {
				t.Error("expected an explanatory note on export invoices")
			}
		})
	}
}

// --- US sales tax ---

func TestResolveInvoiceTax_USSeller_USBuyer_SalesTaxEngine(t *testing.T) {
	// No tenant GST config; env says the company is in the US.
	r := NewTaxResolver(&mockGSTConfigProvider{}, "US", "CA")

	customer := &domain.Customer{
		ID:             uuid.New(),
		BillingAddress: domain.BillingAddress{Country: "United States", State: "CA"},
	}

	res := r.ResolveInvoiceTax(context.Background(), uuid.New(), customer, "USD", 9900, "")

	// Without a wired provider the US engine is still the 0%-rate stub.
	if res.Total != 0 {
		t.Errorf("Total = %d, want 0 (stub engine)", res.Total)
	}
	if res.TaxType != "sales_tax_stub" {
		t.Errorf("TaxType = %q, want 'sales_tax_stub' (engine was consulted)", res.TaxType)
	}
	if res.IGST != 0 || res.CGST != 0 || res.SGST != 0 {
		t.Errorf("GST components = %d/%d/%d, want all zero for US sales tax", res.IGST, res.CGST, res.SGST)
	}
}

// mockSalesTaxProvider implements tax.SalesTaxProvider for resolver tests.
type mockSalesTaxProvider struct {
	calls int
	rate  float64
	err   error
}

func (m *mockSalesTaxProvider) Name() string { return "mocktax" }

func (m *mockSalesTaxProvider) LookupSalesTax(ctx context.Context, q *tax.SalesTaxQuery) (*tax.SalesTaxResult, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return &tax.SalesTaxResult{
		Rate:         m.rate,
		TaxAmount:    int64(math.Round(float64(q.Amount) * m.rate)),
		Jurisdiction: "US/" + q.ToState,
		HasNexus:     true,
	}, nil
}

func usCustomer() *domain.Customer {
	return &domain.Customer{
		ID:             uuid.New(),
		BillingAddress: domain.BillingAddress{Country: "United States", State: "CA", Zip: "90002"},
	}
}

func TestResolveInvoiceTax_USSeller_LiveProvider_RealSalesTax(t *testing.T) {
	provider := &mockSalesTaxProvider{rate: 0.0865}
	r := NewTaxResolver(&mockGSTConfigProvider{}, "US", "CA").WithSalesTaxProvider(provider)

	res := r.ResolveInvoiceTax(context.Background(), uuid.New(), usCustomer(), "USD", 10000, "")

	if res.Total != 865 {
		t.Errorf("Total = %d, want 865 (8.65%% of 10000)", res.Total)
	}
	if res.TaxType != "sales_tax" {
		t.Errorf("TaxType = %q, want 'sales_tax' (live provider replaces the stub marker)", res.TaxType)
	}
	if !strings.Contains(res.Note, "mocktax") {
		t.Errorf("Note = %q, want the provider name in the note", res.Note)
	}
	if res.IGST != 0 || res.CGST != 0 || res.SGST != 0 {
		t.Errorf("GST components = %d/%d/%d, want all zero for US sales tax", res.IGST, res.CGST, res.SGST)
	}
	if provider.calls != 1 {
		t.Errorf("provider calls = %d, want 1", provider.calls)
	}
}

func TestResolveInvoiceTax_USSeller_ProviderError_DegradesNotFails(t *testing.T) {
	provider := &mockSalesTaxProvider{err: errors.New("taxjar 503")}
	r := NewTaxResolver(&mockGSTConfigProvider{}, "US", "CA").WithSalesTaxProvider(provider)

	res := r.ResolveInvoiceTax(context.Background(), uuid.New(), usCustomer(), "USD", 10000, "")

	if res.Total != 0 {
		t.Errorf("Total = %d, want 0 (degrade to 0%% on provider error)", res.Total)
	}
	if res.TaxType != "sales_tax_error" {
		t.Errorf("TaxType = %q, want 'sales_tax_error'", res.TaxType)
	}
	if !strings.Contains(res.Note, "mocktax") {
		t.Errorf("Note = %q, want the provider name for auditability", res.Note)
	}
}

func TestResolveInvoiceTax_USSeller_RatesCachedAcrossInvoices(t *testing.T) {
	// Engines are rebuilt per invoice; the rate cache lives in the
	// resolver-held provider, so a second invoice to the same (state, zip)
	// must not hit the provider again — even for a different amount.
	provider := &mockSalesTaxProvider{rate: 0.10}
	r := NewTaxResolver(&mockGSTConfigProvider{}, "US", "CA").WithSalesTaxProvider(provider)

	first := r.ResolveInvoiceTax(context.Background(), uuid.New(), usCustomer(), "USD", 10000, "")
	second := r.ResolveInvoiceTax(context.Background(), uuid.New(), usCustomer(), "USD", 5000, "")

	if provider.calls != 1 {
		t.Errorf("provider calls = %d, want 1 (second invoice served from cache)", provider.calls)
	}
	if first.Total != 1000 || second.Total != 500 {
		t.Errorf("Totals = %d/%d, want 1000/500 (cached rate reapplied to new amount)", first.Total, second.Total)
	}
}

func TestResolveInvoiceTax_USSeller_ForeignBuyer_NoSalesTax(t *testing.T) {
	r := NewTaxResolver(nil, "US", "CA")

	customer := &domain.Customer{
		ID:             uuid.New(),
		BillingAddress: domain.BillingAddress{Country: "India"},
	}

	res := r.ResolveInvoiceTax(context.Background(), uuid.New(), customer, "USD", 9900, "")

	if res.Total != 0 {
		t.Errorf("Total = %d, want 0", res.Total)
	}
	if res.TaxType != "export" {
		t.Errorf("TaxType = %q, want 'export'", res.TaxType)
	}
}

// --- EU VAT ---

func TestResolveInvoiceTax_EUSeller_DomesticB2C_VAT(t *testing.T) {
	r := NewTaxResolver(nil, "DE", "")

	customer := &domain.Customer{
		ID:             uuid.New(),
		BillingAddress: domain.BillingAddress{Country: "Germany"},
	}

	res := r.ResolveInvoiceTax(context.Background(), uuid.New(), customer, "EUR", 10000, "")

	// Germany standard VAT 19%.
	if res.Total != 1900 {
		t.Errorf("Total = %d, want 1900 (19%% DE VAT)", res.Total)
	}
	if res.TaxType != "vat" {
		t.Errorf("TaxType = %q, want 'vat'", res.TaxType)
	}
	// Non-GST jurisdictions must not populate GST component fields.
	if res.IGST != 0 || res.CGST != 0 || res.SGST != 0 {
		t.Errorf("GST components = %d/%d/%d, want all zero for VAT", res.IGST, res.CGST, res.SGST)
	}
}

func TestResolveInvoiceTax_EUSeller_CrossBorderB2B_ReverseCharge(t *testing.T) {
	r := NewTaxResolver(nil, "DE", "")

	customer := &domain.Customer{
		ID:             uuid.New(),
		BillingAddress: domain.BillingAddress{Country: "FR"},
		TaxType:        "business",
	}

	res := r.ResolveInvoiceTax(context.Background(), uuid.New(), customer, "EUR", 10000, "")

	if res.Total != 0 {
		t.Errorf("Total = %d, want 0 under reverse charge", res.Total)
	}
	if res.TaxType != "vat_reverse_charge" {
		t.Errorf("TaxType = %q, want 'vat_reverse_charge'", res.TaxType)
	}
}

// mockVATValidator implements tax.VATValidator for resolver tests.
type mockVATValidator struct {
	calls  int
	gotCC  string
	gotNum string
	valid  bool
	name   string
	err    error
}

func (m *mockVATValidator) Name() string { return "mockvies" }

func (m *mockVATValidator) ValidateVAT(ctx context.Context, cc, num string) (*tax.VATValidation, error) {
	m.calls++
	m.gotCC, m.gotNum = cc, num
	if m.err != nil {
		return nil, m.err
	}
	return &tax.VATValidation{Valid: m.valid, Name: m.name}, nil
}

func euB2BCustomer(country, vatNumber string) *domain.Customer {
	c := &domain.Customer{
		ID:             uuid.New(),
		BillingAddress: domain.BillingAddress{Country: country},
		TaxType:        "business",
	}
	if vatNumber != "" {
		c.TaxID = domain.StringPtr(vatNumber)
	}
	return c
}

func TestResolveInvoiceTax_EU_VIESValid_ReverseCharge(t *testing.T) {
	val := &mockVATValidator{valid: true, name: "ACME SARL"}
	r := NewTaxResolver(nil, "DE", "").WithVATValidator(val)

	res := r.ResolveInvoiceTax(context.Background(), uuid.New(), euB2BCustomer("FR", "FRXX123456789"), "EUR", 10000, "")

	if res.Total != 0 || res.TaxType != "vat_reverse_charge" {
		t.Errorf("got Total=%d Type=%q, want 0 / vat_reverse_charge", res.Total, res.TaxType)
	}
	if val.calls != 1 || val.gotCC != "FR" || val.gotNum != "XX123456789" {
		t.Errorf("validator saw calls=%d cc=%q num=%q, want 1 / FR / XX123456789", val.calls, val.gotCC, val.gotNum)
	}
	if !strings.Contains(res.Note, "validated") || !strings.Contains(res.Note, "ACME SARL") {
		t.Errorf("Note = %q, want validation confirmation with trader name", res.Note)
	}
}

func TestResolveInvoiceTax_EU_VIESInvalid_ChargesVAT(t *testing.T) {
	val := &mockVATValidator{valid: false}
	r := NewTaxResolver(nil, "DE", "").WithVATValidator(val)

	res := r.ResolveInvoiceTax(context.Background(), uuid.New(), euB2BCustomer("FR", "FRXX123456789"), "EUR", 10000, "")

	// Not eligible for reverse charge -> treated as B2C cross-border, so the
	// buyer-country (FR, 20%) VAT applies and the invoice is still produced.
	if res.Total != 2000 || res.TaxType != "vat" {
		t.Errorf("got Total=%d Type=%q, want 2000 / vat (FR 20%%)", res.Total, res.TaxType)
	}
	if !strings.Contains(res.Note, "not valid") {
		t.Errorf("Note = %q, want an invalid-VAT explanation", res.Note)
	}
}

func TestResolveInvoiceTax_EU_VIESUnavailable_DegradesToReverseCharge(t *testing.T) {
	val := &mockVATValidator{err: tax.ErrVATUnavailable}
	r := NewTaxResolver(nil, "DE", "").WithVATValidator(val)

	res := r.ResolveInvoiceTax(context.Background(), uuid.New(), euB2BCustomer("FR", "FRXX123456789"), "EUR", 10000, "")

	// A VIES outage must never fail the invoice: degrade to presence-based
	// reverse charge with an auditable note.
	if res.Total != 0 || res.TaxType != "vat_reverse_charge" {
		t.Errorf("got Total=%d Type=%q, want 0 / vat_reverse_charge (degraded)", res.Total, res.TaxType)
	}
	if !strings.Contains(res.Note, "unavailable") {
		t.Errorf("Note = %q, want an outage note", res.Note)
	}
}

func TestResolveInvoiceTax_EU_VIESFormatError_ChargesVAT(t *testing.T) {
	val := &mockVATValidator{err: tax.ErrVATInvalidFormat}
	r := NewTaxResolver(nil, "DE", "").WithVATValidator(val)

	res := r.ResolveInvoiceTax(context.Background(), uuid.New(), euB2BCustomer("FR", "FRXX123456789"), "EUR", 10000, "")

	if res.Total != 2000 || res.TaxType != "vat" {
		t.Errorf("got Total=%d Type=%q, want 2000 / vat", res.Total, res.TaxType)
	}
	if !strings.Contains(res.Note, "failed validation") {
		t.Errorf("Note = %q, want a validation-failure note", res.Note)
	}
}

func TestResolveInvoiceTax_EU_B2BNoVATNumber_ChargesVAT(t *testing.T) {
	val := &mockVATValidator{valid: true}
	r := NewTaxResolver(nil, "DE", "").WithVATValidator(val)

	// Business buyer (by tax_type) but no VAT number to validate.
	res := r.ResolveInvoiceTax(context.Background(), uuid.New(), euB2BCustomer("FR", ""), "EUR", 10000, "")

	if res.Total != 2000 || res.TaxType != "vat" {
		t.Errorf("got Total=%d Type=%q, want 2000 / vat (no number -> VAT charged)", res.Total, res.TaxType)
	}
	if val.calls != 0 {
		t.Errorf("validator calls = %d, want 0 (nothing to validate)", val.calls)
	}
}

func TestResolveInvoiceTax_EU_VIESEnabled_DomesticB2B_NotValidated(t *testing.T) {
	// Domestic (DE seller, DE buyer) is not a cross-border reverse-charge
	// candidate, so VIES is never consulted and domestic VAT applies.
	val := &mockVATValidator{valid: true}
	r := NewTaxResolver(nil, "DE", "").WithVATValidator(val)

	res := r.ResolveInvoiceTax(context.Background(), uuid.New(), euB2BCustomer("DE", "DE123456789"), "EUR", 10000, "")

	if res.Total != 1900 || res.TaxType != "vat" {
		t.Errorf("got Total=%d Type=%q, want 1900 / vat (DE 19%%)", res.Total, res.TaxType)
	}
	if val.calls != 0 {
		t.Errorf("validator calls = %d, want 0 for a domestic sale", val.calls)
	}
}

func TestResolveInvoiceTax_EU_VIESDisabled_PresenceBasedUnchanged(t *testing.T) {
	// No validator wired: cross-border B2B with a VAT number present still gets
	// reverse charge (historical presence-based behaviour is preserved).
	r := NewTaxResolver(nil, "DE", "")

	res := r.ResolveInvoiceTax(context.Background(), uuid.New(), euB2BCustomer("FR", "FRXX123456789"), "EUR", 10000, "")

	if res.Total != 0 || res.TaxType != "vat_reverse_charge" {
		t.Errorf("got Total=%d Type=%q, want 0 / vat_reverse_charge", res.Total, res.TaxType)
	}
}

func TestSplitEUVATNumber(t *testing.T) {
	cases := []struct {
		raw, fallback string
		wantCC, wantN string
	}{
		{"DE123456789", "FR", "DE", "123456789"},
		{"de 123.456-789", "FR", "DE", "123456789"},
		{"EL123456789", "IT", "GR", "123456789"}, // EL -> ISO GR
		{"123456789", "FR", "FR", "123456789"},   // no prefix -> fallback
		{"", "FR", "FR", ""},
		{"US123456789", "FR", "FR", "US123456789"}, // non-EU prefix not stripped
	}
	for _, c := range cases {
		gotCC, gotN := splitEUVATNumber(c.raw, c.fallback)
		if gotCC != c.wantCC || gotN != c.wantN {
			t.Errorf("splitEUVATNumber(%q,%q) = (%q,%q), want (%q,%q)", c.raw, c.fallback, gotCC, gotN, c.wantCC, c.wantN)
		}
	}
}

// --- Fallbacks ---

func TestResolveInvoiceTax_ConfigLookupError_EnvFallback(t *testing.T) {
	provider := &mockGSTConfigProvider{err: errors.New("db down")}
	r := NewTaxResolver(provider, "IN", "TN")

	res := r.ResolveInvoiceTax(context.Background(), uuid.New(), inCustomer("TN"), "INR", 100000, "")

	// Env default TN + POS TN -> intra-state despite the failed lookup.
	if res.CGST != 9000 || res.SGST != 9000 {
		t.Errorf("CGST/SGST = %d/%d, want 9000/9000 via env fallback", res.CGST, res.SGST)
	}
	if provider.calls != 1 {
		t.Errorf("GST config lookups = %d, want 1", provider.calls)
	}
}

func TestResolveInvoiceTax_NoConfig_EnvFallback(t *testing.T) {
	provider := &mockGSTConfigProvider{cfg: nil} // repo returns (nil, nil)
	r := NewTaxResolver(provider, "IN", "TN")

	res := r.ResolveInvoiceTax(context.Background(), uuid.New(), inCustomer("KA"), "INR", 100000, "")

	if res.IGST != 18000 {
		t.Errorf("IGST = %d, want 18000 (env TN vs POS KA is inter-state)", res.IGST)
	}
}

func TestResolveInvoiceTax_NilProvider_EnvDefaults(t *testing.T) {
	r := NewTaxResolver(nil, "", "") // empty defaults -> IN/TN

	res := r.ResolveInvoiceTax(context.Background(), uuid.New(), inCustomer("TN"), "INR", 50000, "")

	if res.Total != 9000 {
		t.Errorf("Total = %d, want 9000 (18%% of 50000)", res.Total)
	}
	if res.CGST != 4500 || res.SGST != 4500 {
		t.Errorf("CGST/SGST = %d/%d, want 4500/4500", res.CGST, res.SGST)
	}
}

// --- Normalization helpers ---

func TestNormalizeINState(t *testing.T) {
	cases := map[string]string{
		"33":         "TN",
		"29":         "KA",
		"TN":         "TN",
		"ka":         "KA",
		"Tamil Nadu": "TN",
		"":           "",
	}
	for in, want := range cases {
		if got := normalizeINState(in); got != want {
			t.Errorf("normalizeINState(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeCountry(t *testing.T) {
	cases := map[string]string{
		"India":         "IN",
		"in":            "IN",
		"United States": "US",
		"USA":           "US",
		"Germany":       "DE",
		"":              "",
	}
	for in, want := range cases {
		if got := normalizeCountry(in); got != want {
			t.Errorf("normalizeCountry(%q) = %q, want %q", in, got, want)
		}
	}
}
