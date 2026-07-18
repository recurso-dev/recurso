package taxprovider

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/recurso-dev/recurso/internal/core/service/tax"
)

func avalaraQuery() *tax.SalesTaxQuery {
	return &tax.SalesTaxQuery{
		FromCountry: "US", FromState: "CA", FromZip: "94016",
		ToCountry: "US", ToState: "NY", ToZip: "10001",
		Amount: 10000, Currency: "USD",
	}
}

func TestAvalaraLookupSalesTax(t *testing.T) {
	var body map[string]any
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&body)
		if r.URL.Path != "/api/v2/transactions/create" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"totalTax": 8.88,
			"summary": [
				{"region":"NY","jurisName":"NEW YORK","rate":0.04},
				{"region":"NY","jurisName":"NEW YORK CITY","rate":0.04875}
			]
		}`))
	}))
	defer srv.Close()

	p := NewAvalaraProvider("acct1", "lic1", "", srv.URL)
	res, err := p.LookupSalesTax(context.Background(), avalaraQuery())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// $8.88 -> 888 cents; rate is the jurisdiction sum.
	if res.TaxAmount != 888 {
		t.Fatalf("TaxAmount = %d, want 888", res.TaxAmount)
	}
	if res.Rate < 0.0887 || res.Rate > 0.0888 {
		t.Fatalf("Rate = %f, want ~0.08875", res.Rate)
	}
	if !res.HasNexus {
		t.Fatal("non-zero tax must report nexus")
	}
	if res.Jurisdiction != "US/NY/NEW YORK+NEW YORK CITY" {
		t.Fatalf("Jurisdiction = %q", res.Jurisdiction)
	}

	// Request shape: SalesOrder quote (never committed), basic auth, major units.
	if body["type"] != "SalesOrder" {
		t.Fatalf("type = %v — quotes must never commit to the Avalara ledger", body["type"])
	}
	if auth == "" || auth[:6] != "Basic " {
		t.Fatalf("auth = %q, want Basic", auth)
	}
	lines := body["lines"].([]any)
	if lines[0].(map[string]any)["amount"].(float64) != 100.0 {
		t.Fatalf("line amount = %v, want 100.00 major units", lines[0])
	}
	addrs := body["addresses"].(map[string]any)
	if addrs["shipTo"].(map[string]any)["region"] != "NY" {
		t.Fatalf("shipTo = %+v", addrs["shipTo"])
	}
}

func TestAvalaraErrorTaxonomy(t *testing.T) {
	cases := []struct {
		status int
		kind   error
	}{
		{http.StatusUnauthorized, ErrAvalaraAuth},
		{http.StatusForbidden, ErrAvalaraAuth},
		{http.StatusBadRequest, ErrAvalaraBadRequest},
		{http.StatusInternalServerError, ErrAvalaraUnavailable},
	}
	for _, tc := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tc.status)
		}))
		p := NewAvalaraProvider("a", "l", "", srv.URL)
		_, err := p.LookupSalesTax(context.Background(), avalaraQuery())
		srv.Close()
		if !errors.Is(err, tc.kind) {
			t.Errorf("status %d: err = %v, want kind %v", tc.status, err, tc.kind)
		}
	}
}

func TestAvalaraZeroTaxNoNexus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"totalTax": 0, "summary": []}`))
	}))
	defer srv.Close()
	p := NewAvalaraProvider("a", "l", "", srv.URL)
	res, err := p.LookupSalesTax(context.Background(), avalaraQuery())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.HasNexus || res.TaxAmount != 0 {
		t.Fatalf("res = %+v, want zero tax without nexus", res)
	}
}
