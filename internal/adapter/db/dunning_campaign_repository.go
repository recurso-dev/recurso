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

type DunningCampaignRepository struct {
	db *sql.DB
}

func NewDunningCampaignRepository(db *sql.DB) port.DunningCampaignRepository {
	return &DunningCampaignRepository{db: db}
}

// --- Campaign CRUD ---

func (r *DunningCampaignRepository) CreateCampaign(ctx context.Context, campaign *domain.DunningCampaign) error {
	query := `
		INSERT INTO dunning_campaigns (id, tenant_id, name, is_active, trigger_event, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query,
		campaign.ID, campaign.TenantID, campaign.Name, campaign.IsActive,
		campaign.TriggerEvent, campaign.CreatedAt, campaign.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create dunning campaign: %w", err)
	}
	return nil
}

func (r *DunningCampaignRepository) GetCampaignByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.DunningCampaign, error) {
	// tenant_id scoping (ENG-160 hardening): callers previously relied on a
	// post-fetch TenantID guard in each handler; scoping here makes a forgotten
	// guard impossible.
	query := `
		SELECT id, tenant_id, name, is_active, trigger_event, created_at, updated_at
		FROM dunning_campaigns WHERE id = $1 AND tenant_id = $2
	`
	c := &domain.DunningCampaign{}
	err := r.db.QueryRowContext(ctx, query, id, tenantID).Scan(
		&c.ID, &c.TenantID, &c.Name, &c.IsActive, &c.TriggerEvent, &c.CreatedAt, &c.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get dunning campaign: %w", err)
	}

	steps, err := r.GetStepsByCampaign(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	c.Steps = steps

	return c, nil
}

func (r *DunningCampaignRepository) GetActiveCampaignForTenant(ctx context.Context, tenantID uuid.UUID, triggerEvent string) (*domain.DunningCampaign, error) {
	query := `
		SELECT id, tenant_id, name, is_active, trigger_event, created_at, updated_at
		FROM dunning_campaigns
		WHERE tenant_id = $1 AND trigger_event = $2 AND is_active = TRUE
		ORDER BY created_at DESC LIMIT 1
	`
	c := &domain.DunningCampaign{}
	err := r.db.QueryRowContext(ctx, query, tenantID, triggerEvent).Scan(
		&c.ID, &c.TenantID, &c.Name, &c.IsActive, &c.TriggerEvent, &c.CreatedAt, &c.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active dunning campaign: %w", err)
	}

	steps, err := r.GetStepsByCampaign(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	c.Steps = steps

	return c, nil
}

func (r *DunningCampaignRepository) ListCampaignsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.DunningCampaign, error) {
	query := `
		SELECT id, tenant_id, name, is_active, trigger_event, created_at, updated_at
		FROM dunning_campaigns WHERE tenant_id = $1 ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list dunning campaigns: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var campaigns []*domain.DunningCampaign
	for rows.Next() {
		c := &domain.DunningCampaign{}
		if err := rows.Scan(
			&c.ID, &c.TenantID, &c.Name, &c.IsActive, &c.TriggerEvent, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan dunning campaign: %w", err)
		}
		campaigns = append(campaigns, c)
	}
	return campaigns, rows.Err()
}

func (r *DunningCampaignRepository) UpdateCampaign(ctx context.Context, campaign *domain.DunningCampaign) error {
	query := `
		UPDATE dunning_campaigns SET name = $1, is_active = $2, trigger_event = $3, updated_at = $4
		WHERE id = $5
	`
	_, err := r.db.ExecContext(ctx, query,
		campaign.Name, campaign.IsActive, campaign.TriggerEvent, campaign.UpdatedAt, campaign.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update dunning campaign: %w", err)
	}
	return nil
}

// --- Step CRUD ---

func (r *DunningCampaignRepository) CreateStep(ctx context.Context, step *domain.DunningCampaignStep, tenantID uuid.UUID) error {
	// Guarded insert: the row is only written when the target campaign belongs
	// to tenantID, so a caller cannot add steps (e.g. a payment wall) to another
	// tenant's campaign.
	query := `
		INSERT INTO dunning_campaign_steps (id, campaign_id, step_order, channel, delay_hours, template_name, subject, body, is_payment_wall, created_at)
		SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		WHERE EXISTS (SELECT 1 FROM dunning_campaigns WHERE id = $2 AND tenant_id = $11)
	`
	res, err := r.db.ExecContext(ctx, query,
		step.ID, step.CampaignID, step.StepOrder, step.Channel, step.DelayHours,
		nilIfEmpty(step.TemplateName), nilIfEmpty(step.Subject), nilIfEmpty(step.Body),
		step.IsPaymentWall, step.CreatedAt, tenantID,
	)
	if err != nil {
		return fmt.Errorf("failed to create dunning campaign step: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read insert result: %w", err)
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *DunningCampaignRepository) GetStepsByCampaign(ctx context.Context, campaignID uuid.UUID) ([]domain.DunningCampaignStep, error) {
	query := `
		SELECT id, campaign_id, step_order, channel, delay_hours, template_name, subject, body, is_payment_wall, created_at
		FROM dunning_campaign_steps WHERE campaign_id = $1 ORDER BY step_order ASC
	`
	rows, err := r.db.QueryContext(ctx, query, campaignID)
	if err != nil {
		return nil, fmt.Errorf("failed to get dunning campaign steps: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var steps []domain.DunningCampaignStep
	for rows.Next() {
		var step domain.DunningCampaignStep
		var templateName, subject, body sql.NullString
		if err := rows.Scan(
			&step.ID, &step.CampaignID, &step.StepOrder, &step.Channel, &step.DelayHours,
			&templateName, &subject, &body, &step.IsPaymentWall, &step.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan dunning campaign step: %w", err)
		}
		if templateName.Valid {
			step.TemplateName = templateName.String
		}
		if subject.Valid {
			step.Subject = subject.String
		}
		if body.Valid {
			step.Body = body.String
		}
		steps = append(steps, step)
	}
	return steps, rows.Err()
}

func (r *DunningCampaignRepository) UpdateStep(ctx context.Context, step *domain.DunningCampaignStep, tenantID uuid.UUID) error {
	query := `
		UPDATE dunning_campaign_steps SET step_order = $1, channel = $2, delay_hours = $3,
		template_name = $4, subject = $5, body = $6, is_payment_wall = $7
		WHERE id = $8
		  AND campaign_id IN (SELECT id FROM dunning_campaigns WHERE tenant_id = $9)
	`
	res, err := r.db.ExecContext(ctx, query,
		step.StepOrder, step.Channel, step.DelayHours,
		nilIfEmpty(step.TemplateName), nilIfEmpty(step.Subject), nilIfEmpty(step.Body),
		step.IsPaymentWall, step.ID, tenantID,
	)
	if err != nil {
		return fmt.Errorf("failed to update dunning campaign step: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read update result: %w", err)
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *DunningCampaignRepository) DeleteStep(ctx context.Context, id, tenantID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM dunning_campaign_steps
		 WHERE id = $1
		   AND campaign_id IN (SELECT id FROM dunning_campaigns WHERE tenant_id = $2)`, id, tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete dunning campaign step: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read delete result: %w", err)
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// --- Execution CRUD ---

func (r *DunningCampaignRepository) CreateExecution(ctx context.Context, exec *domain.DunningCampaignExecution) error {
	query := `
		INSERT INTO dunning_campaign_executions (id, tenant_id, invoice_id, campaign_id, current_step_index, status, started_at, next_step_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.ExecContext(ctx, query,
		exec.ID, exec.TenantID, exec.InvoiceID, exec.CampaignID,
		exec.CurrentStepIndex, exec.Status, exec.StartedAt, exec.NextStepAt, exec.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create dunning campaign execution: %w", err)
	}
	return nil
}

func (r *DunningCampaignRepository) GetExecutionByInvoice(ctx context.Context, invoiceID uuid.UUID) (*domain.DunningCampaignExecution, error) {
	query := `
		SELECT id, tenant_id, invoice_id, campaign_id, current_step_index, status, started_at, next_step_at, completed_at
		FROM dunning_campaign_executions WHERE invoice_id = $1 AND status = 'active'
		ORDER BY started_at DESC LIMIT 1
	`
	exec := &domain.DunningCampaignExecution{}
	err := r.db.QueryRowContext(ctx, query, invoiceID).Scan(
		&exec.ID, &exec.TenantID, &exec.InvoiceID, &exec.CampaignID,
		&exec.CurrentStepIndex, &exec.Status, &exec.StartedAt, &exec.NextStepAt, &exec.CompletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get dunning campaign execution: %w", err)
	}
	return exec, nil
}

func (r *DunningCampaignRepository) UpdateExecution(ctx context.Context, exec *domain.DunningCampaignExecution) error {
	query := `
		UPDATE dunning_campaign_executions SET
			current_step_index = $1, status = $2, next_step_at = $3, completed_at = $4
		WHERE id = $5
	`
	_, err := r.db.ExecContext(ctx, query,
		exec.CurrentStepIndex, exec.Status, exec.NextStepAt, exec.CompletedAt, exec.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update dunning campaign execution: %w", err)
	}
	return nil
}

func (r *DunningCampaignRepository) GetDueExecutions(ctx context.Context, now time.Time) ([]*domain.DunningCampaignExecution, error) {
	query := `
		SELECT id, tenant_id, invoice_id, campaign_id, current_step_index, status, started_at, next_step_at, completed_at
		FROM dunning_campaign_executions
		WHERE status = 'active' AND next_step_at <= $1
		ORDER BY next_step_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, now)
	if err != nil {
		return nil, fmt.Errorf("failed to get due executions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var executions []*domain.DunningCampaignExecution
	for rows.Next() {
		exec := &domain.DunningCampaignExecution{}
		if err := rows.Scan(
			&exec.ID, &exec.TenantID, &exec.InvoiceID, &exec.CampaignID,
			&exec.CurrentStepIndex, &exec.Status, &exec.StartedAt, &exec.NextStepAt, &exec.CompletedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan dunning campaign execution: %w", err)
		}
		executions = append(executions, exec)
	}
	return executions, rows.Err()
}
