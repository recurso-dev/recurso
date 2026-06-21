package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
	"github.com/recur-so/recurso/internal/core/port"
)

type AdvancedBillingService struct {
	UnbilledChargeRepo port.UnbilledChargeRepository
	SubscriptionRepo   port.SubscriptionRepository
}

func NewAdvancedBillingService(
	ucRepo port.UnbilledChargeRepository,
	subRepo port.SubscriptionRepository,
) *AdvancedBillingService {
	return &AdvancedBillingService{
		UnbilledChargeRepo: ucRepo,
		SubscriptionRepo:   subRepo,
	}
}

func (s *AdvancedBillingService) AddUnbilledCharge(ctx context.Context, subscriptionID uuid.UUID, amount int64, currency, description string) (*domain.UnbilledCharge, error) {
	// Verify subscription exists
	_, err := s.SubscriptionRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, err // Return error if subscription not found
	}

	charge := &domain.UnbilledCharge{
		ID:             uuid.New(),
		SubscriptionID: subscriptionID,
		Amount:         amount,
		Currency:       currency,
		Description:    description,
		Status:         domain.UnbilledChargeStatusPending,
		CreatedAt:      time.Now(),
	}

	if err := s.UnbilledChargeRepo.Create(charge); err != nil {
		return nil, err
	}

	return charge, nil
}

func (s *AdvancedBillingService) ListUnbilledCharges(subscriptionID uuid.UUID) ([]*domain.UnbilledCharge, error) {
	return s.UnbilledChargeRepo.ListBySubscriptionID(subscriptionID)
}
