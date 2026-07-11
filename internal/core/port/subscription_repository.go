package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

type SubscriptionRepository interface {
	Create(ctx context.Context, sub *domain.Subscription) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Subscription, error)
	// GetByStripeSubscriptionID is cross-tenant by design: the Stripe webhook
	// handler uses it to resolve the owning tenant from the subscription.
	// Never call it from tenant-scoped request paths.
	GetByStripeSubscriptionID(ctx context.Context, stripeSubID string) (*domain.Subscription, error)
	GetActiveSubscriptions(ctx context.Context, tenantID uuid.UUID) ([]*domain.Subscription, error)
	List(ctx context.Context, tenantID uuid.UUID, filter domain.SubscriptionFilter) ([]*domain.Subscription, error)
	Update(ctx context.Context, sub *domain.Subscription) error
	// CountActiveByCustomer returns, for the tenant, how many active
	// subscriptions each customer has (customer_id -> count). Customers with no
	// active subscription are absent from the map.
	CountActiveByCustomer(ctx context.Context, tenantID uuid.UUID) (map[uuid.UUID]int, error)
}
