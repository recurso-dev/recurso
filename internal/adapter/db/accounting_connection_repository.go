package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/secretbox"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

type AccountingConnectionRepository struct {
	db *sql.DB
	// box, when set (GATEWAY_ENCRYPTION_KEY configured), seals OAuth tokens at
	// rest. Reads tolerate legacy plaintext rows (see openToken), so encryption
	// is opportunistic and backward-compatible; a token re-seals on its next
	// write (refresh).
	box *secretbox.Box
}

func NewAccountingConnectionRepository(db *sql.DB) *AccountingConnectionRepository {
	return &AccountingConnectionRepository{db: db}
}

// SetVault enables at-rest encryption of the OAuth tokens.
func (r *AccountingConnectionRepository) SetVault(box *secretbox.Box) { r.box = box }

// sealToken encrypts a token for storage; a no-op when no vault is configured
// or the value is empty.
func (r *AccountingConnectionRepository) sealToken(v string) (string, error) {
	if r.box == nil || v == "" {
		return v, nil
	}
	return r.box.Seal(v)
}

// openToken decrypts a stored token, falling back to the raw value when the
// vault is absent or the value is a legacy plaintext token (Open fails on
// anything not sealed with our key).
func (r *AccountingConnectionRepository) openToken(v string) string {
	if r.box == nil || v == "" {
		return v
	}
	if plain, err := r.box.Open(v); err == nil {
		return plain
	}
	return v
}

// decryptTokens opens the token fields of a freshly-scanned connection.
func (r *AccountingConnectionRepository) decryptTokens(c *domain.AccountingConnection) {
	c.AccessToken = r.openToken(c.AccessToken)
	c.RefreshToken = r.openToken(c.RefreshToken)
}

func (r *AccountingConnectionRepository) Create(ctx context.Context, conn *domain.AccountingConnection) error {
	access, err := r.sealToken(conn.AccessToken)
	if err != nil {
		return err
	}
	refresh, err := r.sealToken(conn.RefreshToken)
	if err != nil {
		return err
	}
	query := `INSERT INTO accounting_connections (id, tenant_id, provider, access_token, refresh_token,
		token_expires_at, realm_id, sync_status, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err = r.db.ExecContext(ctx, query,
		conn.ID, conn.TenantID, conn.Provider, access, refresh,
		conn.TokenExpiresAt, conn.RealmID, conn.SyncStatus, conn.IsActive, conn.CreatedAt,
	)
	return err
}

func (r *AccountingConnectionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.AccountingConnection, error) {
	query := `SELECT id, tenant_id, provider, access_token, COALESCE(refresh_token,''), token_expires_at,
		COALESCE(realm_id,''), last_sync_at, sync_status, COALESCE(last_error,''), is_active, created_at
		FROM accounting_connections WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)
	return r.scanConnection(row)
}

func (r *AccountingConnectionRepository) GetByTenantAndProvider(ctx context.Context, tenantID uuid.UUID, provider string) (*domain.AccountingConnection, error) {
	query := `SELECT id, tenant_id, provider, access_token, COALESCE(refresh_token,''), token_expires_at,
		COALESCE(realm_id,''), last_sync_at, sync_status, COALESCE(last_error,''), is_active, created_at
		FROM accounting_connections WHERE tenant_id = $1 AND provider = $2`
	row := r.db.QueryRowContext(ctx, query, tenantID, provider)
	return r.scanConnection(row)
}

func (r *AccountingConnectionRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.AccountingConnection, error) {
	query := `SELECT id, tenant_id, provider, access_token, COALESCE(refresh_token,''), token_expires_at,
		COALESCE(realm_id,''), last_sync_at, sync_status, COALESCE(last_error,''), is_active, created_at
		FROM accounting_connections WHERE tenant_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var conns []*domain.AccountingConnection
	for rows.Next() {
		var c domain.AccountingConnection
		err := rows.Scan(&c.ID, &c.TenantID, &c.Provider, &c.AccessToken, &c.RefreshToken,
			&c.TokenExpiresAt, &c.RealmID, &c.LastSyncAt, &c.SyncStatus, &c.LastError,
			&c.IsActive, &c.CreatedAt)
		if err != nil {
			return nil, err
		}
		r.decryptTokens(&c)
		conns = append(conns, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return conns, nil
}

func (r *AccountingConnectionRepository) Update(ctx context.Context, conn *domain.AccountingConnection) error {
	// tenant_id scoped in the WHERE (defense-in-depth): conn.TenantID is the
	// DB-loaded owner (connections are fetched via tenant-scoped GetByTen…/
	// ListByTenant), so a token refresh or status write can't touch another
	// tenant's connection.
	access, err := r.sealToken(conn.AccessToken)
	if err != nil {
		return err
	}
	refresh, err := r.sealToken(conn.RefreshToken)
	if err != nil {
		return err
	}
	query := `UPDATE accounting_connections SET access_token = $1, refresh_token = $2, token_expires_at = $3,
		realm_id = $4, last_sync_at = $5, sync_status = $6, last_error = $7, is_active = $8
		WHERE id = $9 AND tenant_id = $10`
	_, err = r.db.ExecContext(ctx, query,
		access, refresh, conn.TokenExpiresAt,
		conn.RealmID, conn.LastSyncAt, conn.SyncStatus, conn.LastError, conn.IsActive, conn.ID, conn.TenantID,
	)
	return err
}

func (r *AccountingConnectionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE accounting_connections SET is_active = FALSE WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *AccountingConnectionRepository) GetActiveConnections(ctx context.Context) ([]*domain.AccountingConnection, error) {
	query := `SELECT id, tenant_id, provider, access_token, COALESCE(refresh_token,''), token_expires_at,
		COALESCE(realm_id,''), last_sync_at, sync_status, COALESCE(last_error,''), is_active, created_at
		FROM accounting_connections WHERE is_active = TRUE`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var conns []*domain.AccountingConnection
	for rows.Next() {
		var c domain.AccountingConnection
		err := rows.Scan(&c.ID, &c.TenantID, &c.Provider, &c.AccessToken, &c.RefreshToken,
			&c.TokenExpiresAt, &c.RealmID, &c.LastSyncAt, &c.SyncStatus, &c.LastError,
			&c.IsActive, &c.CreatedAt)
		if err != nil {
			return nil, err
		}
		r.decryptTokens(&c)
		conns = append(conns, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return conns, nil
}

func (r *AccountingConnectionRepository) CreateSyncLog(ctx context.Context, log *domain.AccountingSyncLog) error {
	query := `INSERT INTO accounting_sync_log (id, tenant_id, connection_id, entity_type, entity_id,
		external_id, action, status, error_message, synced_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := r.db.ExecContext(ctx, query,
		log.ID, log.TenantID, log.ConnectionID, log.EntityType, log.EntityID,
		log.ExternalID, log.Action, log.Status, log.ErrorMessage, log.SyncedAt,
	)
	return err
}

// SyncLogFilter narrows and pages the sync-activity listing.
type SyncLogFilter struct {
	Provider string // "" = all
	Status   string // "" = all ("success"/"error")
	Search   string // matches external_id or the internal entity id (prefix/substring)
	Limit    int
	Offset   int
}

// ListSyncLogsFiltered returns one page of sync activity plus the total row
// count for the same filter, so the dashboard can paginate.
func (r *AccountingConnectionRepository) ListSyncLogsFiltered(ctx context.Context, tenantID uuid.UUID, f SyncLogFilter) ([]*domain.AccountingSyncLog, int, error) {
	where := `FROM accounting_sync_log l
		LEFT JOIN accounting_connections c ON c.id = l.connection_id
		WHERE l.tenant_id = $1`
	args := []interface{}{tenantID}
	if f.Provider != "" {
		args = append(args, f.Provider)
		where += fmt.Sprintf(" AND c.provider = $%d", len(args))
	}
	if f.Status != "" {
		args = append(args, f.Status)
		where += fmt.Sprintf(" AND l.status = $%d", len(args))
	}
	if f.Search != "" {
		args = append(args, "%"+f.Search+"%")
		where += fmt.Sprintf(` AND (l.external_id ILIKE $%d OR l.entity_id::text ILIKE $%d
			OR (l.entity_type = 'customer' AND EXISTS (SELECT 1 FROM customers cu WHERE cu.id = l.entity_id AND cu.name ILIKE $%d))
			OR (l.entity_type = 'invoice' AND EXISTS (SELECT 1 FROM invoices inv WHERE inv.id = l.entity_id AND inv.invoice_number ILIKE $%d)))`,
			len(args), len(args), len(args), len(args))
	}

	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	//nolint:gosec // `where` is built from fixed fragments with $n placeholders; no caller value is spliced
	query := `SELECT l.id, l.tenant_id, l.connection_id, COALESCE(c.provider, ''),
		l.entity_type, l.entity_id,
		COALESCE(l.external_id,''),
		COALESCE(CASE l.entity_type
			WHEN 'customer' THEN (SELECT cu.name FROM customers cu WHERE cu.id = l.entity_id)
			WHEN 'invoice' THEN (SELECT inv.invoice_number FROM invoices inv WHERE inv.id = l.entity_id)
		END, ''),
		l.action, l.status, COALESCE(l.error_message,''), l.synced_at ` +
		where + fmt.Sprintf(" ORDER BY l.synced_at DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, f.Limit, f.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()

	var logs []*domain.AccountingSyncLog
	for rows.Next() {
		var l domain.AccountingSyncLog
		if err := rows.Scan(&l.ID, &l.TenantID, &l.ConnectionID, &l.Provider, &l.EntityType, &l.EntityID,
			&l.ExternalID, &l.EntityName, &l.Action, &l.Status, &l.ErrorMessage, &l.SyncedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, &l)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

func (r *AccountingConnectionRepository) ListSyncLogs(ctx context.Context, tenantID uuid.UUID, limit int) ([]*domain.AccountingSyncLog, error) {
	query := `SELECT l.id, l.tenant_id, l.connection_id, COALESCE(c.provider, ''),
		l.entity_type, l.entity_id,
		COALESCE(l.external_id,''), l.action, l.status, COALESCE(l.error_message,''), l.synced_at
		FROM accounting_sync_log l
		LEFT JOIN accounting_connections c ON c.id = l.connection_id
		WHERE l.tenant_id = $1 ORDER BY l.synced_at DESC LIMIT $2`
	rows, err := r.db.QueryContext(ctx, query, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var logs []*domain.AccountingSyncLog
	for rows.Next() {
		var l domain.AccountingSyncLog
		err := rows.Scan(&l.ID, &l.TenantID, &l.ConnectionID, &l.Provider, &l.EntityType, &l.EntityID,
			&l.ExternalID, &l.Action, &l.Status, &l.ErrorMessage, &l.SyncedAt)
		if err != nil {
			return nil, err
		}
		logs = append(logs, &l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return logs, nil
}

func (r *AccountingConnectionRepository) scanConnection(row *sql.Row) (*domain.AccountingConnection, error) {
	var c domain.AccountingConnection
	err := row.Scan(&c.ID, &c.TenantID, &c.Provider, &c.AccessToken, &c.RefreshToken,
		&c.TokenExpiresAt, &c.RealmID, &c.LastSyncAt, &c.SyncStatus, &c.LastError,
		&c.IsActive, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	r.decryptTokens(&c)
	return &c, nil
}
