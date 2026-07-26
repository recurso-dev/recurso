package db

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// The exemption expiry (migration 000143) survives Create and loads on EVERY
// read path — the fail-open guard: a missed SELECT/Scan would silently drop the
// expiry and keep honoring an expired certificate.
func TestCustomerExemptionExpiry_AllReadPaths_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed exemption-expiry test")
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
		tenantID, "EXP-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	repo := NewCustomerRepository(conn)

	name := "Reseller LLC"
	ref := "REF" + tenantID.String()[:6]
	expiry := time.Date(2027, 6, 30, 0, 0, 0, 0, time.UTC)
	cust := &domain.Customer{
		ID:                    uuid.New(),
		TenantID:              tenantID,
		Email:                 "ap@reseller.example",
		Name:                  &name,
		LedgerAccountID:       uuid.New(),
		ReferralCode:          &ref,
		TaxExempt:             true,
		TaxExemptionNumber:    "RESALE-0001",
		TaxExemptionExpiresAt: &expiry,
	}
	if err := repo.Create(ctx, cust); err != nil {
		t.Fatalf("create: %v", err)
	}

	assertExpiry := func(path string, c *domain.Customer, err error) {
		if err != nil || c == nil {
			t.Fatalf("%s: err=%v c=%v", path, err, c)
		}
		if c.TaxExemptionExpiresAt == nil {
			t.Fatalf("%s: expiry not loaded — a read path drops it (fail-open risk)", path)
		}
		if !c.TaxExemptionExpiresAt.UTC().Truncate(24 * time.Hour).Equal(expiry) {
			t.Errorf("%s: expiry = %v, want %v", path, c.TaxExemptionExpiresAt.UTC(), expiry)
		}
	}

	byID, err := repo.getByIDInternal(ctx, cust.ID, nil)
	assertExpiry("getByID", byID, err)

	byRef, err := repo.GetByReferralCode(ctx, tenantID, ref)
	assertExpiry("getByReferralCode", byRef, err)

	list, err := repo.List(ctx, tenantID, domain.CustomerFilter{Limit: 50})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var found *domain.Customer
	for _, c := range list {
		if c.ID == cust.ID {
			found = c
		}
	}
	assertExpiry("list", found, nil)

	// Update clears the expiry.
	byID.TaxExemptionExpiresAt = nil
	if err := repo.Update(ctx, byID); err != nil {
		t.Fatalf("update: %v", err)
	}
	after, err := repo.getByIDInternal(ctx, cust.ID, nil)
	if err != nil || after == nil {
		t.Fatalf("get after update: %v", err)
	}
	if after.TaxExemptionExpiresAt != nil {
		t.Errorf("expiry not cleared on update: %v", after.TaxExemptionExpiresAt)
	}
}
