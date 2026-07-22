package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// CancelResult contains the result of a subscription cancellation
type CancelResult struct {
	ID               uuid.UUID
	Status           string
	CurrentPeriodEnd time.Time
	CustomerEmail    string
	CustomerName     string
	PlanName         string
}

// Cancel cancels a subscription
func (s *SubscriptionService) Cancel(ctx context.Context, tenantID, subscriptionID uuid.UUID, immediately bool, reason, feedback string) (*CancelResult, error) {
	sub, err := s.subRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}
	if sub == nil {
		return nil, fmt.Errorf("subscription not found")
	}
	if sub.TenantID != tenantID {
		return nil, fmt.Errorf("subscription not found for tenant")
	}

	// Idempotent: an already-canceled subscription is a no-op. Re-running would
	// overwrite CanceledAt with a later time, re-call the gateway cancel, and
	// re-invoke the rev-rec unwind — so guard the terminal state.
	if sub.Status == domain.SubscriptionStatusCanceled {
		return &CancelResult{ID: sub.ID, Status: string(sub.Status), CurrentPeriodEnd: sub.CurrentPeriodEnd}, nil
	}

	// Get customer and plan info for notification (best-effort — the cancel
	// still succeeds if these fail; the notification fields just stay blank).
	customer, err := s.customerRepo.GetByID(ctx, sub.CustomerID)
	if err != nil {
		s.logger.Warn("cancel: customer lookup failed; notification fields may be blank",
			"subscription_id", sub.ID, "error", err)
	}
	plan, err := s.planRepo.GetByID(ctx, sub.PlanID)
	if err != nil {
		s.logger.Warn("cancel: plan lookup failed; notification fields may be blank",
			"subscription_id", sub.ID, "error", err)
	}

	now := time.Now().UTC()

	if immediately {
		sub.Status = domain.SubscriptionStatusCanceled
		sub.CanceledAt = &now
	} else {
		sub.CancelAtPeriodEnd = true
	}

	sub.CancellationReason = reason
	sub.CancellationFeedback = feedback
	sub.UpdatedAt = now

	if err := s.subRepo.Update(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to update subscription: %w", err)
	}

	// Cancel on payment gateway (best-effort)
	if s.gateway != nil {
		if sub.RazorpaySubscriptionID != "" {
			if err := s.gateway.CancelSubscription(ctx, sub.RazorpaySubscriptionID); err != nil {
				s.logger.Error("failed to cancel subscription on payment gateway", "error", err, "gateway", "razorpay", "subscription_id", sub.RazorpaySubscriptionID)
			}
		}
		if sub.StripeSubscriptionID != "" {
			if err := s.gateway.CancelSubscription(ctx, sub.StripeSubscriptionID); err != nil {
				s.logger.Error("failed to cancel subscription on payment gateway", "error", err, "gateway", "stripe", "subscription_id", sub.StripeSubscriptionID)
			}
		}
	}

	// Usage-based billing v1: bill the metered usage of the partial elapsed
	// window on immediate cancel — the flat fee was paid in advance, but the
	// usage since period start would otherwise never be invoiced. Best-effort:
	// a failure logs loudly and leaves the window unclaimed for a manual rerun.
	// Cancel-at-period-end needs nothing here: the normal cycle generator
	// rates the full period when it closes.
	if immediately && s.finalUsageInvoicer != nil {
		if finalInv, err := s.finalUsageInvoicer.GenerateFinalUsageInvoice(ctx, sub, now); err != nil {
			s.logger.Error("final usage invoice on cancel failed", "error", err, "subscription_id", sub.ID)
		} else if finalInv != nil {
			s.logger.Info("final usage invoice generated on cancel",
				"subscription_id", sub.ID, "invoice_id", finalInv.ID, "total", finalInv.Total)
		}
	}

	// Rev-rec unwind on immediate cancel: forfeit (recognize) the remaining
	// deferred revenue and void future recognition events, so a mid-period
	// cancel doesn't leave deferred sitting forever or keep firing recognition
	// (ENG-147). Only for immediate cancels — cancel-at-period-end keeps service
	// (and the natural recognition schedule) running to period end. Best-effort.
	if immediately && s.revrecService != nil {
		if forfeited, err := s.revrecService.UnwindOnCancel(ctx, tenantID, subscriptionID); err != nil {
			s.logger.Error("rev-rec unwind on cancel failed", "error", err, "subscription_id", subscriptionID)
		} else if forfeited > 0 {
			s.logger.Info("rev-rec deferred forfeited on cancel", "subscription_id", subscriptionID, "amount", forfeited)
		}
	}

	result := &CancelResult{
		ID:               sub.ID,
		Status:           string(sub.Status),
		CurrentPeriodEnd: sub.CurrentPeriodEnd,
	}

	if customer != nil {
		result.CustomerEmail = customer.Email
		result.CustomerName = domain.PtrToString(customer.Name)
	}
	if plan != nil {
		result.PlanName = plan.Name
	}

	return result, nil
}

// Reactivate reactivates a cancelled subscription
func (s *SubscriptionService) Reactivate(ctx context.Context, tenantID, subscriptionID uuid.UUID) (*domain.Subscription, error) {
	sub, err := s.subRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}
	if sub == nil {
		return nil, fmt.Errorf("subscription not found")
	}
	if sub.TenantID != tenantID {
		return nil, fmt.Errorf("subscription not found for tenant")
	}

	// Can only reactivate if cancel_at_period_end is true or within grace period
	if !sub.CancelAtPeriodEnd && sub.Status != domain.SubscriptionStatusCanceled {
		return nil, fmt.Errorf("subscription cannot be reactivated")
	}

	// Check if still within period
	if time.Now().After(sub.CurrentPeriodEnd) {
		return nil, fmt.Errorf("subscription period has ended, please create a new subscription")
	}

	sub.CancelAtPeriodEnd = false
	sub.CancellationReason = ""
	sub.CancellationFeedback = ""
	sub.CanceledAt = nil // clear the cancel timestamp — otherwise churn/MRR/rev-rec queries that filter canceled_at IS NOT NULL misclassify the reactivated (live) subscription as churned
	sub.Status = domain.SubscriptionStatusActive
	sub.UpdatedAt = time.Now().UTC()

	if err := s.subRepo.Update(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to reactivate subscription: %w", err)
	}

	return sub, nil
}
