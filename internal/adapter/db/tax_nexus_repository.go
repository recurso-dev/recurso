package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

type TaxNexusRepository struct {
	db *sql.DB
}

func NewTaxNexusRepository(db *sql.DB) *TaxNexusRepository {
	return &TaxNexusRepository{db: db}
}

func (r *TaxNexusRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]domain.TaxNexus, error) {
	// Multi-Entity Books Inc 3b: the tenant/primary nexus set is the entity_id
	// NULL rows; per-entity sets (when the management UI lands) are separate.
	const q = `
		SELECT id, tenant_id, state_code, nexus_type, established_at, created_at
		FROM tenant_tax_nexus
		WHERE tenant_id = $1 AND entity_id IS NULL
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

// SetStates replaces an issuing entity's entire nexus set atomically
// (Multi-Entity Books Inc 3b): a nil entityID manages the tenant/primary set
// (entity_id NULL), a non-primary entity manages its own — each independently.
func (r *TaxNexusRepository) SetStates(ctx context.Context, tenantID uuid.UUID, entityID *uuid.UUID, states []domain.TaxNexus) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM tenant_tax_nexus WHERE tenant_id = $1 AND entity_id IS NOT DISTINCT FROM $2`, tenantID, entityID); err != nil {
		return fmt.Errorf("clear nexus: %w", err)
	}
	conflict := "(tenant_id, state_code) WHERE entity_id IS NULL"
	if entityID != nil {
		conflict = "(tenant_id, entity_id, state_code) WHERE entity_id IS NOT NULL"
	}
	ins := fmt.Sprintf(`
		INSERT INTO tenant_tax_nexus (id, tenant_id, entity_id, state_code, nexus_type, established_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT %s DO NOTHING`, conflict)
	for _, n := range states {
		nt := n.NexusType
		if nt == "" {
			nt = domain.NexusPhysical
		}
		if _, err := tx.ExecContext(ctx, ins, uuid.New(), tenantID, entityID,
			strings.ToUpper(strings.TrimSpace(n.StateCode)), nt, n.EstablishedAt); err != nil {
			return fmt.Errorf("insert nexus: %w", err)
		}
	}
	return tx.Commit()
}

// ListByTenantEntity lists an issuing entity's nexus set (nil = tenant/primary).
func (r *TaxNexusRepository) ListByTenantEntity(ctx context.Context, tenantID uuid.UUID, entityID *uuid.UUID) ([]domain.TaxNexus, error) {
	const q = `
		SELECT id, tenant_id, state_code, nexus_type, established_at, created_at
		FROM tenant_tax_nexus
		WHERE tenant_id = $1 AND entity_id IS NOT DISTINCT FROM $2
		ORDER BY state_code`
	rows, err := r.db.QueryContext(ctx, q, tenantID, entityID)
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

// ListRegistrations returns the tenant's US sales-tax registrations (Track D · D4).
func (r *TaxNexusRepository) ListRegistrations(ctx context.Context, tenantID uuid.UUID) ([]domain.TaxRegistration, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT state_code, registration_number, status, registered_at
		FROM tax_registrations WHERE tenant_id = $1 ORDER BY state_code`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list registrations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []domain.TaxRegistration
	for rows.Next() {
		var reg domain.TaxRegistration
		var status string
		if err := rows.Scan(&reg.StateCode, &reg.RegistrationNumber, &status, &reg.RegisteredAt); err != nil {
			return nil, fmt.Errorf("scan registration: %w", err)
		}
		reg.Status = domain.TaxRegistrationStatus(status)
		out = append(out, reg)
	}
	return out, rows.Err()
}

// SetRegistrations replaces the tenant's entire registration set atomically —
// same full-replacement contract as SetStates.
func (r *TaxNexusRepository) SetRegistrations(ctx context.Context, tenantID uuid.UUID, regs []domain.TaxRegistration) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM tax_registrations WHERE tenant_id = $1`, tenantID); err != nil {
		return fmt.Errorf("clear registrations: %w", err)
	}
	const ins = `
		INSERT INTO tax_registrations (id, tenant_id, state_code, registration_number, status, registered_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (tenant_id, state_code) DO NOTHING`
	for _, reg := range regs {
		status := reg.Status
		if status == "" {
			status = domain.RegistrationRegistered
		}
		if _, err := tx.ExecContext(ctx, ins, uuid.New(), tenantID,
			strings.ToUpper(strings.TrimSpace(reg.StateCode)), strings.TrimSpace(reg.RegistrationNumber),
			string(status), reg.RegisteredAt); err != nil {
			return fmt.Errorf("insert registration: %w", err)
		}
	}
	return tx.Commit()
}

func (r *TaxNexusRepository) Delete(ctx context.Context, tenantID uuid.UUID, stateCode string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM tenant_tax_nexus WHERE tenant_id = $1 AND state_code = $2 AND entity_id IS NULL`,
		tenantID, strings.ToUpper(strings.TrimSpace(stateCode)))
	return err
}

// NexusFor returns (declaredAny, inState) in a single query.
func (r *TaxNexusRepository) NexusFor(ctx context.Context, tenantID uuid.UUID, stateCode string) (bool, bool, error) {
	const q = `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE state_code = $2)
		FROM tenant_tax_nexus WHERE tenant_id = $1 AND entity_id IS NULL`
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
		ON CONFLICT (tenant_id, state_code) WHERE entity_id IS NULL DO NOTHING`,
		uuid.New(), tenantID, strings.ToUpper(stateCode), establishedAt)
	if err != nil {
		return false, fmt.Errorf("failed to establish economic nexus: %w", err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// LiabilityByState aggregates US sales-tax liability per buyer state over
// [from, to) (Track D · D3): gross sales, the taxable/non-taxable split (by
// whether tax was collected), tax collected, and invoice count. Same scoping as
// SalesByState — US buyers, USD, real (non-void/draft) invoices — so the report
// ties to the nexus figures. Ordered by tax collected then state.
func (r *TaxNexusRepository) LiabilityByState(ctx context.Context, tenantID uuid.UUID, from, to time.Time) ([]domain.USLiabilityStateLine, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT UPPER(TRIM(c.state)) AS state,
		       COALESCE(SUM(i.subtotal), 0) AS gross,
		       COALESCE(SUM(CASE WHEN i.tax_amount > 0 THEN i.subtotal ELSE 0 END), 0) AS taxable,
		       COALESCE(SUM(CASE WHEN i.tax_type = 'sales_tax_exempt' THEN i.subtotal ELSE 0 END), 0) AS exempt,
		       COALESCE(SUM(CASE WHEN i.tax_amount = 0 AND i.tax_type <> 'sales_tax_exempt' THEN i.subtotal ELSE 0 END), 0) AS nontaxable,
		       COALESCE(SUM(i.tax_amount), 0) AS tax_collected,
		       COUNT(i.id) AS cnt
		FROM invoices i
		JOIN customers c ON c.id = i.customer_id
		WHERE i.tenant_id = $1
		  AND i.currency = 'USD'
		  AND UPPER(TRIM(c.country)) IN ('US', 'USA', 'UNITED STATES')
		  AND LENGTH(TRIM(c.state)) = 2
		  AND i.status NOT IN ('void', 'draft')
		  AND i.created_at >= $2 AND i.created_at < $3
		GROUP BY UPPER(TRIM(c.state))
		ORDER BY tax_collected DESC, state`, tenantID, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to query liability by state: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []domain.USLiabilityStateLine
	for rows.Next() {
		var l domain.USLiabilityStateLine
		if err := rows.Scan(&l.StateCode, &l.GrossSales, &l.TaxableSales, &l.ExemptSales, &l.NonTaxableSales, &l.TaxCollected, &l.InvoiceCount); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// ClaimNexusAlert atomically records that a (tenant, state, year, level) alert is
// being sent, returning true only the first time. The INSERT ... ON CONFLICT DO
// NOTHING is the dedup primitive — robust even when the scheduler lock is a no-op
// without Redis — so a threshold that stays crossed is alerted at most once per
// calendar year, per level.
func (r *TaxNexusRepository) ClaimNexusAlert(ctx context.Context, tenantID uuid.UUID, stateCode string, year int, level string, proximityPct int) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO nexus_alerts (id, tenant_id, state_code, year, level, proximity_pct, sent_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (tenant_id, state_code, year, level) DO NOTHING`,
		uuid.New(), tenantID, strings.ToUpper(stateCode), year, level, proximityPct)
	if err != nil {
		return false, fmt.Errorf("failed to claim nexus alert: %w", err)
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}
