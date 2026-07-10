package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

type qtMockQuoteRepo struct {
	port.QuoteRepository
	quote   *domain.Quote
	updated *domain.Quote
}

func (m *qtMockQuoteRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Quote, error) {
	return m.quote, nil
}

func (m *qtMockQuoteRepo) Update(ctx context.Context, q *domain.Quote) error {
	m.updated = q
	return nil
}

type qtMockInvoiceRepo struct {
	port.InvoiceRepository
	created *domain.Invoice
}

func (m *qtMockInvoiceRepo) Create(ctx context.Context, inv *domain.Invoice) error {
	m.created = inv
	return nil
}

// TestQuoteConvertToInvoice_CarriesMoneyFields proves the ENG-144 fix: a
// converted quote's invoice carries the quote's Subtotal/Tax/Total, not $0.
func TestQuoteConvertToInvoice_CarriesMoneyFields(t *testing.T) {
	quote := &domain.Quote{
		ID:          uuid.New(),
		TenantID:    uuid.New(),
		CustomerID:  uuid.New(),
		QuoteNumber: "Q-1",
		Status:      domain.QuoteStatusAccepted, // convertible
		Subtotal:    1000000,
		TaxAmount:   180000,
		Total:       1180000,
		Currency:    "INR",
	}
	qr := &qtMockQuoteRepo{quote: quote}
	ir := &qtMockInvoiceRepo{}
	svc := NewQuoteService(qr, ir)

	if _, err := svc.ConvertToInvoice(context.Background(), quote.ID); err != nil {
		t.Fatalf("ConvertToInvoice: %v", err)
	}
	if ir.created == nil {
		t.Fatal("no invoice was created")
	}
	if ir.created.Total != 1180000 {
		t.Errorf("invoice Total = %d, want 1180000 (from the quote, not $0)", ir.created.Total)
	}
	if ir.created.Subtotal != 1000000 {
		t.Errorf("invoice Subtotal = %d, want 1000000", ir.created.Subtotal)
	}
	if ir.created.TaxAmount != 180000 {
		t.Errorf("invoice TaxAmount = %d, want 180000", ir.created.TaxAmount)
	}
}
