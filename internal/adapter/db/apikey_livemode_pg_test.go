package db

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// TestAPIKeyLivemode_Postgres proves the rsk_test_/rsk_live_ key round-trip:
// CreateAPIKey persists livemode, GetTenantByKey returns it for the matched key,
// and ListAPIKeys surfaces it with the right prefix.
//
// Skipped unless TEST_DATABASE_URL points at a scratch database.
func TestAPIKeyLivemode_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed api-key test")
	}
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations (000078 livemode): %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = conn.Close() }()

	repo := NewTenantRepository(conn)
	ctx := context.Background()
	now := time.Now().UTC()

	tenantID := uuid.New()
	if err := repo.CreateTenant(ctx, &domain.Tenant{
		ID: tenantID, Name: "LM-" + tenantID.String()[:8],
		Email: tenantID.String()[:8] + "@lm.com", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	testKey := domain.NewAPIKeyValue(false, uuid.New().String())
	liveKey := domain.NewAPIKeyValue(true, uuid.New().String())
	if !strings.HasPrefix(testKey, "rsk_test_") || !strings.HasPrefix(liveKey, "rsk_live_") {
		t.Fatalf("unexpected key prefixes: test=%q live=%q", testKey, liveKey)
	}
	for _, kd := range []struct {
		val  string
		live bool
	}{{testKey, false}, {liveKey, true}} {
		if err := repo.CreateAPIKey(ctx, &domain.APIKey{
			ID: uuid.New(), TenantID: tenantID, KeyValue: kd.val,
			Type: "secret", IsActive: true, Livemode: kd.live, CreatedAt: now,
		}); err != nil {
			t.Fatalf("create api key (live=%v): %v", kd.live, err)
		}
	}

	// GetTenantByKey returns the matched key's livemode.
	if tn, lm, err := repo.GetTenantByKey(ctx, testKey); err != nil || lm != false || tn.ID != tenantID {
		t.Fatalf("GetTenantByKey(test) = (%v, %v, %v), want (tenant, false, nil)", tn, lm, err)
	}
	if tn, lm, err := repo.GetTenantByKey(ctx, liveKey); err != nil || lm != true || tn.ID != tenantID {
		t.Fatalf("GetTenantByKey(live) = (%v, %v, %v), want (tenant, true, nil)", tn, lm, err)
	}

	// ListAPIKeys surfaces livemode + the mode-distinguishing prefix.
	keys, err := repo.ListAPIKeys(ctx, tenantID)
	if err != nil {
		t.Fatalf("ListAPIKeys: %v", err)
	}
	byPrefix := map[string]*domain.APIKey{}
	for _, k := range keys {
		byPrefix[k.KeyPrefix] = k
	}
	if k := byPrefix["rsk_test"]; k == nil || k.Livemode {
		t.Errorf("rsk_test key = %+v, want present with livemode=false", k)
	}
	if k := byPrefix["rsk_live"]; k == nil || !k.Livemode {
		t.Errorf("rsk_live key = %+v, want present with livemode=true", k)
	}
}
