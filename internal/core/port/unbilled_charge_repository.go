package port

import (
	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
)

type UnbilledChargeRepository interface {
	Create(charge *domain.UnbilledCharge) error
	ListBySubscriptionID(subscriptionID uuid.UUID) ([]*domain.UnbilledCharge, error)
	MarkAsInvoiced(chargeIDs []uuid.UUID) error
}
