package db

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestEUConfig_PerEntityResolution_Postgres proves Multi-Entity Books Inc 3b
// (EU): the EU seller config resolves to the issuing entity's own registration
// — the primary (nil) entity gets the tenant default, a non-primary entity gets
// its own row, and an unconfigured non-primary entity resolves to nil (so its
// EN 16931 e-invoice is skipped, never filed under the default's VAT id).
func TestEUConfig_PerEntityResolution_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed eu-config-entity test")
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
		tenantID, "EU-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	entityRepo := NewEntityRepository(conn)
	configured := &domain.Entity{TenantID: tenantID, Name: "ACME GmbH", InvoicePrefix: "DE"}
	if err := entityRepo.Create(ctx, configured); err != nil {
		t.Fatalf("create configured entity: %v", err)
	}
	unconfigured := &domain.Entity{TenantID: tenantID, Name: "ACME SARL", InvoicePrefix: "FR"}
	if err := entityRepo.Create(ctx, unconfigured); err != nil {
		t.Fatalf("create unconfigured entity: %v", err)
	}

	repo := NewTenantEUConfigRepository(conn)

	// Tenant/primary default config (entity_id NULL) via the settings Upsert.
	if err := repo.Upsert(ctx, &domain.TenantEUConfig{
		TenantID: tenantID, Enabled: true, LegalName: "ACME Primary BV",
		VATNumber: "NL000099998B57", CountryCode: "NL",
	}); err != nil {
		t.Fatalf("upsert default config: %v", err)
	}
	// The configured non-primary entity's own registration.
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenant_eu_config (tenant_id, entity_id, enabled, legal_name, vat_number, country_code, updated_at)
		 VALUES ($1,$2,TRUE,$3,$4,$5,NOW())`,
		tenantID, configured.ID, "ACME GmbH", "DE123456789", "DE"); err != nil {
		t.Fatalf("insert entity config: %v", err)
	}

	// Primary (nil) → tenant default VAT.
	if cfg, err := repo.GetByTenantEntity(ctx, tenantID, nil); err != nil || cfg == nil || cfg.VATNumber != "NL000099998B57" {
		t.Fatalf("primary resolution = %+v, %v; want default VAT NL000099998B57", cfg, err)
	}
	// Configured non-primary entity → its own VAT.
	if cfg, err := repo.GetByTenantEntity(ctx, tenantID, &configured.ID); err != nil || cfg == nil || cfg.VATNumber != "DE123456789" {
		t.Fatalf("configured entity resolution = %+v, %v; want own VAT DE123456789", cfg, err)
	}
	// Unconfigured non-primary entity → nil (strict: no borrowing the default).
	if cfg, err := repo.GetByTenantEntity(ctx, tenantID, &unconfigured.ID); err != nil {
		t.Fatalf("unconfigured entity resolution error: %v", err)
	} else if cfg != nil {
		t.Errorf("unconfigured entity resolved to %q; want nil (strict, no default borrowing)", cfg.VATNumber)
	}

	// The default reader still returns the tenant default.
	if cfg, err := repo.GetByTenantID(ctx, tenantID); err != nil || cfg == nil || cfg.VATNumber != "NL000099998B57" {
		t.Fatalf("GetByTenantID = %+v, %v; want the default VAT", cfg, err)
	}
}
