package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
)

type MandateRepository interface {
	Create(ctx context.Context, mandate *domain.Mandate) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Mandate, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Mandate, error)
	Update(ctx context.Context, mandate *domain.Mandate) error
	GetByRazorpayTokenID(ctx context.Context, tokenID string) (*domain.Mandate, error)
	GetDueForPreNotification(ctx context.Context) ([]*domain.Mandate, error)
	GetReadyForDebit(ctx context.Context) ([]*domain.Mandate, error)
}
