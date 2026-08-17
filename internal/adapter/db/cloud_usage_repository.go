package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// CloudUsageRepository measures and stores each tenant's per-period activity for
// Recurso Cloud self-billing. It is read-only against the tenants' own billing
// data (invoices, payment attempts) and writes only to cloud_tenant_usage — it
// never touches the money path.
type CloudUsageRepository struct {
	db *sql.DB
}

func NewCloudUsageRepository(database *sql.DB) *CloudUsageRepository {
	return &CloudUsageRepository{db: database}
}

// AggregateUsage computes, per (tenant, currency), the tracked revenue and
// collected volume for the [start, end) window across ALL tenants. Tracked
// revenue is every invoice raised in the window (paid or not — matching the
// published definition); collected volume is the amount of succeeded payment
// attempts, taking the currency from their invoice. Rows come back with only the
// measurement fields set; the caller stamps id/period/computed_at.
func (r *CloudUsageRepository) AggregateUsage(ctx context.Context, start, end time.Time) ([]*domain.CloudTenantUsage, error) {
	rows, err := r.db.QueryContext(ctx, `
		WITH inv AS (
			SELECT tenant_id, currency, SUM(total)::bigint AS tracked
			FROM invoices
			WHERE created_at >= $1 AND created_at < $2
			GROUP BY tenant_id, currency
		),
		col AS (
			SELECT pa.tenant_id, i.currency, SUM(pa.amount)::bigint AS collected
			FROM payment_attempts pa
			JOIN invoices i ON i.id = pa.invoice_id
			WHERE pa.status = 'succeeded' AND pa.created_at >= $1 AND pa.created_at < $2
			GROUP BY pa.tenant_id, i.currency
		)
		SELECT
			COALESCE(inv.tenant_id, col.tenant_id)  AS tenant_id,
			COALESCE(inv.currency,  col.currency)   AS currency,
			COALESCE(inv.tracked,   0)              AS tracked_revenue_minor,
			COALESCE(col.collected, 0)              AS collected_volume_minor
		FROM inv
		FULL OUTER JOIN col
			ON inv.tenant_id = col.tenant_id AND inv.currency = col.currency`,
		start, end,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []*domain.CloudTenantUsage
	for rows.Next() {
		u := &domain.CloudTenantUsage{}
		if err := rows.Scan(&u.TenantID, &u.Currency, &u.TrackedRevenueMinor, &u.CollectedVolumeMinor); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// Upsert writes each reading, replacing the current value for its
// (tenant, period_start, currency) — so re-measuring the current period as it
// accrues just refreshes the row.
func (r *CloudUsageRepository) Upsert(ctx context.Context, rows []*domain.CloudTenantUsage) error {
	for _, u := range rows {
		if _, err := r.db.ExecContext(ctx, `
			INSERT INTO cloud_tenant_usage
				(id, tenant_id, period_start, period_end, currency, tracked_revenue_minor, collected_volume_minor, computed_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (tenant_id, period_start, currency) DO UPDATE SET
				tracked_revenue_minor  = EXCLUDED.tracked_revenue_minor,
				collected_volume_minor = EXCLUDED.collected_volume_minor,
				period_end             = EXCLUDED.period_end,
				computed_at            = EXCLUDED.computed_at`,
			u.ID, u.TenantID, u.PeriodStart, u.PeriodEnd, u.Currency, u.TrackedRevenueMinor, u.CollectedVolumeMinor, u.ComputedAt,
		); err != nil {
			return err
		}
	}
	return nil
}
