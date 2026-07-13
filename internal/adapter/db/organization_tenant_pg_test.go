package db_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

func openOrgTestDB(t *testing.T) (*db.OrganizationRepository, *sql.DB) {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed organization isolation test")
	}
	if err := db.RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return db.NewOrganizationRepository(conn), conn
}

func seedOrgTenant(t *testing.T, conn *sql.DB) uuid.UUID {
	t.Helper()
	tenantID := uuid.New()
	if _, err := conn.ExecContext(context.Background(),
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`,
		tenantID, "ORG-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	return tenantID
}

// TestOrganization_TenantIsolation proves the ENG-165 fix: the Organizations
// subsystem is scoped to the owning tenant. A non-owning tenant cannot read,
// list, mutate, or attach itself to another tenant's organization, and no
// tenant can pull a foreign tenant into an org it owns.
func TestOrganization_TenantIsolation(t *testing.T) {
	repo, conn := openOrgTestDB(t)
	svc := service.NewOrganizationService(repo, nil, nil)
	ctx := context.Background()

	owner := seedOrgTenant(t, conn)
	attacker := seedOrgTenant(t, conn)

	org, err := svc.Create(ctx, owner, "Acme Group", "boss@acme.com")
	if err != nil {
		t.Fatalf("create org: %v", err)
	}

	// Owner can read its own org.
	if _, err := svc.GetByID(ctx, owner, org.ID); err != nil {
		t.Fatalf("owner GetByID: %v", err)
	}
	// Attacker cannot read it — reported as not-found (no existence oracle).
	if _, err := svc.GetByID(ctx, attacker, org.ID); !errors.Is(err, domain.ErrOrganizationNotFound) {
		t.Fatalf("attacker GetByID: want ErrOrganizationNotFound, got %v", err)
	}

	// List is scoped: owner sees the org, attacker sees none.
	if orgs, err := svc.List(ctx, owner); err != nil || len(orgs) != 1 {
		t.Fatalf("owner List: got %d orgs, err %v; want 1", len(orgs), err)
	}
	if orgs, err := svc.List(ctx, attacker); err != nil || len(orgs) != 0 {
		t.Fatalf("attacker List: got %d orgs, err %v; want 0", len(orgs), err)
	}

	// Attacker cannot update, delete, list tenants, or read consolidated MRR.
	if _, err := svc.Update(ctx, attacker, org.ID, "Pwned", ""); !errors.Is(err, domain.ErrOrganizationNotFound) {
		t.Errorf("attacker Update: want ErrOrganizationNotFound, got %v", err)
	}
	if err := svc.Delete(ctx, attacker, org.ID); !errors.Is(err, domain.ErrOrganizationNotFound) {
		t.Errorf("attacker Delete: want ErrOrganizationNotFound, got %v", err)
	}
	if _, err := svc.ListTenants(ctx, attacker, org.ID); !errors.Is(err, domain.ErrOrganizationNotFound) {
		t.Errorf("attacker ListTenants: want ErrOrganizationNotFound, got %v", err)
	}

	// Attacker cannot attach itself to the owner's org.
	if err := svc.AddTenant(ctx, attacker, org.ID, attacker); !errors.Is(err, domain.ErrOrganizationNotFound) {
		t.Errorf("attacker AddTenant(self): want ErrOrganizationNotFound, got %v", err)
	}
	// Owner cannot pull the attacker's tenant into its own org (no consent).
	if err := svc.AddTenant(ctx, owner, org.ID, attacker); !errors.Is(err, domain.ErrCrossTenantAttach) {
		t.Errorf("owner AddTenant(foreign): want ErrCrossTenantAttach, got %v", err)
	}

	// Owner can attach itself and then see it in the org's tenant list.
	if err := svc.AddTenant(ctx, owner, org.ID, owner); err != nil {
		t.Fatalf("owner AddTenant(self): %v", err)
	}
	members, err := svc.ListTenants(ctx, owner, org.ID)
	if err != nil {
		t.Fatalf("owner ListTenants: %v", err)
	}
	if len(members) != 1 || members[0].ID != owner {
		t.Fatalf("owner ListTenants: got %+v, want [owner]", members)
	}

	// Owner delete succeeds.
	if err := svc.Delete(ctx, owner, org.ID); err != nil {
		t.Fatalf("owner Delete: %v", err)
	}
}
