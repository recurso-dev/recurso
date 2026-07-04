package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

type UsageRepository struct {
	db *sql.DB
}

func NewUsageRepository(db *sql.DB) *UsageRepository {
	return &UsageRepository{db: db}
}

func (r *UsageRepository) RecordEvent(ctx context.Context, event *domain.UsageEvent) error {
	query := `
		INSERT INTO usage_events (id, subscription_id, customer_id, dimension, quantity, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, query,
		event.ID, event.SubscriptionID, event.CustomerID, event.Dimension, event.Quantity, event.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to insert usage event: %w", err)
	}
	return nil
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
