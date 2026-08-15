package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// CancelPreview is the deterministic financial forecast of canceling a
// subscription — shown BEFORE the mutation so an operator never cancels blind.
// It exposes ONLY values the engine can compute deterministically without
// changing behavior:
//
//   - effective time + resulting status (immediate vs at-period-end)
//   - the still-deferred revenue an immediate cancel forfeits and recognizes as
//     breakage (the exact figure UnwindOnCancel would post, computed read-only)
//   - the future recurring amount that will no longer be billed (plan list price)
//   - flat_fee_refund, a constant 0 — the current flat fee is paid in advance and
//     is NOT refunded on cancel (surfaced explicitly rather than omitted, so the
//     UI can state plainly that no refund occurs)
//
// It deliberately does NOT include an unused-time proration credit (the cancel
// mutation computes/posts none — that would be inventing money that never
// moves) nor a final metered-usage figure (only computable by the mutating
// invoice path). Both are documented gaps, not fabricated here.
type CancelPreview struct {
	SubscriptionID    uuid.UUID `json:"subscription_id"`
	Immediately       bool      `json:"immediately"`
	EffectiveDate     time.Time `json:"effective_date"`
	ResultingStatus   string    `json:"resulting_status"`
	CancelAtPeriodEnd bool      `json:"cancel_at_period_end"`
	Currency          string    `json:"currency"`
	// DeferredRevenueForfeited is the collected-but-unearned revenue an immediate
	// cancel forfeits (0 for at-period-end — natural recognition continues).
	DeferredRevenueForfeited int64 `json:"deferred_revenue_forfeited"`
	// RecognizedAsBreakage equals DeferredRevenueForfeited (DR Deferred / CR
	// Recognized) — the accounting consequence of an immediate cancel.
	RecognizedAsBreakage int64 `json:"recognized_as_breakage"`
	// AvoidedFutureRecurring is the plan list price that will no longer be billed
	// once the cancellation takes effect (base recurring value, not tax-inclusive).
	AvoidedFutureRecurring int64 `json:"avoided_future_recurring"`
	// FlatFeeRefund is always 0: the flat fee is paid in advance and not refunded.
	FlatFeeRefund int64 `json:"flat_fee_refund"`
}

// PreviewCancel forecasts the financial consequence of canceling a subscription
// without mutating anything. It reuses the mutation's own arithmetic in
// read-only form (the rev-rec forfeit sum) and never touches billing/proration/
// recognition logic. Returns (nil, nil) when the subscription doesn't exist for
// the tenant, so the handler returns a flat 404. Repeated calls are naturally
// idempotent (pure read).
func (s *SubscriptionService) PreviewCancel(ctx context.Context, tenantID, subscriptionID uuid.UUID, immediately bool) (*CancelPreview, error) {
	sub, err := s.subRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}
	if sub == nil || sub.TenantID != tenantID {
		return nil, nil // not found / cross-tenant → 404
	}

	// Currency + the recurring amount that would no longer be billed.
	currency := ""
	var recurring int64
	if plan, perr := s.planRepo.GetByID(ctx, sub.PlanID); perr == nil && plan != nil && len(plan.Prices) > 0 {
		currency = plan.Prices[0].Currency
		recurring = plan.Prices[0].Amount
	}
	if currency == "" {
		currency = "USD"
	}

	now := time.Now().UTC()

	// Already canceled: canceling again is a no-op (matches the mutation's
	// idempotent guard) — nothing is forfeited or avoided.
	if sub.Status == domain.SubscriptionStatusCanceled {
		eff := now
		if sub.CanceledAt != nil {
			eff = *sub.CanceledAt
		}
		return &CancelPreview{
			SubscriptionID:  sub.ID,
			Immediately:     immediately,
			EffectiveDate:   eff,
			ResultingStatus: string(domain.SubscriptionStatusCanceled),
			Currency:        currency,
		}, nil
	}

	preview := &CancelPreview{
		SubscriptionID:         sub.ID,
		Immediately:            immediately,
		CancelAtPeriodEnd:      !immediately,
		Currency:               currency,
		AvoidedFutureRecurring: recurring,
		FlatFeeRefund:          0,
	}
	if immediately {
		preview.EffectiveDate = now
		preview.ResultingStatus = string(domain.SubscriptionStatusCanceled)
		// Only an immediate cancel forfeits still-deferred revenue. Compute the
		// exact figure UnwindOnCancel would recognize as breakage — read-only.
		if s.revrecService != nil {
			forfeited, err := s.revrecService.RemainingDeferredForSubscription(ctx, tenantID, subscriptionID)
			if err != nil {
				return nil, err
			}
			preview.DeferredRevenueForfeited = forfeited
			preview.RecognizedAsBreakage = forfeited
		}
	} else {
		// At period end: status is unchanged until the period ends; recognition
		// continues naturally, so nothing is forfeited now.
		preview.EffectiveDate = sub.CurrentPeriodEnd
		preview.ResultingStatus = string(sub.Status)
	}
	return preview, nil
}
