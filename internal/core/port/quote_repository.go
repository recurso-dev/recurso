package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// QuoteRepository handles quote persistence
type QuoteRepository interface {
	Create(ctx context.Context, quote *domain.Quote) error
	GetByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.Quote, error)
	Update(ctx context.Context, quote *domain.Quote) error
	Delete(ctx context.Context, id, tenantID uuid.UUID) error
	List(ctx context.Context, tenantID uuid.UUID, filter domain.QuoteFilter) ([]*domain.Quote, error)
	GetNextQuoteNumber(ctx context.Context, tenantID uuid.UUID) (string, error)
	// ClaimForConversion atomically stamps invoice_id on an accepted, not-yet-
	// converted quote, returning true only for the caller that won. It is the
	// exclusivity gate so two concurrent conversions can't each mint an invoice.
	ClaimForConversion(ctx context.Context, id, tenantID, invoiceID uuid.UUID) (bool, error)
	// ReleaseConversion clears invoice_id, used to undo a claim when the invoice
	// creation then fails.
	ReleaseConversion(ctx context.Context, id, tenantID uuid.UUID) error
}
