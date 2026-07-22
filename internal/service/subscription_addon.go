package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// AddAddon attaches an add-on plan to a subscription with a quantity. It guards
// tenant ownership of the subscription, validates the quantity, confirms the
// add-on plan exists, and requires the add-on plan's currency to match the
// subscription's base-plan currency. The add-on takes effect from the next
// recurring invoice.
func (s *SubscriptionService) AddAddon(ctx context.Context, tenantID, subscriptionID, planID uuid.UUID, quantity int) (*domain.SubscriptionAddon, error) {
	if s.addonRepo == nil {
		return nil, fmt.Errorf("add-ons are not enabled")
	}
	if quantity <= 0 {
		return nil, ErrInvalidQuantity
	}

	sub, err := s.requireOwnedSubscription(ctx, tenantID, subscriptionID)
	if err != nil {
		return nil, err
	}

	subCurrency, err := s.subscriptionCurrency(ctx, sub)
	if err != nil {
		return nil, err
	}

	addonPlan, err := s.planRepo.GetByID(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("failed to get add-on plan: %w", err)
	}
	if addonPlan == nil || len(addonPlan.Prices) == 0 {
		return nil, ErrPlanNotFound
	}
	if !strings.EqualFold(addonPlan.Prices[0].Currency, subCurrency) {
		return nil, ErrAddonCurrencyMismatch
	}

	addon := &domain.SubscriptionAddon{
		ID:             uuid.New(),
		TenantID:       tenantID,
		SubscriptionID: subscriptionID,
		PlanID:         planID,
		Quantity:       quantity,
		CreatedAt:      time.Now().UTC(),
	}
	if err := s.addonRepo.Create(ctx, addon); err != nil {
		return nil, err
	}

	s.logger.Info("subscription add-on attached",
		"subscription_id", subscriptionID, "add_on_plan_id", planID, "quantity", quantity)

	return addon, nil
}

// ListAddons returns the add-ons attached to a subscription (tenant-scoped).
func (s *SubscriptionService) ListAddons(ctx context.Context, tenantID, subscriptionID uuid.UUID) ([]*domain.SubscriptionAddon, error) {
	if s.addonRepo == nil {
		return nil, fmt.Errorf("add-ons are not enabled")
	}
	if _, err := s.requireOwnedSubscription(ctx, tenantID, subscriptionID); err != nil {
		return nil, err
	}
	return s.addonRepo.ListBySubscriptionID(ctx, tenantID, subscriptionID)
}

// RemoveAddon detaches an add-on from a subscription. It guards subscription
// ownership and confirms the add-on belongs to that subscription before
// deleting, so a valid add-on ID from another subscription cannot be removed.
func (s *SubscriptionService) RemoveAddon(ctx context.Context, tenantID, subscriptionID, addonID uuid.UUID) error {
	if s.addonRepo == nil {
		return fmt.Errorf("add-ons are not enabled")
	}
	if _, err := s.requireOwnedSubscription(ctx, tenantID, subscriptionID); err != nil {
		return err
	}

	addon, err := s.addonRepo.GetByID(ctx, tenantID, addonID)
	if err != nil {
		return err
	}
	if addon == nil || addon.SubscriptionID != subscriptionID {
		return ErrAddonNotFound
	}

	return s.addonRepo.Delete(ctx, tenantID, addonID)
}
