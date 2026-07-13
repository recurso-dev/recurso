package db

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

type DunningRepository struct {
	db *sql.DB
}

func NewDunningRepository(db *sql.DB) *DunningRepository {
	return &DunningRepository{db: db}
}

func (r *DunningRepository) GetWeights(ctx context.Context, contextKey string) ([]domain.DunningWeight, error) {
	query := `
		SELECT context_key, action_id, average_reward, sample_count, updated_at
		FROM dunning_weights
		WHERE context_key = $1
	`
	rows, err := r.db.QueryContext(ctx, query, contextKey)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var weights []domain.DunningWeight
	for rows.Next() {
		var w domain.DunningWeight
		if err := rows.Scan(&w.ContextKey, &w.ActionID, &w.AverageReward, &w.SampleCount, &w.UpdatedAt); err != nil {
			return nil, err
		}
		weights = append(weights, w)
	}
	return weights, nil
}

// ApplyOutcome atomically folds one reward into the arm's running average. For
// a new arm the first observation seeds average=reward, sample_count=1; for an
// existing arm it computes the incremental average from the CURRENT row values
// (sample_count+1) in a single statement, so concurrent recordings can't lose an
// update the way a read-average-then-overwrite did.
func (r *DunningRepository) ApplyOutcome(ctx context.Context, contextKey, actionID string, reward float64) error {
	query := `
		INSERT INTO dunning_weights (context_key, action_id, average_reward, sample_count, updated_at)
		VALUES ($1, $2, $3, 1, NOW())
		ON CONFLICT (context_key, action_id) DO UPDATE SET
			sample_count = dunning_weights.sample_count + 1,
			average_reward = dunning_weights.average_reward
				+ ($3 - dunning_weights.average_reward) / (dunning_weights.sample_count + 1),
			updated_at = NOW()
	`
	_, err := r.db.ExecContext(ctx, query, contextKey, actionID, reward)
	return err
}

func (r *DunningRepository) RecordHistory(ctx context.Context, history domain.DunningHistory) error {
	query := `
		INSERT INTO dunning_history (id, tenant_id, invoice_id, context_key, action_id, retry_interval, outcome, reward, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.ExecContext(ctx, query,
		history.ID, history.TenantID, history.InvoiceID, history.ContextKey, history.ActionID, history.RetryInterval, history.Outcome, history.Reward, history.CreatedAt,
	)
	return err
}

// GetAllWeights returns all dunning weights across all contexts
func (r *DunningRepository) GetAllWeights(ctx context.Context) ([]domain.DunningWeight, error) {
	query := `
		SELECT context_key, action_id, average_reward, sample_count, updated_at
		FROM dunning_weights
		ORDER BY context_key, action_id
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var weights []domain.DunningWeight
	for rows.Next() {
		var w domain.DunningWeight
		if err := rows.Scan(&w.ContextKey, &w.ActionID, &w.AverageReward, &w.SampleCount, &w.UpdatedAt); err != nil {
			return nil, err
		}
		weights = append(weights, w)
	}
	return weights, nil
}

// GetRecentHistory returns the most recent dunning history entries for a tenant.
func (r *DunningRepository) GetRecentHistory(ctx context.Context, tenantID uuid.UUID, limit int) ([]domain.DunningHistory, error) {
	query := `
		SELECT id, tenant_id, invoice_id, context_key, action_id, retry_interval, outcome, reward, created_at
		FROM dunning_history
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := r.db.QueryContext(ctx, query, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var history []domain.DunningHistory
	for rows.Next() {
		var h domain.DunningHistory
		if err := rows.Scan(&h.ID, &h.TenantID, &h.InvoiceID, &h.ContextKey, &h.ActionID, &h.RetryInterval, &h.Outcome, &h.Reward, &h.CreatedAt); err != nil {
			return nil, err
		}
		history = append(history, h)
	}
	return history, nil
}

// GetHistoryStats returns aggregate dunning-history stats for a single tenant.
func (r *DunningRepository) GetHistoryStats(ctx context.Context, tenantID uuid.UUID) (totalRetries int, totalSuccesses int, err error) {
	query := `
		SELECT
			COUNT(*) as total_retries,
			COUNT(*) FILTER (WHERE outcome = 'success') as total_successes
		FROM dunning_history
		WHERE tenant_id = $1
	`
	err = r.db.QueryRowContext(ctx, query, tenantID).Scan(&totalRetries, &totalSuccesses)
	return
}
