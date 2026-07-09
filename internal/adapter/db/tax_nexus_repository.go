package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

type TaxNexusRepository struct {
	db *sql.DB
}

func NewTaxNexusRepository(db *sql.DB) *TaxNexusRepository {
	return &TaxNexusRepository{db: db}
}

func (r *TaxNexusRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]domain.TaxNexus, error) {
	const q = `
		SELECT id, tenant_id, state_code, nexus_type, established_at, created_at
		FROM tenant_tax_nexus
		WHERE tenant_id = $1
		ORDER BY state_code`
	rows, err := r.db.QueryContext(ctx, q, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list nexus: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []domain.TaxNexus
	for rows.Next() {
		var n domain.TaxNexus
		if err := rows.Scan(&n.ID, &n.TenantID, &n.StateCode, &n.NexusType, &n.EstablishedAt, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan nexus: %w", err)
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// SetStates replaces the tenant's entire nexus set atomically.
func (r *TaxNexusRepository) SetStates(ctx context.Context, tenantID uuid.UUID, states []domain.TaxNexus) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM tenant_tax_nexus WHERE tenant_id = $1`, tenantID); err != nil {
		return fmt.Errorf("clear nexus: %w", err)
	}
	const ins = `
		INSERT INTO tenant_tax_nexus (id, tenant_id, state_code, nexus_type, established_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (tenant_id, state_code) DO NOTHING`
	for _, n := range states {
		nt := n.NexusType
		if nt == "" {
			nt = domain.NexusPhysical
		}
		if _, err := tx.ExecContext(ctx, ins, uuid.New(), tenantID,
			strings.ToUpper(strings.TrimSpace(n.StateCode)), nt, n.EstablishedAt); err != nil {
			return fmt.Errorf("insert nexus: %w", err)
		}
	}
	return tx.Commit()
}

func (r *TaxNexusRepository) Delete(ctx context.Context, tenantID uuid.UUID, stateCode string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM tenant_tax_nexus WHERE tenant_id = $1 AND state_code = $2`,
		tenantID, strings.ToUpper(strings.TrimSpace(stateCode)))
	return err
}

// NexusFor returns (declaredAny, inState) in a single query.
func (r *TaxNexusRepository) NexusFor(ctx context.Context, tenantID uuid.UUID, stateCode string) (bool, bool, error) {
	const q = `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE state_code = $2)
		FROM tenant_tax_nexus WHERE tenant_id = $1`
	var total, inState int
	if err := r.db.QueryRowContext(ctx, q, tenantID, strings.ToUpper(strings.TrimSpace(stateCode))).Scan(&total, &inState); err != nil {
		return false, false, fmt.Errorf("nexus lookup: %w", err)
	}
	return total > 0, inState > 0, nil
}
