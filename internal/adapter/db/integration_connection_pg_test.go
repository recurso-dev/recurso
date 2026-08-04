package db

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestIntegrationConnection_OneActivePerCategory_Postgres proves the
// preferred-provider fix: connecting a second tax provider deactivates the
// first, so a tenant never has two active tax providers and the resolver's
// choice is unambiguous. Before this, Upsert only deactivated the same
// (category, provider), so TaxJar + Ziptax could both be active and TaxJar
// would silently win by iteration order.
//
// Skipped unless TEST_DATABASE_URL points at a scratch database.
func TestIntegrationConnection_OneActivePerCategory_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed integration-connection test")
	}
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = conn.Close() }()
	ctx := context.Background()

	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`,
		tenantID, "IC-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	repo := NewIntegrationConnectionRepository(conn)
	mk := func(provider string) *domain.IntegrationConnection {
		return &domain.IntegrationConnection{
			ID: uuid.New(), TenantID: tenantID, Category: domain.IntegrationTax,
			Provider: provider, ConfigEnc: "sealed-" + provider, Active: true,
		}
	}

	// Connect TaxJar, then Ziptax — the same tax category.
	if err := repo.Upsert(ctx, mk("taxjar")); err != nil {
		t.Fatalf("connect taxjar: %v", err)
	}
	if got, _ := repo.GetActive(ctx, tenantID, domain.IntegrationTax, "taxjar"); got == nil {
		t.Fatal("taxjar should be active after connecting it")
	}
	if err := repo.Upsert(ctx, mk("ziptax")); err != nil {
		t.Fatalf("connect ziptax: %v", err)
	}

	// Ziptax is now the single active tax provider; TaxJar was deactivated.
	if got, _ := repo.GetActive(ctx, tenantID, domain.IntegrationTax, "ziptax"); got == nil {
		t.Fatal("ziptax should be active after connecting it")
	}
	if got, _ := repo.GetActive(ctx, tenantID, domain.IntegrationTax, "taxjar"); got != nil {
		t.Error("taxjar must be deactivated when ziptax is connected (one active tax provider per tenant)")
	}
	active, err := repo.ListByTenant(ctx, tenantID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	taxCount := 0
	for _, c := range active {
		if c.Category == domain.IntegrationTax {
			taxCount++
		}
	}
	if taxCount != 1 {
		t.Errorf("active tax providers = %d, want exactly 1", taxCount)
	}
}
