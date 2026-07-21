package port

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ProgressiveBillingRepository backs interim (progressive) billing (A5). The
// watermark records how much has already been invoiced per (subscription,
// charge, period); the next bill is rate(cumulative_now) - billed_amount. The
// compare-and-swap advance is the idempotency mechanism that prevents a
// concurrent or retried sweep from billing the same delta twice.
type ProgressiveBillingRepository interface {
	// GetThreshold returns the subscription's progressive_billing_threshold in
	// minor units, or nil when progressive billing is off for it.
	GetThreshold(ctx context.Context, subscriptionID uuid.UUID) (*int64, error)

	// GetWatermark returns the amount already invoiced for (subscription,
	// charge, period); 0 when no watermark row exists yet.
	GetWatermark(ctx context.Context, subscriptionID, chargeID uuid.UUID, periodStart time.Time) (int64, error)

	// AdvanceWatermarkCAS atomically moves the watermark from oldAmount to
	// newAmount for (subscription, charge, period), creating the row when it is
	// absent and oldAmount is 0. It returns true IFF it advanced — false means a
	// concurrent or retried run already moved the watermark past oldAmount, so
	// THIS run must not bill the delta (the guarantee that no unit is billed
	// twice).
	AdvanceWatermarkCAS(ctx context.Context, tenantID, subscriptionID, chargeID uuid.UUID, periodStart time.Time, oldAmount, newAmount int64) (bool, error)
}
