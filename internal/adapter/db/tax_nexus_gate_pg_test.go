package db

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestNexusFor_AutoEconomicDoesNotGate_Postgres proves the US collection gate is
// NOT flipped into "collect only in listed states" by an auto-established
// economic-nexus row. A tenant deferring to a provider account (no MANUAL
// declarations) must keep declaredAny=false after the scheduler auto-establishes
// economic nexus — otherwise the gate silently halts collection in every state
// with real provider nexus not mirrored here (ENG-171).
func TestNexusFor_AutoEconomicDoesNotGate_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed nexus-gate test")
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

	// The scheduler auto-establishes economic nexus in WA after a crossing.
	established, err := repo.EstablishEconomic(ctx, tenantID, "WA", time.Now())
	if err != nil {
		t.Fatalf("EstablishEconomic: %v", err)
	}
	if !established {
		t.Fatal("expected economic nexus to be established")
	}

	// A buyer in CA — a state the tenant has NO Recurso nexus in but real
	// provider nexus. declaredAny MUST be false so the gate defers to the
	// provider; the auto WA row must not count as a manual declaration.
	// (Old code counted ALL rows → declaredAny=true → CA collection halted.)
	declaredCA, inCA, err := repo.NexusFor(ctx, tenantID, "CA")
	if err != nil {
		t.Fatalf("NexusFor CA: %v", err)
	}
	if declaredCA {
		t.Error("declaredAny = true for a tenant with only an AUTO-established economic row — the gate would halt provider collection in CA")
	}
	if inCA {
		t.Error("inState = true for CA, which has no nexus row")
	}

	// WA itself: still not a manual declaration, but inState is true so the
	// economic-nexus state is collected.
	declaredWA, inWA, err := repo.NexusFor(ctx, tenantID, "WA")
	if err != nil {
		t.Fatalf("NexusFor WA: %v", err)
	}
	if declaredWA {
		t.Error("declaredAny = true from an auto economic row (should stay false until a manual declaration)")
	}
	if !inWA {
		t.Error("inState = false for WA, which has the economic nexus row")
	}

	// A MANUAL declaration flips declaredAny true (the tenant opts into
	// Recurso-managed nexus gating).
	if err := repo.SetStates(ctx, tenantID, nil, []domain.TaxNexus{
		{StateCode: "CA", NexusType: domain.NexusPhysical},
	}); err != nil {
		t.Fatalf("SetStates: %v", err)
	}
	declaredCA2, inCA2, err := repo.NexusFor(ctx, tenantID, "CA")
	if err != nil {
		t.Fatalf("NexusFor CA (after manual): %v", err)
	}
	if !declaredCA2 || !inCA2 {
		t.Errorf("after a manual CA declaration: declaredAny=%v inState=%v, want true/true", declaredCA2, inCA2)
	}
}
