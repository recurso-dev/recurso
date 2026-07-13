package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// ErrInvalidChargeAmount is returned when an unbilled charge amount is not
// positive. A negative amount would reduce the next invoice total (a caller
// could zero out or credit an invoice through the charge path).
var ErrInvalidChargeAmount = errors.New("charge amount must be greater than zero")

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

// AddUnbilledCharge records an ad-hoc charge to be folded onto the
// subscription's next invoice as its own line item. hsn is the optional HSN/SAC
// code the charge is taxed at; empty falls back to the tenant SAC at invoice
// time.
func (s *AdvancedBillingService) AddUnbilledCharge(ctx context.Context, subscriptionID uuid.UUID, amount int64, currency, description, hsn string) (*domain.UnbilledCharge, error) {
	if amount <= 0 {
		return nil, ErrInvalidChargeAmount
	}

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
		HSNCode:        hsn,
		Status:         domain.UnbilledChargeStatusPending,
		CreatedAt:      time.Now(),
	}

	if err := s.UnbilledChargeRepo.Create(charge); err != nil {
		return nil, err
	}

	return charge, nil
}

func (s *AdvancedBillingService) ListUnbilledCharges(ctx context.Context, subscriptionID uuid.UUID) ([]*domain.UnbilledCharge, error) {
	// Confirm the subscription belongs to the caller's tenant before exposing
	// its charges — GetByID is tenant-scoped and fails closed. Without this any
	// tenant could read another tenant's unbilled charges by subscription ID.
	if _, err := s.SubscriptionRepo.GetByID(ctx, subscriptionID); err != nil {
		return nil, err
	}
	return s.UnbilledChargeRepo.ListBySubscriptionID(subscriptionID)
}
