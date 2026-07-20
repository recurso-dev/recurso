package db

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// GatewayConnectionRepository is the Postgres store for per-tenant BYO gateway
// credentials. Secret columns hold ciphertext produced upstream by the service;
// this layer only reads and writes rows.
type GatewayConnectionRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewGatewayConnectionRepository(db *sql.DB) *GatewayConnectionRepository {
	return &GatewayConnectionRepository{db: db, logger: slog.Default().With("repo", "gateway_connection")}
}

const gatewayConnCols = `id, tenant_id, provider, mode, public_key, secret_key_enc, webhook_secret_enc, active, created_at, updated_at`

func scanGatewayConn(row interface{ Scan(...any) error }) (*domain.GatewayConnection, error) {
	var c domain.GatewayConnection
	err := row.Scan(&c.ID, &c.TenantID, &c.Provider, &c.Mode, &c.PublicKey,
		&c.SecretKeyEnc, &c.WebhookSecretEnc, &c.Active, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrGatewayConnectionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// Upsert replaces the tenant's active connection for a provider. A prior active
// row is deactivated first (preserving it for the audit trail) so the partial
// unique index (tenant, provider) WHERE active is never violated.
func (r *GatewayConnectionRepository) Upsert(ctx context.Context, conn *domain.GatewayConnection) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		`UPDATE gateway_connections SET active = FALSE, updated_at = now()
		 WHERE tenant_id = $1 AND provider = $2 AND active`,
		conn.TenantID, conn.Provider); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO gateway_connections (`+gatewayConnCols+`)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		conn.ID, conn.TenantID, conn.Provider, conn.Mode, conn.PublicKey,
		conn.SecretKeyEnc, conn.WebhookSecretEnc, conn.Active, conn.CreatedAt, conn.UpdatedAt,
	); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *GatewayConnectionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.GatewayConnection, error) {
	return scanGatewayConn(r.db.QueryRowContext(ctx,
		`SELECT `+gatewayConnCols+` FROM gateway_connections WHERE id = $1`, id))
}

func (r *GatewayConnectionRepository) GetActive(ctx context.Context, tenantID uuid.UUID, provider domain.GatewayProvider) (*domain.GatewayConnection, error) {
	return scanGatewayConn(r.db.QueryRowContext(ctx,
		`SELECT `+gatewayConnCols+` FROM gateway_connections
		 WHERE tenant_id = $1 AND provider = $2 AND active`, tenantID, provider))
}

func (r *GatewayConnectionRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.GatewayConnection, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+gatewayConnCols+` FROM gateway_connections
		 WHERE tenant_id = $1 AND active ORDER BY provider`, tenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var conns []*domain.GatewayConnection
	for rows.Next() {
		c, err := scanGatewayConn(rows)
		if err != nil {
			return nil, err
		}
		conns = append(conns, c)
	}
	return conns, rows.Err()
}

func (r *GatewayConnectionRepository) Deactivate(ctx context.Context, tenantID uuid.UUID, provider domain.GatewayProvider) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE gateway_connections SET active = FALSE, updated_at = now()
		 WHERE tenant_id = $1 AND provider = $2 AND active`, tenantID, provider)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrGatewayConnectionNotFound
	}
	return nil
}
