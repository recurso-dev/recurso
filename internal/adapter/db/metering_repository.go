package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// BillableMetricRepository is the Postgres implementation of
// port.BillableMetricRepository.
type BillableMetricRepository struct {
	db *sql.DB
}

func NewBillableMetricRepository(db *sql.DB) port.BillableMetricRepository {
	return &BillableMetricRepository{db: db}
}

const metricColumns = `id, tenant_id, name, code, aggregation_type, field_name, created_at, updated_at`

func scanMetric(row interface{ Scan(...any) error }) (*domain.BillableMetric, error) {
	var m domain.BillableMetric
	if err := row.Scan(&m.ID, &m.TenantID, &m.Name, &m.Code, &m.AggregationType, &m.FieldName, &m.CreatedAt, &m.UpdatedAt); err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *BillableMetricRepository) Create(ctx context.Context, m *domain.BillableMetric) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO billable_metrics (id, tenant_id, name, code, aggregation_type, field_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		m.ID, m.TenantID, m.Name, m.Code, m.AggregationType, m.FieldName, m.CreatedAt, m.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert billable metric: %w", err)
	}
	return nil
}

func (r *BillableMetricRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.BillableMetric, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+metricColumns+` FROM billable_metrics WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	)
	m, err := scanMetric(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get billable metric: %w", err)
	}
	return m, nil
}

func (r *BillableMetricRepository) GetByCode(ctx context.Context, tenantID uuid.UUID, code string) (*domain.BillableMetric, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+metricColumns+` FROM billable_metrics WHERE tenant_id = $1 AND code = $2`,
		tenantID, code,
	)
	m, err := scanMetric(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get billable metric by code: %w", err)
	}
	return m, nil
}

func (r *BillableMetricRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]domain.BillableMetric, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+metricColumns+` FROM billable_metrics WHERE tenant_id = $1 ORDER BY code`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list billable metrics: %w", err)
	}
	defer func() { _ = rows.Close() }()

	metrics := []domain.BillableMetric{}
	for rows.Next() {
		m, err := scanMetric(rows)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, *m)
	}
	return metrics, rows.Err()
}

func (r *BillableMetricRepository) Update(ctx context.Context, m *domain.BillableMetric) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE billable_metrics
		SET name = $3, aggregation_type = $4, field_name = $5, updated_at = $6
		WHERE tenant_id = $1 AND id = $2`,
		m.TenantID, m.ID, m.Name, m.AggregationType, m.FieldName, m.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update billable metric: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *BillableMetricRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM billable_metrics WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	)
	if err != nil {
		return fmt.Errorf("failed to delete billable metric: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// IsUniqueViolation reports whether err is a Postgres unique-constraint
// violation (duplicate metric code, already-rated window, ...).
func IsUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}

// IsForeignKeyViolation reports whether err is a Postgres FK violation
// (e.g. deleting a metric still referenced by a plan charge).
func IsForeignKeyViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23503"
}

// ChargeRepository is the Postgres implementation of port.ChargeRepository.
type ChargeRepository struct {
	db *sql.DB
}

func NewChargeRepository(db *sql.DB) port.ChargeRepository {
	return &ChargeRepository{db: db}
}

func (r *ChargeRepository) ReplaceForPlan(ctx context.Context, tenantID, planID uuid.UUID, charges []domain.Charge) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM plan_charges WHERE tenant_id = $1 AND plan_id = $2`,
		tenantID, planID,
	); err != nil {
		return fmt.Errorf("failed to clear plan charges: %w", err)
	}

	const insert = `
		INSERT INTO plan_charges
			(id, tenant_id, plan_id, metric_id, charge_model, amounts, hsn_code, pay_in_advance, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	for _, ch := range charges {
		amounts, err := json.Marshal(ch.Amounts)
		if err != nil {
			return fmt.Errorf("failed to encode charge amounts: %w", err)
		}
		if _, err := tx.ExecContext(ctx, insert,
			ch.ID, tenantID, planID, ch.MetricID, ch.ChargeModel,
			amounts, ch.HSNCode, ch.PayInAdvance, ch.CreatedAt, ch.UpdatedAt,
		); err != nil {
			return fmt.Errorf("failed to insert plan charge: %w", err)
		}
	}

	return tx.Commit()
}

func (r *ChargeRepository) ListByPlan(ctx context.Context, tenantID, planID uuid.UUID) ([]domain.Charge, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT c.id, c.tenant_id, c.plan_id, c.metric_id, c.charge_model, c.amounts, c.hsn_code,
		       c.pay_in_advance, c.created_at, c.updated_at,
		       m.id, m.tenant_id, m.name, m.code, m.aggregation_type, m.field_name, m.created_at, m.updated_at
		FROM plan_charges c
		JOIN billable_metrics m ON m.id = c.metric_id
		WHERE c.tenant_id = $1 AND c.plan_id = $2
		ORDER BY m.code`,
		tenantID, planID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list plan charges: %w", err)
	}
	defer func() { _ = rows.Close() }()

	charges := []domain.Charge{}
	for rows.Next() {
		var (
			ch      domain.Charge
			m       domain.BillableMetric
			amounts []byte
		)
		if err := rows.Scan(
			&ch.ID, &ch.TenantID, &ch.PlanID, &ch.MetricID, &ch.ChargeModel, &amounts, &ch.HSNCode,
			&ch.PayInAdvance, &ch.CreatedAt, &ch.UpdatedAt,
			&m.ID, &m.TenantID, &m.Name, &m.Code, &m.AggregationType, &m.FieldName, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(amounts, &ch.Amounts); err != nil {
			return nil, fmt.Errorf("failed to decode charge amounts: %w", err)
		}
		ch.Metric = &m
		charges = append(charges, ch)
	}
	return charges, rows.Err()
}

// UsageRatingRepository is the Postgres implementation of
// port.UsageRatingRepository.
type UsageRatingRepository struct {
	db *sql.DB
}

func NewUsageRatingRepository(db *sql.DB) port.UsageRatingRepository {
	return &UsageRatingRepository{db: db}
}

func (r *UsageRatingRepository) Create(ctx context.Context, rating *domain.UsageRating) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO usage_ratings
			(id, tenant_id, subscription_id, charge_id, period_start, period_end, invoice_id, quantity, amount, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (subscription_id, charge_id, period_start) DO NOTHING`,
		rating.ID, rating.TenantID, rating.SubscriptionID, rating.ChargeID,
		rating.PeriodStart, rating.PeriodEnd, rating.InvoiceID,
		rating.Quantity, rating.Amount, rating.CreatedAt,
	)
	if err != nil {
		return false, fmt.Errorf("failed to insert usage rating: %w", err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (r *UsageRatingRepository) Exists(ctx context.Context, subscriptionID, chargeID uuid.UUID, periodStart time.Time) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM usage_ratings
			WHERE subscription_id = $1 AND charge_id = $2 AND period_start = $3
		)`,
		subscriptionID, chargeID, periodStart,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check usage rating: %w", err)
	}
	return exists, nil
}
