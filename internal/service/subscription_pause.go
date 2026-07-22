package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// Pause / resume lifecycle (Phase 49 + issue #111 auto-resume). Kept in its own
// file so subscription.go doesn't keep growing; all methods hang off
// *SubscriptionService.

// PauseSubscription pauses an active subscription. resumeAt schedules an
// automatic return to active (issue #111) — e.g. a retention "pause N months"
// offer passes now+N months; a nil resumeAt is an indefinite, manual-resume
// pause (the historical behaviour). The resume time is written with a targeted
// SetResumeAt (not the full-row Update) so it can't be clobbered later.
func (s *SubscriptionService) PauseSubscription(ctx context.Context, tenantID, subscriptionID uuid.UUID, resumeAt *time.Time) (*domain.Subscription, error) {
	sub, err := s.subRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, fmt.Errorf("subscription not found")
	}
	if sub.TenantID != tenantID {
		return nil, fmt.Errorf("subscription not found for tenant")
	}

	if sub.Status != domain.SubscriptionStatusActive {
		return nil, fmt.Errorf("only active subscriptions can be paused")
	}

	sub.Status = domain.SubscriptionStatusPaused

	if err := s.subRepo.Update(ctx, sub); err != nil {
		return nil, err
	}

	// Persist the scheduled resume (or clear any stale value) separately from the
	// status write so a future full-row Update that hasn't loaded resume_at can't
	// wipe it.
	if err := s.subRepo.SetResumeAt(ctx, tenantID, subscriptionID, resumeAt); err != nil {
		return nil, err
	}
	sub.ResumeAt = resumeAt

	return sub, nil
}

// ResumeSubscription returns a paused subscription to active and clears any
// scheduled auto-resume. Used by both the manual /resume endpoint and the
// auto-resume scheduler (issue #111).
func (s *SubscriptionService) ResumeSubscription(ctx context.Context, tenantID, subscriptionID uuid.UUID) (*domain.Subscription, error) {
	sub, err := s.subRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, fmt.Errorf("subscription not found")
	}
	if sub.TenantID != tenantID {
		return nil, fmt.Errorf("subscription not found for tenant")
	}

	if sub.Status != domain.SubscriptionStatusPaused {
		return nil, fmt.Errorf("only paused subscriptions can be resumed")
	}

	sub.Status = domain.SubscriptionStatusActive

	// If the billing period fully elapsed while paused, roll it forward so
	// billing resumes fresh from now. Otherwise the renewal scheduler — which
	// claims active subscriptions with current_period_end <= now and catches up
	// one period per tick — would retroactively bill the ENTIRE paused window
	// (e.g. 3 back-invoices for a 3-month pause). A pause that ends within the
	// original period keeps its remaining time.
	now := time.Now().UTC()
	if sub.CurrentPeriodEnd.Before(now) {
		plan, err := s.planRepo.GetByID(ctx, sub.PlanID)
		if err != nil || plan == nil {
			return nil, fmt.Errorf("plan unavailable for resume: %w", err)
		}
		sub.CurrentPeriodStart = now
		sub.CurrentPeriodEnd = domain.AddInterval(now, string(plan.IntervalUnit), plan.IntervalCount)
	}

	if err := s.subRepo.Update(ctx, sub); err != nil {
		return nil, err
	}

	// Clear the scheduled resume so the resume scheduler won't re-claim this row
	// (and so a leased-but-now-resumed value doesn't linger).
	if err := s.subRepo.SetResumeAt(ctx, tenantID, subscriptionID, nil); err != nil {
		return nil, err
	}
	sub.ResumeAt = nil

	return sub, nil
}
