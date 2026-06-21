package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
	"github.com/recur-so/recurso/internal/core/port"
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
func (s *QuoteService) CreateQuote(ctx context.Context, tenantID uuid.UUID, req domain.CreateQuoteRequest) (*domain.Quote, error) {
	// Generate quote number
	quoteNumber, err := s.quoteRepo.GetNextQuoteNumber(ctx, tenantID)
	if err != nil {
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

// GetQuote retrieves a quote by ID
func (s *QuoteService) GetQuote(ctx context.Context, id uuid.UUID) (*domain.Quote, error) {
	return s.quoteRepo.GetByID(ctx, id)
}

// ListQuotes lists quotes with filters
func (s *QuoteService) ListQuotes(ctx context.Context, tenantID uuid.UUID, filter domain.QuoteFilter) ([]*domain.Quote, error) {
	return s.quoteRepo.List(ctx, tenantID, filter)
}

// UpdateQuote updates a quote (only if draft)
func (s *QuoteService) UpdateQuote(ctx context.Context, id uuid.UUID, req domain.CreateQuoteRequest) (*domain.Quote, error) {
	quote, err := s.quoteRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if !quote.IsEditable() {
		return nil, ErrQuoteNotEditable
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
func (s *QuoteService) SendQuote(ctx context.Context, id uuid.UUID) (*domain.Quote, error) {
	quote, err := s.quoteRepo.GetByID(ctx, id)
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
func (s *QuoteService) AcceptQuote(ctx context.Context, id uuid.UUID) (*domain.Quote, error) {
	quote, err := s.quoteRepo.GetByID(ctx, id)
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
func (s *QuoteService) DeclineQuote(ctx context.Context, id uuid.UUID) (*domain.Quote, error) {
	quote, err := s.quoteRepo.GetByID(ctx, id)
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
func (s *QuoteService) ConvertToInvoice(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	quote, err := s.quoteRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if !quote.CanConvertToInvoice() {
		return nil, ErrCannotConvertQuote
	}

	// Create invoice from quote
	dueDate := time.Now().AddDate(0, 0, 30) // Net 30
	invoice := &domain.Invoice{
		ID:         uuid.New(),
		TenantID:   quote.TenantID,
		CustomerID: quote.CustomerID,
		Status:     "open",
		AmountDue:  int64(quote.Total),
		Currency:   quote.Currency,
		DueDate:    dueDate,
		CreatedAt:  time.Now(),
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
func (s *QuoteService) DeleteQuote(ctx context.Context, id uuid.UUID) error {
	quote, err := s.quoteRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if !quote.IsEditable() {
		return ErrQuoteNotEditable
	}

	return s.quoteRepo.Delete(ctx, id)
}

// Quote errors
type QuoteError string

func (e QuoteError) Error() string { return string(e) }

const (
	ErrQuoteNotEditable   = QuoteError("quote is not editable")
	ErrInvalidQuoteStatus = QuoteError("invalid quote status for this action")
	ErrCannotConvertQuote = QuoteError("quote cannot be converted to invoice")
)
