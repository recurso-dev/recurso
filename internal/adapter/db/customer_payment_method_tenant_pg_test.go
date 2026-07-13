package db

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

func openCustomerPMTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed payment-method isolation test")
	}
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sqlx.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func seedPMTenantCustomer(t *testing.T, conn *sql.DB) (tenantID, customerID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	tenantID = uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`,
		tenantID, "PM-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	customerID = uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1, $2, $3, $4, NOW())`,
		customerID, tenantID, customerID.String()[:8]+"@c.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}
	return tenantID, customerID
}

// TestUpdatePaymentMethod_TenantIsolation proves the ENG-178 fix: a caller can
// only update its own tenant's customer's card. A cross-tenant update touches
// zero rows (sql.ErrNoRows) and leaves the card untouched.
func TestUpdatePaymentMethod_TenantIsolation(t *testing.T) {
	dbx := openCustomerPMTestDB(t)
	conn := dbx.DB
	repo := NewCustomerRepository(dbx)
	ctx := context.Background()

	ownerTenant, custID := seedPMTenantCustomer(t, conn)
	attackerTenant, _ := seedPMTenantCustomer(t, conn)

	// Attacker (different tenant) cannot update the owner's customer card.
	actx := context.WithValue(ctx, domain.TenantIDKey, attackerTenant)
	if err := repo.UpdatePaymentMethod(actx, custID, "hacked", "0000", 1, 2099); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("cross-tenant UpdatePaymentMethod: want sql.ErrNoRows, got %v", err)
	}
	var brand sql.NullString
	if err := conn.QueryRowContext(ctx, `SELECT card_brand FROM customers WHERE id = $1`, custID).Scan(&brand); err != nil {
		t.Fatalf("read card_brand: %v", err)
	}
	if brand.String == "hacked" {
		t.Fatal("cross-tenant update mutated the card brand")
	}

	// Missing tenant in context fails closed.
	if err := repo.UpdatePaymentMethod(ctx, custID, "visa", "4242", 12, 2030); err == nil {
		t.Fatal("UpdatePaymentMethod without tenant in context: expected error")
	}

	// The owning tenant updates successfully.
	octx := context.WithValue(ctx, domain.TenantIDKey, ownerTenant)
	if err := repo.UpdatePaymentMethod(octx, custID, "visa", "4242", 12, 2030); err != nil {
		t.Fatalf("owner UpdatePaymentMethod: %v", err)
	}
	if err := conn.QueryRowContext(ctx, `SELECT card_brand FROM customers WHERE id = $1`, custID).Scan(&brand); err != nil {
		t.Fatalf("read card_brand after owner update: %v", err)
	}
	if brand.String != "visa" {
		t.Fatalf("owner update did not persist: card_brand = %q, want visa", brand.String)
	}
}
