package service

import (
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
// quantity so far and the amount already billed this period (the watermark).
// Returns the non-negative delta to bill now and the new watermark
// (= max(fee-at-cumulative, prior watermark)). Pure — the safety proof lives in
// TestProgressiveDelta_NeverDoubleOrUnderBills.
func progressiveDelta(model domain.ChargeModel, amounts domain.ChargeAmounts, cumulativeQty, billedAmount int64) (delta int64, newWatermark int64, err error) {
	feeNow, err := RateCharge(model, amounts, cumulativeQty)
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
