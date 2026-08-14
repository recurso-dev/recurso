package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// SubscriptionHistoryRepository reads the append-only subscription timeline
// captured by the subscription_history trigger (status + plan transitions).
// Read-only — writes happen in the database, never from Go.
type SubscriptionHistoryRepository struct {
	db *sql.DB
}

func NewSubscriptionHistoryRepository(db *sql.DB) *SubscriptionHistoryRepository {
	return &SubscriptionHistoryRepository{db: db}
}

// ListBySubscription returns a subscription's recorded changes oldest-first,
// tenant-scoped. from_value is null on the creation row.
func (r *SubscriptionHistoryRepository) ListBySubscription(ctx context.Context, tenantID, subscriptionID uuid.UUID) ([]domain.SubscriptionChange, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, subscription_id, change_type, from_value, to_value, changed_at
		 FROM subscription_history
		 WHERE subscription_id = $1 AND tenant_id = $2
		 ORDER BY changed_at, id`, subscriptionID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscription history: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := []domain.SubscriptionChange{}
	for rows.Next() {
		var c domain.SubscriptionChange
		var from, to sql.NullString
		if err := rows.Scan(&c.ID, &c.SubscriptionID, &c.ChangeType, &from, &to, &c.ChangedAt); err != nil {
			return nil, fmt.Errorf("failed to scan subscription change: %w", err)
		}
		if from.Valid {
			c.FromValue = &from.String
		}
		if to.Valid {
			c.ToValue = &to.String
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
