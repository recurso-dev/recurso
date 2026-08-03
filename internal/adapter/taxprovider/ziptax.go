package taxprovider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/recurso-dev/recurso/internal/core/service/tax"
)

// DefaultZiptaxURL is Ziptax's production API base. Point ZIPTAX_API_URL
// elsewhere to exercise the adapter against a stub (the tests do).
const DefaultZiptaxURL = "https://api.zip-tax.com"

const ziptaxTimeout = 10 * time.Second

// ziptaxPath is the v60 rate endpoint. v60 is targeted deliberately: earlier
// versions report the application result as top-level rCode/rName/rMessage,
// whereas v60 moves it under metadata.response, which is what this adapter
// branches on.
const ziptaxPath = "/request/v60"

// Typed error kinds, mirroring the TaxJar/Avalara sentinel pattern so callers
// keep one degradation policy across providers.
var (
	// ErrZiptaxAuth: the API key was rejected. Retrying won't help.
	ErrZiptaxAuth = errors.New("ziptax: authentication failed")
	// ErrZiptaxBadRequest: Ziptax rejected the request shape/params, or the
	// query cannot be expressed against the rates endpoint at all.
	ErrZiptaxBadRequest = errors.New("ziptax: invalid request")
	// ErrZiptaxUnavailable: network failure, rate limiting, or a server error
	// after the single retry.
	ErrZiptaxUnavailable = errors.New("ziptax: service unavailable")
	// ErrZiptaxNotEntitled: the key is valid but the caller's Ziptax plan does
	// not include the requested feature.
	//
	// This is a fourth sentinel where TaxJar and Avalara have three, and it is
	// deliberate. Ziptax reports plan-entitlement failures (107/112/113) with
	// their own response codes, and folding them into ErrZiptaxBadRequest gives
	// a self-hoster "invalid request" when the true answer is "your plan does
	// not include this", the one message that cannot lead them to a fix.
	ErrZiptaxNotEntitled = errors.New("ziptax: feature not enabled for this plan")
)

// Ziptax application response codes. Ziptax documents the numeric codes as the
// stable contract (messages may be reworded), so the adapter switches on these
// rather than on the HTTP status.
const (
	ziptaxCodeSuccess                 = 100
	ziptaxCodeInvalidKey              = 101
	ziptaxCodeInvalidState            = 102
	ziptaxCodeInvalidCity             = 103
	ziptaxCodeInvalidPostalCode       = 104
	ziptaxCodeInvalidFormat           = 105
	ziptaxCodeAPIError                = 106
	ziptaxCodeFeatureNotEnabled       = 107
	ziptaxCodeRequestLimitMet         = 108
	ziptaxCodeAddressIncomplete       = 109
	ziptaxCodeNoResult                = 110
	ziptaxCodeInvalidHistorical       = 111
	ziptaxCodeInternationalNotEnabled = 112
	ziptaxCodeProductRulesNotEnabled  = 113
)

// ZiptaxError carries the detail behind one of the sentinel kinds.
//
// ResponseCode has no equivalent on TaxJarError or AvalaraError. It is carried
// because it is the field an operator needs to self-diagnose: Ziptax publishes
// the code table, so "ResponseCode 107" is directly actionable in a way that an
// HTTP status is not.
type ZiptaxError struct {
	Kind         error // one of the sentinels above
	StatusCode   int   // HTTP status; 0 for transport errors and pre-flight rejections
	ResponseCode int   // Ziptax metadata.response.code; 0 when absent/unparseable
	Detail       string
}

func (e *ZiptaxError) Error() string {
	switch {
	case e.StatusCode == 0 && e.ResponseCode == 0:
		return fmt.Sprintf("%v: %s", e.Kind, e.Detail)
	case e.ResponseCode == 0:
		return fmt.Sprintf("%v (HTTP %d): %s", e.Kind, e.StatusCode, e.Detail)
	default:
		return fmt.Sprintf("%v (HTTP %d, code %d): %s", e.Kind, e.StatusCode, e.ResponseCode, e.Detail)
	}
}

func (e *ZiptaxError) Unwrap() error { return e.Kind }

// ZiptaxProvider implements tax.SalesTaxProvider against Ziptax's
// GET /request/v60 rate endpoint.
//
// Scope: Ziptax answers "what does this jurisdiction charge". It does not file
// or remit, it is not told the product's taxability, and it is not asked about
// nexus. See LookupSalesTax for how each of those is handled.
type ZiptaxProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

var _ tax.SalesTaxProvider = (*ZiptaxProvider)(nil)

// NewZiptaxProvider creates a Ziptax client. baseURL "" means production
// (DefaultZiptaxURL).
func NewZiptaxProvider(apiKey, baseURL string) *ZiptaxProvider {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = DefaultZiptaxURL
	}
	return &ZiptaxProvider{
		apiKey:     apiKey,
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: ziptaxTimeout},
	}
}

// Name implements tax.SalesTaxProvider.
func (p *ZiptaxProvider) Name() string { return "ziptax" }

// ziptaxCountryAlpha3 maps the ISO alpha-2 country codes SalesTaxQuery carries
// to the alpha-3 codes Ziptax's countryCode parameter expects.
//
// The trap this exists to prevent: alpha-2 "CA" is Canada, but a ToState of
// "CA" is California. Passing ToCountry through unmapped would send
// countryCode=CA, which Ziptax does not recognise as USA, so a plain
// California sale would fail or resolve against the wrong country. Never pass
// ToCountry through without this map.
var ziptaxCountryAlpha3 = map[string]string{
	"US": "USA", // United States
	"CA": "CAN", // Canada, NOT California
	"PR": "PRI", // Puerto Rico
	"AS": "ASM", // American Samoa
	"GU": "GUM", // Guam
	"MP": "MNP", // Northern Mariana Islands
	"VI": "VIR", // U.S. Virgin Islands
}

// ziptaxResponse is the subset of the v60 response this adapter consumes.
type ziptaxResponse struct {
	Metadata struct {
		Version  string `json:"version"`
		Response struct {
			Code    int    `json:"code"`
			Name    string `json:"name"`
			Message string `json:"message"`
		} `json:"response"`
	} `json:"metadata"`
	// BaseRates is the per-jurisdiction breakdown. It carries a PARALLEL sales
	// and use entry for every level, so it must never be summed. See
	// ziptaxSalesRate.
	BaseRates []struct {
		Rate    float64 `json:"rate"`
		JurType string  `json:"jurType"`
		JurName string  `json:"jurName"`
	} `json:"baseRates"`
	TaxSummaries []struct {
		Rate    float64 `json:"rate"`
		TaxType string  `json:"taxType"`
	} `json:"taxSummaries"`
}

// LookupSalesTax implements tax.SalesTaxProvider via GET /request/v60.
//
// Accuracy note: SalesTaxQuery carries no street address, so this adapter
// performs a postal-code lookup. Ziptax documents that as its least precise
// mode: a ZIP can span several tax jurisdictions, and the postal-code result
// is not adjusted for unincorporated areas or special districts the way an
// address or lat/lng lookup is. Callers needing rooftop accuracy need a street
// address on the query; that is a change to the port, not to this adapter.
//
// Three query fields are deliberately not sent:
//
//   - FromCountry/FromState/FromZip. The v60 rates endpoint takes no origin
//     parameter; Ziptax resolves origin-vs-destination sourcing server-side and
//     reports which rule it applied in sourcingRules. This differs from the
//     TaxJar adapter, which does pass from_* through.
//   - NexusStates. Nexus is a registration fact: which states the seller is
//     registered in, and since when. It is not derivable from an address, and a
//     rate API cannot answer it. Recurso already owns that determination, so the
//     adapter forwards nothing and derives HasNexus from the query instead.
//   - Exempt/ExemptionNo/EntityUseCode. The rates endpoint has no exemption
//     parameter (see the short-circuit below).
func (p *ZiptaxProvider) LookupSalesTax(ctx context.Context, q *tax.SalesTaxQuery) (*tax.SalesTaxResult, error) {
	if q == nil {
		return nil, &ZiptaxError{Kind: ErrZiptaxBadRequest, Detail: "nil query"}
	}

	// An exempt sale is zero tax by definition, and the rates endpoint has no
	// exemption parameter to send, so the lookup would tell us nothing we do
	// not already know. Short-circuiting saves an API call, which matters: the
	// free tier is 100 calls/month, so quota is the binding constraint.
	//
	// Limitation worth stating plainly: unlike TaxJar (exemption_type) and
	// Avalara (entityUseCode/exemptionNo), a Ziptax lookup does not record the
	// exempt sale anywhere. Recurso holds the exemption record itself so nothing
	// is lost functionally, but there is no provider-side audit artifact.
	if q.IsExempt() {
		return &tax.SalesTaxResult{
			Rate:         0,
			TaxAmount:    0,
			Jurisdiction: joinJurisdiction(strings.ToUpper(strings.TrimSpace(q.ToCountry)), strings.ToUpper(strings.TrimSpace(q.ToState))),
			HasNexus:     ziptaxHasNexus(q),
		}, nil
	}

	// Ziptax quotes US (and US-territory) rates in USD. Anything else is a
	// caller error rather than something to discover via a wasted API call.
	if cur := strings.ToUpper(strings.TrimSpace(q.Currency)); cur != "" && cur != "USD" {
		return nil, &ZiptaxError{Kind: ErrZiptaxBadRequest, Detail: "unsupported currency " + cur + " (ziptax quotes USD)"}
	}

	zip := strings.TrimSpace(q.ToZip)
	if zip == "" {
		// A state alone will not resolve to a rate; the call would spend quota
		// to come back 109/110. Fail before the request.
		return nil, &ZiptaxError{Kind: ErrZiptaxBadRequest, Detail: "destination ZIP is required for a ziptax rate lookup"}
	}

	countryAlpha2 := strings.ToUpper(strings.TrimSpace(q.ToCountry))
	if countryAlpha2 == "" {
		countryAlpha2 = "US"
	}
	countryAlpha3, ok := ziptaxCountryAlpha3[countryAlpha2]
	if !ok {
		return nil, &ZiptaxError{Kind: ErrZiptaxBadRequest, Detail: "unsupported destination country " + countryAlpha2}
	}

	params := url.Values{}
	// postalcode, NOT address. The address parameter routes to Ziptax's
	// geocoded lookup, which requires the geo entitlement, and a key without it
	// gets 107 on every call. postalcode is the core path and needs no
	// entitlement beyond an active key, which is what makes the free tier
	// actually usable here.
	params.Set("postalcode", zip)
	params.Set("countryCode", countryAlpha3)

	res, err := p.doWithOneRetry(ctx, params, q, countryAlpha2)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// doWithOneRetry performs the GET, retrying at most once on transport errors
// and on the response codes Ziptax marks transient (106 API error, 108 rate
// limit). Everything else returns on the first attempt.
//
// No backoff: Recurso owns degradation policy (provider.go: "Errors are
// returned as-is; callers decide degradation policy"), and an adapter that
// sleeps inside a checkout path is worse than one that returns a typed error
// promptly. Ziptax's docs do recommend exponential backoff for 108, but Ziptax
// sends no Retry-After, so any backoff here would be a guess made in the wrong
// layer.
func (p *ZiptaxProvider) doWithOneRetry(ctx context.Context, params url.Values, q *tax.SalesTaxQuery, countryAlpha2 string) (*tax.SalesTaxResult, error) {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		res, retryable, err := p.doOnce(ctx, params, q, countryAlpha2)
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

func (p *ZiptaxProvider) doOnce(ctx context.Context, params url.Values, q *tax.SalesTaxQuery, countryAlpha2 string) (res *tax.SalesTaxResult, retryable bool, err error) {
	endpoint := p.baseURL + ziptaxPath + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, false, &ZiptaxError{Kind: ErrZiptaxBadRequest, Detail: err.Error()}
	}
	// Header auth, never the key= query parameter. A key in the URL leaks into
	// access logs, proxy logs, and any error string that echoes the request URL.
	req.Header.Set("X-API-KEY", p.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, true, &ZiptaxError{Kind: ErrZiptaxUnavailable, Detail: err.Error()}
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, true, &ZiptaxError{Kind: ErrZiptaxUnavailable, StatusCode: resp.StatusCode, Detail: err.Error()}
	}

	var parsed ziptaxResponse
	parseErr := json.Unmarshal(raw, &parsed)
	code := parsed.Metadata.Response.Code

	// Branch on the application code first. Ziptax returns a meaningful code in
	// the body, and it is the authority. The HTTP status is a coarser echo of
	// it. The status is only consulted when no code could be read (a proxy error
	// page, say), so an infrastructure failure in front of Ziptax still maps to
	// something sane.
	if parseErr != nil || code == 0 {
		return nil, ziptaxStatusRetryable(resp.StatusCode), ziptaxErrorFromStatus(resp.StatusCode, raw)
	}

	if code != ziptaxCodeSuccess {
		kind, retry := ziptaxKindForCode(code)
		return nil, retry, &ZiptaxError{
			Kind:         kind,
			StatusCode:   resp.StatusCode,
			ResponseCode: code,
			Detail:       ziptaxDetail(parsed, raw),
		}
	}

	rate, ok := ziptaxSalesRate(parsed)
	if !ok {
		// Deliberately not a positional fallback to taxSummaries[0]: the array
		// also carries USE_TAX, and quoting a use-tax rate as sales tax would be
		// a silent wrong answer rather than a visible failure.
		return nil, false, &ZiptaxError{
			Kind:         ErrZiptaxBadRequest,
			StatusCode:   resp.StatusCode,
			ResponseCode: code,
			Detail:       "response carried no SALES_TAX summary",
		}
	}

	return &tax.SalesTaxResult{
		Rate: rate,
		// Computed locally, in minor units. Ziptax returns a rate rather than an
		// amount, and this is the same expression CachedSalesTaxProvider uses on
		// a cache hit, so a cached result is bit-identical to a live one.
		TaxAmount:    int64(math.Round(float64(q.Amount) * rate)),
		Jurisdiction: ziptaxJurisdiction(parsed, countryAlpha2),
		HasNexus:     ziptaxHasNexus(q),
	}, false, nil
}

// ziptaxKindForCode maps a Ziptax application response code to a sentinel and
// reports whether the code is worth one retry.
func ziptaxKindForCode(code int) (kind error, retryable bool) {
	switch code {
	case ziptaxCodeInvalidKey:
		return ErrZiptaxAuth, false
	case ziptaxCodeAPIError:
		return ErrZiptaxUnavailable, true
	case ziptaxCodeRequestLimitMet:
		// A 4xx that is genuinely transient. The TaxJar convention is "never
		// retry 4xx", but that rule is a proxy for "never retry what cannot
		// succeed", and on a per-minute rate limit the next attempt often can.
		return ErrZiptaxUnavailable, true
	case ziptaxCodeFeatureNotEnabled, ziptaxCodeInternationalNotEnabled, ziptaxCodeProductRulesNotEnabled:
		return ErrZiptaxNotEntitled, false
	case ziptaxCodeInvalidState, ziptaxCodeInvalidCity, ziptaxCodeInvalidPostalCode,
		ziptaxCodeInvalidFormat, ziptaxCodeAddressIncomplete, ziptaxCodeNoResult,
		ziptaxCodeInvalidHistorical:
		return ErrZiptaxBadRequest, false
	default:
		return ErrZiptaxUnavailable, false
	}
}

// ziptaxErrorFromStatus classifies a response whose body carried no usable
// application code, falling back to the HTTP status.
func ziptaxErrorFromStatus(status int, raw []byte) *ZiptaxError {
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return &ZiptaxError{Kind: ErrZiptaxAuth, StatusCode: status, Detail: errDetail(raw)}
	case status == http.StatusOK:
		return &ZiptaxError{Kind: ErrZiptaxBadRequest, StatusCode: status, Detail: "malformed response: " + errDetail(raw)}
	case status >= 400 && status < 500:
		return &ZiptaxError{Kind: ErrZiptaxBadRequest, StatusCode: status, Detail: errDetail(raw)}
	default:
		return &ZiptaxError{Kind: ErrZiptaxUnavailable, StatusCode: status, Detail: errDetail(raw)}
	}
}

func ziptaxStatusRetryable(status int) bool {
	return status >= 500 || status == http.StatusTooManyRequests
}

// ziptaxDetail prefers Ziptax's own response name/message over a raw body dump.
func ziptaxDetail(parsed ziptaxResponse, raw []byte) string {
	name := strings.TrimSpace(parsed.Metadata.Response.Name)
	msg := strings.TrimSpace(parsed.Metadata.Response.Message)
	if detail := strings.TrimSpace(name + " " + msg); detail != "" {
		return detail
	}
	return errDetail(raw)
}

// ziptaxSalesRate reads the combined sales-tax rate from taxSummaries.
//
// This is THE correctness-critical function in the adapter. baseRates carries a
// parallel sales-tax and use-tax entry for every jurisdiction level, so the
// obvious-looking "sum the breakdown" approach, which is what the Avalara
// adapter in this package does over its own, differently-shaped Summary array,
// double-counts and returns roughly 15.5% where the answer is 7.75%.
//
// The rate is already a 0.0–1.0 decimal; no conversion.
func ziptaxSalesRate(parsed ziptaxResponse) (float64, bool) {
	for _, s := range parsed.TaxSummaries {
		if strings.EqualFold(strings.TrimSpace(s.TaxType), "SALES_TAX") {
			return s.Rate, true
		}
	}
	return 0, false
}

// ziptaxJurisdictionLevel orders a jurType into the country/state/county/city
// hierarchy used for display. Unrecognised levels sort last, preserving their
// relative order.
func ziptaxJurisdictionLevel(jurType string) int {
	switch {
	case strings.Contains(jurType, "_STATE_"):
		return 0
	case strings.Contains(jurType, "_COUNTY_"):
		return 1
	case strings.Contains(jurType, "_CITY_"):
		return 2
	case strings.Contains(jurType, "_DISTRICT_"):
		return 3
	default:
		return 4
	}
}

// ziptaxJurisdiction builds the human-readable breakdown ("US/CA/ORANGE/IRVINE")
// from the sales-tax half of baseRates, matching the TaxJar adapter's output
// style.
//
// Zero-rate levels are kept: a city that charges nothing is still the city the
// sale landed in, and dropping it would make the string disagree with the
// jurisdiction the rate was actually resolved against. Only blank names are
// skipped. Use-tax entries are excluded so each level appears once.
func ziptaxJurisdiction(parsed ziptaxResponse, countryAlpha2 string) string {
	type level struct {
		order int
		name  string
	}
	levels := make([]level, 0, len(parsed.BaseRates))
	for _, br := range parsed.BaseRates {
		jurType := strings.ToUpper(strings.TrimSpace(br.JurType))
		if !strings.HasSuffix(jurType, "_SALES_TAX") {
			continue
		}
		name := strings.TrimSpace(br.JurName)
		if name == "" {
			continue
		}
		levels = append(levels, level{order: ziptaxJurisdictionLevel(jurType), name: name})
	}
	sort.SliceStable(levels, func(i, j int) bool { return levels[i].order < levels[j].order })

	parts := make([]string, 0, len(levels)+1)
	parts = append(parts, countryAlpha2)
	for _, l := range levels {
		parts = append(parts, l.name)
	}
	return joinJurisdiction(parts...)
}

// ziptaxHasNexus reports whether Recurso considers the seller to have nexus in
// the destination state.
//
// Ziptax cannot answer this and is never asked: nexus depends on where the
// seller is registered and which thresholds they have crossed, none of which is
// derivable from an address. So the value is derived from the query's own
// NexusStates, which Recurso populates.
//
// An empty NexusStates means "Recurso has not declared any states", and this
// returns true. The adapter is not contradicting Recurso; it simply has no
// grounds to withhold nexus. It is deliberately NOT hardcoded true, and it does
// NOT infer nexus from a non-zero rate the way the Avalara adapter does, which
// conflates a genuine 0% jurisdiction with an absence of nexus.
func ziptaxHasNexus(q *tax.SalesTaxQuery) bool {
	if len(q.NexusStates) == 0 {
		return true
	}
	dest := strings.ToUpper(strings.TrimSpace(q.ToState))
	for _, st := range q.NexusStates {
		if strings.EqualFold(strings.TrimSpace(st), dest) && dest != "" {
			return true
		}
	}
	return false
}
