package port

import (
	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

type UnbilledChargeRepository interface {
	Create(charge *domain.UnbilledCharge) error
	ListBySubscriptionID(subscriptionID uuid.UUID) ([]*domain.UnbilledCharge, error)
	MarkAsInvoiced(chargeIDs []uuid.UUID) error
}
