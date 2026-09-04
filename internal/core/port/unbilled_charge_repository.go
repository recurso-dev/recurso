package port

import (
	"context"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

type UnbilledChargeRepository interface {
	Create(ctx context.Context, charge *domain.UnbilledCharge) error
	ListBySubscriptionID(ctx context.Context, subscriptionID uuid.UUID) ([]*domain.UnbilledCharge, error)
	// ListBySubscriptionIDPaged bounds the API-facing list. Invoice generation
	// keeps the unbounded ListBySubscriptionID — billing must sweep every
	// pending charge, and a paged read there would silently leave charges
	// unbilled.
	ListBySubscriptionIDPaged(ctx context.Context, subscriptionID uuid.UUID, limit, offset int) ([]*domain.UnbilledCharge, error)
	MarkAsInvoiced(ctx context.Context, chargeIDs []uuid.UUID) error
}
