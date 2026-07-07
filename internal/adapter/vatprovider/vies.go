// Package vatprovider contains adapters for external VAT-number validation
// services implementing tax.VATValidator. The only implementation today is
// VIES, the European Commission's VAT Information Exchange System.
package vatprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/swapnull-in/recur-so/internal/core/service/tax"
)

// DefaultVIESURL is the European Commission's VIES REST API base. The lookup
// path is {base}/ms/{country}/vat/{number}. Point VIES_API_URL at a test
// server to exercise the adapter without hitting the live service.
//
// The Commission also publishes a SOAP endpoint at
// .../vies/services/checkVatService; the REST API returns the same registry
// data as JSON with no credentials, so we use REST for simplicity.
const DefaultVIESURL = "https://ec.europa.eu/taxation_customs/vies/rest-api"

const viesTimeout = 10 * time.Second

// viesError wraps one of the tax sentinel kinds with HTTP/service detail.
// Callers classify failures via errors.Is against the tax.ErrVAT* sentinels.
type viesError struct {
	Kind       error // one of tax.ErrVATInvalidFormat / ErrVATInvalidInput / ErrVATUnavailable
	StatusCode int   // 0 for transport errors
	Detail     string
}

func (e *viesError) Error() string {
	if e.StatusCode == 0 {
		return fmt.Sprintf("%v: %s", e.Kind, e.Detail)
	}
	return fmt.Sprintf("%v (HTTP %d): %s", e.Kind, e.StatusCode, e.Detail)
}

func (e *viesError) Unwrap() error { return e.Kind }

// VIESValidator implements tax.VATValidator against the VIES REST API.
type VIESValidator struct {
	baseURL    string
	httpClient *http.Client
}

var _ tax.VATValidator = (*VIESValidator)(nil)

// NewVIESValidator creates a VIES client. baseURL "" means the production
// Commission endpoint (DefaultVIESURL); pass a test-server URL to override.
func NewVIESValidator(baseURL string) *VIESValidator {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = DefaultVIESURL
	}
	return &VIESValidator{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: viesTimeout},
	}
}

// Name implements tax.VATValidator.
func (v *VIESValidator) Name() string { return "vies" }

// viesResponse is the subset of the VIES REST response we consume.
// userError carries the registry status ("VALID", "INVALID",
// "MS_UNAVAILABLE", ...) alongside the isValid boolean.
type viesResponse struct {
	IsValid   bool   `json:"isValid"`
	UserError string `json:"userError"`
	Name      string `json:"name"`
	Address   string `json:"address"`
}

// ValidateVAT implements tax.VATValidator. It normalises and format-checks the
// number locally FIRST (no network call on a malformed number), then queries
// VIES. Transport errors and 5xx are retried exactly once; the member-state
// registry being unavailable maps to tax.ErrVATUnavailable so the caller can
// degrade gracefully.
func (v *VIESValidator) ValidateVAT(ctx context.Context, countryCode, vatNumber string) (*tax.VATValidation, error) {
	cc := strings.ToUpper(strings.TrimSpace(countryCode))
	num := normalizeVATNumber(vatNumber)

	// VIES addresses Greece as "EL", not the ISO "GR".
	viesCC := cc
	if viesCC == "GR" {
		viesCC = "EL"
	}

	re, ok := vatFormats[viesCC]
	if !ok {
		return nil, &viesError{Kind: tax.ErrVATInvalidInput, Detail: "unsupported country code: " + cc}
	}
	if !re.MatchString(num) {
		return nil, &viesError{Kind: tax.ErrVATInvalidFormat, Detail: cc + " VAT number " + num + " has an invalid format"}
	}

	return v.doWithOneRetry(ctx, viesCC, num)
}

func (v *VIESValidator) doWithOneRetry(ctx context.Context, viesCC, num string) (*tax.VATValidation, error) {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		res, retryable, err := v.doOnce(ctx, viesCC, num)
		if err == nil {
			return res, nil
		}
		lastErr = err
		if !retryable || ctx.Err() != nil {
			return nil, err
		}
	}
	return nil, lastErr
}

func (v *VIESValidator) doOnce(ctx context.Context, viesCC, num string) (res *tax.VATValidation, retryable bool, err error) {
	endpoint := v.baseURL + "/ms/" + url.PathEscape(viesCC) + "/vat/" + url.PathEscape(num)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, false, &viesError{Kind: tax.ErrVATInvalidInput, Detail: err.Error()}
	}
	req.Header.Set("Accept", "application/json")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, true, &viesError{Kind: tax.ErrVATUnavailable, Detail: err.Error()}
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, true, &viesError{Kind: tax.ErrVATUnavailable, StatusCode: resp.StatusCode, Detail: err.Error()}
	}

	switch {
	case resp.StatusCode == http.StatusOK:
		// parsed below
	case resp.StatusCode == http.StatusBadRequest:
		return nil, false, &viesError{Kind: tax.ErrVATInvalidInput, StatusCode: resp.StatusCode, Detail: truncate(raw)}
	case resp.StatusCode >= 400 && resp.StatusCode < 500:
		// 404/429 and friends: transient from our side, safe to retry once.
		return nil, true, &viesError{Kind: tax.ErrVATUnavailable, StatusCode: resp.StatusCode, Detail: truncate(raw)}
	default:
		return nil, true, &viesError{Kind: tax.ErrVATUnavailable, StatusCode: resp.StatusCode, Detail: truncate(raw)}
	}

	var parsed viesResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, false, &viesError{Kind: tax.ErrVATUnavailable, StatusCode: resp.StatusCode, Detail: "malformed response: " + err.Error()}
	}

	// The registry answers via userError even on HTTP 200.
	switch strings.ToUpper(strings.TrimSpace(parsed.UserError)) {
	case "VALID", "":
		if !parsed.IsValid {
			// isValid=false with no explicit error: treat as not registered.
			return &tax.VATValidation{Valid: false}, false, nil
		}
		return &tax.VATValidation{
			Valid:   true,
			Name:    cleanField(parsed.Name),
			Address: cleanField(parsed.Address),
		}, false, nil
	case "INVALID":
		return &tax.VATValidation{Valid: false}, false, nil
	case "INVALID_INPUT":
		return nil, false, &viesError{Kind: tax.ErrVATInvalidInput, StatusCode: resp.StatusCode, Detail: parsed.UserError}
	default:
		// MS_UNAVAILABLE, SERVICE_UNAVAILABLE, TIMEOUT, MS_MAX_CONCURRENT_REQ,
		// GLOBAL_MAX_CONCURRENT_REQ, and anything unrecognised: validity is
		// unknown, so surface as unavailable (retryable once).
		return nil, true, &viesError{Kind: tax.ErrVATUnavailable, StatusCode: resp.StatusCode, Detail: parsed.UserError}
	}
}

// normalizeVATNumber strips spaces, dots, and dashes and uppercases the
// number so "DE 123 456 789" and "de123456789" validate identically. Any
// leading country prefix is expected to be removed by the caller (the resolver
// passes the national number only).
func normalizeVATNumber(s string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(s) {
		if r == ' ' || r == '.' || r == '-' || r == '\t' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// cleanField normalises VIES's "---" placeholder (used when a member state
// does not disclose trader details) to an empty string.
func cleanField(s string) string {
	s = strings.TrimSpace(s)
	if s == "---" {
		return ""
	}
	return s
}

func truncate(raw []byte) string {
	s := strings.TrimSpace(string(raw))
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}

// vatFormats holds the per-country local format regex for the NATIONAL VAT
// number (country prefix already stripped). Keys use VIES country codes, so
// Greece is "EL". These are deliberately permissive structural checks (length
// and character classes) — enough to reject obvious typos before a network
// call, not a checksum validation.
var vatFormats = map[string]*regexp.Regexp{
	"AT": regexp.MustCompile(`^U\d{8}$`),                // Austria: U + 8 digits
	"BE": regexp.MustCompile(`^[01]\d{9}$`),             // Belgium: 10 digits (0/1 lead)
	"BG": regexp.MustCompile(`^\d{9,10}$`),              // Bulgaria: 9-10 digits
	"CY": regexp.MustCompile(`^\d{8}[A-Z]$`),            // Cyprus: 8 digits + letter
	"CZ": regexp.MustCompile(`^\d{8,10}$`),              // Czechia: 8-10 digits
	"DE": regexp.MustCompile(`^\d{9}$`),                 // Germany: 9 digits
	"DK": regexp.MustCompile(`^\d{8}$`),                 // Denmark: 8 digits
	"EE": regexp.MustCompile(`^\d{9}$`),                 // Estonia: 9 digits
	"EL": regexp.MustCompile(`^\d{9}$`),                 // Greece: 9 digits
	"ES": regexp.MustCompile(`^[A-Z0-9]\d{7}[A-Z0-9]$`), // Spain: char+7 digits+char
	"FI": regexp.MustCompile(`^\d{8}$`),                 // Finland: 8 digits
	"FR": regexp.MustCompile(`^[A-Z0-9]{2}\d{9}$`),      // France: 2 chars + 9 digits
	"HR": regexp.MustCompile(`^\d{11}$`),                // Croatia: 11 digits
	"HU": regexp.MustCompile(`^\d{8}$`),                 // Hungary: 8 digits
	"IE": regexp.MustCompile(`^[0-9A-Z+*]{8,9}$`),       // Ireland: 8-9 alnum (+/*)
	"IT": regexp.MustCompile(`^\d{11}$`),                // Italy: 11 digits
	"LT": regexp.MustCompile(`^(\d{9}|\d{12})$`),        // Lithuania: 9 or 12 digits
	"LU": regexp.MustCompile(`^\d{8}$`),                 // Luxembourg: 8 digits
	"LV": regexp.MustCompile(`^\d{11}$`),                // Latvia: 11 digits
	"MT": regexp.MustCompile(`^\d{8}$`),                 // Malta: 8 digits
	"NL": regexp.MustCompile(`^\d{9}B\d{2}$`),           // Netherlands: 9 digits + B + 2 digits
	"PL": regexp.MustCompile(`^\d{10}$`),                // Poland: 10 digits
	"PT": regexp.MustCompile(`^\d{9}$`),                 // Portugal: 9 digits
	"RO": regexp.MustCompile(`^\d{2,10}$`),              // Romania: 2-10 digits
	"SE": regexp.MustCompile(`^\d{12}$`),                // Sweden: 12 digits
	"SI": regexp.MustCompile(`^\d{8}$`),                 // Slovenia: 8 digits
	"SK": regexp.MustCompile(`^\d{10}$`),                // Slovakia: 10 digits
}
