package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// Pay-in-advance billing: a charge marked pay_in_advance is
// rated PER usage event at ingestion time and captured immediately as a pending
// unbilled charge. GenerateInvoice already folds pending unbilled charges onto
// the subscription's next invoice as tax-resolved lines and posts the ledger
// through the normal RecordInvoice path — so pay-in-advance adds NO new ledger
// code and the double-entry invariant holds by construction. meteredLines skips
// pay_in_advance charges at period close so they are never billed twice.
//
// Only non-cumulative models (per_unit, percentage, dynamic) may be
// pay-in-advance; SetPlanCharges rejects the cumulative ones, whose per-event
// price is undefined (their tier/bundle math needs the whole period).

// PayInAdvanceBiller rates pay-in-advance charges for a single event and writes
// one unbilled charge per matching charge. It is an optional dependency of
// UsageService (nil = disabled), wired via SetPayInAdvanceBiller.
type PayInAdvanceBiller struct {
	charges  port.ChargeRepository
	plans    port.PlanRepository
	unbilled port.UnbilledChargeRepository
	now      func() time.Time
}

// NewPayInAdvanceBiller builds the biller over the charge, plan, and
// unbilled-charge repositories.
func NewPayInAdvanceBiller(charges port.ChargeRepository, plans port.PlanRepository, unbilled port.UnbilledChargeRepository) *PayInAdvanceBiller {
	return &PayInAdvanceBiller{
		charges:  charges,
		plans:    plans,
		unbilled: unbilled,
		now:      func() time.Time { return time.Now().UTC() },
	}
}

// BillEvent rates every pay-in-advance charge on the subscription's plan whose
// metric matches the event's dimension, and captures each as a pending unbilled
// charge in the plan's currency. Returns the number of charges captured. The
// caller decides how to handle an error (the ingestion path logs it for
// reconciliation and does not fail the event write).
func (b *PayInAdvanceBiller) BillEvent(ctx context.Context, sub *domain.Subscription, event *domain.UsageEvent) (int, error) {
	plan, err := b.plans.GetByID(ctx, sub.PlanID)
	if err != nil {
		return 0, fmt.Errorf("pay-in-advance: load plan: %w", err)
	}
	if plan == nil || len(plan.Prices) == 0 {
		return 0, nil
	}
	currency := strings.ToUpper(strings.TrimSpace(plan.Prices[0].Currency))

	charges, err := b.charges.ListByPlan(ctx, sub.TenantID, sub.PlanID)
	if err != nil {
		return 0, fmt.Errorf("pay-in-advance: list charges: %w", err)
	}

	captured := 0
	for _, ch := range charges {
		if !ch.PayInAdvance || ch.Metric == nil || ch.Metric.Code != event.Dimension {
			continue
		}
		amounts, ok := ch.Amounts[currency]
		if !ok {
			continue // no pricing for the plan currency
		}

		// The per-event quantity fed to the (non-cumulative) model: the event's
		// units for per_unit/percentage, its supplied price for dynamic.
		qty := event.Quantity
		if ch.ChargeModel == domain.ChargeDynamic {
			qty = event.DynamicAmount
		}
		fee, err := RateCharge(ch.ChargeModel, amounts, qty)
		if err != nil {
			return captured, fmt.Errorf("pay-in-advance: rate charge %s: %w", ch.ID, err)
		}
		if fee <= 0 {
			continue
		}

		if err := b.unbilled.Create(&domain.UnbilledCharge{
			ID:             uuid.New(),
			SubscriptionID: sub.ID,
			Amount:         fee,
			Currency:       currency,
			Description:    fmt.Sprintf("%s — usage (pay in advance)", ch.Metric.Name),
			HSNCode:        ch.HSNCode,
			Status:         domain.UnbilledChargeStatusPending,
			CreatedAt:      b.now(),
		}); err != nil {
			return captured, fmt.Errorf("pay-in-advance: create unbilled charge: %w", err)
		}
		captured++
	}
	return captured, nil
}
