package db

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// IntegrationConnectionRepository is the Postgres store for per-tenant
// integration credentials. The config blob is sealed upstream by the service.
type IntegrationConnectionRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewIntegrationConnectionRepository(db *sql.DB) *IntegrationConnectionRepository {
	return &IntegrationConnectionRepository{db: db, logger: slog.Default().With("repo", "integration_connection")}
}

const integrationConnCols = `id, tenant_id, category, provider, config_enc, active, created_at, updated_at`

func scanIntegrationConn(row interface{ Scan(...any) error }) (*domain.IntegrationConnection, error) {
	var c domain.IntegrationConnection
	err := row.Scan(&c.ID, &c.TenantID, &c.Category, &c.Provider, &c.ConfigEnc, &c.Active, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrIntegrationConnectionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// Upsert deactivates any prior active row for the (category, provider) then
// inserts the new one, in a tx, honoring the partial-unique index.
func (r *IntegrationConnectionRepository) Upsert(ctx context.Context, conn *domain.IntegrationConnection) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		`UPDATE integration_connections SET active = FALSE, updated_at = now()
		 WHERE tenant_id = $1 AND category = $2 AND provider = $3 AND active`,
		conn.TenantID, conn.Category, conn.Provider); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO integration_connections (`+integrationConnCols+`)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		conn.ID, conn.TenantID, conn.Category, conn.Provider, conn.ConfigEnc, conn.Active, conn.CreatedAt, conn.UpdatedAt,
	); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *IntegrationConnectionRepository) GetActive(ctx context.Context, tenantID uuid.UUID, category domain.IntegrationCategory, provider string) (*domain.IntegrationConnection, error) {
	return scanIntegrationConn(r.db.QueryRowContext(ctx,
		`SELECT `+integrationConnCols+` FROM integration_connections
		 WHERE tenant_id = $1 AND category = $2 AND provider = $3 AND active`, tenantID, category, provider))
}

func (r *IntegrationConnectionRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.IntegrationConnection, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+integrationConnCols+` FROM integration_connections
		 WHERE tenant_id = $1 AND active ORDER BY category, provider`, tenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var conns []*domain.IntegrationConnection
	for rows.Next() {
		c, err := scanIntegrationConn(rows)
		if err != nil {
			return nil, err
		}
		conns = append(conns, c)
	}
	return conns, rows.Err()
}

func (r *IntegrationConnectionRepository) Deactivate(ctx context.Context, tenantID uuid.UUID, category domain.IntegrationCategory, provider string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE integration_connections SET active = FALSE, updated_at = now()
		 WHERE tenant_id = $1 AND category = $2 AND provider = $3 AND active`, tenantID, category, provider)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrIntegrationConnectionNotFound
	}
	return nil
}
