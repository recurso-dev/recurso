package recurso

import (
	"context"
	"net/http"
	"time"
)

// CreditNote is a customer credit.
type CreditNote struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	CustomerID string    `json:"customer_id"`
	InvoiceID  string    `json:"invoice_id"`
	Reference  string    `json:"reference"`
	Amount     int64     `json:"amount"`
	Balance    int64     `json:"balance"`
	Currency   string    `json:"currency"`
	Status     string    `json:"status"`
	Reason     string    `json:"reason"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Customer   *Customer `json:"customer,omitempty"`
}

// CreditNoteCreateParams is the body for issuing a credit note. Amount is in the
// currency's smallest unit.
type CreditNoteCreateParams struct {
	CustomerID string `json:"customer_id"`
	InvoiceID  string `json:"invoice_id,omitempty"`
	Amount     int64  `json:"amount"`
	Currency   string `json:"currency"`
	Reason     string `json:"reason,omitempty"`
}

// CreditNoteListParams filters the credit-note list.
type CreditNoteListParams struct {
	CustomerID string
}

// CreditNotesService groups the credit-note endpoints.
type CreditNotesService struct{ client *Client }

// Create issues a credit note.
func (s *CreditNotesService) Create(ctx context.Context, params *CreditNoteCreateParams) (*CreditNote, error) {
	out, err := getData[CreditNote](ctx, s.client, http.MethodPost, "/credit-notes", params)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// List returns the tenant's credit notes.
func (s *CreditNotesService) List(ctx context.Context, params *CreditNoteListParams) ([]CreditNote, error) {
	path := "/credit-notes"
	if params != nil {
		path = newQuery().str("customer_id", params.CustomerID).apply(path)
	}
	return getData[[]CreditNote](ctx, s.client, http.MethodGet, path, nil)
}
