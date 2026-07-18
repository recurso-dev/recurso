package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

type UsageRepository struct {
	db *sql.DB
}

func NewUsageRepository(db *sql.DB) *UsageRepository {
	return &UsageRepository{db: db}
}

func (r *UsageRepository) RecordEvent(ctx context.Context, event *domain.UsageEvent) error {
	// properties is NULL (not '{}') for property-less events so the column
	// stays cheap for the overwhelmingly common bare event.
	var props any
	if len(event.Properties) > 0 {
		b, err := json.Marshal(event.Properties)
		if err != nil {
			return fmt.Errorf("failed to encode event properties: %w", err)
		}
		props = b
	}
	query := `
		INSERT INTO usage_events (id, subscription_id, customer_id, dimension, quantity, timestamp, properties)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query,
		event.ID, event.SubscriptionID, event.CustomerID, event.Dimension, event.Quantity, event.Timestamp, props,
	)
	if err != nil {
		return fmt.Errorf("failed to insert usage event: %w", err)
	}
	return nil
}

// AggregateForMetric reduces the subscription's events for the metric's
// dimension (== metric.Code) inside [start, end) to one quantity:
//
//	count:  number of events
//	sum:    Σ quantity
//	max:    largest single-event quantity
//	unique: COUNT(DISTINCT properties->>field_name), NULL-property events excluded
//
// The aggregation SQL is selected from a fixed set (never interpolated from
// caller input); metric.AggregationType must be pre-validated by the service.
func (r *UsageRepository) AggregateForMetric(ctx context.Context, subscriptionID uuid.UUID, metric domain.BillableMetric, start, end time.Time) (int64, error) {
	var agg string
	args := []interface{}{subscriptionID, metric.Code, start, end}
	switch metric.AggregationType {
	case domain.AggregationCount:
		agg = `COUNT(*)`
	case domain.AggregationSum:
		agg = `COALESCE(SUM(quantity), 0)`
	case domain.AggregationMax:
		agg = `COALESCE(MAX(quantity), 0)`
	case domain.AggregationUnique:
		agg = `COUNT(DISTINCT properties->>$5)`
		args = append(args, metric.FieldName)
	default:
		return 0, fmt.Errorf("unsupported aggregation type %q", metric.AggregationType)
	}

	query := `
		SELECT ` + agg + `
		FROM usage_events
		WHERE subscription_id = $1
		AND dimension = $2
		AND timestamp >= $3
		AND timestamp < $4
	`
	var total int64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("failed to aggregate usage for metric %q: %w", metric.Code, err)
	}
	return total, nil
}

// GetUsageForPeriod aggregates usage (SUM) for billing.
func (r *UsageRepository) GetUsageForPeriod(ctx context.Context, subID string, dimension string, start, end time.Time) (int64, error) {
	query := `
		SELECT COALESCE(SUM(quantity), 0) 
		FROM usage_events 
		WHERE subscription_id = $1 
		AND dimension = $2
		AND timestamp >= $3
		AND timestamp < $4
	`
	var total int64
	err := r.db.QueryRowContext(ctx, query, subID, dimension, start, end).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to aggregate usage: %w", err)
	}
	return total, nil
}

// QueryUsage aggregates usage into date_trunc'd buckets.
//
// Tenant scoping: usage_events has no tenant_id column, so isolation is
// enforced by joining subscriptions and filtering on subscriptions.tenant_id.
// Granularity is passed as a bind parameter to date_trunc and must be
// pre-validated by the service layer ("day" | "month").
func (r *UsageRepository) QueryUsage(ctx context.Context, tenantID uuid.UUID, filter domain.UsageQueryFilter) ([]domain.UsageBucket, error) {
	query := `
		SELECT date_trunc($1, ue.timestamp) AS period, ue.dimension, COALESCE(SUM(ue.quantity), 0) AS quantity
		FROM usage_events ue
		JOIN subscriptions s ON s.id = ue.subscription_id
		WHERE s.tenant_id = $2
		AND ue.timestamp >= $3
		AND ue.timestamp < $4
	`
	args := []interface{}{filter.Granularity, tenantID, filter.From, filter.To}

	if filter.SubscriptionID != nil {
		args = append(args, *filter.SubscriptionID)
		query += fmt.Sprintf(" AND ue.subscription_id = $%d", len(args))
	}
	if filter.CustomerID != nil {
		args = append(args, *filter.CustomerID)
		query += fmt.Sprintf(" AND ue.customer_id = $%d", len(args))
	}
	if filter.Dimension != "" {
		args = append(args, filter.Dimension)
		query += fmt.Sprintf(" AND ue.dimension = $%d", len(args))
	}
	query += " GROUP BY 1, 2 ORDER BY 1, 2"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query usage buckets: %w", err)
	}
	defer func() { _ = rows.Close() }()

	buckets := []domain.UsageBucket{}
	for rows.Next() {
		var b domain.UsageBucket
		if err := rows.Scan(&b.Period, &b.Dimension, &b.Quantity); err != nil {
			return nil, err
		}
		buckets = append(buckets, b)
	}
	return buckets, rows.Err()
}

// GetSubscriptionUsageByDimension returns, per dimension, the quantity
// inside [periodStart, periodEnd) plus the lifetime total, in one
// set-based pass (FILTER clause). Tenant-scoped via the subscriptions join
// (usage_events has no tenant_id).
func (r *UsageRepository) GetSubscriptionUsageByDimension(ctx context.Context, tenantID, subscriptionID uuid.UUID, periodStart, periodEnd time.Time) ([]domain.SubscriptionDimensionUsage, error) {
	query := `
		SELECT ue.dimension,
			COALESCE(SUM(ue.quantity) FILTER (WHERE ue.timestamp >= $3 AND ue.timestamp < $4), 0) AS period_quantity,
			COALESCE(SUM(ue.quantity), 0) AS lifetime_quantity
		FROM usage_events ue
		JOIN subscriptions s ON s.id = ue.subscription_id
		WHERE s.tenant_id = $1
		AND ue.subscription_id = $2
		GROUP BY ue.dimension
		ORDER BY ue.dimension
	`
	rows, err := r.db.QueryContext(ctx, query, tenantID, subscriptionID, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to query subscription usage: %w", err)
	}
	defer func() { _ = rows.Close() }()

	usages := []domain.SubscriptionDimensionUsage{}
	for rows.Next() {
		var u domain.SubscriptionDimensionUsage
		if err := rows.Scan(&u.Dimension, &u.PeriodQuantity, &u.LifetimeQuantity); err != nil {
			return nil, err
		}
		usages = append(usages, u)
	}
	return usages, rows.Err()
}

// ListDimensions returns the tenant's distinct dimensions with event
// counts and first/last seen. Tenant-scoped via the subscriptions join.
func (r *UsageRepository) ListDimensions(ctx context.Context, tenantID uuid.UUID) ([]domain.UsageDimension, error) {
	query := `
		SELECT ue.dimension, COUNT(*) AS event_count, MIN(ue.timestamp) AS first_seen, MAX(ue.timestamp) AS last_seen
		FROM usage_events ue
		JOIN subscriptions s ON s.id = ue.subscription_id
		WHERE s.tenant_id = $1
		GROUP BY ue.dimension
		ORDER BY ue.dimension
	`
	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list usage dimensions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	dims := []domain.UsageDimension{}
	for rows.Next() {
		var d domain.UsageDimension
		if err := rows.Scan(&d.Dimension, &d.EventCount, &d.FirstSeen, &d.LastSeen); err != nil {
			return nil, err
		}
		dims = append(dims, d)
	}
	return dims, rows.Err()
}

func (r *UsageRepository) GetUsageStats(ctx context.Context, tenantID uuid.UUID) ([]*domain.UsageStats, error) {
	query := `
		SELECT ue.dimension, COALESCE(SUM(ue.quantity), 0) as total_quantity
		FROM usage_events ue
		JOIN subscriptions s ON ue.subscription_id = s.id
		WHERE s.tenant_id = $1
		GROUP BY ue.dimension
	`
	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query usage stats: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var stats []*domain.UsageStats
	for rows.Next() {
		var s domain.UsageStats
		if err := rows.Scan(&s.Dimension, &s.TotalQuantity); err != nil {
			return nil, err
		}
		stats = append(stats, &s)
	}
	return stats, nil
}
