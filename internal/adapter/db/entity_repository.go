package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// EntityRepository persists a tenant's legal entities and their invoice-number
// sequences.
type EntityRepository struct {
	db *sql.DB
}

func NewEntityRepository(db *sql.DB) *EntityRepository {
	return &EntityRepository{db: db}
}

const entityColumns = `id, tenant_id, name, legal_name, is_primary, tb_ledger_id, invoice_prefix, country_code, created_at, updated_at`

func scanEntity(row interface{ Scan(...any) error }) (*domain.Entity, error) {
	e := &domain.Entity{}
	err := row.Scan(&e.ID, &e.TenantID, &e.Name, &e.LegalName, &e.IsPrimary,
		&e.TBLedgerID, &e.InvoicePrefix, &e.CountryCode, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return e, nil
}

// List returns a tenant's entities, primary first then by creation order.
func (r *EntityRepository) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Entity, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+entityColumns+` FROM entities WHERE tenant_id = $1 ORDER BY is_primary DESC, created_at`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list entities: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []*domain.Entity
	for rows.Next() {
		e, err := scanEntity(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// GetByID returns one entity scoped to the tenant, or nil when not found.
func (r *EntityRepository) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.Entity, error) {
	e, err := scanEntity(r.db.QueryRowContext(ctx,
		`SELECT `+entityColumns+` FROM entities WHERE id = $1 AND tenant_id = $2`, id, tenantID))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}
	return e, nil
}

// GetPrimary returns the tenant's primary entity (the backfill target on ledger 1).
func (r *EntityRepository) GetPrimary(ctx context.Context, tenantID uuid.UUID) (*domain.Entity, error) {
	e, err := scanEntity(r.db.QueryRowContext(ctx,
		`SELECT `+entityColumns+` FROM entities WHERE tenant_id = $1 AND is_primary`, tenantID))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get primary entity: %w", err)
	}
	return e, nil
}

// nextLedgerID returns max(tb_ledger_id)+1 for the tenant (1-based; primary is 1).
func (r *EntityRepository) nextLedgerID(ctx context.Context, tx *sql.Tx, tenantID uuid.UUID) (int, error) {
	var maxID sql.NullInt64
	if err := tx.QueryRowContext(ctx,
		`SELECT MAX(tb_ledger_id) FROM entities WHERE tenant_id = $1`, tenantID).Scan(&maxID); err != nil {
		return 0, err
	}
	if !maxID.Valid {
		return 1, nil
	}
	return int(maxID.Int64) + 1, nil
}

// Create inserts a non-primary entity, allocating the next ledger id for the
// tenant and seeding its invoice sequence — atomically. TBLedgerID is assigned
// here (any value on the input is ignored).
func (r *EntityRepository) Create(ctx context.Context, e *domain.Entity) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	ledgerID, err := r.nextLedgerID(ctx, tx, e.TenantID)
	if err != nil {
		return fmt.Errorf("failed to allocate ledger id: %w", err)
	}
	e.TBLedgerID = ledgerID

	err = tx.QueryRowContext(ctx,
		`INSERT INTO entities (tenant_id, name, legal_name, is_primary, tb_ledger_id, invoice_prefix, country_code)
		 VALUES ($1,$2,$3,FALSE,$4,$5,$6)
		 RETURNING id, created_at, updated_at`,
		e.TenantID, e.Name, e.LegalName, e.TBLedgerID, e.InvoicePrefix, e.CountryCode,
	).Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create entity: %w", err)
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO entity_invoice_sequences (entity_id, next_number) VALUES ($1, 1)`, e.ID); err != nil {
		return fmt.Errorf("failed to seed invoice sequence: %w", err)
	}
	return tx.Commit()
}

// Update changes an entity's editable fields (name, legal name, invoice prefix,
// country). Primary status and ledger id are immutable here.
func (r *EntityRepository) Update(ctx context.Context, e *domain.Entity) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE entities SET name = $1, legal_name = $2, invoice_prefix = $3, country_code = $4, updated_at = NOW()
		 WHERE id = $5 AND tenant_id = $6`,
		e.Name, e.LegalName, e.InvoicePrefix, e.CountryCode, e.ID, e.TenantID)
	if err != nil {
		return fmt.Errorf("failed to update entity: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("entity %s not found", e.ID)
	}
	return nil
}

// Delete removes a non-primary entity. Callers must guard against deleting the
// primary or an entity that still owns data.
func (r *EntityRepository) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM entities WHERE id = $1 AND tenant_id = $2 AND is_primary = FALSE`, id, tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete entity: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("entity %s not found or is primary", id)
	}
	return nil
}
