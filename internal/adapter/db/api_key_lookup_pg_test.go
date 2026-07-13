package db

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"golang.org/x/crypto/bcrypt"
)

func openAPIKeyLookupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed api-key lookup test")
	}
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// TestAPIKeyLookup_FastPathAndInvalid proves the ENG-174 fast path: a key
// created through CreateAPIKey stores a SHA-256 key_lookup and authenticates via
// the O(1) indexed lookup; a wrong key is rejected.
func TestAPIKeyLookup_FastPathAndInvalid(t *testing.T) {
	conn := openAPIKeyLookupTestDB(t)
	repo := NewTenantRepository(conn)
	ctx := context.Background()
	now := time.Now().UTC()

	tenantID := uuid.New()
	if err := repo.CreateTenant(ctx, &domain.Tenant{
		ID: tenantID, Name: "AK-" + tenantID.String()[:8], Email: tenantID.String()[:8] + "@ak.com",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	key := domain.NewAPIKeyValue(true, uuid.New().String())
	keyID := uuid.New()
	if err := repo.CreateAPIKey(ctx, &domain.APIKey{
		ID: keyID, TenantID: tenantID, KeyValue: key, Type: "secret", IsActive: true, Livemode: true, CreatedAt: now,
	}); err != nil {
		t.Fatalf("create api key: %v", err)
	}

	// key_lookup is populated with the SHA-256 of the key.
	var lookup sql.NullString
	if err := conn.QueryRowContext(ctx, `SELECT key_lookup FROM api_keys WHERE id = $1`, keyID).Scan(&lookup); err != nil {
		t.Fatalf("read key_lookup: %v", err)
	}
	if lookup.String != apiKeyLookup(key) {
		t.Fatalf("stored key_lookup = %q, want %q", lookup.String, apiKeyLookup(key))
	}

	// Correct key authenticates via the fast path.
	if tn, lm, err := repo.GetTenantByKey(ctx, key); err != nil || !lm || tn.ID != tenantID {
		t.Fatalf("GetTenantByKey(valid) = (%v, %v, %v), want (tenant, true, nil)", tn, lm, err)
	}
	// A wrong key is rejected.
	if _, _, err := repo.GetTenantByKey(ctx, domain.NewAPIKeyValue(true, uuid.New().String())); err == nil {
		t.Fatal("GetTenantByKey(wrong key): expected error, got nil")
	}
}

// TestAPIKeyLookup_LegacyFallbackBackfills proves that a pre-migration key (bcrypt
// hash + prefix, but key_lookup NULL) still authenticates via the legacy prefix
// scan, and that a successful auth backfills key_lookup so the next auth uses the
// fast path.
func TestAPIKeyLookup_LegacyFallbackBackfills(t *testing.T) {
	conn := openAPIKeyLookupTestDB(t)
	repo := NewTenantRepository(conn)
	ctx := context.Background()
	now := time.Now().UTC()

	tenantID := uuid.New()
	if err := repo.CreateTenant(ctx, &domain.Tenant{
		ID: tenantID, Name: "LG-" + tenantID.String()[:8], Email: tenantID.String()[:8] + "@lg.com",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	// Insert a legacy key by hand: bcrypt hash + 8-char prefix, key_lookup NULL.
	key := domain.NewAPIKeyValue(false, uuid.New().String())
	hash, err := bcrypt.GenerateFromPassword([]byte(key), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	keyID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO api_keys (id, tenant_id, key_value, key_hash, key_prefix, key_lookup, type, is_active, livemode, created_at)
		 VALUES ($1, $2, '', $3, $4, NULL, 'secret', TRUE, FALSE, NOW())`,
		keyID, tenantID, string(hash), key[:8]); err != nil {
		t.Fatalf("insert legacy key: %v", err)
	}

	// Authenticates via the legacy fallback.
	if tn, lm, err := repo.GetTenantByKey(ctx, key); err != nil || lm || tn.ID != tenantID {
		t.Fatalf("legacy GetTenantByKey = (%v, %v, %v), want (tenant, false, nil)", tn, lm, err)
	}

	// The successful auth backfilled key_lookup.
	var lookup sql.NullString
	if err := conn.QueryRowContext(ctx, `SELECT key_lookup FROM api_keys WHERE id = $1`, keyID).Scan(&lookup); err != nil {
		t.Fatalf("read key_lookup: %v", err)
	}
	if lookup.String != apiKeyLookup(key) {
		t.Fatalf("key_lookup after auth = %q, want backfilled %q", lookup.String, apiKeyLookup(key))
	}

	// The now-backfilled key still authenticates (fast path).
	if tn, _, err := repo.GetTenantByKey(ctx, key); err != nil || tn.ID != tenantID {
		t.Fatalf("post-backfill GetTenantByKey = (%v, %v), want tenant", tn, err)
	}
}
