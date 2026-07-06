package taxprovider

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/swapnull-in/recur-so/internal/core/service/tax"
)

func caQuery() *tax.SalesTaxQuery {
	return &tax.SalesTaxQuery{
		FromCountry: "US",
		FromState:   "CA",
		ToCountry:   "US",
		ToState:     "CA",
		ToZip:       "90002",
		Amount:      16_50, // $16.50
		Currency:    "USD",
	}
}

const caResponse = `{
  "tax": {
    "order_total_amount": 16.5,
    "shipping": 0,
    "taxable_amount": 16.5,
    "amount_to_collect": 1.43,
    "rate": 0.0865,
    "has_nexus": true,
    "freight_taxable": false,
    "tax_source": "destination",
    "jurisdictions": {
      "country": "US",
      "state": "CA",
      "county": "LOS ANGELES",
      "city": "LOS ANGELES"
    }
  }
}`

func TestTaxJar_LookupSalesTax_RequestShapeAndParsing(t *testing.T) {
	var gotPath, gotMethod, gotAuth, gotContentType string
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(caResponse))
	}))
	defer srv.Close()

	p := NewTaxJarProvider("test-key", srv.URL)
	res, err := p.LookupSalesTax(context.Background(), caQuery())
	if err != nil {
		t.Fatalf("LookupSalesTax: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/v2/taxes" {
		t.Errorf("request = %s %s, want POST /v2/taxes", gotMethod, gotPath)
	}
	if gotAuth != "Bearer test-key" {
		t.Errorf("Authorization = %q, want 'Bearer test-key'", gotAuth)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotContentType)
	}

	// TaxJar takes decimal dollars, cents converted.
	if gotBody["amount"] != 16.5 {
		t.Errorf("body amount = %v, want 16.5 (dollars)", gotBody["amount"])
	}
	if _, ok := gotBody["shipping"]; !ok {
		t.Error("body must carry the required 'shipping' parameter")
	}
	for k, want := range map[string]string{
		"from_country": "US", "from_state": "CA",
		"to_country": "US", "to_state": "CA", "to_zip": "90002",
	} {
		if gotBody[k] != want {
			t.Errorf("body %s = %v, want %q", k, gotBody[k], want)
		}
	}

	if res.Rate != 0.0865 {
		t.Errorf("Rate = %v, want 0.0865", res.Rate)
	}
	if res.TaxAmount != 143 {
		t.Errorf("TaxAmount = %d cents, want 143", res.TaxAmount)
	}
	if !res.HasNexus {
		t.Error("HasNexus = false, want true")
	}
	if res.Jurisdiction != "US/CA/LOS ANGELES/LOS ANGELES" {
		t.Errorf("Jurisdiction = %q", res.Jurisdiction)
	}
}

func TestTaxJar_AuthError_NotRetried(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"Unauthorized","detail":"Not authorized for route","status":401}`))
	}))
	defer srv.Close()

	p := NewTaxJarProvider("bad-key", srv.URL)
	_, err := p.LookupSalesTax(context.Background(), caQuery())

	if !errors.Is(err, ErrTaxJarAuth) {
		t.Fatalf("err = %v, want ErrTaxJarAuth", err)
	}
	if calls != 1 {
		t.Errorf("HTTP calls = %d, want 1 (4xx must not be retried)", calls)
	}
}

func TestTaxJar_BadRequest_TypedError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"Bad Request","detail":"to_zip is invalid","status":400}`))
	}))
	defer srv.Close()

	p := NewTaxJarProvider("key", srv.URL)
	_, err := p.LookupSalesTax(context.Background(), caQuery())

	if !errors.Is(err, ErrTaxJarBadRequest) {
		t.Fatalf("err = %v, want ErrTaxJarBadRequest", err)
	}
	var tjErr *TaxJarError
	if !errors.As(err, &tjErr) || tjErr.StatusCode != 400 {
		t.Errorf("err = %#v, want *TaxJarError with StatusCode 400", err)
	}
}

func TestTaxJar_ServerError_RetriedExactlyOnce(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := NewTaxJarProvider("key", srv.URL)
	_, err := p.LookupSalesTax(context.Background(), caQuery())

	if !errors.Is(err, ErrTaxJarUnavailable) {
		t.Fatalf("err = %v, want ErrTaxJarUnavailable", err)
	}
	if calls != 2 {
		t.Errorf("HTTP calls = %d, want 2 (one retry, no more)", calls)
	}
}

func TestTaxJar_ServerError_ThenSuccess_RetryRecovers(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte(caResponse))
	}))
	defer srv.Close()

	p := NewTaxJarProvider("key", srv.URL)
	res, err := p.LookupSalesTax(context.Background(), caQuery())
	if err != nil {
		t.Fatalf("LookupSalesTax after retry: %v", err)
	}
	if res.TaxAmount != 143 {
		t.Errorf("TaxAmount = %d, want 143", res.TaxAmount)
	}
	if calls != 2 {
		t.Errorf("HTTP calls = %d, want 2", calls)
	}
}

func TestTaxJar_DefaultBaseURL(t *testing.T) {
	p := NewTaxJarProvider("key", "")
	if p.baseURL != DefaultTaxJarURL {
		t.Errorf("baseURL = %q, want %q", p.baseURL, DefaultTaxJarURL)
	}
	if p.Name() != "taxjar" {
		t.Errorf("Name() = %q, want 'taxjar'", p.Name())
	}
}
