// Package taxprovider contains adapters for external sales-tax rate
// services implementing tax.SalesTaxProvider.
package taxprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/recurso-dev/recurso/internal/core/service/tax"
)

// DefaultTaxJarURL is TaxJar's production API base. Point TAXJAR_API_URL at
// https://api.sandbox.taxjar.com for the sandbox.
const DefaultTaxJarURL = "https://api.taxjar.com"

const taxJarTimeout = 10 * time.Second

// Typed error kinds. Callers can errors.Is against these sentinels to tell
// configuration problems (bad key) from transient outages.
var (
	// ErrTaxJarAuth: the API key was rejected (401/403). Retrying won't help.
	ErrTaxJarAuth = errors.New("taxjar: authentication failed")
	// ErrTaxJarBadRequest: TaxJar rejected the request shape/params (4xx).
	ErrTaxJarBadRequest = errors.New("taxjar: invalid request")
	// ErrTaxJarUnavailable: network failure or 5xx after the single retry.
	ErrTaxJarUnavailable = errors.New("taxjar: service unavailable")
)

// TaxJarError carries the HTTP detail behind one of the sentinel kinds.
type TaxJarError struct {
	Kind       error // one of the sentinels above
	StatusCode int   // 0 for transport errors
	Detail     string
}

func (e *TaxJarError) Error() string {
	if e.StatusCode == 0 {
		return fmt.Sprintf("%v: %s", e.Kind, e.Detail)
	}
	return fmt.Sprintf("%v (HTTP %d): %s", e.Kind, e.StatusCode, e.Detail)
}

func (e *TaxJarError) Unwrap() error { return e.Kind }

// TaxJarProvider implements tax.SalesTaxProvider against TaxJar's
// POST /v2/taxes "tax for order" endpoint.
type TaxJarProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

var _ tax.SalesTaxProvider = (*TaxJarProvider)(nil)

// NewTaxJarProvider creates a TaxJar client. baseURL "" means production
// (DefaultTaxJarURL); pass the sandbox URL for test keys.
func NewTaxJarProvider(apiKey, baseURL string) *TaxJarProvider {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = DefaultTaxJarURL
	}
	return &TaxJarProvider{
		apiKey:     apiKey,
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: taxJarTimeout},
	}
}

// Name implements tax.SalesTaxProvider.
func (p *TaxJarProvider) Name() string { return "taxjar" }

// taxJarOrderRequest is the /v2/taxes request body. TaxJar amounts are
// decimal dollars; shipping is a required parameter (0 for SaaS invoices).
type taxJarOrderRequest struct {
	FromCountry string  `json:"from_country,omitempty"`
	FromState   string  `json:"from_state,omitempty"`
	FromZip     string  `json:"from_zip,omitempty"`
	ToCountry   string  `json:"to_country"`
	ToState     string  `json:"to_state,omitempty"`
	ToZip       string  `json:"to_zip,omitempty"`
	Amount      float64 `json:"amount"`
	Shipping    float64 `json:"shipping"`
	// ExemptionType, when set, tells TaxJar the order is exempt and it returns
	// amount_to_collect 0 (Track D · D2). One of TaxJar's categories:
	// wholesale, government, marketplace, other, non_exempt.
	ExemptionType string `json:"exemption_type,omitempty"`
}

// taxJarExemptionType maps a provider-agnostic entity-use code to TaxJar's
// exemption_type category. TaxJar's set is coarse, so an unrecognized code
// falls back to "other" — still exempt, just uncategorized.
func taxJarExemptionType(entityUseCode string) string {
	switch strings.ToLower(strings.TrimSpace(entityUseCode)) {
	case "wholesale", "resale":
		return "wholesale"
	case "government", "federal", "state_govt", "a", "b", "c":
		return "government"
	case "marketplace":
		return "marketplace"
	case "", "non_exempt":
		return "other"
	default:
		return "other"
	}
}

// taxJarOrderResponse is the subset of the /v2/taxes response we consume.
type taxJarOrderResponse struct {
	Tax struct {
		AmountToCollect float64 `json:"amount_to_collect"`
		Rate            float64 `json:"rate"`
		HasNexus        bool    `json:"has_nexus"`
		TaxSource       string  `json:"tax_source"`
		Jurisdictions   struct {
			Country string `json:"country"`
			State   string `json:"state"`
			County  string `json:"county"`
			City    string `json:"city"`
		} `json:"jurisdictions"`
	} `json:"tax"`
}

type taxJarErrorResponse struct {
	Error  string `json:"error"`
	Detail string `json:"detail"`
}

// LookupSalesTax implements tax.SalesTaxProvider via POST /v2/taxes.
// Transport errors and 5xx responses are retried exactly once; 4xx never.
func (p *TaxJarProvider) LookupSalesTax(ctx context.Context, q *tax.SalesTaxQuery) (*tax.SalesTaxResult, error) {
	reqBody := taxJarOrderRequest{
		FromCountry: q.FromCountry,
		FromState:   q.FromState,
		FromZip:     q.FromZip,
		ToCountry:   q.ToCountry,
		ToState:     q.ToState,
		ToZip:       q.ToZip,
		Amount:      centsToDollars(q.Amount),
		Shipping:    0,
	}
	if q.IsExempt() {
		reqBody.ExemptionType = taxJarExemptionType(q.EntityUseCode)
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, &TaxJarError{Kind: ErrTaxJarBadRequest, Detail: err.Error()}
	}

	res, err := p.doWithOneRetry(ctx, body)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// doWithOneRetry performs the POST, retrying once on transport errors and
// 5xx responses. 4xx responses map to typed errors immediately.
func (p *TaxJarProvider) doWithOneRetry(ctx context.Context, body []byte) (*tax.SalesTaxResult, error) {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		res, retryable, err := p.doOnce(ctx, body)
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

func (p *TaxJarProvider) doOnce(ctx context.Context, body []byte) (res *tax.SalesTaxResult, retryable bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v2/taxes", bytes.NewReader(body))
	if err != nil {
		return nil, false, &TaxJarError{Kind: ErrTaxJarBadRequest, Detail: err.Error()}
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, true, &TaxJarError{Kind: ErrTaxJarUnavailable, Detail: err.Error()}
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, true, &TaxJarError{Kind: ErrTaxJarUnavailable, StatusCode: resp.StatusCode, Detail: err.Error()}
	}

	switch {
	case resp.StatusCode == http.StatusOK:
		// parsed below
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return nil, false, &TaxJarError{Kind: ErrTaxJarAuth, StatusCode: resp.StatusCode, Detail: errDetail(raw)}
	case resp.StatusCode >= 400 && resp.StatusCode < 500:
		return nil, false, &TaxJarError{Kind: ErrTaxJarBadRequest, StatusCode: resp.StatusCode, Detail: errDetail(raw)}
	default:
		return nil, true, &TaxJarError{Kind: ErrTaxJarUnavailable, StatusCode: resp.StatusCode, Detail: errDetail(raw)}
	}

	var parsed taxJarOrderResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, false, &TaxJarError{Kind: ErrTaxJarBadRequest, StatusCode: resp.StatusCode, Detail: "malformed response: " + err.Error()}
	}

	return &tax.SalesTaxResult{
		Rate:         parsed.Tax.Rate,
		TaxAmount:    dollarsToCents(parsed.Tax.AmountToCollect),
		Jurisdiction: joinJurisdiction(parsed.Tax.Jurisdictions.Country, parsed.Tax.Jurisdictions.State, parsed.Tax.Jurisdictions.County, parsed.Tax.Jurisdictions.City),
		HasNexus:     parsed.Tax.HasNexus,
	}, false, nil
}

func errDetail(raw []byte) string {
	var e taxJarErrorResponse
	if json.Unmarshal(raw, &e) == nil && (e.Error != "" || e.Detail != "") {
		return strings.TrimSpace(strings.TrimSpace(e.Error) + " " + e.Detail)
	}
	s := strings.TrimSpace(string(raw))
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}

func joinJurisdiction(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, "/")
}

// centsToDollars converts the lowest-currency-unit amount used across the
// codebase to TaxJar's decimal-dollar representation.
func centsToDollars(cents int64) float64 {
	return float64(cents) / 100.0
}

// dollarsToCents converts TaxJar's decimal dollars back to cents, rounding
// half away from zero the way tax collection rounds.
func dollarsToCents(dollars float64) int64 {
	return int64(math.Round(dollars * 100.0))
}
