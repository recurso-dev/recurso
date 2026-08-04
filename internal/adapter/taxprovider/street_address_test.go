package taxprovider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/recurso-dev/recurso/internal/core/service/tax"
)

// The street line is optional on SalesTaxQuery. These tests pin both halves of
// that contract for the existing adapters: forwarded when present, absent from
// the request when not, so a ZIP-only deployment sends exactly what it sent
// before.

func TestTaxJar_ToStreet_ForwardedWhenPresent(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(caResponse))
	}))
	defer srv.Close()

	q := caQuery()
	q.ToStreet = "200 Spectrum Center Dr"

	if _, err := NewTaxJarProvider("k", srv.URL).LookupSalesTax(context.Background(), q); err != nil {
		t.Fatalf("LookupSalesTax: %v", err)
	}
	if got := gotBody["to_street"]; got != "200 Spectrum Center Dr" {
		t.Errorf("to_street = %v, want the street line", got)
	}
}

func TestTaxJar_ToStreet_OmittedWhenEmpty(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(caResponse))
	}))
	defer srv.Close()

	if _, err := NewTaxJarProvider("k", srv.URL).LookupSalesTax(context.Background(), caQuery()); err != nil {
		t.Fatalf("LookupSalesTax: %v", err)
	}
	if _, present := gotBody["to_street"]; present {
		t.Errorf("to_street must be omitted when the query carries no street, got %v", gotBody["to_street"])
	}
}

func TestAvalara_ShipToLine1_ForwardedWhenPresent(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"totalTax":1.43,"summary":[{"region":"CA","jurisName":"LOS ANGELES","rate":0.0865}]}`))
	}))
	defer srv.Close()

	q := &tax.SalesTaxQuery{
		FromCountry: "US", FromState: "CA",
		ToCountry: "US", ToState: "CA", ToZip: "90002",
		ToStreet: "1 Infinite Loop",
		Amount:   16_50, Currency: "USD",
	}

	if _, err := NewAvalaraProvider("acct", "lic", "DEFAULT", srv.URL).LookupSalesTax(context.Background(), q); err != nil {
		t.Fatalf("LookupSalesTax: %v", err)
	}
	addrs, _ := gotBody["addresses"].(map[string]any)
	shipTo, _ := addrs["shipTo"].(map[string]any)
	if got := shipTo["line1"]; got != "1 Infinite Loop" {
		t.Errorf("shipTo.line1 = %v, want the street line", got)
	}
}

func TestAvalara_ShipToLine1_OmittedWhenEmpty(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"totalTax":1.43,"summary":[{"region":"CA","jurisName":"LOS ANGELES","rate":0.0865}]}`))
	}))
	defer srv.Close()

	q := &tax.SalesTaxQuery{
		FromCountry: "US", FromState: "CA",
		ToCountry: "US", ToState: "CA", ToZip: "90002",
		Amount: 16_50, Currency: "USD",
	}

	if _, err := NewAvalaraProvider("acct", "lic", "DEFAULT", srv.URL).LookupSalesTax(context.Background(), q); err != nil {
		t.Fatalf("LookupSalesTax: %v", err)
	}
	addrs, _ := gotBody["addresses"].(map[string]any)
	shipTo, _ := addrs["shipTo"].(map[string]any)
	if _, present := shipTo["line1"]; present {
		t.Errorf("shipTo.line1 must be omitted when the query carries no street, got %v", shipTo["line1"])
	}
}
