package db

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestIRPConfig_PerEntityResolution_Postgres proves Multi-Entity Books Inc 3b
// (IRP): submission credentials resolve to the issuing entity's own IRP account
// — the primary (nil) entity gets the tenant default, a non-primary entity gets
// its own row, and an unconfigured non-primary entity resolves to nil (so its
// IRN is not submitted under another entity's account).
func TestIRPConfig_PerEntityResolution_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed irp-config-entity test")
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
		tenantID, "IRP-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	entityRepo := NewEntityRepository(conn)
	configured := &domain.Entity{TenantID: tenantID, Name: "ACME India Two", InvoicePrefix: "IN2"}
	if err := entityRepo.Create(ctx, configured); err != nil {
		t.Fatalf("create configured entity: %v", err)
	}
	unconfigured := &domain.Entity{TenantID: tenantID, Name: "ACME India Three", InvoicePrefix: "IN3"}
	if err := entityRepo.Create(ctx, unconfigured); err != nil {
		t.Fatalf("create unconfigured entity: %v", err)
	}

	repo := NewIRPConfigRepository(conn)

	// Tenant/primary default credentials (entity_id NULL) via the settings Upsert.
	if err := repo.Upsert(ctx, nil, &domain.IRPConfig{
		TenantID: tenantID.String(), Environment: "production",
		ClientID: "primary-client", GSTIN: "29AAAAA0000A1Z5", IsEnabled: true,
	}); err != nil {
		t.Fatalf("upsert default config: %v", err)
	}
	// The configured non-primary entity's own IRP account.
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenant_irp_configs (tenant_id, entity_id, environment, client_id, gstin, is_enabled, updated_at)
		 VALUES ($1,$2,'production',$3,$4,TRUE,NOW())`,
		tenantID, configured.ID, "entity-client", "33BBBBB0000B1Z5"); err != nil {
		t.Fatalf("insert entity config: %v", err)
	}

	// Primary (nil) → tenant default credentials.
	if cfg, err := repo.GetByTenantEntity(ctx, tenantID, nil, "production"); err != nil || cfg == nil || cfg.ClientID != "primary-client" {
		t.Fatalf("primary resolution = %+v, %v; want default client primary-client", cfg, err)
	}
	// Configured non-primary entity → its own credentials.
	if cfg, err := repo.GetByTenantEntity(ctx, tenantID, &configured.ID, "production"); err != nil || cfg == nil || cfg.ClientID != "entity-client" {
		t.Fatalf("configured entity resolution = %+v, %v; want entity-client", cfg, err)
	}
	// Unconfigured non-primary entity → nil (strict: no borrowing the default).
	if cfg, err := repo.GetByTenantEntity(ctx, tenantID, &unconfigured.ID, "production"); err != nil {
		t.Fatalf("unconfigured entity resolution error: %v", err)
	} else if cfg != nil {
		t.Errorf("unconfigured entity resolved to %q; want nil (strict, no default borrowing)", cfg.ClientID)
	}

	// The default reader still returns the tenant default.
	if cfg, err := repo.GetByTenantID(ctx, tenantID, "production"); err != nil || cfg == nil || cfg.ClientID != "primary-client" {
		t.Fatalf("GetByTenantID = %+v, %v; want the default client", cfg, err)
	}
}
