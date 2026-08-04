package taxprovider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/recurso-dev/recurso/internal/core/service/tax"
)

func ziptaxCAQuery() *tax.SalesTaxQuery {
	return &tax.SalesTaxQuery{
		FromCountry: "US",
		FromState:   "CA",
		ToCountry:   "US",
		ToState:     "CA",
		ToZip:       "92618",
		Amount:      16_50, // $16.50
		Currency:    "USD",
	}
}

// ziptaxCAResponse is a real-shaped v60 payload for 200 Spectrum Center Dr,
// Irvine CA. Note that baseRates carries BOTH a sales-tax and a use-tax entry
// at every level. Summing it yields ~15.5% against a true rate of 7.75%.
const ziptaxCAResponse = `{
  "metadata": { "version": "v60", "response": { "code": 100, "name": "RESPONSE_CODE_SUCCESS", "message": "Successful API Request." } },
  "baseRates": [
    { "rate": 0.0725, "jurType": "US_STATE_SALES_TAX",  "jurName": "CA",     "jurTaxCode": "06" },
    { "rate": 0.0725, "jurType": "US_STATE_USE_TAX",    "jurName": "CA",     "jurTaxCode": "06" },
    { "rate": 0.005,  "jurType": "US_COUNTY_SALES_TAX", "jurName": "ORANGE", "jurTaxCode": "30" },
    { "rate": 0.005,  "jurType": "US_COUNTY_USE_TAX",   "jurName": "ORANGE", "jurTaxCode": "30" },
    { "rate": 0,      "jurType": "US_CITY_SALES_TAX",   "jurName": "IRVINE", "jurTaxCode": null },
    { "rate": 0,      "jurType": "US_CITY_USE_TAX",     "jurName": "IRVINE", "jurTaxCode": null }
  ],
  "taxSummaries": [
    { "rate": 0.0775, "taxType": "SALES_TAX", "summaryName": "Total Base Sales Tax" },
    { "rate": 0.0775, "taxType": "USE_TAX",   "summaryName": "Total Base Use Tax" }
  ],
  "sourcingRules": { "adjustmentType": "ORIGIN_DESTINATION", "description": "Destination Based Taxation", "value": "D" },
  "shipping": { "taxable": "N" },
  "service":  { "taxable": "N" },
  "addressDetail": { "normalizedAddress": "200 Spectrum Center Dr, Irvine, CA 92618-5003, United States", "incorporated": "true" }
}`

// ziptaxErrorResponse builds an error payload carrying a given application code.
func ziptaxErrorResponse(code int, name string) string {
	return fmt.Sprintf(`{"metadata":{"version":"v60","response":{"code":%d,"name":%q,"message":"test"}},"baseRates":[],"taxSummaries":[]}`, code, name)
}

func TestZiptax_LookupSalesTax_RequestShapeAndParsing(t *testing.T) {
	var gotPath, gotMethod, gotKey, gotPostalCode, gotCountry, gotRawQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotKey = r.Header.Get("X-API-KEY")
		gotPostalCode = r.URL.Query().Get("postalcode")
		gotCountry = r.URL.Query().Get("countryCode")
		gotRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(ziptaxCAResponse))
	}))
	defer srv.Close()

	p := NewZiptaxProvider("zt-test-key", srv.URL)
	res, err := p.LookupSalesTax(context.Background(), ziptaxCAQuery())
	if err != nil {
		t.Fatalf("LookupSalesTax: %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if gotPath != "/request/v60" {
		t.Errorf("path = %q, want /request/v60", gotPath)
	}
	if gotKey != "zt-test-key" {
		t.Errorf("X-API-KEY = %q, want zt-test-key", gotKey)
	}
	if gotPostalCode != "92618" {
		t.Errorf("postalcode = %q, want 92618", gotPostalCode)
	}
	if gotCountry != "USA" {
		t.Errorf("countryCode = %q, want alpha-3 USA", gotCountry)
	}
	// The API key must never travel in the URL, where it would land in access
	// and proxy logs.
	if q := gotRawQuery; strings.Contains(q, "key=") {
		t.Errorf("raw query %q must not carry the API key", q)
	}

	if res.Rate != 0.0775 {
		t.Errorf("Rate = %v, want 0.0775", res.Rate)
	}
	if res.TaxAmount != 128 { // round(1650 * 0.0775) = 128
		t.Errorf("TaxAmount = %d, want 128", res.TaxAmount)
	}
	if res.Jurisdiction != "US/CA/ORANGE/IRVINE" {
		t.Errorf("Jurisdiction = %q, want US/CA/ORANGE/IRVINE", res.Jurisdiction)
	}
}

// TestZiptax_BaseRatesAreNotSummed_DoubleCountRegression is a named guard
// against the single most likely bug in this adapter: baseRates holds parallel
// sales-tax and use-tax entries at every jurisdiction level, so summing it
// returns ~0.155 instead of 0.0775. The rate must come from the SALES_TAX
// entry of taxSummaries. Do not delete this test.
func TestZiptax_BaseRatesAreNotSummed_DoubleCountRegression(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(ziptaxCAResponse))
	}))
	defer srv.Close()

	res, err := NewZiptaxProvider("k", srv.URL).LookupSalesTax(context.Background(), ziptaxCAQuery())
	if err != nil {
		t.Fatalf("LookupSalesTax: %v", err)
	}
	if res.Rate != 0.0775 {
		t.Fatalf("Rate = %v, want 0.0775", res.Rate)
	}
	// The specific wrong answer this test exists to catch.
	const summedBaseRates = 0.0725 + 0.0725 + 0.005 + 0.005
	if res.Rate == summedBaseRates {
		t.Fatalf("Rate = %v: baseRates were summed (sales + use double-counted)", res.Rate)
	}
}

func TestZiptax_MissingSalesTaxSummary_IsAnError(t *testing.T) {
	// Only a USE_TAX summary. A positional taxSummaries[0] fallback would
	// silently quote the use-tax rate as sales tax.
	const body = `{
      "metadata": { "version": "v60", "response": { "code": 100, "name": "RESPONSE_CODE_SUCCESS" } },
      "baseRates": [],
      "taxSummaries": [ { "rate": 0.0775, "taxType": "USE_TAX", "summaryName": "Total Base Use Tax" } ]
    }`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	_, err := NewZiptaxProvider("k", srv.URL).LookupSalesTax(context.Background(), ziptaxCAQuery())
	if !errors.Is(err, ErrZiptaxBadRequest) {
		t.Fatalf("err = %v, want ErrZiptaxBadRequest", err)
	}
}

func TestZiptax_ErrorTaxonomy(t *testing.T) {
	cases := []struct {
		code   int
		name   string
		status int
		want   error
	}{
		{101, "RESPONSE_CODE_INVALID_KEY", 401, ErrZiptaxAuth},
		{104, "RESPONSE_CODE_INVALID_POSTAL_CODE", 422, ErrZiptaxBadRequest},
		{106, "RESPONSE_CODE_API_ERROR", 500, ErrZiptaxUnavailable},
		{107, "RESPONSE_CODE_FEATURE_NOT_ENABLED", 405, ErrZiptaxNotEntitled},
		{108, "RESPONSE_CODE_REQUEST_LIMIT_MET", 429, ErrZiptaxUnavailable},
		{109, "RESPONSE_CODE_ADDRESS_INCOMPLETE", 422, ErrZiptaxBadRequest},
		{110, "RESPONSE_CODE_NO_RESULT", 422, ErrZiptaxBadRequest},
		{112, "RESPONSE_CODE_INTERNATIONAL_NOT_ENABLED", 403, ErrZiptaxNotEntitled},
		{113, "RESPONSE_CODE_PRODUCT_RULES_NOT_ENABLED", 403, ErrZiptaxNotEntitled},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(ziptaxErrorResponse(tc.code, tc.name)))
			}))
			defer srv.Close()

			_, err := NewZiptaxProvider("k", srv.URL).LookupSalesTax(context.Background(), ziptaxCAQuery())
			if !errors.Is(err, tc.want) {
				t.Fatalf("code %d: err = %v, want %v", tc.code, err, tc.want)
			}
			var zerr *ZiptaxError
			if !errors.As(err, &zerr) {
				t.Fatalf("code %d: err is not *ZiptaxError: %v", tc.code, err)
			}
			if zerr.ResponseCode != tc.code {
				t.Errorf("ResponseCode = %d, want %d", zerr.ResponseCode, tc.code)
			}
			if zerr.StatusCode != tc.status {
				t.Errorf("StatusCode = %d, want %d", zerr.StatusCode, tc.status)
			}
		})
	}
}

// TestZiptax_BranchesOnApplicationCodeNotHTTPStatus pins the central design
// decision: Ziptax can return a failure code under HTTP 200, so the body is the
// authority.
func TestZiptax_BranchesOnApplicationCodeNotHTTPStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK) // 200 OK ...
		_, _ = w.Write([]byte(ziptaxErrorResponse(101, "RESPONSE_CODE_INVALID_KEY")))
	}))
	defer srv.Close()

	_, err := NewZiptaxProvider("k", srv.URL).LookupSalesTax(context.Background(), ziptaxCAQuery())
	if !errors.Is(err, ErrZiptaxAuth) {
		t.Fatalf("err = %v, want ErrZiptaxAuth despite HTTP 200", err)
	}
}

func TestZiptax_APIError_RetriedExactlyOnce(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(500)
		_, _ = w.Write([]byte(ziptaxErrorResponse(106, "RESPONSE_CODE_API_ERROR")))
	}))
	defer srv.Close()

	_, err := NewZiptaxProvider("k", srv.URL).LookupSalesTax(context.Background(), ziptaxCAQuery())
	if !errors.Is(err, ErrZiptaxUnavailable) {
		t.Fatalf("err = %v, want ErrZiptaxUnavailable", err)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("calls = %d, want 2 (one retry)", got)
	}
}

func TestZiptax_RateLimit_RetriedExactlyOnce(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(429)
		_, _ = w.Write([]byte(ziptaxErrorResponse(108, "RESPONSE_CODE_REQUEST_LIMIT_MET")))
	}))
	defer srv.Close()

	_, _ = NewZiptaxProvider("k", srv.URL).LookupSalesTax(context.Background(), ziptaxCAQuery())
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("calls = %d, want 2 (one retry)", got)
	}
}

func TestZiptax_AuthError_NotRetried(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(401)
		_, _ = w.Write([]byte(ziptaxErrorResponse(101, "RESPONSE_CODE_INVALID_KEY")))
	}))
	defer srv.Close()

	_, _ = NewZiptaxProvider("k", srv.URL).LookupSalesTax(context.Background(), ziptaxCAQuery())
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("calls = %d, want 1 (auth failures are not retried)", got)
	}
}

func TestZiptax_APIError_ThenSuccess_RetryRecovers(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(500)
			_, _ = w.Write([]byte(ziptaxErrorResponse(106, "RESPONSE_CODE_API_ERROR")))
			return
		}
		_, _ = w.Write([]byte(ziptaxCAResponse))
	}))
	defer srv.Close()

	res, err := NewZiptaxProvider("k", srv.URL).LookupSalesTax(context.Background(), ziptaxCAQuery())
	if err != nil {
		t.Fatalf("LookupSalesTax: %v", err)
	}
	if res.Rate != 0.0775 {
		t.Errorf("Rate = %v, want 0.0775", res.Rate)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("calls = %d, want 2", got)
	}
}

func TestZiptax_ExemptQuery_ShortCircuitsWithNoHTTPCall(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		_, _ = w.Write([]byte(ziptaxCAResponse))
	}))
	defer srv.Close()

	q := ziptaxCAQuery()
	q.Exempt = true
	q.ExemptionNo = "RESALE-123"

	res, err := NewZiptaxProvider("k", srv.URL).LookupSalesTax(context.Background(), q)
	if err != nil {
		t.Fatalf("LookupSalesTax: %v", err)
	}
	if res.Rate != 0 || res.TaxAmount != 0 {
		t.Errorf("exempt result = rate %v amount %d, want 0/0", res.Rate, res.TaxAmount)
	}
	// Quota is the binding constraint on the free tier; an exempt sale is zero
	// by definition and must not spend a call.
	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Fatalf("calls = %d, want 0 (exempt queries must not hit the API)", got)
	}
}

func TestZiptax_HasNexus(t *testing.T) {
	cases := []struct {
		name        string
		nexusStates []string
		toState     string
		want        bool
	}{
		{"destination in nexus states", []string{"NY", "CA"}, "CA", true},
		{"destination not in nexus states", []string{"NY", "TX"}, "CA", false},
		{"empty nexus states defers to Recurso", nil, "CA", true},
		{"case and whitespace insensitive", []string{" ca "}, "CA", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(ziptaxCAResponse))
			}))
			defer srv.Close()

			q := ziptaxCAQuery()
			q.NexusStates = tc.nexusStates
			q.ToState = tc.toState

			res, err := NewZiptaxProvider("k", srv.URL).LookupSalesTax(context.Background(), q)
			if err != nil {
				t.Fatalf("LookupSalesTax: %v", err)
			}
			if res.HasNexus != tc.want {
				t.Errorf("HasNexus = %v, want %v", res.HasNexus, tc.want)
			}
		})
	}
}

// TestZiptax_CountryMapping covers the California/Canada trap: alpha-2 "CA" is
// Canada, but a ToState of "CA" is California. Passing ToCountry through
// unmapped would send countryCode=CA for a plain California sale.
func TestZiptax_CountryMapping(t *testing.T) {
	cases := []struct {
		toCountry   string
		wantCountry string
		wantErr     bool
	}{
		{"US", "USA", false},
		{"CA", "CAN", false}, // Canada, not California
		{"PR", "PRI", false},
		{"", "USA", false}, // empty defaults to the US
		{"XX", "", true},
	}
	for _, tc := range cases {
		t.Run("ToCountry="+tc.toCountry, func(t *testing.T) {
			var calls int32
			var gotCountry string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&calls, 1)
				gotCountry = r.URL.Query().Get("countryCode")
				_, _ = w.Write([]byte(ziptaxCAResponse))
			}))
			defer srv.Close()

			q := ziptaxCAQuery()
			q.ToCountry = tc.toCountry

			_, err := NewZiptaxProvider("k", srv.URL).LookupSalesTax(context.Background(), q)
			if tc.wantErr {
				if !errors.Is(err, ErrZiptaxBadRequest) {
					t.Fatalf("err = %v, want ErrZiptaxBadRequest", err)
				}
				if got := atomic.LoadInt32(&calls); got != 0 {
					t.Fatalf("calls = %d, want 0 (unknown country must not spend quota)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("LookupSalesTax: %v", err)
			}
			if gotCountry != tc.wantCountry {
				t.Errorf("countryCode = %q, want %q", gotCountry, tc.wantCountry)
			}
		})
	}
}

// TestZiptax_NexusStatesNotForwarded pins the documented decision that nexus is
// Recurso's determination, not the rate API's.
func TestZiptax_NexusStatesNotForwarded(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(ziptaxCAResponse))
	}))
	defer srv.Close()

	q := ziptaxCAQuery()
	q.NexusStates = []string{"NY", "TX"}

	if _, err := NewZiptaxProvider("k", srv.URL).LookupSalesTax(context.Background(), q); err != nil {
		t.Fatalf("LookupSalesTax: %v", err)
	}
	for _, forbidden := range []string{"nexus", "NY", "TX"} {
		if strings.Contains(gotQuery, forbidden) {
			t.Errorf("query %q must not carry %q", gotQuery, forbidden)
		}
	}
}

// TestZiptax_OriginFieldsNotForwarded documents that the v60 rates endpoint has
// no origin parameter. Ziptax resolves sourcing server-side. This differs from
// the TaxJar adapter, which does pass from_* through.
func TestZiptax_OriginFieldsNotForwarded(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(ziptaxCAResponse))
	}))
	defer srv.Close()

	q := ziptaxCAQuery()
	q.FromZip = "99501"

	if _, err := NewZiptaxProvider("k", srv.URL).LookupSalesTax(context.Background(), q); err != nil {
		t.Fatalf("LookupSalesTax: %v", err)
	}
	if strings.Contains(gotQuery, "99501") || strings.Contains(gotQuery, "from") {
		t.Errorf("query %q must not carry origin fields", gotQuery)
	}
}

func TestZiptax_NonUSDCurrency_RejectedWithNoHTTPCall(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		_, _ = w.Write([]byte(ziptaxCAResponse))
	}))
	defer srv.Close()

	q := ziptaxCAQuery()
	q.Currency = "EUR"

	_, err := NewZiptaxProvider("k", srv.URL).LookupSalesTax(context.Background(), q)
	if !errors.Is(err, ErrZiptaxBadRequest) {
		t.Fatalf("err = %v, want ErrZiptaxBadRequest", err)
	}
	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Fatalf("calls = %d, want 0", got)
	}
}

func TestZiptax_MissingZip_RejectedWithNoHTTPCall(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		_, _ = w.Write([]byte(ziptaxCAResponse))
	}))
	defer srv.Close()

	q := ziptaxCAQuery()
	q.ToZip = ""

	_, err := NewZiptaxProvider("k", srv.URL).LookupSalesTax(context.Background(), q)
	if !errors.Is(err, ErrZiptaxBadRequest) {
		t.Fatalf("err = %v, want ErrZiptaxBadRequest", err)
	}
	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Fatalf("calls = %d, want 0 (a doomed call must not spend quota)", got)
	}
}

func TestZiptax_DefaultBaseURL(t *testing.T) {
	p := NewZiptaxProvider("k", "")
	if p.baseURL != DefaultZiptaxURL {
		t.Errorf("baseURL = %q, want %q", p.baseURL, DefaultZiptaxURL)
	}
	if got := NewZiptaxProvider("k", "https://example.test/").baseURL; got != "https://example.test" {
		t.Errorf("baseURL = %q, want trailing slash trimmed", got)
	}
	if p.Name() != "ziptax" {
		t.Errorf("Name() = %q, want ziptax", p.Name())
	}
}

// TestZiptax_TransportError_Retried covers the transport-failure leg of the
// retry policy (the server is closed before the call).
func TestZiptax_TransportError_Retried(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	url := srv.URL
	srv.Close()

	_, err := NewZiptaxProvider("k", url).LookupSalesTax(context.Background(), ziptaxCAQuery())
	if !errors.Is(err, ErrZiptaxUnavailable) {
		t.Fatalf("err = %v, want ErrZiptaxUnavailable", err)
	}
}

// TestZiptax_MalformedBodyUnderHTTP200 falls back to the HTTP status when no
// application code can be read (a proxy error page, say).
func TestZiptax_MalformedBodyFallsBackToStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(502)
		_, _ = w.Write([]byte("<html>bad gateway</html>"))
	}))
	defer srv.Close()

	_, err := NewZiptaxProvider("k", srv.URL).LookupSalesTax(context.Background(), ziptaxCAQuery())
	if !errors.Is(err, ErrZiptaxUnavailable) {
		t.Fatalf("err = %v, want ErrZiptaxUnavailable", err)
	}
}
