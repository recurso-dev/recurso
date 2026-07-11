package db

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// SSOConnectionRepository is the Postgres-backed store for per-tenant SAML IdP
// configuration.
type SSOConnectionRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewSSOConnectionRepository(db *sql.DB) *SSOConnectionRepository {
	return &SSOConnectionRepository{db: db, logger: slog.Default().With("repo", "sso_connection")}
}

func (r *SSOConnectionRepository) GetByTenant(ctx context.Context, tenantID uuid.UUID) (*domain.SSOConnection, error) {
	var c domain.SSOConnection
	var metadataXML sql.NullString
	err := r.db.QueryRowContext(ctx,
		`SELECT id, tenant_id, idp_metadata_xml, idp_entity_id, idp_sso_url, idp_certificate, enabled, created_at, updated_at
		 FROM sso_connections WHERE tenant_id = $1`, tenantID,
	).Scan(&c.ID, &c.TenantID, &metadataXML, &c.IDPEntityID, &c.IDPSSOURL, &c.IDPCertificate, &c.Enabled, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrSSOConnectionNotFound
	}
	if err != nil {
		return nil, err
	}
	if metadataXML.Valid {
		c.IDPMetadataXML = metadataXML.String
	}
	return &c, nil
}

func (r *SSOConnectionRepository) Upsert(ctx context.Context, c *domain.SSOConnection) error {
	var metadataXML sql.NullString
	if c.IDPMetadataXML != "" {
		metadataXML = sql.NullString{String: c.IDPMetadataXML, Valid: true}
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO sso_connections
		   (id, tenant_id, idp_metadata_xml, idp_entity_id, idp_sso_url, idp_certificate, enabled, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		 ON CONFLICT (tenant_id) DO UPDATE SET
		   idp_metadata_xml = EXCLUDED.idp_metadata_xml,
		   idp_entity_id    = EXCLUDED.idp_entity_id,
		   idp_sso_url      = EXCLUDED.idp_sso_url,
		   idp_certificate  = EXCLUDED.idp_certificate,
		   enabled          = EXCLUDED.enabled,
		   updated_at       = NOW()`,
		c.ID, c.TenantID, metadataXML, c.IDPEntityID, c.IDPSSOURL, c.IDPCertificate, c.Enabled,
	)
	return err
}

func (r *SSOConnectionRepository) Delete(ctx context.Context, tenantID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM sso_connections WHERE tenant_id = $1`, tenantID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrSSOConnectionNotFound
	}
	return nil
}
