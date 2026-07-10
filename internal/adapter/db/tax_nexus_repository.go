package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

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

// ListThresholds returns the seeded US economic-nexus threshold dataset.
func (r *TaxNexusRepository) ListThresholds(ctx context.Context) ([]domain.NexusThreshold, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT state_code, sales_threshold, txn_threshold, combinator, measurement_period, certified
		FROM us_nexus_thresholds ORDER BY state_code`)
	if err != nil {
		return nil, fmt.Errorf("failed to query nexus thresholds: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []domain.NexusThreshold
	for rows.Next() {
		var t domain.NexusThreshold
		var sales sql.NullInt64
		var txns sql.NullInt32
		if err := rows.Scan(&t.StateCode, &sales, &txns, &t.Combinator, &t.MeasurementPeriod, &t.Certified); err != nil {
			return nil, err
		}
		if sales.Valid {
			v := sales.Int64
			t.SalesThreshold = &v
		}
		if txns.Valid {
			v := int(txns.Int32)
			t.TxnThreshold = &v
		}
		t.StateCode = strings.TrimSpace(t.StateCode)
		out = append(out, t)
	}
	return out, nil
}

// SalesByState returns cumulative taxable sales (invoice subtotals, USD cents)
// and transaction counts per US buyer state for a tenant's calendar year.
// Computed from posted invoices — void and draft invoices don't count. The
// tenant is an explicit parameter (not ctx-derived) so background callers are
// immune to the tenant-context bug class.
func (r *TaxNexusRepository) SalesByState(ctx context.Context, tenantID uuid.UUID, year int) ([]domain.NexusStateSales, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT UPPER(TRIM(c.state)) AS state, COALESCE(SUM(i.subtotal), 0), COUNT(i.id)
		FROM invoices i
		JOIN customers c ON c.id = i.customer_id
		WHERE i.tenant_id = $1
		  AND i.currency = 'USD'
		  AND UPPER(TRIM(c.country)) IN ('US', 'USA', 'UNITED STATES')
		  AND LENGTH(TRIM(c.state)) = 2
		  AND i.status NOT IN ('void', 'draft')
		  AND EXTRACT(YEAR FROM i.created_at) = $2
		GROUP BY UPPER(TRIM(c.state))`, tenantID, year)
	if err != nil {
		return nil, fmt.Errorf("failed to query nexus sales: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []domain.NexusStateSales
	for rows.Next() {
		var s domain.NexusStateSales
		if err := rows.Scan(&s.StateCode, &s.TaxableSales, &s.TxnCount); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

// EstablishEconomic records economic nexus for a state a threshold was crossed
// in. Existing rows (physical/voluntary/economic) are left untouched — a
// declared nexus is never downgraded to economic.
func (r *TaxNexusRepository) EstablishEconomic(ctx context.Context, tenantID uuid.UUID, stateCode string, establishedAt time.Time) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO tenant_tax_nexus (id, tenant_id, state_code, nexus_type, established_at)
		VALUES ($1, $2, $3, 'economic', $4)
		ON CONFLICT (tenant_id, state_code) DO NOTHING`,
		uuid.New(), tenantID, strings.ToUpper(stateCode), establishedAt)
	if err != nil {
		return false, fmt.Errorf("failed to establish economic nexus: %w", err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}
