package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

// EntitlementRepository is the Postgres implementation of
// port.EntitlementRepository.
type EntitlementRepository struct {
	db *sql.DB
}

func NewEntitlementRepository(db *sql.DB) port.EntitlementRepository {
	return &EntitlementRepository{db: db}
}

// ReplaceForPlan deletes the plan's existing entitlement rows and inserts
// the new set in a single transaction (PUT replace semantics).
func (r *EntitlementRepository) ReplaceForPlan(ctx context.Context, tenantID, planID uuid.UUID, ents []domain.Entitlement) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM plan_entitlements WHERE tenant_id = $1 AND plan_id = $2`,
		tenantID, planID,
	); err != nil {
		return fmt.Errorf("failed to clear plan entitlements: %w", err)
	}

	const insert = `
		INSERT INTO plan_entitlements
			(id, tenant_id, plan_id, feature_key, kind, bool_value, limit_value, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	for _, e := range ents {
		if _, err := tx.ExecContext(ctx, insert,
			e.ID, tenantID, planID, e.FeatureKey, e.Kind,
			e.BoolValue, e.LimitValue, e.CreatedAt, e.UpdatedAt,
		); err != nil {
			return fmt.Errorf("failed to insert plan entitlement %q: %w", e.FeatureKey, err)
		}
	}

	return tx.Commit()
}

func (r *EntitlementRepository) ListByPlan(ctx context.Context, tenantID, planID uuid.UUID) ([]domain.Entitlement, error) {
	return r.list(ctx,
		`SELECT id, tenant_id, plan_id, feature_key, kind, bool_value, limit_value, created_at, updated_at
		 FROM plan_entitlements
		 WHERE tenant_id = $1 AND plan_id = $2
		 ORDER BY feature_key`,
		tenantID, planID,
	)
}

func (r *EntitlementRepository) ListByPlanIDs(ctx context.Context, tenantID uuid.UUID, planIDs []uuid.UUID) ([]domain.Entitlement, error) {
	if len(planIDs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(planIDs))
	args := make([]interface{}, 0, len(planIDs)+1)
	args = append(args, tenantID)
	for i, id := range planIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, id)
	}

	query := fmt.Sprintf(
		`SELECT id, tenant_id, plan_id, feature_key, kind, bool_value, limit_value, created_at, updated_at
		 FROM plan_entitlements
		 WHERE tenant_id = $1 AND plan_id IN (%s)
		 ORDER BY feature_key, plan_id`,
		strings.Join(placeholders, ", "),
	)
	return r.list(ctx, query, args...)
}

// CheckFeature resolves a single feature for a customer in one indexed
// query: join the customer's ACTIVE/TRIALING subscriptions to the plans'
// entitlement rows and aggregate.
//
// Aggregation mirrors the union semantics:
//   - granted: bool_or over rows — boolean rows contribute bool_value,
//     limit rows always grant access (the cap is the value, not the gate).
//   - limit_value: MAX across rows (most generous plan wins); NULL when
//     no limit-kind row matched.
//
// No rows (no subscriptions, or no plan grants the feature) yields
// granted=false, limit_value=NULL.
func (r *EntitlementRepository) CheckFeature(ctx context.Context, tenantID, customerID uuid.UUID, featureKey string) (*domain.EntitlementCheck, error) {
	const query = `
		SELECT
			COALESCE(bool_or(CASE WHEN pe.kind = 'boolean' THEN pe.bool_value ELSE TRUE END), FALSE) AS granted,
			MAX(pe.limit_value) AS limit_value
		FROM subscriptions s
		JOIN plan_entitlements pe
			ON pe.plan_id = s.plan_id AND pe.tenant_id = s.tenant_id
		WHERE s.tenant_id = $1
		  AND s.customer_id = $2
		  AND s.status IN ('active', 'trialing')
		  AND pe.feature_key = $3
	`

	check := &domain.EntitlementCheck{FeatureKey: featureKey}
	var limit sql.NullInt64
	if err := r.db.QueryRowContext(ctx, query, tenantID, customerID, featureKey).
		Scan(&check.Granted, &limit); err != nil {
		return nil, fmt.Errorf("failed to check entitlement: %w", err)
	}
	if limit.Valid {
		check.LimitValue = &limit.Int64
	}
	return check, nil
}

func (r *EntitlementRepository) list(ctx context.Context, query string, args ...interface{}) ([]domain.Entitlement, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var ents []domain.Entitlement
	for rows.Next() {
		var e domain.Entitlement
		var boolVal sql.NullBool
		var limitVal sql.NullInt64
		if err := rows.Scan(
			&e.ID, &e.TenantID, &e.PlanID, &e.FeatureKey, &e.Kind,
			&boolVal, &limitVal, &e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if boolVal.Valid {
			e.BoolValue = &boolVal.Bool
		}
		if limitVal.Valid {
			e.LimitValue = &limitVal.Int64
		}
		ents = append(ents, e)
	}
	return ents, rows.Err()
}
