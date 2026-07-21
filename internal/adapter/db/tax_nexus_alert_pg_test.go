package db

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"
)

// The claim is the dedup primitive: the first claim for a (tenant, state, year,
// level) wins, repeats return false, and a different level is independent —
// exercised against the real UNIQUE constraint from migration 000121.
func TestClaimNexusAlert_Dedup_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed nexus-alert test")
	}
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	ctx := context.Background()

	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "NX-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	repo := NewTaxNexusRepository(conn)

	// First claim wins.
	first, err := repo.ClaimNexusAlert(ctx, tenantID, "CA", 2026, "approaching", 82)
	if err != nil {
		t.Fatalf("claim 1: %v", err)
	}
	if !first {
		t.Fatal("first claim should win (true)")
	}

	// Same (tenant, state, year, level) is deduped.
	again, err := repo.ClaimNexusAlert(ctx, tenantID, "CA", 2026, "approaching", 90)
	if err != nil {
		t.Fatalf("claim 2: %v", err)
	}
	if again {
		t.Error("repeat claim should be deduped (false)")
	}

	// A different level for the same state is an independent alert.
	crossed, err := repo.ClaimNexusAlert(ctx, tenantID, "CA", 2026, "crossed", 120)
	if err != nil {
		t.Fatalf("claim crossed: %v", err)
	}
	if !crossed {
		t.Error("a different level should claim independently (true)")
	}

	// A different calendar year resets.
	nextYear, err := repo.ClaimNexusAlert(ctx, tenantID, "CA", 2027, "approaching", 85)
	if err != nil {
		t.Fatalf("claim next year: %v", err)
	}
	if !nextYear {
		t.Error("a new calendar year should claim independently (true)")
	}
}
