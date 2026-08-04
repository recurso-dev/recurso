package port

import (
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

type UnbilledChargeRepository interface {
	Create(charge *domain.UnbilledCharge) error
	ListBySubscriptionID(subscriptionID uuid.UUID) ([]*domain.UnbilledCharge, error)
	// ListBySubscriptionIDPaged bounds the API-facing list. Invoice generation
	// keeps the unbounded ListBySubscriptionID — billing must sweep every
	// pending charge, and a paged read there would silently leave charges
	// unbilled.
	ListBySubscriptionIDPaged(subscriptionID uuid.UUID, limit, offset int) ([]*domain.UnbilledCharge, error)
	MarkAsInvoiced(chargeIDs []uuid.UUID) error
}
