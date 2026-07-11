package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// SubscriptionAddonRepository persists add-ons attached to a subscription.
// Every method is tenant-scoped: callers pass the tenant explicitly so the
// invoice-generation path (which resolves the tenant from the subscription)
// and request paths share one contract.
type SubscriptionAddonRepository interface {
	Create(ctx context.Context, addon *domain.SubscriptionAddon) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SubscriptionAddon, error)
	ListBySubscriptionID(ctx context.Context, tenantID, subscriptionID uuid.UUID) ([]*domain.SubscriptionAddon, error)
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}
