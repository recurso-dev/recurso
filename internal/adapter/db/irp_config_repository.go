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

func (r *IRPConfigRepository) GetByTenantID(ctx context.Context, tenantID uuid.UUID, env string) (*domain.IRPConfig, error) {
	config := &domain.IRPConfig{}
	query := `
		SELECT id, tenant_id, environment, client_id, client_secret, username, password, gstin, is_enabled
		FROM tenant_irp_configs
		WHERE tenant_id = $1 AND environment = $2
	`
	err := r.db.QueryRowContext(ctx, query, tenantID, env).Scan(
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

func (r *IRPConfigRepository) Upsert(ctx context.Context, config *domain.IRPConfig) error {
	query := `
		INSERT INTO tenant_irp_configs (tenant_id, environment, client_id, client_secret, username, password, gstin, is_enabled, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (tenant_id, environment)
		DO UPDATE SET
			client_id = EXCLUDED.client_id,
			client_secret = EXCLUDED.client_secret,
			username = EXCLUDED.username,
			password = EXCLUDED.password,
			gstin = EXCLUDED.gstin,
			is_enabled = EXCLUDED.is_enabled,
			updated_at = NOW()
	`
	_, err := r.db.ExecContext(ctx, query,
		config.TenantID, config.Environment, config.ClientID, config.ClientSecret,
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
