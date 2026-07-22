package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// MCPSettingsRepository persists a tenant's MCP opt-in.
type MCPSettingsRepository struct {
	db *sql.DB
}

func NewMCPSettingsRepository(db *sql.DB) *MCPSettingsRepository {
	return &MCPSettingsRepository{db: db}
}

// GetByTenantID returns the tenant's MCP settings, or nil when none is set (the
// caller then treats Tier-3 as disabled — fail-closed).
func (r *MCPSettingsRepository) GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.MCPSettings, error) {
	s := &domain.MCPSettings{}
	err := r.db.QueryRowContext(ctx,
		`SELECT tenant_id, tier3_enabled, updated_at FROM mcp_settings WHERE tenant_id = $1`, tenantID,
	).Scan(&s.TenantID, &s.Tier3Enabled, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP settings: %w", err)
	}
	return s, nil
}

// Upsert creates or replaces the tenant's MCP settings.
func (r *MCPSettingsRepository) Upsert(ctx context.Context, tenantID uuid.UUID, tier3Enabled bool) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO mcp_settings (tenant_id, tier3_enabled, updated_at)
		 VALUES ($1, $2, NOW())
		 ON CONFLICT (tenant_id) DO UPDATE SET
		   tier3_enabled = EXCLUDED.tier3_enabled, updated_at = NOW()`,
		tenantID, tier3Enabled,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert MCP settings: %w", err)
	}
	return nil
}
