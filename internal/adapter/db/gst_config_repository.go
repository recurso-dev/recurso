package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

type GSTConfigRepository struct {
	db *sql.DB
}

func NewGSTConfigRepository(db *sql.DB) *GSTConfigRepository {
	return &GSTConfigRepository{db: db}
}

// GetByTenantEntity resolves the GST seller config for a specific issuing
// entity (Multi-Entity Books Inc 3b). A nil entityID matches the tenant/primary
// default (entity_id IS NULL); a non-primary entity matches only its own row,
// so an unconfigured non-primary entity returns (nil, nil) and its e-invoice is
// skipped rather than filed under the default's GSTIN.
func (r *GSTConfigRepository) GetByTenantEntity(ctx context.Context, tenantID uuid.UUID, entityID *uuid.UUID) (*domain.TenantGSTConfig, error) {
	config := &domain.TenantGSTConfig{}
	query := `
		SELECT tenant_id, gstin, state_code, state_name, sac_code, gst_rate, pan,
		       legal_name, trade_name, address, has_lut
		FROM tenant_gst_configs
		WHERE tenant_id = $1 AND entity_id IS NOT DISTINCT FROM $2
	`
	err := r.db.QueryRowContext(ctx, query, tenantID, entityID).Scan(
		&config.TenantID, &config.GSTIN, &config.StateCode, &config.StateName,
		&config.SACCode, &config.GSTRate, &config.PAN,
		&config.LegalName, &config.TradeName, &config.Address, &config.HasLUT,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get GST config: %w", err)
	}
	return config, nil
}

// GetByTenantID returns the tenant/primary default GST config (entity_id NULL).
func (r *GSTConfigRepository) GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.TenantGSTConfig, error) {
	return r.GetByTenantEntity(ctx, tenantID, nil)
}

func (r *GSTConfigRepository) Upsert(ctx context.Context, tenantID uuid.UUID, config *domain.TenantGSTConfig) error {
	query := `
		INSERT INTO tenant_gst_configs (tenant_id, gstin, state_code, state_name, sac_code, gst_rate, pan,
		                                legal_name, trade_name, address, has_lut, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
		ON CONFLICT (tenant_id) WHERE entity_id IS NULL
		DO UPDATE SET
			gstin = EXCLUDED.gstin,
			state_code = EXCLUDED.state_code,
			state_name = EXCLUDED.state_name,
			sac_code = EXCLUDED.sac_code,
			gst_rate = EXCLUDED.gst_rate,
			pan = EXCLUDED.pan,
			legal_name = EXCLUDED.legal_name,
			trade_name = EXCLUDED.trade_name,
			address = EXCLUDED.address,
			has_lut = EXCLUDED.has_lut,
			updated_at = NOW()
	`
	_, err := r.db.ExecContext(ctx, query,
		tenantID, config.GSTIN, config.StateCode, config.StateName,
		config.SACCode, config.GSTRate, config.PAN,
		config.LegalName, config.TradeName, config.Address, config.HasLUT,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert GST config: %w", err)
	}
	return nil
}
