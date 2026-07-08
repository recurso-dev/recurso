package recurso

import (
	"context"
	"net/http"
	"time"
)

// LineItem is a quote line item. Monetary fields are in the smallest unit.
type LineItem struct {
	Description string `json:"description"`
	Quantity    int    `json:"quantity"`
	UnitPrice   int64  `json:"unit_price"`
	Amount      int64  `json:"amount"`
}

// Quote is a sales quote in the quote-to-invoice lifecycle.
type Quote struct {
	ID             string     `json:"id"`
	TenantID       string     `json:"tenant_id"`
	CustomerID     string     `json:"customer_id"`
	QuoteNumber    string     `json:"quote_number"`
	Status         string     `json:"status"`
	LineItems      []LineItem `json:"line_items"`
	Subtotal       int64      `json:"subtotal"`
	TaxAmount      int64      `json:"tax_amount"`
	DiscountAmount int64      `json:"discount_amount"`
	Total          int64      `json:"total"`
	Currency       string     `json:"currency"`
	ValidUntil     *time.Time `json:"valid_until"`
	Notes          string     `json:"notes"`
	Terms          string     `json:"terms"`
	InvoiceID      string     `json:"invoice_id"`
	AcceptedAt     *time.Time `json:"accepted_at"`
	DeclinedAt     *time.Time `json:"declined_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// QuoteCreateParams is the body for creating or updating a quote.
type QuoteCreateParams struct {
	CustomerID     string     `json:"customer_id"`
	LineItems      []LineItem `json:"line_items"`
	Currency       string     `json:"currency,omitempty"`
	ValidUntil     string     `json:"valid_until,omitempty"`
	Notes          string     `json:"notes,omitempty"`
	Terms          string     `json:"terms,omitempty"`
	TaxAmount      int64      `json:"tax_amount,omitempty"`
	DiscountAmount int64      `json:"discount_amount,omitempty"`
}

// QuoteListParams filters the quote list.
type QuoteListParams struct {
	Status     string
	CustomerID string
	Search     string
}

// quoteActionResponse is the {"data": Quote, "message": ...} shape returned by
// the send/accept/decline actions.
type quoteActionResponse struct {
	Data    Quote  `json:"data"`
	Message string `json:"message"`
}

// QuotesService groups the quote endpoints.
type QuotesService struct{ client *Client }

// Create creates a draft quote.
func (s *QuotesService) Create(ctx context.Context, params *QuoteCreateParams) (*Quote, error) {
	out, err := getData[Quote](ctx, s.client, http.MethodPost, "/quotes", params)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// List returns the tenant's quotes.
func (s *QuotesService) List(ctx context.Context, params *QuoteListParams) ([]Quote, error) {
	path := "/quotes"
	if params != nil {
		path = newQuery().
			str("status", params.Status).
			str("customer_id", params.CustomerID).
			str("search", params.Search).
			apply(path)
	}
	return getData[[]Quote](ctx, s.client, http.MethodGet, path, nil)
}

// Get retrieves a quote by ID.
func (s *QuotesService) Get(ctx context.Context, id string) (*Quote, error) {
	out, err := getData[Quote](ctx, s.client, http.MethodGet, "/quotes/"+id, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Update updates a draft quote.
func (s *QuotesService) Update(ctx context.Context, id string, params *QuoteCreateParams) (*Quote, error) {
	out, err := getData[Quote](ctx, s.client, http.MethodPut, "/quotes/"+id, params)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete deletes a draft quote.
func (s *QuotesService) Delete(ctx context.Context, id string) (*MessageResponse, error) {
	var out MessageResponse
	if err := s.client.do(ctx, http.MethodDelete, "/quotes/"+id, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *QuotesService) action(ctx context.Context, id, action string) (*Quote, error) {
	var out quoteActionResponse
	if err := s.client.do(ctx, http.MethodPost, "/quotes/"+id+"/"+action, nil, &out); err != nil {
		return nil, err
	}
	return &out.Data, nil
}

// Send transitions a quote from draft to sent.
func (s *QuotesService) Send(ctx context.Context, id string) (*Quote, error) {
	return s.action(ctx, id, "send")
}

// Accept marks a quote as accepted.
func (s *QuotesService) Accept(ctx context.Context, id string) (*Quote, error) {
	return s.action(ctx, id, "accept")
}

// Decline marks a quote as declined.
func (s *QuotesService) Decline(ctx context.Context, id string) (*Quote, error) {
	return s.action(ctx, id, "decline")
}

// Convert converts an accepted quote into an invoice.
func (s *QuotesService) Convert(ctx context.Context, id string) (*Invoice, error) {
	out, err := getData[Invoice](ctx, s.client, http.MethodPost, "/quotes/"+id+"/convert", nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
