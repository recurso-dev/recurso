package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// AccountingMappingRepository stores internal-to-external entity ID mappings
// for accounting connections (accounting_entity_mappings table).
type AccountingMappingRepository struct {
	db *sql.DB
}

func NewAccountingMappingRepository(db *sql.DB) *AccountingMappingRepository {
	return &AccountingMappingRepository{db: db}
}

// Upsert inserts the mapping or, when a row already exists for
// (connection_id, entity_type, entity_id), refreshes external_id and
// updated_at.
func (r *AccountingMappingRepository) Upsert(ctx context.Context, m *domain.AccountingEntityMapping) error {
	query := `INSERT INTO accounting_entity_mappings
		(id, tenant_id, connection_id, entity_type, entity_id, external_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		ON CONFLICT (connection_id, entity_type, entity_id)
		DO UPDATE SET external_id = EXCLUDED.external_id, updated_at = NOW()`
	_, err := r.db.ExecContext(ctx, query,
		m.ID, m.TenantID, m.ConnectionID, m.EntityType, m.EntityID, m.ExternalID,
	)
	return err
}

// Delete removes the mapping for the given connection/entity. Deleting a
// mapping that does not exist is a no-op.
func (r *AccountingMappingRepository) Delete(ctx context.Context, connectionID uuid.UUID, entityType string, entityID uuid.UUID) error {
	query := `DELETE FROM accounting_entity_mappings
		WHERE connection_id = $1 AND entity_type = $2 AND entity_id = $3`
	_, err := r.db.ExecContext(ctx, query, connectionID, entityType, entityID)
	return err
}

// Get returns the mapping for the given connection/entity, or (nil, nil)
// when no mapping exists.
func (r *AccountingMappingRepository) Get(ctx context.Context, connectionID uuid.UUID, entityType string, entityID uuid.UUID) (*domain.AccountingEntityMapping, error) {
	query := `SELECT id, tenant_id, connection_id, entity_type, entity_id, external_id, created_at, updated_at
		FROM accounting_entity_mappings
		WHERE connection_id = $1 AND entity_type = $2 AND entity_id = $3`
	var m domain.AccountingEntityMapping
	err := r.db.QueryRowContext(ctx, query, connectionID, entityType, entityID).Scan(
		&m.ID, &m.TenantID, &m.ConnectionID, &m.EntityType, &m.EntityID,
		&m.ExternalID, &m.CreatedAt, &m.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}
