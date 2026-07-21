package db

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// Customer tax-exemption fields survive Create → read → Update (Track D · D2):
// the migration-000122 columns plus the repo's insert/scan/update plumbing.
func TestCustomerTaxExemption_RoundTrip_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed customer exemption test")
	}
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sqlx.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	ctx := context.Background()

	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "EX-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	repo := NewCustomerRepository(conn)

	// Create with an exemption on file.
	name := "Reseller LLC"
	cust := &domain.Customer{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		Email:              "ap@reseller.example",
		Name:               &name,
		LedgerAccountID:    uuid.New(),
		TaxExempt:          true,
		TaxExemptionNumber: "RESALE-0001",
		TaxExemptionCode:   "A",
	}
	if err := repo.Create(ctx, cust); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.getByIDInternal(ctx, cust.ID, nil)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil {
		t.Fatal("customer not found after create")
	}
	if !got.TaxExempt || got.TaxExemptionNumber != "RESALE-0001" || got.TaxExemptionCode != "A" {
		t.Fatalf("exemption not persisted on create: %+v", got)
	}

	// Update: revoke the exemption.
	got.TaxExempt = false
	got.TaxExemptionNumber = ""
	got.TaxExemptionCode = ""
	if err := repo.Update(ctx, got); err != nil {
		t.Fatalf("update: %v", err)
	}

	after, err := repo.getByIDInternal(ctx, cust.ID, nil)
	if err != nil {
		t.Fatalf("get after update: %v", err)
	}
	if after.TaxExempt || after.TaxExemptionNumber != "" || after.TaxExemptionCode != "" {
		t.Fatalf("exemption not cleared on update: %+v", after)
	}

	// Sanity: a customer created without exemption defaults to not-exempt.
	plainID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1,$2,$3,$4,NOW())`,
		plainID, tenantID, "plain@example", uuid.New()); err != nil {
		t.Fatalf("seed plain customer: %v", err)
	}
	plain, err := repo.getByIDInternal(ctx, plainID, nil)
	if err != nil || plain == nil {
		t.Fatalf("get plain: %v", err)
	}
	if plain.TaxExempt || plain.TaxExemptionNumber != "" || plain.TaxExemptionCode != "" {
		t.Errorf("defaults wrong: %+v", plain)
	}
}
