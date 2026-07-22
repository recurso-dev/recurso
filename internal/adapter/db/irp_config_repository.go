package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

type IRPConfigRepository struct {
	db *sql.DB
}

func NewIRPConfigRepository(db *sql.DB) *IRPConfigRepository {
	return &IRPConfigRepository{db: db}
}

// GetByTenantEntity resolves the IRP submission credentials for a specific
// issuing entity + environment (Multi-Entity Books Inc 3b). A nil entityID
// matches the tenant/primary default (entity_id IS NULL); a non-primary entity
// matches only its own row, so an unconfigured non-primary entity returns
// (nil, nil) and its IRN submission is not attempted under another entity's
// IRP account.
func (r *IRPConfigRepository) GetByTenantEntity(ctx context.Context, tenantID uuid.UUID, entityID *uuid.UUID, env string) (*domain.IRPConfig, error) {
	config := &domain.IRPConfig{}
	query := `
		SELECT id, tenant_id, environment, client_id, client_secret, username, password, gstin, is_enabled
		FROM tenant_irp_configs
		WHERE tenant_id = $1 AND environment = $2 AND entity_id IS NOT DISTINCT FROM $3
	`
	err := r.db.QueryRowContext(ctx, query, tenantID, env, entityID).Scan(
		&config.ID, &config.TenantID, &config.Environment,
		&config.ClientID, &config.ClientSecret, &config.Username, &config.Password,
		&config.GSTIN, &config.IsEnabled,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get IRP config: %w", err)
	}
	return config, nil
}

// GetByTenantID returns the tenant/primary default IRP config (entity_id NULL).
func (r *IRPConfigRepository) GetByTenantID(ctx context.Context, tenantID uuid.UUID, env string) (*domain.IRPConfig, error) {
	return r.GetByTenantEntity(ctx, tenantID, nil, env)
}

// Upsert writes the IRP config for an issuing entity + environment (Multi-Entity
// Books Inc 3b): a nil entityID writes the tenant/primary default (entity_id
// NULL), a non-primary entity writes its own row. The ON CONFLICT target matches
// the partial unique index for whichever case applies.
func (r *IRPConfigRepository) Upsert(ctx context.Context, entityID *uuid.UUID, config *domain.IRPConfig) error {
	conflict := "(tenant_id, environment) WHERE entity_id IS NULL"
	if entityID != nil {
		conflict = "(tenant_id, environment, entity_id) WHERE entity_id IS NOT NULL"
	}
	query := fmt.Sprintf(`
		INSERT INTO tenant_irp_configs (tenant_id, entity_id, environment, client_id, client_secret, username, password, gstin, is_enabled, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		ON CONFLICT %s
		DO UPDATE SET
			client_id = EXCLUDED.client_id,
			client_secret = EXCLUDED.client_secret,
			username = EXCLUDED.username,
			password = EXCLUDED.password,
			gstin = EXCLUDED.gstin,
			is_enabled = EXCLUDED.is_enabled,
			updated_at = NOW()
	`, conflict)
	_, err := r.db.ExecContext(ctx, query,
		config.TenantID, entityID, config.Environment, config.ClientID, config.ClientSecret,
		config.Username, config.Password, config.GSTIN, config.IsEnabled,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert IRP config: %w", err)
	}
	return nil
}

func (r *IRPConfigRepository) Delete(ctx context.Context, tenantID uuid.UUID, env string) error {
	query := `DELETE FROM tenant_irp_configs WHERE tenant_id = $1 AND environment = $2`
	_, err := r.db.ExecContext(ctx, query, tenantID, env)
	if err != nil {
		return fmt.Errorf("failed to delete IRP config: %w", err)
	}
	return nil
}
