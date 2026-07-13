package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// QuoteService handles quote business logic
type QuoteService struct {
	quoteRepo   port.QuoteRepository
	invoiceRepo port.InvoiceRepository
}

func NewQuoteService(quoteRepo port.QuoteRepository, invoiceRepo port.InvoiceRepository) *QuoteService {
	return &QuoteService{
		quoteRepo:   quoteRepo,
		invoiceRepo: invoiceRepo,
	}
}

// CreateQuote creates a new quote
// validateQuoteAmounts rejects negative quantities/prices/tax/discount and a
// discount larger than the subtotal+tax (which would make the quote — and the
// invoice it converts to — a negative total, i.e. a credit to the customer).
func validateQuoteAmounts(lineItems []domain.LineItem, taxAmount, discountAmount int) error {
	if taxAmount < 0 || discountAmount < 0 {
		return ErrInvalidQuoteAmount
	}
	subtotal := 0
	for _, li := range lineItems {
		if li.Quantity < 0 || li.UnitPrice < 0 {
			return ErrInvalidQuoteAmount
		}
		subtotal += li.Quantity * li.UnitPrice
	}
	if discountAmount > subtotal+taxAmount {
		return ErrInvalidQuoteAmount
	}
	return nil
}

func (s *QuoteService) CreateQuote(ctx context.Context, tenantID uuid.UUID, req domain.CreateQuoteRequest) (*domain.Quote, error) {
	// Generate quote number
	quoteNumber, err := s.quoteRepo.GetNextQuoteNumber(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	if err := validateQuoteAmounts(req.LineItems, req.TaxAmount, req.DiscountAmount); err != nil {
		return nil, err
	}

	// Set default currency
	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}

	quote := &domain.Quote{
		ID:             uuid.New(),
		TenantID:       tenantID,
		CustomerID:     req.CustomerID,
		QuoteNumber:    quoteNumber,
		Status:         domain.QuoteStatusDraft,
		LineItems:      req.LineItems,
		Currency:       currency,
		ValidUntil:     req.ValidUntil,
		Notes:          req.Notes,
		Terms:          req.Terms,
		TaxAmount:      req.TaxAmount,
		DiscountAmount: req.DiscountAmount,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Calculate totals
	quote.CalculateTotals()

	if err := s.quoteRepo.Create(ctx, quote); err != nil {
		return nil, err
	}

	return quote, nil
}

// GetQuote retrieves a quote by ID, scoped to the tenant.
func (s *QuoteService) GetQuote(ctx context.Context, id, tenantID uuid.UUID) (*domain.Quote, error) {
	return s.quoteRepo.GetByID(ctx, id, tenantID)
}

// ListQuotes lists quotes with filters
func (s *QuoteService) ListQuotes(ctx context.Context, tenantID uuid.UUID, filter domain.QuoteFilter) ([]*domain.Quote, error) {
	return s.quoteRepo.List(ctx, tenantID, filter)
}

// UpdateQuote updates a quote (only if draft)
func (s *QuoteService) UpdateQuote(ctx context.Context, id, tenantID uuid.UUID, req domain.CreateQuoteRequest) (*domain.Quote, error) {
	quote, err := s.quoteRepo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	if !quote.IsEditable() {
		return nil, ErrQuoteNotEditable
	}

	if err := validateQuoteAmounts(req.LineItems, req.TaxAmount, req.DiscountAmount); err != nil {
		return nil, err
	}

	quote.LineItems = req.LineItems
	quote.TaxAmount = req.TaxAmount
	quote.DiscountAmount = req.DiscountAmount
	quote.ValidUntil = req.ValidUntil
	quote.Notes = req.Notes
	quote.Terms = req.Terms
	quote.UpdatedAt = time.Now()

	quote.CalculateTotals()

	if err := s.quoteRepo.Update(ctx, quote); err != nil {
		return nil, err
	}

	return quote, nil
}

// SendQuote marks a quote as sent
func (s *QuoteService) SendQuote(ctx context.Context, id, tenantID uuid.UUID) (*domain.Quote, error) {
	quote, err := s.quoteRepo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	if quote.Status != domain.QuoteStatusDraft {
		return nil, ErrInvalidQuoteStatus
	}

	quote.Status = domain.QuoteStatusSent
	quote.UpdatedAt = time.Now()

	if err := s.quoteRepo.Update(ctx, quote); err != nil {
		return nil, err
	}

	return quote, nil
}

// AcceptQuote marks a quote as accepted
func (s *QuoteService) AcceptQuote(ctx context.Context, id, tenantID uuid.UUID) (*domain.Quote, error) {
	quote, err := s.quoteRepo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	if quote.Status != domain.QuoteStatusSent {
		return nil, ErrInvalidQuoteStatus
	}

	now := time.Now()
	quote.Status = domain.QuoteStatusAccepted
	quote.AcceptedAt = &now
	quote.UpdatedAt = now

	if err := s.quoteRepo.Update(ctx, quote); err != nil {
		return nil, err
	}

	return quote, nil
}

// DeclineQuote marks a quote as declined
func (s *QuoteService) DeclineQuote(ctx context.Context, id, tenantID uuid.UUID) (*domain.Quote, error) {
	quote, err := s.quoteRepo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	if quote.Status != domain.QuoteStatusSent {
		return nil, ErrInvalidQuoteStatus
	}

	now := time.Now()
	quote.Status = domain.QuoteStatusDeclined
	quote.DeclinedAt = &now
	quote.UpdatedAt = now

	if err := s.quoteRepo.Update(ctx, quote); err != nil {
		return nil, err
	}

	return quote, nil
}

// ConvertToInvoice converts an accepted quote to an invoice
func (s *QuoteService) ConvertToInvoice(ctx context.Context, id, tenantID uuid.UUID) (*domain.Invoice, error) {
	quote, err := s.quoteRepo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	if !quote.CanConvertToInvoice() {
		return nil, ErrCannotConvertQuote
	}

	// Create invoice from quote
	dueDate := time.Now().AddDate(0, 0, 30) // Net 30
	invID := uuid.New()

	// Itemization (Phase 1): carry the quote's own line items onto the invoice so
	// the converted invoice is itemized like every other path. Quotes have no
	// per-line GST in Phase 1, so tax fields stay zero. (Note: the quote->invoice
	// conversion currently only sets AmountDue, leaving Subtotal/TaxAmount at 0 —
	// a pre-existing quirk unrelated to itemization; the lines reflect the quote.)
	var lines []domain.InvoiceItem
	for _, li := range quote.LineItems {
		desc := li.Description
		if desc == "" {
			desc = fmt.Sprintf("Quote %s", quote.QuoteNumber)
		}
		lines = append(lines, newInvoiceLine(invID, desc, "", li.Quantity, int64(li.UnitPrice), int64(li.Amount), InvoiceTax{}, time.Time{}))
	}
	if len(lines) == 0 {
		// No quote lines: emit a single line for the quote total so the invoice
		// is still itemized.
		lines = []domain.InvoiceItem{
			newInvoiceLine(invID, fmt.Sprintf("Quote %s", quote.QuoteNumber), "", 1, int64(quote.Total), int64(quote.Total), InvoiceTax{}, time.Time{}),
		}
	}

	invoice := &domain.Invoice{
		ID:         invID,
		TenantID:   quote.TenantID,
		CustomerID: quote.CustomerID,
		Status:     "open",
		// Carry the quote's money fields onto the invoice. Setting only AmountDue
		// left Subtotal/Total/TaxAmount at zero, so the PDF, MarkInvoicePaid, and
		// the ledger all saw a $0 invoice (ENG-144).
		Subtotal:  int64(quote.Subtotal),
		TaxAmount: int64(quote.TaxAmount),
		Total:     int64(quote.Total),
		AmountDue: int64(quote.Total),
		Currency:  quote.Currency,
		LineItems: lines,
		DueDate:   dueDate,
		CreatedAt: time.Now(),
	}

	if err := s.invoiceRepo.Create(ctx, invoice); err != nil {
		return nil, err
	}

	// Update quote with invoice reference
	quote.InvoiceID = &invoice.ID
	quote.UpdatedAt = time.Now()

	if err := s.quoteRepo.Update(ctx, quote); err != nil {
		return nil, err
	}

	return invoice, nil
}

// DeleteQuote deletes a draft quote
func (s *QuoteService) DeleteQuote(ctx context.Context, id, tenantID uuid.UUID) error {
	quote, err := s.quoteRepo.GetByID(ctx, id, tenantID)
	if err != nil {
		return err
	}

	if !quote.IsEditable() {
		return ErrQuoteNotEditable
	}

	return s.quoteRepo.Delete(ctx, id, tenantID)
}

// Quote errors
type QuoteError string

func (e QuoteError) Error() string { return string(e) }

const (
	ErrQuoteNotEditable   = QuoteError("quote is not editable")
	ErrInvalidQuoteStatus = QuoteError("invalid quote status for this action")
	ErrInvalidQuoteAmount = QuoteError("quote amounts must be non-negative and the discount cannot exceed the subtotal plus tax")
	ErrCannotConvertQuote = QuoteError("quote cannot be converted to invoice")
)
