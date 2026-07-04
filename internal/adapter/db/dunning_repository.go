package db

import (
	"context"
	"database/sql"

	"github.com/swapnull-in/recur-so/internal/core/domain"
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

func (r *DunningRepository) UpdateWeight(ctx context.Context, weight domain.DunningWeight) error {
	query := `
		INSERT INTO dunning_weights (context_key, action_id, average_reward, sample_count, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (context_key, action_id) DO UPDATE SET
			average_reward = EXCLUDED.average_reward,
			sample_count = EXCLUDED.sample_count,
			updated_at = EXCLUDED.updated_at
	`
	_, err := r.db.ExecContext(ctx, query,
		weight.ContextKey, weight.ActionID, weight.AverageReward, weight.SampleCount, weight.UpdatedAt,
	)
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

// GetRecentHistory returns the most recent dunning history entries
func (r *DunningRepository) GetRecentHistory(ctx context.Context, limit int) ([]domain.DunningHistory, error) {
	query := `
		SELECT id, tenant_id, invoice_id, context_key, action_id, retry_interval, outcome, reward, created_at
		FROM dunning_history
		ORDER BY created_at DESC
		LIMIT $1
	`
	rows, err := r.db.QueryContext(ctx, query, limit)
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

// GetHistoryStats returns aggregate stats from dunning history
func (r *DunningRepository) GetHistoryStats(ctx context.Context) (totalRetries int, totalSuccesses int, err error) {
	query := `
		SELECT
			COUNT(*) as total_retries,
			COUNT(*) FILTER (WHERE outcome = 'success') as total_successes
		FROM dunning_history
	`
	err = r.db.QueryRowContext(ctx, query).Scan(&totalRetries, &totalSuccesses)
	return
}
