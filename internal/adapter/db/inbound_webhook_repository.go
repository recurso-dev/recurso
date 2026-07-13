package db

import (
	"context"
	"database/sql"
	"fmt"
)

// InboundWebhookRepository records which gateway webhook events have already
// been fully processed, so redeliveries can be acknowledged without re-running
// non-idempotent side effects (ENG-162).
type InboundWebhookRepository struct {
	db *sql.DB
}

func NewInboundWebhookRepository(db *sql.DB) *InboundWebhookRepository {
	return &InboundWebhookRepository{db: db}
}

// WasProcessed reports whether (gateway, eventID) has already been recorded as
// processed.
func (r *InboundWebhookRepository) WasProcessed(ctx context.Context, gateway, eventID string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM inbound_webhook_events WHERE gateway = $1 AND event_id = $2)`,
		gateway, eventID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check processed webhook event: %w", err)
	}
	return exists, nil
}

// MarkProcessed records (gateway, eventID) as processed. Idempotent: a
// concurrent duplicate that races past WasProcessed simply conflicts here and is
// a no-op.
func (r *InboundWebhookRepository) MarkProcessed(ctx context.Context, gateway, eventID, eventType string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO inbound_webhook_events (gateway, event_id, event_type)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (gateway, event_id) DO NOTHING`,
		gateway, eventID, eventType)
	if err != nil {
		return fmt.Errorf("record processed webhook event: %w", err)
	}
	return nil
}
