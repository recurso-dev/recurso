package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// QuoteRepository handles quote persistence
type QuoteRepository interface {
	Create(ctx context.Context, quote *domain.Quote) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Quote, error)
	Update(ctx context.Context, quote *domain.Quote) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, tenantID uuid.UUID, filter domain.QuoteFilter) ([]*domain.Quote, error)
	GetNextQuoteNumber(ctx context.Context, tenantID uuid.UUID) (string, error)
}
