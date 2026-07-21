package service

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// Progressive billing (A5): a subscription with a threshold bills its metered
// charges incrementally — whenever accrued usage reaches the threshold, an
// interim invoice bills the delta since the last bill. Correctness rests on a
// per-(subscription, charge, period) billed-amount WATERMARK: each bill is
// exactly rate(cumulative_now) - billed_amount, and the watermark advances to
// rate(cumulative_now). Because the charge fee is monotonic non-decreasing in
// the cumulative quantity, every unit is billed exactly once at its marginal
// rate — no double-bill, no under-bill — regardless of how many interim points
// there are or which charge model is used. The final period-close bill is just
// the last delta (rate(final) - watermark), so Σ(deltas) == rate(final), the
// same total classic arrears would produce.

// progressiveDelta computes what to bill for one charge given the cumulative
// quantity so far (an exact rational, so a fractional aggregation like custom or
// weighted_sum is priced without pre-rounding) and the amount already billed
// this period (the watermark, in minor units). Returns the non-negative delta to
// bill now and the new watermark (= max(fee-at-cumulative, prior watermark)).
// Pure — the safety proof lives in TestProgressiveDelta_NeverDoubleOrUnderBills.
func progressiveDelta(model domain.ChargeModel, amounts domain.ChargeAmounts, cumulativeQty *big.Rat, billedAmount int64) (delta int64, newWatermark int64, err error) {
	feeNow, err := RateChargeRat(model, amounts, cumulativeQty)
	if err != nil {
		return 0, billedAmount, err
	}
	if feeNow <= billedAmount {
		// Fee never decreases as usage grows; if it hasn't advanced past what's
		// already billed, bill nothing and hold the watermark (never rewind it).
		return 0, billedAmount, nil
	}
	return feeNow - billedAmount, feeNow, nil
}

// isProgressive reports whether the subscription bills progressively (has a
// threshold set). Nil-safe: false when the repo is unwired.
func (s *InvoiceService) isProgressive(ctx context.Context, subID uuid.UUID) bool {
	if s.ProgressiveRepo == nil {
		return false
	}
	th, err := s.ProgressiveRepo.GetThreshold(ctx, subID)
	if err != nil {
		slog.Warn("progressive: threshold lookup failed", "subscription_id", subID, "error", err)
		return false
	}
	return th != nil
}

// claimProgressiveDelta computes rate(cumulative to upTo) - watermark for one
// eligible charge and ATOMICALLY advances the watermark via compare-and-swap.
// It returns (delta, true) only when this run WON the CAS and must bill the
// delta. Because the CAS runs before any invoice is created, a later
// invoice-creation failure under-bills (recoverable) rather than double-bills —
// the guarantee that no usage unit is billed twice. Shared by the period-close
// path (progressiveCloseLine) and the interim path (billProgressive), so the
// watermark is advanced in exactly one place.
func (s *InvoiceService) claimProgressiveDelta(ctx context.Context, sub *domain.Subscription, ch domain.Charge, amounts domain.ChargeAmounts, periodStart, upTo time.Time) (int64, bool) {
	cumQty, err := meteredQuantity(ctx, s.UsageRepo, sub.ID, ch, periodStart, upTo)
	if err != nil {
		slog.Warn("progressive: aggregation failed", "charge_id", ch.ID, "error", err)
		return 0, false
	}
	oldWM, err := s.ProgressiveRepo.GetWatermark(ctx, sub.ID, ch.ID, periodStart)
	if err != nil {
		slog.Warn("progressive: watermark read failed", "charge_id", ch.ID, "error", err)
		return 0, false
	}
	delta, newWM, err := progressiveDelta(ch.ChargeModel, amounts, cumQty, oldWM)
	if err != nil {
		slog.Warn("progressive: rating failed", "charge_id", ch.ID, "error", err)
		return 0, false
	}
	if delta == 0 {
		return 0, false
	}
	won, err := s.ProgressiveRepo.AdvanceWatermarkCAS(ctx, sub.TenantID, sub.ID, ch.ID, periodStart, oldWM, newWM)
	if err != nil {
		slog.Warn("progressive: watermark CAS failed", "charge_id", ch.ID, "error", err)
		return 0, false
	}
	return delta, won
}

// progressiveCloseLine settles one eligible charge's remaining delta at period
// close as a line on the renewal invoice: rate(usage over the whole period)
// minus what interims already billed (the watermark). Returns (line, true) when
// a positive delta was claimed. The rating claim is nil — the watermark, not
// usage_ratings, is this charge's double-billing guard.
func (s *InvoiceService) progressiveCloseLine(ctx context.Context, sub *domain.Subscription, customer *domain.Customer, plan *domain.Plan, currency, cur string, invID uuid.UUID, ch domain.Charge, periodStart, periodEnd, now time.Time) (meteredLine, bool) {
	amounts, ok := ch.Amounts[cur]
	if !ok {
		return meteredLine{}, false
	}
	delta, ok := s.claimProgressiveDelta(ctx, sub, ch, amounts, periodStart, periodEnd)
	if !ok {
		return meteredLine{}, false
	}
	hsn := ch.HSNCode
	if hsn == "" {
		hsn = plan.HSNCode
	}
	tax := s.TaxResolver.ResolveInvoiceTax(ctx, sub.TenantID, customer, currency, delta, hsn)
	desc := fmt.Sprintf("%s — usage (progressive, to %s)", ch.Metric.Name, periodEnd.Format("2 Jan 2006"))
	return meteredLine{
		item: newInvoiceLine(invID, desc, tax.HSN, 1, delta, delta, tax, time.Time{}),
		tax:  tax,
	}, true
}

// GenerateProgressiveInvoice creates an interim invoice for a progressive
// subscription when its accrued unbilled usage has reached the threshold.
// Returns (nil, nil) when the subscription is not progressive or the threshold
// is not yet met. Read-only threshold gate; the actual billing (with the CAS)
// happens in billProgressive.
func (s *InvoiceService) GenerateProgressiveInvoice(ctx context.Context, sub *domain.Subscription) (*domain.Invoice, error) {
	if s.ProgressiveRepo == nil {
		return nil, nil
	}
	th, err := s.ProgressiveRepo.GetThreshold(ctx, sub.ID)
	if err != nil || th == nil {
		return nil, err
	}
	upTo := time.Now()
	unbilled, err := s.progressiveUnbilled(ctx, sub, upTo)
	if err != nil {
		return nil, err
	}
	if unbilled < *th {
		return nil, nil // threshold not yet crossed
	}
	return s.billProgressive(ctx, sub, upTo)
}

// GenerateProgressiveInvoiceForSub loads the subscription and, when it bills
// progressively and its accrued usage has reached the threshold, generates the
// interim invoice. Returns (nil, nil) when nothing is due. The handler for
// POST /v1/subscriptions/:id/bill-usage.
func (s *InvoiceService) GenerateProgressiveInvoiceForSub(ctx context.Context, subID uuid.UUID) (*domain.Invoice, error) {
	sub, err := s.SubscriptionRepo.GetByID(ctx, subID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, fmt.Errorf("subscription not found")
	}
	return s.GenerateProgressiveInvoice(ctx, sub)
}

// progressiveUnbilled sums rate(cumulative to upTo) - watermark across the
// subscription's eligible charges WITHOUT advancing anything — the read-only
// gate for the interim threshold.
func (s *InvoiceService) progressiveUnbilled(ctx context.Context, sub *domain.Subscription, upTo time.Time) (int64, error) {
	plan, err := s.PlanRepo.GetByID(ctx, sub.PlanID)
	if err != nil || plan == nil || len(plan.Prices) == 0 {
		return 0, err
	}
	cur := strings.ToUpper(strings.TrimSpace(plan.Prices[0].Currency))
	charges, err := s.ChargeRepo.ListByPlan(ctx, sub.TenantID, sub.PlanID)
	if err != nil {
		return 0, err
	}
	periodStart := sub.CurrentPeriodStart
	var total int64
	for _, ch := range charges {
		if ch.Metric == nil || !domain.ProgressiveBillingEligible(ch.ChargeModel) {
			continue
		}
		amounts, ok := ch.Amounts[cur]
		if !ok {
			continue
		}
		cumQty, err := meteredQuantity(ctx, s.UsageRepo, sub.ID, ch, periodStart, upTo)
		if err != nil {
			return 0, err
		}
		oldWM, err := s.ProgressiveRepo.GetWatermark(ctx, sub.ID, ch.ID, periodStart)
		if err != nil {
			return 0, err
		}
		delta, _, err := progressiveDelta(ch.ChargeModel, amounts, cumQty, oldWM)
		if err != nil {
			return 0, err
		}
		total += delta
	}
	return total, nil
}

// billProgressive bills every eligible charge's outstanding delta up to `upTo`
// on ONE interim invoice, owning the full transaction (CAS -> create invoice ->
// post ledger) so no caller can drop the ledger legs. Returns (nil, nil) when
// nothing new is due. The CAS runs before invoice creation (never double-bill).
func (s *InvoiceService) billProgressive(ctx context.Context, sub *domain.Subscription, upTo time.Time) (*domain.Invoice, error) {
	plan, err := s.PlanRepo.GetByID(ctx, sub.PlanID)
	if err != nil {
		return nil, fmt.Errorf("progressive: get plan: %w", err)
	}
	if plan == nil || len(plan.Prices) == 0 {
		return nil, nil
	}
	customer, err := s.CustomerRepo.GetByID(ctx, sub.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("progressive: get customer: %w", err)
	}
	currency := plan.Prices[0].Currency
	cur := strings.ToUpper(strings.TrimSpace(currency))
	charges, err := s.ChargeRepo.ListByPlan(ctx, sub.TenantID, sub.PlanID)
	if err != nil {
		return nil, fmt.Errorf("progressive: list charges: %w", err)
	}

	periodStart := sub.CurrentPeriodStart
	invID := uuid.New()
	now := time.Now()
	var lines []domain.InvoiceItem
	var subtotal, taxTotal, igst, cgst, sgst int64
	var invTaxType string // D3c: resolved tax type for the liability report
	for _, ch := range charges {
		if ch.Metric == nil || !domain.ProgressiveBillingEligible(ch.ChargeModel) {
			continue
		}
		amounts, ok := ch.Amounts[cur]
		if !ok {
			continue
		}
		delta, ok := s.claimProgressiveDelta(ctx, sub, ch, amounts, periodStart, upTo)
		if !ok {
			continue
		}
		hsn := ch.HSNCode
		if hsn == "" {
			hsn = plan.HSNCode
		}
		tax := s.TaxResolver.ResolveInvoiceTax(ctx, sub.TenantID, customer, currency, delta, hsn)
		if invTaxType == "" && tax.TaxType != "" {
			invTaxType = tax.TaxType
		}
		desc := fmt.Sprintf("%s — usage (progressive, to %s)", ch.Metric.Name, upTo.Format("2 Jan 2006"))
		lines = append(lines, newInvoiceLine(invID, desc, tax.HSN, 1, delta, delta, tax, time.Time{}))
		subtotal += delta
		taxTotal += tax.Total
		igst += tax.IGST
		cgst += tax.CGST
		sgst += tax.SGST
	}
	if len(lines) == 0 {
		return nil, nil
	}

	inv := &domain.Invoice{
		ID:             invID,
		TenantID:       sub.TenantID,
		SubscriptionID: &sub.ID,
		CustomerID:     sub.CustomerID,
		InvoiceNumber:  fmt.Sprintf("INV-%d-%s", now.UnixNano(), invID.String()[:8]),
		BillingReason:  domain.BillingReasonProgressiveUsage,
		Status:         domain.InvoiceStatusOpen,
		Currency:       currency,
		Subtotal:       subtotal,
		TaxAmount:      taxTotal,
		TaxType:        invTaxType,
		Total:          subtotal + taxTotal,
		IGSTAmount:     igst,
		CGSTAmount:     cgst,
		SGSTAmount:     sgst,
		LineItems:      lines,
		CreatedAt:      now,
		DueDate:        domain.CalculateDueDate(now, "net0"),
		PaymentTerms:   "net0",
	}
	if err := s.InvoiceRepo.Create(ctx, inv); err != nil {
		// The watermark advanced but no invoice was created — this under-bills
		// (recoverable by a later sweep/reconciliation), never double-bills.
		return nil, fmt.Errorf("progressive: create interim invoice: %w", err)
	}
	// Post the interim invoice's ledger legs (DR AR / CR Revenue) ourselves so
	// no caller can drop them. Best-effort with reconciliation, like every other
	// ledger dual-write.
	if s.LedgerPoster != nil {
		if err := s.LedgerPoster.RecordInvoice(ctx, inv); err != nil {
			slog.Error("progressive: ledger post failed — reconciliation needed", "invoice_id", inv.ID, "error", err)
		}
	}
	return inv, nil
}
