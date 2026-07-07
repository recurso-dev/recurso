package vatprovider

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/swapnull-in/recur-so/internal/core/service/tax"
)

func TestVIES_ValidNumber_RequestShapeAndParsing(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"isValid":true,"userError":"VALID","name":"ACME GMBH","address":"Berlin"}`))
	}))
	defer srv.Close()

	v := NewVIESValidator(srv.URL)
	res, err := v.ValidateVAT(context.Background(), "DE", "123456789")
	if err != nil {
		t.Fatalf("ValidateVAT: %v", err)
	}
	if gotMethod != http.MethodGet || gotPath != "/ms/DE/vat/123456789" {
		t.Errorf("request = %s %s, want GET /ms/DE/vat/123456789", gotMethod, gotPath)
	}
	if !res.Valid {
		t.Error("Valid = false, want true")
	}
	if res.Name != "ACME GMBH" {
		t.Errorf("Name = %q, want 'ACME GMBH'", res.Name)
	}
}

func TestVIES_GreeceMappedToEL(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"isValid":true,"userError":"VALID"}`))
	}))
	defer srv.Close()

	v := NewVIESValidator(srv.URL)
	if _, err := v.ValidateVAT(context.Background(), "GR", "123456789"); err != nil {
		t.Fatalf("ValidateVAT: %v", err)
	}
	if gotPath != "/ms/EL/vat/123456789" {
		t.Errorf("path = %q, want /ms/EL/vat/123456789 (Greece uses VIES code EL)", gotPath)
	}
}

func TestVIES_NormalizesSpacesAndDots(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"isValid":true,"userError":"VALID"}`))
	}))
	defer srv.Close()

	v := NewVIESValidator(srv.URL)
	if _, err := v.ValidateVAT(context.Background(), "de", " 123.456-789 "); err != nil {
		t.Fatalf("ValidateVAT: %v", err)
	}
	if gotPath != "/ms/DE/vat/123456789" {
		t.Errorf("path = %q, want normalized /ms/DE/vat/123456789", gotPath)
	}
}

func TestVIES_InvalidNumber_ReturnsNotValid(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"isValid":false,"userError":"INVALID"}`))
	}))
	defer srv.Close()

	v := NewVIESValidator(srv.URL)
	res, err := v.ValidateVAT(context.Background(), "DE", "000000000")
	if err != nil {
		t.Fatalf("ValidateVAT: unexpected error %v", err)
	}
	if res.Valid {
		t.Error("Valid = true, want false for an unregistered number")
	}
}

func TestVIES_LocalFormatRejection_NoNetworkCall(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		_, _ = w.Write([]byte(`{"isValid":true,"userError":"VALID"}`))
	}))
	defer srv.Close()

	v := NewVIESValidator(srv.URL)
	// DE requires exactly 9 digits; "12345" is too short.
	_, err := v.ValidateVAT(context.Background(), "DE", "12345")
	if !errors.Is(err, tax.ErrVATInvalidFormat) {
		t.Fatalf("err = %v, want ErrVATInvalidFormat", err)
	}
	if called {
		t.Error("network was called despite a local format failure")
	}
}

func TestVIES_UnsupportedCountry_InvalidInput(t *testing.T) {
	v := NewVIESValidator("http://unused.example")
	_, err := v.ValidateVAT(context.Background(), "US", "123456789")
	if !errors.Is(err, tax.ErrVATInvalidInput) {
		t.Fatalf("err = %v, want ErrVATInvalidInput for a non-EU country", err)
	}
}

func TestVIES_MSUnavailable_MapsToUnavailable_RetriesOnce(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = w.Write([]byte(`{"isValid":false,"userError":"MS_UNAVAILABLE"}`))
	}))
	defer srv.Close()

	v := NewVIESValidator(srv.URL)
	_, err := v.ValidateVAT(context.Background(), "DE", "123456789")
	if !errors.Is(err, tax.ErrVATUnavailable) {
		t.Fatalf("err = %v, want ErrVATUnavailable", err)
	}
	if calls != 2 {
		t.Errorf("HTTP calls = %d, want 2 (MS_UNAVAILABLE retried once)", calls)
	}
}

func TestVIES_ServerError_RetriedExactlyOnce(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	v := NewVIESValidator(srv.URL)
	_, err := v.ValidateVAT(context.Background(), "DE", "123456789")
	if !errors.Is(err, tax.ErrVATUnavailable) {
		t.Fatalf("err = %v, want ErrVATUnavailable", err)
	}
	if calls != 2 {
		t.Errorf("HTTP calls = %d, want 2 (one retry, no more)", calls)
	}
}

func TestVIES_ServerError_ThenSuccess_RetryRecovers(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte(`{"isValid":true,"userError":"VALID"}`))
	}))
	defer srv.Close()

	v := NewVIESValidator(srv.URL)
	res, err := v.ValidateVAT(context.Background(), "DE", "123456789")
	if err != nil {
		t.Fatalf("ValidateVAT after retry: %v", err)
	}
	if !res.Valid || calls != 2 {
		t.Errorf("Valid=%v calls=%d, want true / 2", res.Valid, calls)
	}
}

func TestVIES_MaskedFields_CleanedToEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"isValid":true,"userError":"VALID","name":"---","address":"---"}`))
	}))
	defer srv.Close()

	v := NewVIESValidator(srv.URL)
	res, err := v.ValidateVAT(context.Background(), "DE", "123456789")
	if err != nil {
		t.Fatalf("ValidateVAT: %v", err)
	}
	if res.Name != "" || res.Address != "" {
		t.Errorf("Name/Address = %q/%q, want empty (--- placeholder cleaned)", res.Name, res.Address)
	}
}

func TestVIES_DefaultBaseURL(t *testing.T) {
	v := NewVIESValidator("")
	if v.baseURL != DefaultVIESURL {
		t.Errorf("baseURL = %q, want %q", v.baseURL, DefaultVIESURL)
	}
	if v.Name() != "vies" {
		t.Errorf("Name() = %q, want 'vies'", v.Name())
	}
}

func TestVIES_LocalFormats_PerCountry(t *testing.T) {
	// Structural format acceptance/rejection, exercised without any network by
	// relying on the fact that a format failure short-circuits before the call.
	v := NewVIESValidator("http://unused.invalid")
	cases := []struct {
		cc, num string
		wantOK  bool // true => passes local format (would proceed to network)
	}{
		{"DE", "123456789", true},
		{"DE", "12345678", false},   // 8 digits, needs 9
		{"AT", "U12345678", true},   // U + 8 digits
		{"AT", "12345678", false},   // missing U
		{"FR", "XX123456789", true}, // 2 chars + 9 digits
		{"NL", "123456789B01", true},
		{"NL", "123456789012", false}, // missing B block
		{"IT", "12345678901", true},   // 11 digits
		{"IT", "1234567890", false},   // 10 digits
		{"GR", "123456789", true},     // maps to EL
	}
	for _, c := range cases {
		_, err := v.ValidateVAT(context.Background(), c.cc, c.num)
		formatFailed := errors.Is(err, tax.ErrVATInvalidFormat)
		if c.wantOK && formatFailed {
			t.Errorf("%s %s: format rejected, want accepted", c.cc, c.num)
		}
		if !c.wantOK && !formatFailed {
			// wantOK=false must yield a format error (network never dialed here,
			// so a transport error would only appear if format passed).
			if !strings.Contains(errStr(err), "format") {
				t.Errorf("%s %s: err = %v, want ErrVATInvalidFormat", c.cc, c.num, err)
			}
		}
	}
}

func errStr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
