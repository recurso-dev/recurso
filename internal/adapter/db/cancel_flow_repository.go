package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
	"github.com/recur-so/recurso/internal/core/port"
)

type CancelFlowRepository struct {
	db *sql.DB
}

func NewCancelFlowRepository(db *sql.DB) port.CancelFlowRepository {
	return &CancelFlowRepository{db: db}
}

// --- Flow CRUD ---

func (r *CancelFlowRepository) CreateFlow(ctx context.Context, flow *domain.CancelFlow) error {
	query := `
		INSERT INTO cancel_flows (id, tenant_id, name, is_active, is_default, cooldown_days, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.ExecContext(ctx, query,
		flow.ID, flow.TenantID, flow.Name, flow.IsActive, flow.IsDefault,
		flow.CooldownDays, flow.CreatedAt, flow.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create cancel flow: %w", err)
	}
	return nil
}

func (r *CancelFlowRepository) GetFlowByID(ctx context.Context, id uuid.UUID) (*domain.CancelFlow, error) {
	query := `
		SELECT id, tenant_id, name, is_active, is_default, cooldown_days, created_at, updated_at
		FROM cancel_flows WHERE id = $1
	`
	flow := &domain.CancelFlow{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&flow.ID, &flow.TenantID, &flow.Name, &flow.IsActive, &flow.IsDefault,
		&flow.CooldownDays, &flow.CreatedAt, &flow.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get cancel flow: %w", err)
	}

	steps, err := r.GetStepsByFlow(ctx, flow.ID)
	if err != nil {
		return nil, err
	}
	flow.Steps = steps

	return flow, nil
}

func (r *CancelFlowRepository) GetDefaultFlowForTenant(ctx context.Context, tenantID uuid.UUID) (*domain.CancelFlow, error) {
	query := `
		SELECT id, tenant_id, name, is_active, is_default, cooldown_days, created_at, updated_at
		FROM cancel_flows WHERE tenant_id = $1 AND is_default = TRUE AND is_active = TRUE
		LIMIT 1
	`
	flow := &domain.CancelFlow{}
	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(
		&flow.ID, &flow.TenantID, &flow.Name, &flow.IsActive, &flow.IsDefault,
		&flow.CooldownDays, &flow.CreatedAt, &flow.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get default cancel flow: %w", err)
	}

	steps, err := r.GetStepsByFlow(ctx, flow.ID)
	if err != nil {
		return nil, err
	}
	flow.Steps = steps

	return flow, nil
}

func (r *CancelFlowRepository) ListFlowsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.CancelFlow, error) {
	query := `
		SELECT id, tenant_id, name, is_active, is_default, cooldown_days, created_at, updated_at
		FROM cancel_flows WHERE tenant_id = $1 ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list cancel flows: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var flows []*domain.CancelFlow
	for rows.Next() {
		flow := &domain.CancelFlow{}
		if err := rows.Scan(
			&flow.ID, &flow.TenantID, &flow.Name, &flow.IsActive, &flow.IsDefault,
			&flow.CooldownDays, &flow.CreatedAt, &flow.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan cancel flow: %w", err)
		}
		flows = append(flows, flow)
	}
	return flows, rows.Err()
}

func (r *CancelFlowRepository) UpdateFlow(ctx context.Context, flow *domain.CancelFlow) error {
	query := `
		UPDATE cancel_flows SET name = $1, is_active = $2, is_default = $3,
		cooldown_days = $4, updated_at = $5 WHERE id = $6
	`
	_, err := r.db.ExecContext(ctx, query,
		flow.Name, flow.IsActive, flow.IsDefault, flow.CooldownDays, flow.UpdatedAt, flow.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update cancel flow: %w", err)
	}
	return nil
}

// --- Step CRUD ---

func (r *CancelFlowRepository) CreateStep(ctx context.Context, step *domain.CancelFlowStep) error {
	query := `
		INSERT INTO cancel_flow_steps (id, flow_id, step_order, step_type, config, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, query,
		step.ID, step.FlowID, step.StepOrder, step.StepType, step.Config, step.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create cancel flow step: %w", err)
	}
	return nil
}

func (r *CancelFlowRepository) GetStepsByFlow(ctx context.Context, flowID uuid.UUID) ([]domain.CancelFlowStep, error) {
	query := `
		SELECT id, flow_id, step_order, step_type, config, created_at
		FROM cancel_flow_steps WHERE flow_id = $1 ORDER BY step_order ASC
	`
	rows, err := r.db.QueryContext(ctx, query, flowID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cancel flow steps: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var steps []domain.CancelFlowStep
	for rows.Next() {
		var step domain.CancelFlowStep
		if err := rows.Scan(
			&step.ID, &step.FlowID, &step.StepOrder, &step.StepType, &step.Config, &step.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan cancel flow step: %w", err)
		}
		steps = append(steps, step)
	}
	return steps, rows.Err()
}

func (r *CancelFlowRepository) UpdateStep(ctx context.Context, step *domain.CancelFlowStep) error {
	query := `
		UPDATE cancel_flow_steps SET step_order = $1, step_type = $2, config = $3 WHERE id = $4
	`
	_, err := r.db.ExecContext(ctx, query, step.StepOrder, step.StepType, step.Config, step.ID)
	if err != nil {
		return fmt.Errorf("failed to update cancel flow step: %w", err)
	}
	return nil
}

func (r *CancelFlowRepository) DeleteStep(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM cancel_flow_steps WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete cancel flow step: %w", err)
	}
	return nil
}

// --- Session CRUD ---

func (r *CancelFlowRepository) CreateSession(ctx context.Context, session *domain.CancelFlowSession) error {
	query := `
		INSERT INTO cancel_flow_sessions (
			id, tenant_id, customer_id, subscription_id, flow_id, status,
			current_step_index, cancellation_reason, feedback,
			offer_presented, offer_accepted, saved_by_offer, started_at, completed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	_, err := r.db.ExecContext(ctx, query,
		session.ID, session.TenantID, session.CustomerID, session.SubscriptionID,
		session.FlowID, session.Status, session.CurrentStepIndex,
		nilIfEmpty(session.CancellationReason), nilIfEmpty(session.Feedback),
		session.OfferPresented, session.OfferAccepted, session.SavedByOffer,
		session.StartedAt, session.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create cancel flow session: %w", err)
	}
	return nil
}

func (r *CancelFlowRepository) GetSessionByID(ctx context.Context, id uuid.UUID) (*domain.CancelFlowSession, error) {
	query := `
		SELECT id, tenant_id, customer_id, subscription_id, flow_id, status,
			current_step_index, cancellation_reason, feedback,
			offer_presented, offer_accepted, saved_by_offer, started_at, completed_at
		FROM cancel_flow_sessions WHERE id = $1
	`
	session := &domain.CancelFlowSession{}
	var reason, feedback sql.NullString
	var offerPresented []byte
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&session.ID, &session.TenantID, &session.CustomerID, &session.SubscriptionID,
		&session.FlowID, &session.Status, &session.CurrentStepIndex,
		&reason, &feedback, &offerPresented,
		&session.OfferAccepted, &session.SavedByOffer, &session.StartedAt, &session.CompletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get cancel flow session: %w", err)
	}
	if reason.Valid {
		session.CancellationReason = reason.String
	}
	if feedback.Valid {
		session.Feedback = feedback.String
	}
	if offerPresented != nil {
		session.OfferPresented = offerPresented
	}
	return session, nil
}

func (r *CancelFlowRepository) UpdateSession(ctx context.Context, session *domain.CancelFlowSession) error {
	query := `
		UPDATE cancel_flow_sessions SET
			status = $1, current_step_index = $2, cancellation_reason = $3,
			feedback = $4, offer_presented = $5, offer_accepted = $6,
			saved_by_offer = $7, completed_at = $8
		WHERE id = $9
	`
	_, err := r.db.ExecContext(ctx, query,
		session.Status, session.CurrentStepIndex,
		nilIfEmpty(session.CancellationReason), nilIfEmpty(session.Feedback),
		session.OfferPresented, session.OfferAccepted,
		session.SavedByOffer, session.CompletedAt, session.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update cancel flow session: %w", err)
	}
	return nil
}

func (r *CancelFlowRepository) GetRecentSessionByCustomer(ctx context.Context, customerID uuid.UUID) (*domain.CancelFlowSession, error) {
	query := `
		SELECT id, tenant_id, customer_id, subscription_id, flow_id, status,
			current_step_index, cancellation_reason, feedback,
			offer_presented, offer_accepted, saved_by_offer, started_at, completed_at
		FROM cancel_flow_sessions
		WHERE customer_id = $1 AND saved_by_offer = TRUE
		ORDER BY started_at DESC LIMIT 1
	`
	session := &domain.CancelFlowSession{}
	var reason, feedback sql.NullString
	var offerPresented []byte
	err := r.db.QueryRowContext(ctx, query, customerID).Scan(
		&session.ID, &session.TenantID, &session.CustomerID, &session.SubscriptionID,
		&session.FlowID, &session.Status, &session.CurrentStepIndex,
		&reason, &feedback, &offerPresented,
		&session.OfferAccepted, &session.SavedByOffer, &session.StartedAt, &session.CompletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get recent session: %w", err)
	}
	if reason.Valid {
		session.CancellationReason = reason.String
	}
	if feedback.Valid {
		session.Feedback = feedback.String
	}
	if offerPresented != nil {
		session.OfferPresented = offerPresented
	}
	return session, nil
}

// --- Analytics ---

func (r *CancelFlowRepository) GetSessionStats(ctx context.Context, tenantID uuid.UUID, flowID uuid.UUID) (*domain.FlowStats, error) {
	stats := &domain.FlowStats{
		ReasonBreakdown: make(map[string]int),
	}

	// Total sessions and save stats
	query := `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'completed') as completed,
			COUNT(*) FILTER (WHERE status = 'saved') as saved,
			COUNT(*) FILTER (WHERE offer_accepted = TRUE) as offer_accepted,
			COUNT(*) FILTER (WHERE offer_presented IS NOT NULL) as offer_presented
		FROM cancel_flow_sessions
		WHERE tenant_id = $1 AND flow_id = $2
	`
	var total, completed, saved, offerAccepted, offerPresented int
	err := r.db.QueryRowContext(ctx, query, tenantID, flowID).Scan(
		&total, &completed, &saved, &offerAccepted, &offerPresented,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get session stats: %w", err)
	}

	stats.TotalSessions = total
	stats.CompletedCount = completed
	stats.SavedCount = saved
	if total > 0 {
		stats.SaveRate = float64(saved) / float64(total) * 100
	}
	if offerPresented > 0 {
		stats.OfferAcceptRate = float64(offerAccepted) / float64(offerPresented) * 100
	}

	// Reason breakdown
	reasonQuery := `
		SELECT cancellation_reason, COUNT(*) as cnt
		FROM cancel_flow_sessions
		WHERE tenant_id = $1 AND flow_id = $2 AND cancellation_reason IS NOT NULL AND cancellation_reason != ''
		GROUP BY cancellation_reason ORDER BY cnt DESC
	`
	rows, err := r.db.QueryContext(ctx, reasonQuery, tenantID, flowID)
	if err != nil {
		return nil, fmt.Errorf("failed to get reason breakdown: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var reason string
		var cnt int
		if err := rows.Scan(&reason, &cnt); err != nil {
			return nil, err
		}
		stats.ReasonBreakdown[reason] = cnt
	}

	return stats, rows.Err()
}
