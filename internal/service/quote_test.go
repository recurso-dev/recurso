package service

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

type qtMockQuoteRepo struct {
	port.QuoteRepository
	quote   *domain.Quote
	updated *domain.Quote
	deleted bool
}

func (m *qtMockQuoteRepo) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.Quote, error) {
	if m.quote == nil || m.quote.ID != id || m.quote.TenantID != tenantID {
		return nil, sql.ErrNoRows
	}
	return m.quote, nil
}

func (m *qtMockQuoteRepo) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	if m.quote == nil || m.quote.ID != id || m.quote.TenantID != tenantID {
		return sql.ErrNoRows
	}
	m.deleted = true
	return nil
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

	if _, err := svc.ConvertToInvoice(context.Background(), quote.ID, quote.TenantID); err != nil {
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

// TestQuote_TenantIsolation proves ENG-160: none of the quote read/mutate paths
// touch a quote belonging to another tenant. A wrong tenant_id resolves to
// "not found" (sql.ErrNoRows) at every entry point, and no invoice/delete/update
// side effect fires.
func TestQuote_TenantIsolation(t *testing.T) {
	owner := uuid.New()
	quote := &domain.Quote{
		ID:          uuid.New(),
		TenantID:    owner,
		CustomerID:  uuid.New(),
		QuoteNumber: "Q-ISO-1",
		Status:      domain.QuoteStatusSent, // accept/decline eligible
		Total:       500000,
		Currency:    "USD",
	}
	attacker := uuid.New()

	newSvc := func() (*QuoteService, *qtMockQuoteRepo, *qtMockInvoiceRepo) {
		qr := &qtMockQuoteRepo{quote: quote}
		ir := &qtMockInvoiceRepo{}
		return NewQuoteService(qr, ir), qr, ir
	}

	t.Run("GetQuote", func(t *testing.T) {
		svc, _, _ := newSvc()
		if _, err := svc.GetQuote(context.Background(), quote.ID, attacker); err != sql.ErrNoRows {
			t.Fatalf("cross-tenant GetQuote: want sql.ErrNoRows, got %v", err)
		}
	})

	t.Run("UpdateQuote", func(t *testing.T) {
		svc, qr, _ := newSvc()
		if _, err := svc.UpdateQuote(context.Background(), quote.ID, attacker, domain.CreateQuoteRequest{}); err != sql.ErrNoRows {
			t.Fatalf("cross-tenant UpdateQuote: want sql.ErrNoRows, got %v", err)
		}
		if qr.updated != nil {
			t.Error("cross-tenant UpdateQuote must not persist changes")
		}
	})

	t.Run("SendQuote", func(t *testing.T) {
		svc, qr, _ := newSvc()
		if _, err := svc.SendQuote(context.Background(), quote.ID, attacker); err != sql.ErrNoRows {
			t.Fatalf("cross-tenant SendQuote: want sql.ErrNoRows, got %v", err)
		}
		if qr.updated != nil {
			t.Error("cross-tenant SendQuote must not persist changes")
		}
	})

	t.Run("AcceptQuote", func(t *testing.T) {
		svc, qr, _ := newSvc()
		if _, err := svc.AcceptQuote(context.Background(), quote.ID, attacker); err != sql.ErrNoRows {
			t.Fatalf("cross-tenant AcceptQuote: want sql.ErrNoRows, got %v", err)
		}
		if qr.updated != nil {
			t.Error("cross-tenant AcceptQuote must not persist changes")
		}
	})

	t.Run("DeclineQuote", func(t *testing.T) {
		svc, qr, _ := newSvc()
		if _, err := svc.DeclineQuote(context.Background(), quote.ID, attacker); err != sql.ErrNoRows {
			t.Fatalf("cross-tenant DeclineQuote: want sql.ErrNoRows, got %v", err)
		}
		if qr.updated != nil {
			t.Error("cross-tenant DeclineQuote must not persist changes")
		}
	})

	t.Run("ConvertToInvoice", func(t *testing.T) {
		svc, _, ir := newSvc()
		if _, err := svc.ConvertToInvoice(context.Background(), quote.ID, attacker); err != sql.ErrNoRows {
			t.Fatalf("cross-tenant ConvertToInvoice: want sql.ErrNoRows, got %v", err)
		}
		if ir.created != nil {
			t.Error("cross-tenant ConvertToInvoice must not create an invoice")
		}
	})

	t.Run("DeleteQuote", func(t *testing.T) {
		draft := *quote
		draft.Status = domain.QuoteStatusDraft // editable → passes the IsEditable gate
		qr := &qtMockQuoteRepo{quote: &draft}
		svc := NewQuoteService(qr, &qtMockInvoiceRepo{})
		if err := svc.DeleteQuote(context.Background(), draft.ID, attacker); err != sql.ErrNoRows {
			t.Fatalf("cross-tenant DeleteQuote: want sql.ErrNoRows, got %v", err)
		}
		if qr.deleted {
			t.Error("cross-tenant DeleteQuote must not delete the quote")
		}
	})

	t.Run("OwnerStillWorks", func(t *testing.T) {
		svc, _, _ := newSvc()
		if _, err := svc.GetQuote(context.Background(), quote.ID, owner); err != nil {
			t.Fatalf("owner GetQuote should succeed, got %v", err)
		}
	})
}
