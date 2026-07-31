package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// ApplyRetentionDiscount honors a "discount" retention offer accepted in the
// cancel flow: it mints a percentage coupon and attaches it to the
// subscription, so the promised discount actually lands on upcoming invoices.
// Previously the cancel flow only LOGGED an accepted discount offer and billed
// the retained customer at full price — a broken promise.
//
// The renewal path applies a subscription's coupon per couponAppliesThisPeriod
// (see InvoiceService.GenerateInvoice), so the current (already-invoiced)
// period is untouched and the discount begins on the next renewal:
//   - durationMonths > 0  → a repeating coupon for that many renewals
//   - durationMonths == 0 → a forever coupon (every renewal)
//
// The applied-periods counter is reset to 0 so the discount runs for its full
// duration from the next renewal. subRepo.Update persists coupon_id and
// coupon_periods_applied (see the subscription repository).
func (s *SubscriptionService) ApplyRetentionDiscount(ctx context.Context, tenantID, subscriptionID uuid.UUID, percent, durationMonths int) (*domain.Coupon, error) {
	if percent <= 0 || percent > 100 {
		return nil, fmt.Errorf("retention discount percent must be 1-100, got %d", percent)
	}
	if durationMonths < 0 {
		return nil, fmt.Errorf("retention discount duration months must be >= 0, got %d", durationMonths)
	}
	if s.couponRepo == nil {
		return nil, fmt.Errorf("coupon repository not configured; cannot apply retention discount")
	}

	sub, err := s.subRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}
	if sub == nil || sub.TenantID != tenantID {
		return nil, fmt.Errorf("subscription not found")
	}
	if sub.Status == domain.SubscriptionStatusCanceled {
		return nil, fmt.Errorf("cannot apply a retention discount to a canceled subscription")
	}

	codeBytes := make([]byte, 5)
	if _, err := rand.Read(codeBytes); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	coupon := &domain.Coupon{
		ID:            uuid.New(),
		TenantID:      tenantID,
		Code:          fmt.Sprintf("RETAIN-%s", hex.EncodeToString(codeBytes)),
		DiscountType:  domain.DiscountTypePercent,
		DiscountValue: int64(percent),
		Duration:      domain.DurationForever,
		Active:        true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if durationMonths > 0 {
		months := durationMonths
		coupon.Duration = domain.DurationRepeating
		coupon.DurationMonths = &months
	}
	if err := s.couponRepo.Create(ctx, coupon); err != nil {
		return nil, fmt.Errorf("failed to create retention coupon: %w", err)
	}

	sub.CouponID = &coupon.ID
	sub.CouponPeriodsApplied = 0
	sub.UpdatedAt = now
	if err := s.subRepo.Update(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to attach retention coupon to subscription: %w", err)
	}
	return coupon, nil
}
