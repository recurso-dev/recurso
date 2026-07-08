package recurso

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// InvoiceItem is one itemized line on an invoice.
type InvoiceItem struct {
	ID            string    `json:"id"`
	InvoiceID     string    `json:"invoice_id"`
	Description   string    `json:"description"`
	HSNCode       string    `json:"hsn_code"`
	Quantity      int       `json:"quantity"`
	UnitAmount    int64     `json:"unit_amount"`
	Amount        int64     `json:"amount"`
	TaxRate       float64   `json:"tax_rate"`
	CGSTAmount    int64     `json:"cgst_amount"`
	SGSTAmount    int64     `json:"sgst_amount"`
	IGSTAmount    int64     `json:"igst_amount"`
	TaxableAmount int64     `json:"taxable_amount"`
	CreatedAt     time.Time `json:"created_at"`
}

// Invoice is a billing invoice.
type Invoice struct {
	ID                string        `json:"id"`
	TenantID          string        `json:"tenant_id"`
	SubscriptionID    string        `json:"subscription_id"`
	CustomerID        string        `json:"customer_id"`
	InvoiceNumber     string        `json:"invoice_number"`
	BillingReason     string        `json:"billing_reason"`
	AmountDue         int64         `json:"amount_due"`
	AmountPaid        int64         `json:"amount_paid"`
	Currency          string        `json:"currency"`
	Subtotal          int64         `json:"subtotal"`
	TaxAmount         int64         `json:"tax_amount"`
	Total             int64         `json:"total"`
	IGSTAmount        int64         `json:"igst_amount"`
	CGSTAmount        int64         `json:"cgst_amount"`
	SGSTAmount        int64         `json:"sgst_amount"`
	HSNCode           string        `json:"hsn_code"`
	IRN               string        `json:"irn"`
	AckNo             string        `json:"ack_no"`
	SignedQRCode      string        `json:"signed_qr_code"`
	EInvoiceStatus    string        `json:"e_invoice_status"`
	TDSAmount         int64         `json:"tds_amount"`
	Status            string        `json:"status"`
	CreatedAt         time.Time     `json:"created_at"`
	DueDate           time.Time     `json:"due_date"`
	PaidAt            time.Time     `json:"paid_at"`
	PaymentTerms      string        `json:"payment_terms"`
	ExchangeRate      float64       `json:"exchange_rate"`
	BaseCurrencyTotal int64         `json:"base_currency_total"`
	BaseCurrency      string        `json:"base_currency"`
	RetryCount        int           `json:"retry_count"`
	PaymentWallActive bool          `json:"payment_wall_active"`
	LineItems         []InvoiceItem `json:"line_items"`
}

// EInvoiceStatus is the IRN (Indian e-invoicing) status for an invoice.
type EInvoiceStatus struct {
	InvoiceID      string     `json:"invoice_id"`
	InvoiceNumber  string     `json:"invoice_number"`
	EInvoiceStatus string     `json:"e_invoice_status"`
	IRN            string     `json:"irn"`
	AckNo          string     `json:"ack_no"`
	AckDate        string     `json:"ack_date"`
	SignedQRCode   string     `json:"signed_qr_code"`
	RetryCount     int        `json:"retry_count"`
	NextRetryAt    *time.Time `json:"next_retry_at"`
	ErrorMessage   string     `json:"error_message"`
}

// EInvoiceRetryResult is returned by RetryEInvoice.
type EInvoiceRetryResult struct {
	Data    json.RawMessage `json:"data"`
	Message string          `json:"message"`
}

// MessageResponse is a generic {"message": ...} acknowledgement.
type MessageResponse struct {
	Message string `json:"message"`
}

// EInvoiceCancelParams cancels an e-invoice (IRN).
type EInvoiceCancelParams struct {
	CancelCode int    `json:"cancel_code"`
	Reason     string `json:"reason"`
}

// InvoiceListParams filters the invoice list.
type InvoiceListParams struct {
	Limit int
	Page  int
}

// InvoicesService groups the invoice endpoints.
type InvoicesService struct{ client *Client }

// List returns the tenant's invoices.
func (s *InvoicesService) List(ctx context.Context, params *InvoiceListParams) ([]Invoice, error) {
	path := "/invoices"
	if params != nil {
		path = newQuery().int("limit", params.Limit).int("page", params.Page).apply(path)
	}
	return getData[[]Invoice](ctx, s.client, http.MethodGet, path, nil)
}

// PDFURL returns the public URL for an invoice's printable document. It does
// not perform a request.
func (s *InvoicesService) PDFURL(id string) string {
	return fmt.Sprintf("%s/invoices/%s/pdf", s.client.baseURL, id)
}

// EInvoiceStatus returns the e-invoice (IRN) status for an invoice.
func (s *InvoicesService) EInvoiceStatus(ctx context.Context, id string) (*EInvoiceStatus, error) {
	out, err := getData[EInvoiceStatus](ctx, s.client, http.MethodGet, fmt.Sprintf("/invoices/%s/einvoice", id), nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// RetryEInvoice retries IRN generation for a failed e-invoice.
func (s *InvoicesService) RetryEInvoice(ctx context.Context, id string) (*EInvoiceRetryResult, error) {
	var out EInvoiceRetryResult
	if err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/invoices/%s/einvoice/retry", id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CancelEInvoice cancels an e-invoice (IRN).
func (s *InvoicesService) CancelEInvoice(ctx context.Context, id string, params *EInvoiceCancelParams) (*MessageResponse, error) {
	var out MessageResponse
	if err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/invoices/%s/einvoice/cancel", id), params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
