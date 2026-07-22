package db

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestGSTConfig_PerEntityResolution_Postgres proves Multi-Entity Books Inc 3b:
// the GST seller config resolves to the issuing entity's own registration —
// the primary (nil) entity gets the tenant default, a non-primary entity gets
// its own row, and an unconfigured non-primary entity resolves to nil (so its
// e-invoice is skipped, never filed under the default GSTIN).
func TestGSTConfig_PerEntityResolution_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed gst-config-entity test")
	}
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	dbx, err := sqlx.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = dbx.Close() }()
	conn := dbx.DB
	ctx := context.Background()

	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "GST-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	entityRepo := NewEntityRepository(conn)
	// A configured non-primary entity and an unconfigured one.
	configured := &domain.Entity{TenantID: tenantID, Name: "ACME India Two", InvoicePrefix: "IN2"}
	if err := entityRepo.Create(ctx, configured); err != nil {
		t.Fatalf("create configured entity: %v", err)
	}
	unconfigured := &domain.Entity{TenantID: tenantID, Name: "ACME India Three", InvoicePrefix: "IN3"}
	if err := entityRepo.Create(ctx, unconfigured); err != nil {
		t.Fatalf("create unconfigured entity: %v", err)
	}

	repo := NewGSTConfigRepository(conn)

	// Tenant/primary default config (entity_id NULL) via the settings Upsert.
	if err := repo.Upsert(ctx, tenantID, nil, &domain.TenantGSTConfig{
		GSTIN: "29AAAAA0000A1Z5", StateCode: "29", LegalName: "ACME Primary",
	}); err != nil {
		t.Fatalf("upsert default config: %v", err)
	}
	// The configured non-primary entity's own registration.
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenant_gst_configs (tenant_id, entity_id, gstin, state_code, legal_name, updated_at)
		 VALUES ($1,$2,$3,$4,$5,NOW())`,
		tenantID, configured.ID, "33BBBBB0000B1Z5", "33", "ACME India Two"); err != nil {
		t.Fatalf("insert entity config: %v", err)
	}

	// Primary (nil) → tenant default GSTIN.
	if cfg, err := repo.GetByTenantEntity(ctx, tenantID, nil); err != nil || cfg == nil || cfg.GSTIN != "29AAAAA0000A1Z5" {
		t.Fatalf("primary resolution = %+v, %v; want default GSTIN 29AAAAA0000A1Z5", cfg, err)
	}
	// Configured non-primary entity → its own GSTIN.
	if cfg, err := repo.GetByTenantEntity(ctx, tenantID, &configured.ID); err != nil || cfg == nil || cfg.GSTIN != "33BBBBB0000B1Z5" {
		t.Fatalf("configured entity resolution = %+v, %v; want own GSTIN 33BBBBB0000B1Z5", cfg, err)
	}
	// Unconfigured non-primary entity → nil (strict: no borrowing the default).
	if cfg, err := repo.GetByTenantEntity(ctx, tenantID, &unconfigured.ID); err != nil {
		t.Fatalf("unconfigured entity resolution error: %v", err)
	} else if cfg != nil {
		t.Errorf("unconfigured entity resolved to %q; want nil (strict, no default borrowing)", cfg.GSTIN)
	}

	// The default reader still returns the tenant default.
	if cfg, err := repo.GetByTenantID(ctx, tenantID); err != nil || cfg == nil || cfg.GSTIN != "29AAAAA0000A1Z5" {
		t.Fatalf("GetByTenantID = %+v, %v; want the default GSTIN", cfg, err)
	}
}

// TestGSTConfig_PerEntityWrite_Postgres proves the per-entity write path (Inc 3b
// UI backend): Upsert with a non-nil entity writes that entity's own row without
// touching the tenant default, and updates it in place on a second call.
func TestGSTConfig_PerEntityWrite_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed gst-config-write test")
	}
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	dbx, err := sqlx.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = dbx.Close() }()
	conn := dbx.DB
	ctx := context.Background()

	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "GW-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	entityRepo := NewEntityRepository(conn)
	ent := &domain.Entity{TenantID: tenantID, Name: "ACME Two", InvoicePrefix: "AC2"}
	if err := entityRepo.Create(ctx, ent); err != nil {
		t.Fatalf("create entity: %v", err)
	}
	repo := NewGSTConfigRepository(conn)

	// Write the tenant default and the entity's own config via the write path.
	if err := repo.Upsert(ctx, tenantID, nil, &domain.TenantGSTConfig{GSTIN: "29DEFAULT0000Z5"}); err != nil {
		t.Fatalf("upsert default: %v", err)
	}
	if err := repo.Upsert(ctx, tenantID, &ent.ID, &domain.TenantGSTConfig{GSTIN: "33ENTITY00000Z5"}); err != nil {
		t.Fatalf("upsert entity: %v", err)
	}

	// Each row is independent.
	if cfg, _ := repo.GetByTenantEntity(ctx, tenantID, nil); cfg == nil || cfg.GSTIN != "29DEFAULT0000Z5" {
		t.Fatalf("default = %+v; want 29DEFAULT0000Z5", cfg)
	}
	if cfg, _ := repo.GetByTenantEntity(ctx, tenantID, &ent.ID); cfg == nil || cfg.GSTIN != "33ENTITY00000Z5" {
		t.Fatalf("entity = %+v; want 33ENTITY00000Z5", cfg)
	}

	// Updating the entity row in place does not touch the default.
	if err := repo.Upsert(ctx, tenantID, &ent.ID, &domain.TenantGSTConfig{GSTIN: "33UPDATED0000Z5"}); err != nil {
		t.Fatalf("update entity: %v", err)
	}
	if cfg, _ := repo.GetByTenantEntity(ctx, tenantID, &ent.ID); cfg == nil || cfg.GSTIN != "33UPDATED0000Z5" {
		t.Errorf("entity after update = %+v; want 33UPDATED0000Z5", cfg)
	}
	if cfg, _ := repo.GetByTenantEntity(ctx, tenantID, nil); cfg == nil || cfg.GSTIN != "29DEFAULT0000Z5" {
		t.Errorf("default after entity update = %+v; want unchanged 29DEFAULT0000Z5", cfg)
	}
}
