package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

type PlanRepository struct {
	db *sql.DB
}

func NewPlanRepository(db *sql.DB) port.PlanRepository {
	return &PlanRepository{db: db}
}

func (r *PlanRepository) Create(ctx context.Context, plan *domain.Plan) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// 1. Insert Plan
	query := `
		INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active, hsn_code, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err = tx.ExecContext(ctx, query,
		plan.ID, plan.TenantID, plan.Name, plan.Code,
		plan.IntervalUnit, plan.IntervalCount, plan.Active, plan.HSNCode, plan.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert plan: %w", err)
	}

	// 2. Insert Prices
	priceQuery := `
		INSERT INTO prices (id, plan_id, currency, amount, type, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	for _, price := range plan.Prices {
		_, err = tx.ExecContext(ctx, priceQuery,
			price.ID, price.PlanID, price.Currency, price.Amount, price.Type, price.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to insert price: %w", err)
		}
	}

	return tx.Commit()
}

func (r *PlanRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Plan, error) {
	tenantID, ok := ctx.Value(domain.TenantIDKey).(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("tenant_id missing from context")

	}

	plan := &domain.Plan{}
	query := `
		SELECT id, tenant_id, name, code, interval_unit, interval_count, active, hsn_code, created_at
		FROM plans WHERE id = $1 AND tenant_id = $2
	`
	err := r.db.QueryRowContext(ctx, query, id, tenantID).Scan(
		&plan.ID, &plan.TenantID, &plan.Name, &plan.Code,
		&plan.IntervalUnit, &plan.IntervalCount, &plan.Active, &plan.HSNCode, &plan.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, err
	}

	// Fetch Prices
	priceQuery := `SELECT id, plan_id, currency, amount, type, created_at FROM prices WHERE plan_id = $1`
	rows, err := r.db.QueryContext(ctx, priceQuery, id)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var p domain.Price
		if err := rows.Scan(&p.ID, &p.PlanID, &p.Currency, &p.Amount, &p.Type, &p.CreatedAt); err != nil {
			return nil, err
		}
		plan.Prices = append(plan.Prices, p)
	}

	return plan, nil
}

// Update persists mutable plan columns. Scoped by tenant_id so a caller can
// never edit another tenant's plan. `code` is intentionally not updated — it
// is the plan's stable external identifier.
func (r *PlanRepository) Update(ctx context.Context, plan *domain.Plan) error {
	query := `
		UPDATE plans
		SET name = $1, interval_unit = $2, interval_count = $3, active = $4, hsn_code = $5
		WHERE id = $6 AND tenant_id = $7
	`
	res, err := r.db.ExecContext(ctx, query,
		plan.Name, plan.IntervalUnit, plan.IntervalCount, plan.Active, plan.HSNCode,
		plan.ID, plan.TenantID,
	)
	if err != nil {
		return fmt.Errorf("failed to update plan: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *PlanRepository) GetByCode(ctx context.Context, tenantID uuid.UUID, code string) (*domain.Plan, error) {
	// Implementation similar to GetByID but using tenant_id + code index
	// Skipping for brevity in P0 implementation
	return nil, nil
}

func (r *PlanRepository) List(ctx context.Context, tenantID uuid.UUID, filter domain.PlanFilter) ([]*domain.Plan, error) {
	query := `
		SELECT id, tenant_id, name, code, interval_unit, interval_count, active, hsn_code, created_at
		FROM plans WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}
	argIdx := 2

	if filter.Search != "" {
		query += fmt.Sprintf(" AND (name ILIKE $%d OR code ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, filter.Limit)
		argIdx++
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var plans []*domain.Plan
	for rows.Next() {
		var p domain.Plan
		if err := rows.Scan(&p.ID, &p.TenantID, &p.Name, &p.Code, &p.IntervalUnit, &p.IntervalCount, &p.Active, &p.HSNCode, &p.CreatedAt); err != nil {
			return nil, err
		}

		// Fetch Prices for each plan (N+1 but acceptable for MVP Catalog size)
		priceQuery := `SELECT id, plan_id, currency, amount, type, created_at FROM prices WHERE plan_id = $1`
		pRows, err := r.db.QueryContext(ctx, priceQuery, p.ID)
		if err != nil {
			return nil, err
		}

		for pRows.Next() {
			var price domain.Price
			if err := pRows.Scan(&price.ID, &price.PlanID, &price.Currency, &price.Amount, &price.Type, &price.CreatedAt); err != nil {
				_ = pRows.Close()
				return nil, err
			}
			p.Prices = append(p.Prices, price)
		}
		_ = pRows.Close()

		plans = append(plans, &p)
	}

	return plans, nil
}
