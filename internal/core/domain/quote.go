package domain

import (
	"time"

	"github.com/google/uuid"
)

// Quote represents a price quote that can be converted to an invoice
type Quote struct {
	ID             uuid.UUID   `json:"id" db:"id"`
	TenantID       uuid.UUID   `json:"tenant_id" db:"tenant_id"`
	CustomerID     uuid.UUID   `json:"customer_id" db:"customer_id"`
	QuoteNumber    string      `json:"quote_number" db:"quote_number"`
	Status         QuoteStatus `json:"status" db:"status"`
	LineItems      []LineItem  `json:"line_items" db:"line_items"`
	Subtotal       int         `json:"subtotal" db:"subtotal"`               // cents
	TaxAmount      int         `json:"tax_amount" db:"tax_amount"`           // cents
	DiscountAmount int         `json:"discount_amount" db:"discount_amount"` // cents
	Total          int         `json:"total" db:"total"`                     // cents
	Currency       string      `json:"currency" db:"currency"`
	ValidUntil     *time.Time  `json:"valid_until" db:"valid_until"`
	Notes          string      `json:"notes" db:"notes"`
	Terms          string      `json:"terms" db:"terms"`
	InvoiceID      *uuid.UUID  `json:"invoice_id" db:"invoice_id"` // Set when converted
	AcceptedAt     *time.Time  `json:"accepted_at" db:"accepted_at"`
	DeclinedAt     *time.Time  `json:"declined_at" db:"declined_at"`
	CreatedAt      time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at" db:"updated_at"`
}

// QuoteStatus represents the status of a quote
type QuoteStatus string

const (
	QuoteStatusDraft    QuoteStatus = "draft"
	QuoteStatusSent     QuoteStatus = "sent"
	QuoteStatusAccepted QuoteStatus = "accepted"
	QuoteStatusDeclined QuoteStatus = "declined"
	QuoteStatusExpired  QuoteStatus = "expired"
)

// LineItem represents an item on a quote/invoice
type LineItem struct {
	Description string `json:"description"`
	Quantity    int    `json:"quantity"`
	UnitPrice   int    `json:"unit_price"` // cents
	Amount      int    `json:"amount"`     // quantity * unit_price
}

// IsEditable returns true if the quote can be modified
func (q *Quote) IsEditable() bool {
	return q.Status == QuoteStatusDraft
}

// IsExpired returns true if the quote has passed its valid_until date
func (q *Quote) IsExpired() bool {
	if q.ValidUntil == nil {
		return false
	}
	return time.Now().After(*q.ValidUntil)
}

// CanConvertToInvoice returns true if the quote can be converted to an invoice
func (q *Quote) CanConvertToInvoice() bool {
	return q.Status == QuoteStatusAccepted && q.InvoiceID == nil
}

// CalculateTotals recalculates subtotal and total from line items
func (q *Quote) CalculateTotals() {
	q.Subtotal = 0
	for i := range q.LineItems {
		q.LineItems[i].Amount = q.LineItems[i].Quantity * q.LineItems[i].UnitPrice
		q.Subtotal += q.LineItems[i].Amount
	}
	q.Total = q.Subtotal + q.TaxAmount - q.DiscountAmount
}

// CreateQuoteRequest for creating new quotes
type CreateQuoteRequest struct {
	CustomerID     uuid.UUID  `json:"customer_id" binding:"required"`
	LineItems      []LineItem `json:"line_items" binding:"required,min=1"`
	Currency       string     `json:"currency"`
	ValidUntil     *time.Time `json:"valid_until"`
	Notes          string     `json:"notes"`
	Terms          string     `json:"terms"`
	TaxAmount      int        `json:"tax_amount"`
	DiscountAmount int        `json:"discount_amount"`
}

// QuoteFilter for filtering quotes
type QuoteFilter struct {
	Status     string
	CustomerID string
	Search     string
	Limit      int
	Offset     int
}
