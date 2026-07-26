package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TenantUSTaxConfigRepository persists a tenant's US tax identity (W-9), kept
// isolated from the India GST and EU configs.
type TenantUSTaxConfigRepository struct {
	db *sql.DB
}

func NewTenantUSTaxConfigRepository(db *sql.DB) *TenantUSTaxConfigRepository {
	return &TenantUSTaxConfigRepository{db: db}
}

// GetByTenantID returns the tenant's US tax config, or (nil, nil) when none is
// set — the US invoice then falls back to the env seller identity.
func (r *TenantUSTaxConfigRepository) GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.TenantUSTaxConfig, error) {
	c := &domain.TenantUSTaxConfig{}
	err := r.db.QueryRowContext(ctx,
		`SELECT tenant_id, legal_name, ein, address FROM tenant_us_tax_config WHERE tenant_id = $1`, tenantID,
	).Scan(&c.TenantID, &c.LegalName, &c.EIN, &c.Address)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get US tax config: %w", err)
	}
	return c, nil
}

// Upsert writes the tenant's US tax config.
func (r *TenantUSTaxConfigRepository) Upsert(ctx context.Context, c *domain.TenantUSTaxConfig) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO tenant_us_tax_config (tenant_id, legal_name, ein, address, updated_at)
		 VALUES ($1,$2,$3,$4,NOW())
		 ON CONFLICT (tenant_id) DO UPDATE SET
		   legal_name = EXCLUDED.legal_name, ein = EXCLUDED.ein, address = EXCLUDED.address, updated_at = NOW()`,
		c.TenantID, c.LegalName, c.EIN, c.Address,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert US tax config: %w", err)
	}
	return nil
}
