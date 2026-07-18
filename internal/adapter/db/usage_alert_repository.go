package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// UsageAlertRepository is the Postgres implementation of
// port.UsageAlertRepository.
type UsageAlertRepository struct {
	db *sql.DB
}

func NewUsageAlertRepository(db *sql.DB) port.UsageAlertRepository {
	return &UsageAlertRepository{db: db}
}

const alertColumns = `id, tenant_id, subscription_id, metric_code, threshold_type, threshold, last_fired_period_start, created_at, updated_at`

func scanAlert(row interface{ Scan(...any) error }) (*domain.UsageAlert, error) {
	var a domain.UsageAlert
	if err := row.Scan(&a.ID, &a.TenantID, &a.SubscriptionID, &a.MetricCode, &a.ThresholdType,
		&a.Threshold, &a.LastFiredPeriodStart, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *UsageAlertRepository) Create(ctx context.Context, a *domain.UsageAlert) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO usage_alerts (id, tenant_id, subscription_id, metric_code, threshold_type, threshold, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		a.ID, a.TenantID, a.SubscriptionID, a.MetricCode, a.ThresholdType, a.Threshold, a.CreatedAt, a.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert usage alert: %w", err)
	}
	return nil
}

func (r *UsageAlertRepository) list(ctx context.Context, query string, args ...any) ([]domain.UsageAlert, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list usage alerts: %w", err)
	}
	defer func() { _ = rows.Close() }()
	alerts := []domain.UsageAlert{}
	for rows.Next() {
		a, err := scanAlert(rows)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, *a)
	}
	return alerts, rows.Err()
}

func (r *UsageAlertRepository) ListBySubscription(ctx context.Context, tenantID, subscriptionID uuid.UUID) ([]domain.UsageAlert, error) {
	return r.list(ctx,
		`SELECT `+alertColumns+` FROM usage_alerts WHERE tenant_id = $1 AND subscription_id = $2 ORDER BY metric_code, threshold`,
		tenantID, subscriptionID)
}

func (r *UsageAlertRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]domain.UsageAlert, error) {
	return r.list(ctx,
		`SELECT `+alertColumns+` FROM usage_alerts WHERE tenant_id = $1 ORDER BY subscription_id, metric_code, threshold`,
		tenantID)
}

func (r *UsageAlertRepository) ListAll(ctx context.Context, limit int) ([]domain.UsageAlert, error) {
	if limit <= 0 || limit > 5000 {
		limit = 1000
	}
	return r.list(ctx, `SELECT `+alertColumns+` FROM usage_alerts ORDER BY id LIMIT $1`, limit)
}

func (r *UsageAlertRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM usage_alerts WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return fmt.Errorf("failed to delete usage alert: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *UsageAlertRepository) MarkFired(ctx context.Context, id uuid.UUID, periodStart time.Time) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE usage_alerts SET last_fired_period_start = $2, updated_at = NOW()
		WHERE id = $1
		AND (last_fired_period_start IS NULL OR last_fired_period_start <> $2)`,
		id, periodStart)
	if err != nil {
		return false, fmt.Errorf("failed to mark usage alert fired: %w", err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}
