// Package recurso is the official Go client for the Recurso billing API.
//
// It is hand-crafted over the same OpenAPI spec as the Node and Python SDKs,
// uses only the standard library, and mirrors their resource/method surface.
// Every method takes a context.Context first and returns (T, error); any
// non-2xx response is returned as *APIError carrying the API's standard
// {"error": {"code", "message"}} envelope and the HTTP status.
//
//	client := recurso.NewClient("rsk_test_...", recurso.WithBaseURL("https://billing.example.com/v1"))
//	plan, err := client.Plans.Create(ctx, &recurso.PlanCreateParams{Name: "Pro", Code: "PRO", Amount: 2900, Currency: "USD", IntervalUnit: "month"})
//
// Monetary amounts are int64 in the currency's smallest unit.
package recurso

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultBaseURL is used when no WithBaseURL option is supplied. It already
// includes the /v1 prefix, so resource paths are appended without it.
const DefaultBaseURL = "https://api.recurso.dev/v1"

// Client is a Recurso API client. Create one with NewClient; it is safe for
// concurrent use.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client

	Account       *AccountService
	Customers     *CustomersService
	Plans         *PlansService
	Subscriptions *SubscriptionsService
	Invoices      *InvoicesService
	Coupons       *CouponsService
	Usage         *UsageService
	CreditNotes   *CreditNotesService
	Quotes        *QuotesService
	Webhooks      *WebhooksService
	Events        *EventsService
	Mandates      *MandatesService
	Gifts         *GiftsService
	Referrals     *ReferralsService
	Entitlements  *EntitlementsService
	Analytics     *AnalyticsService
	Ledger        *LedgerService
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL overrides the API base URL (including the /v1 prefix), e.g. a
// self-hosted instance: WithBaseURL("https://billing.example.com/v1").
func WithBaseURL(baseURL string) Option {
	return func(c *Client) { c.baseURL = strings.TrimRight(baseURL, "/") }
}

// WithHTTPClient supplies a custom *http.Client (timeouts, transport, proxies).
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		if hc != nil {
			c.httpClient = hc
		}
	}
}

// NewClient returns a Client authenticating with apiKey as a bearer token.
func NewClient(apiKey string, opts ...Option) *Client {
	c := &Client{
		apiKey:     apiKey,
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	c.Account = &AccountService{c}
	c.Customers = &CustomersService{c}
	c.Plans = &PlansService{c}
	c.Subscriptions = &SubscriptionsService{c}
	c.Invoices = &InvoicesService{c}
	c.Coupons = &CouponsService{c}
	c.Usage = &UsageService{c}
	c.CreditNotes = &CreditNotesService{c}
	c.Quotes = &QuotesService{c}
	c.Webhooks = &WebhooksService{c}
	c.Events = &EventsService{c}
	c.Mandates = &MandatesService{c}
	c.Gifts = &GiftsService{c}
	c.Referrals = &ReferralsService{c}
	c.Entitlements = &EntitlementsService{c}
	c.Analytics = &AnalyticsService{c}
	c.Ledger = &LedgerService{c}
	return c
}

// APIError is returned for any non-2xx response. It carries the API's error
// envelope ({"error": {"code", "message"}}) and the HTTP status code.
type APIError struct {
	StatusCode int    // HTTP status code
	Code       string // machine-readable error code from the envelope
	Message    string // human-readable message from the envelope
	Body       string // raw response body, for diagnostics
}

func (e *APIError) Error() string {
	if e.Code != "" || e.Message != "" {
		return fmt.Sprintf("recurso: %s (%s, HTTP %d)", e.Message, e.Code, e.StatusCode)
	}
	return fmt.Sprintf("recurso: HTTP %d: %s", e.StatusCode, e.Body)
}

// rawResponse is the standard error envelope shape.
type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// do performs an HTTP request and, on success, returns the raw response body.
// A non-2xx status is decoded into *APIError.
func (c *Client) do(ctx context.Context, method, path string, query url.Values, body any) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("recurso: marshal request body: %w", err)
		}
		reader = bytes.NewReader(buf)
	}

	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u, reader)
	if err != nil {
		return nil, fmt.Errorf("recurso: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("recurso: %s %s: %w", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("recurso: read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
		var env errorEnvelope
		if json.Unmarshal(respBody, &env) == nil {
			apiErr.Code = env.Error.Code
			apiErr.Message = env.Error.Message
		}
		return nil, apiErr
	}
	return respBody, nil
}

// doResource decodes a single-resource response. The API is inconsistent about
// wrapping single resources in a {"data": ...} envelope (bare on create, wrapped
// on get), so this transparently unwraps a data envelope when present.
func doResource[T any](ctx context.Context, c *Client, method, path string, query url.Values, body any) (*T, error) {
	raw, err := c.do(ctx, method, path, query, body)
	if err != nil {
		return nil, err
	}
	if len(bytesTrim(raw)) == 0 {
		return nil, nil
	}
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if json.Unmarshal(raw, &env) == nil && len(env.Data) > 0 {
		raw = env.Data
	}
	var t T
	if err := json.Unmarshal(raw, &t); err != nil {
		return nil, fmt.Errorf("recurso: decode response: %w", err)
	}
	return &t, nil
}

// doList decodes a {"data": [...]} list response into a slice.
func doList[T any](ctx context.Context, c *Client, method, path string, query url.Values, body any) ([]T, error) {
	raw, err := c.do(ctx, method, path, query, body)
	if err != nil {
		return nil, err
	}
	var env struct {
		Data []T `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("recurso: decode list response: %w", err)
	}
	return env.Data, nil
}

func bytesTrim(b []byte) []byte { return bytes.TrimSpace(b) }

// decodeMap decodes a JSON object response into a map, for endpoints whose
// response is a small ad-hoc object rather than a named resource.
func decodeMap(raw []byte) (map[string]any, error) {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("recurso: decode response: %w", err)
	}
	return m, nil
}

// doAnyList decodes a {"data": [...]} response into an untyped slice, for
// catalog endpoints without a dedicated resource type.
func doAnyList(ctx context.Context, c *Client, method, path string) ([]any, error) {
	return doList[any](ctx, c, method, path, nil, nil)
}
