package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
)

type SubscriptionRepository interface {
	Create(ctx context.Context, sub *domain.Subscription) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Subscription, error)
	GetActiveSubscriptions(ctx context.Context) ([]*domain.Subscription, error)
	List(ctx context.Context, tenantID uuid.UUID, filter domain.SubscriptionFilter) ([]*domain.Subscription, error)
	Update(ctx context.Context, sub *domain.Subscription) error
}
