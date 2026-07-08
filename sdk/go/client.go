package recurso

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// DefaultBaseURL is the production Recurso API base URL. It already includes
// the /v1 version prefix, so resource paths are joined without it.
const DefaultBaseURL = "https://api.recurso.dev/v1"

// Client is the entry point for the Recurso API. Create one with NewClient
// and reach every endpoint through its resource fields (Plans, Customers,
// Subscriptions, ...). A Client is safe for concurrent use.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client

	Account       *AccountService
	Analytics     *AnalyticsService
	Coupons       *CouponsService
	CreditNotes   *CreditNotesService
	Customers     *CustomersService
	Developer     *DeveloperService
	Entitlements  *EntitlementsService
	Events        *EventsService
	Gifts         *GiftsService
	Invoices      *InvoicesService
	Ledger        *LedgerService
	Mandates      *MandatesService
	Plans         *PlansService
	Quotes        *QuotesService
	Referrals     *ReferralsService
	Subscriptions *SubscriptionsService
	Usage         *UsageService
	Webhooks      *WebhooksService
}

// Option configures a Client. Pass options to NewClient.
type Option func(*Client)

// WithBaseURL overrides the API base URL (default DefaultBaseURL). A trailing
// slash is trimmed. Use this to target a self-hosted or staging deployment.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		c.baseURL = strings.TrimRight(baseURL, "/")
	}
}

// WithHTTPClient sets a custom *http.Client (for timeouts, transports, or
// instrumentation). The default client has no timeout.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		if hc != nil {
			c.httpClient = hc
		}
	}
}

// NewClient returns a Client authenticated with the given API key. The key is
// sent as a bearer token on every request.
func NewClient(apiKey string, opts ...Option) *Client {
	c := &Client{
		apiKey:     apiKey,
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{},
	}
	for _, opt := range opts {
		opt(c)
	}

	c.Account = &AccountService{client: c}
	c.Analytics = &AnalyticsService{client: c}
	c.Coupons = &CouponsService{client: c}
	c.CreditNotes = &CreditNotesService{client: c}
	c.Customers = &CustomersService{client: c}
	c.Developer = &DeveloperService{client: c}
	c.Entitlements = &EntitlementsService{client: c}
	c.Events = &EventsService{client: c}
	c.Gifts = &GiftsService{client: c}
	c.Invoices = &InvoicesService{client: c}
	c.Ledger = &LedgerService{client: c}
	c.Mandates = &MandatesService{client: c}
	c.Plans = &PlansService{client: c}
	c.Quotes = &QuotesService{client: c}
	c.Referrals = &ReferralsService{client: c}
	c.Subscriptions = &SubscriptionsService{client: c}
	c.Usage = &UsageService{client: c}
	c.Webhooks = &WebhooksService{client: c}
	return c
}

// do builds and sends an authenticated request. body, when non-nil, is
// JSON-encoded as the request body. On a 2xx response the JSON body is
// decoded into out (when out is non-nil). On a non-2xx response it decodes the
// error envelope into a *APIError.
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var reqBody io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		var env errorEnvelope
		if json.Unmarshal(data, &env) == nil && env.Error.Code != "" {
			apiErr.Code = env.Error.Code
			apiErr.Message = env.Error.Message
		} else {
			apiErr.Message = strings.TrimSpace(string(data))
		}
		return apiErr
	}

	if out != nil && len(data) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}

// dataEnvelope wraps the common {"data": ...} response shape.
type dataEnvelope[T any] struct {
	Data T    `json:"data"`
	Meta Meta `json:"meta"`
}

// Meta is the pagination metadata returned by paged list endpoints.
type Meta struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
}

// getData performs a request whose successful body is wrapped in {"data": T}
// and returns the unwrapped value.
func getData[T any](ctx context.Context, c *Client, method, path string, body any) (T, error) {
	var env dataEnvelope[T]
	err := c.do(ctx, method, path, body, &env)
	return env.Data, err
}

// query is a small helper for building URL query strings for list endpoints.
type query struct {
	values url.Values
}

func newQuery() *query { return &query{values: url.Values{}} }

func (q *query) str(key, val string) *query {
	if val != "" {
		q.values.Set(key, val)
	}
	return q
}

func (q *query) int(key string, val int) *query {
	if val != 0 {
		q.values.Set(key, strconv.Itoa(val))
	}
	return q
}

// apply appends the encoded query string to path, if any params were set.
func (q *query) apply(path string) string {
	if len(q.values) == 0 {
		return path
	}
	return path + "?" + q.values.Encode()
}
