package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// ProgressiveBillingRepository is the Postgres implementation of
// port.ProgressiveBillingRepository (A5).
type ProgressiveBillingRepository struct {
	db *sql.DB
}

func NewProgressiveBillingRepository(db *sql.DB) *ProgressiveBillingRepository {
	return &ProgressiveBillingRepository{db: db}
}

var _ port.ProgressiveBillingRepository = (*ProgressiveBillingRepository)(nil)

func (r *ProgressiveBillingRepository) GetThreshold(ctx context.Context, subscriptionID uuid.UUID) (*int64, error) {
	var threshold sql.NullInt64
	err := r.db.QueryRowContext(ctx,
		`SELECT progressive_billing_threshold FROM subscriptions WHERE id = $1`,
		subscriptionID,
	).Scan(&threshold)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get progressive threshold: %w", err)
	}
	if !threshold.Valid {
		return nil, nil
	}
	v := threshold.Int64
	return &v, nil
}

// ListActiveProgressiveSubscriptionIDs returns active subscriptions with a
// progressive_billing_threshold set — the sweep's candidate set. The threshold
// gate and watermark CAS inside billing decide whether each actually bills, so
// this query only needs to narrow the scan, not be exact.
func (r *ProgressiveBillingRepository) ListActiveProgressiveSubscriptionIDs(ctx context.Context) ([]uuid.UUID, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id FROM subscriptions
		 WHERE progressive_billing_threshold IS NOT NULL AND status = 'active'
		 ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list progressive subscriptions: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan progressive subscription id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *ProgressiveBillingRepository) GetWatermark(ctx context.Context, subscriptionID, chargeID uuid.UUID, periodStart time.Time) (int64, error) {
	var billed int64
	err := r.db.QueryRowContext(ctx,
		`SELECT billed_amount FROM progressive_billing_watermarks
		 WHERE subscription_id = $1 AND charge_id = $2 AND period_start = $3`,
		subscriptionID, chargeID, periodStart,
	).Scan(&billed)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get watermark: %w", err)
	}
	return billed, nil
}

// AdvanceWatermarkCAS advances the watermark only when the stored billed_amount
// still equals oldAmount (compare-and-swap), creating the row when it is absent
// and oldAmount is 0. Rows-affected == 1 means this run won the advance and must
// bill the delta; 0 means a concurrent/retried run already advanced past
// oldAmount, so this run must NOT bill it.
func (r *ProgressiveBillingRepository) AdvanceWatermarkCAS(ctx context.Context, tenantID, subscriptionID, chargeID uuid.UUID, periodStart time.Time, oldAmount, newAmount int64) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO progressive_billing_watermarks
			(id, tenant_id, subscription_id, charge_id, period_start, billed_amount, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (subscription_id, charge_id, period_start)
		DO UPDATE SET billed_amount = $6, updated_at = NOW()
		WHERE progressive_billing_watermarks.billed_amount = $7`,
		uuid.New(), tenantID, subscriptionID, chargeID, periodStart, newAmount, oldAmount,
	)
	if err != nil {
		return false, fmt.Errorf("failed to advance watermark: %w", err)
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}
