package port

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
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
	// SetResumeAt records (or clears, when nil) a paused subscription's scheduled
	// auto-resume time (issue #111). A targeted write, not routed through Update,
	// so it can't be clobbered by a caller that didn't load resume_at.
	SetResumeAt(ctx context.Context, tenantID, subID uuid.UUID, resumeAt *time.Time) error
	// CountActiveByCustomer returns, for the tenant, how many active
	// subscriptions each customer has (customer_id -> count). Customers with no
	// active subscription are absent from the map.
	CountActiveByCustomer(ctx context.Context, tenantID uuid.UUID) (map[uuid.UUID]int, error)
}
