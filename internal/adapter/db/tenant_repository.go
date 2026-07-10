package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"golang.org/x/crypto/bcrypt"
)

type TenantRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewTenantRepository(db *sql.DB) *TenantRepository {
	return &TenantRepository{
		db:     db,
		logger: slog.Default().With("repo", "tenant"),
	}
}

func (r *TenantRepository) CreateTenant(ctx context.Context, tenant *domain.Tenant) error {
	query := `INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.ExecContext(ctx, query, tenant.ID, tenant.Name, tenant.Email, tenant.CreatedAt, tenant.UpdatedAt)
	return err
}

// CreateAPIKey hashes the key with bcrypt before storing. The original key is
// returned in the APIKey struct but is never stored in the database.
func (r *TenantRepository) CreateAPIKey(ctx context.Context, key *domain.APIKey) error {
	// Hash the key value with bcrypt
	hash, err := bcrypt.GenerateFromPassword([]byte(key.KeyValue), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash API key: %w", err)
	}

	// Store prefix (first 8 chars) for efficient lookup
	prefix := key.KeyValue
	if len(prefix) > 8 {
		prefix = prefix[:8]
	}

	query := `INSERT INTO api_keys (id, tenant_id, key_value, key_hash, key_prefix, type, is_active, created_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err = r.db.ExecContext(ctx, query,
		key.ID, key.TenantID, "", string(hash), prefix, key.Type, key.IsActive, key.CreatedAt,
	)
	if err != nil {
		return err
	}

	// Set the hash and prefix on the struct for reference
	key.KeyHash = string(hash)
	key.KeyPrefix = prefix
	return nil
}

// GetTenantByKey validates an API key using prefix lookup + bcrypt compare.
func (r *TenantRepository) GetTenantByKey(ctx context.Context, keyValue string) (*domain.Tenant, error) {
	// Try hashed lookup first (prefix match + bcrypt verify)
	prefix := keyValue
	if len(prefix) > 8 {
		prefix = prefix[:8]
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT t.id, t.name, t.email, k.key_hash
		FROM tenants t
		JOIN api_keys k ON t.id = k.tenant_id
		WHERE k.key_prefix = $1 AND k.is_active = TRUE AND k.key_hash IS NOT NULL
	`, prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to query API keys: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var t domain.Tenant
		var keyHash string
		if err := rows.Scan(&t.ID, &t.Name, &t.Email, &keyHash); err != nil {
			continue
		}
		// bcrypt compare
		if bcrypt.CompareHashAndPassword([]byte(keyHash), []byte(keyValue)) == nil {
			return &t, nil
		}
	}

	return nil, fmt.Errorf("invalid API key")
}

func (r *TenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	query := `SELECT id, name, email, created_at, updated_at FROM tenants WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)
	var t domain.Tenant
	if err := row.Scan(&t.ID, &t.Name, &t.Email, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TenantRepository) Update(ctx context.Context, tenant *domain.Tenant) error {
	query := `UPDATE tenants SET name = $1, email = $2, updated_at = $3 WHERE id = $4`
	_, err := r.db.ExecContext(ctx, query, tenant.Name, tenant.Email, tenant.UpdatedAt, tenant.ID)
	return err
}

func (r *TenantRepository) ListTenants(ctx context.Context) ([]*domain.Tenant, error) {
	query := `SELECT id, name, email, created_at, updated_at FROM tenants`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var tenants []*domain.Tenant
	for rows.Next() {
		var t domain.Tenant
		// name/email are nullable — scanning a NULL into a plain string aborts
		// the whole sweep (breaks the nexus + churn schedulers, which iterate
		// every tenant).
		var name, email sql.NullString
		if err := rows.Scan(&t.ID, &name, &email, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		t.Name = name.String
		t.Email = email.String
		tenants = append(tenants, &t)
	}
	return tenants, nil
}

func (r *TenantRepository) ListAPIKeys(ctx context.Context, tenantID uuid.UUID) ([]*domain.APIKey, error) {
	query := `SELECT id, tenant_id, key_prefix, type, is_active, created_at FROM api_keys WHERE tenant_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var keys []*domain.APIKey
	for rows.Next() {
		var k domain.APIKey
		var prefix sql.NullString
		if err := rows.Scan(&k.ID, &k.TenantID, &prefix, &k.Type, &k.IsActive, &k.CreatedAt); err != nil {
			return nil, err
		}
		// Show prefix with mask for display: "rk_1a2b...****"
		if prefix.Valid {
			k.KeyPrefix = prefix.String
			k.KeyValue = prefix.String + "****"
		}
		keys = append(keys, &k)
	}
	return keys, nil
}
