package db

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestUSTaxConfig_UpsertGet_Postgres proves the per-tenant US tax identity round-trips.
func TestUSTaxConfig_UpsertGet_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed us-tax-config test")
	}
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	database, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	repo := NewTenantUSTaxConfigRepository(database)

	// A tenant with no config resolves to nil.
	tenantID := uuid.New()
	if _, err := database.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email) VALUES ($1, 'US Co', $2)`,
		tenantID, tenantID.String()+"@test.co"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	defer func() { _, _ = database.ExecContext(ctx, `DELETE FROM tenants WHERE id = $1`, tenantID) }()

	got, err := repo.GetByTenantID(ctx, tenantID)
	if err != nil {
		t.Fatalf("get (empty): %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for unconfigured tenant, got %+v", got)
	}

	// Upsert then read back.
	want := &domain.TenantUSTaxConfig{TenantID: tenantID, LegalName: "Acme Inc", EIN: "12-3456789", Address: "1 Market St"}
	if err := repo.Upsert(ctx, want); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	got, err = repo.GetByTenantID(ctx, tenantID)
	if err != nil || got == nil {
		t.Fatalf("get after upsert: %v (got=%v)", err, got)
	}
	if got.LegalName != "Acme Inc" || got.EIN != "12-3456789" || got.Address != "1 Market St" {
		t.Errorf("round-trip mismatch: %+v", got)
	}

	// Upsert again updates in place (no duplicate row / PK conflict).
	want.EIN = "98-7654321"
	if err := repo.Upsert(ctx, want); err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	got, _ = repo.GetByTenantID(ctx, tenantID)
	if got.EIN != "98-7654321" {
		t.Errorf("update-in-place failed: EIN = %q", got.EIN)
	}
}
