package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// CloudChargeRepository persists the Recurso Cloud dry-run charge previews.
// Preview data only — it never touches invoices or the ledger.
type CloudChargeRepository struct {
	db *sql.DB
}

func NewCloudChargeRepository(database *sql.DB) *CloudChargeRepository {
	return &CloudChargeRepository{db: database}
}

// UpsertPreview writes each tenant's dry-run charge, replacing the prior value
// for its (period_start, tenant_id) so a recompute just refreshes it.
func (r *CloudChargeRepository) UpsertPreview(ctx context.Context, rows []*domain.CloudChargePreview) error {
	for _, p := range rows {
		if _, err := r.db.ExecContext(ctx, `
			INSERT INTO cloud_charge_preview
				(id, period_start, period_end, tenant_id, currency,
				 tracked_revenue_minor, collected_volume_minor, would_charge_minor, reason, computed_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT (period_start, tenant_id) DO UPDATE SET
				period_end             = EXCLUDED.period_end,
				currency               = EXCLUDED.currency,
				tracked_revenue_minor  = EXCLUDED.tracked_revenue_minor,
				collected_volume_minor = EXCLUDED.collected_volume_minor,
				would_charge_minor     = EXCLUDED.would_charge_minor,
				reason                 = EXCLUDED.reason,
				computed_at            = EXCLUDED.computed_at`,
			p.ID, p.PeriodStart, p.PeriodEnd, p.TenantID, p.Currency,
			p.TrackedRevenueMinor, p.CollectedVolumeMinor, p.WouldChargeMinor, p.Reason, p.ComputedAt,
		); err != nil {
			return err
		}
	}
	return nil
}

// ListPreviewByPeriod returns all dry-run charges for a period (by period_start),
// newest computation reflected. Read-only.
func (r *CloudChargeRepository) ListPreviewByPeriod(ctx context.Context, periodStart time.Time) ([]*domain.CloudChargePreview, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, period_start, period_end, tenant_id, currency,
		       tracked_revenue_minor, collected_volume_minor, would_charge_minor, reason, computed_at
		FROM cloud_charge_preview
		WHERE period_start = $1
		ORDER BY would_charge_minor DESC`,
		periodStart,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []*domain.CloudChargePreview
	for rows.Next() {
		p := &domain.CloudChargePreview{}
		if err := rows.Scan(&p.ID, &p.PeriodStart, &p.PeriodEnd, &p.TenantID, &p.Currency,
			&p.TrackedRevenueMinor, &p.CollectedVolumeMinor, &p.WouldChargeMinor, &p.Reason, &p.ComputedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
