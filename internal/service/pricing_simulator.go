package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// Pricing simulator: a read-only "what would this pricing
// bill?" calculator. It validates a PROPOSED charge set the same way
// SetPlanCharges does, rates it against sample usage (explicit quantities, or a
// subscription's current-period usage), and returns the rated lines plus a
// balanced general-ledger preview. Nothing is persisted and no ledger legs are
// posted — the simulator is a pure function of its inputs.

// SimulateUsage is a sample usage quantity for one metric.
type SimulateUsage struct {
	MetricID string `json:"metric_id"`
	Quantity int64  `json:"quantity"`
}

// SimulateRequest is a pricing-simulation request for a plan.
type SimulateRequest struct {
	// Currency to rate in; defaults to the plan's first price currency.
	Currency string `json:"currency"`
	// SubscriptionID (optional): when set, metrics without an explicit Usage
	// entry take that subscription's current-period usage.
	SubscriptionID string `json:"subscription_id"`
	// Charges is the proposed charge set (same shape as SetPlanCharges).
	Charges []ChargeInput `json:"charges"`
	// Usage supplies sample quantities per metric_id (overrides subscription
	// usage for that metric).
	Usage []SimulateUsage `json:"usage"`
}

// SimulatedCharge is one proposed charge's rated result.
type SimulatedCharge struct {
	MetricID    string `json:"metric_id"`
	MetricCode  string `json:"metric_code"`
	MetricName  string `json:"metric_name"`
	ChargeModel string `json:"charge_model"`
	Quantity    int64  `json:"quantity"`
	Amount      int64  `json:"amount"` // minor units
}

// SimulatedGLLine is one line of the general-ledger preview.
type SimulatedGLLine struct {
	AccountCode int    `json:"account_code"`
	AccountName string `json:"account_name"`
	Debit       int64  `json:"debit"`
	Credit      int64  `json:"credit"`
}

// ChargeSimulation is the simulator's result: the rated charges, the subtotal,
// and a balanced GL preview. Pre-tax — GST/tax is resolved at invoice time.
type ChargeSimulation struct {
	PlanID    uuid.UUID         `json:"plan_id"`
	Currency  string            `json:"currency"`
	Charges   []SimulatedCharge `json:"charges"`
	Subtotal  int64             `json:"subtotal"`
	GLPreview []SimulatedGLLine `json:"gl_preview"`
	Balanced  bool              `json:"balanced"`
	Note      string            `json:"note"`
}

// SimulateCharges rates a proposed charge set against sample usage and returns
// a pre-tax preview with a balanced GL projection. Read-only: no persistence,
// no ledger posting.
func (s *MeteringService) SimulateCharges(ctx context.Context, tenantID, planID uuid.UUID, req SimulateRequest) (*ChargeSimulation, error) {
	plan, err := s.plans.GetByID(ctx, planID)
	if err != nil || plan == nil || plan.TenantID != tenantID {
		return nil, ErrMeteringPlanNotFound
	}

	currency := strings.ToUpper(strings.TrimSpace(req.Currency))
	if currency == "" && len(plan.Prices) > 0 {
		currency = strings.ToUpper(strings.TrimSpace(plan.Prices[0].Currency))
	}
	if len(currency) != 3 {
		return nil, MeteringValidationError("currency is required (or the plan must have a price)")
	}

	// Optional subscription for real usage.
	var sub *domain.Subscription
	if req.SubscriptionID != "" {
		sid, err := uuid.Parse(req.SubscriptionID)
		if err != nil {
			return nil, MeteringValidationError("invalid subscription_id")
		}
		sub, err = s.subscriptions.GetByID(ctx, sid)
		if err != nil {
			return nil, err
		}
		if sub == nil || sub.TenantID != tenantID {
			return nil, MeteringValidationError("subscription not found")
		}
	}

	usageByMetric := make(map[string]int64, len(req.Usage))
	for _, u := range req.Usage {
		usageByMetric[u.MetricID] = u.Quantity
	}

	out := &ChargeSimulation{
		PlanID:   planID,
		Currency: currency,
		Charges:  []SimulatedCharge{},
	}
	seen := map[uuid.UUID]bool{}
	for i, in := range req.Charges {
		metric, model, normalized, err := s.resolveChargeInput(ctx, tenantID, i, in)
		if err != nil {
			return nil, err
		}
		if seen[metric.ID] {
			return nil, MeteringValidationError(fmt.Sprintf("charges[%d]: duplicate charge for metric %s", i, metric.Code))
		}
		seen[metric.ID] = true

		amounts, ok := normalized[currency]
		if !ok {
			return nil, MeteringValidationError(fmt.Sprintf("charges[%d]: no pricing for currency %s", i, currency))
		}

		qty, explicit := usageByMetric[metric.ID.String()]
		if !explicit && sub != nil {
			ch := domain.Charge{ChargeModel: model, Metric: metric}
			qty, err = meteredQuantity(ctx, s.usage, sub.ID, ch, sub.CurrentPeriodStart, s.now())
			if err != nil {
				return nil, err
			}
		}
		if qty < 0 {
			qty = 0
		}

		amount, err := RateCharge(model, amounts, qty)
		if err != nil {
			return nil, MeteringValidationError(fmt.Sprintf("charges[%d]: %v", i, err))
		}

		out.Charges = append(out.Charges, SimulatedCharge{
			MetricID:    metric.ID.String(),
			MetricCode:  metric.Code,
			MetricName:  metric.Name,
			ChargeModel: string(model),
			Quantity:    qty,
			Amount:      amount,
		})
		out.Subtotal += amount
	}

	// GL preview mirrors how a real invoice posts its metered revenue (ADR-002,
	// Code-1): DR Accounts Receivable / CR Revenue for the subtotal. Balanced by
	// construction. Tax is resolved per line at invoice time, not here.
	out.GLPreview = []SimulatedGLLine{
		{AccountCode: domain.AccountCodeAR, AccountName: "Accounts Receivable", Debit: out.Subtotal, Credit: 0},
		{AccountCode: domain.AccountCodeRevenue, AccountName: "Revenue", Debit: 0, Credit: out.Subtotal},
	}
	out.Balanced = true
	out.Note = "Pre-tax preview. GST/tax is resolved per line at invoice time; nothing is persisted."
	return out, nil
}
