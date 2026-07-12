package db

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

func openCancelFlowTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed cancel-flow step test")
	}
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	dbx, err := sqlx.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	return dbx
}

func seedCancelFlowTenant(t *testing.T, conn *sql.DB) uuid.UUID {
	t.Helper()
	tenantID := uuid.New()
	if _, err := conn.ExecContext(context.Background(),
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`,
		tenantID, "CF-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	return tenantID
}

// TestCancelFlowStep_TenantIsolation proves the ENG-160 hardening: a step's
// parent flow gates every write. Tenant B cannot update or delete a step that
// belongs to tenant A's flow — the scoped write affects zero rows and returns
// sql.ErrNoRows — while the owning tenant's writes still succeed.
func TestCancelFlowStep_TenantIsolation(t *testing.T) {
	dbx := openCancelFlowTestDB(t)
	defer func() { _ = dbx.Close() }()
	conn := dbx.DB
	repo := NewCancelFlowRepository(conn)
	ctx := context.Background()

	owner := seedCancelFlowTenant(t, conn)
	attacker := seedCancelFlowTenant(t, conn)

	flow := &domain.CancelFlow{
		ID:           uuid.New(),
		TenantID:     owner,
		Name:         "Retention",
		IsActive:     true,
		IsDefault:    true,
		CooldownDays: 30,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if err := repo.CreateFlow(ctx, flow); err != nil {
		t.Fatalf("create flow: %v", err)
	}

	step := &domain.CancelFlowStep{
		ID:        uuid.New(),
		FlowID:    flow.ID,
		StepOrder: 1,
		StepType:  domain.StepTypeOffer,
		Config:    []byte(`{}`),
		CreatedAt: time.Now().UTC(),
	}
	if err := repo.CreateStep(ctx, step); err != nil {
		t.Fatalf("create step: %v", err)
	}

	// Attacker update -> no rows, sql.ErrNoRows, and the row is untouched.
	tampered := &domain.CancelFlowStep{ID: step.ID, StepOrder: 99, StepType: step.StepType, Config: []byte(`{"hacked":true}`)}
	if err := repo.UpdateStep(ctx, tampered, attacker); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("cross-tenant UpdateStep: want sql.ErrNoRows, got %v", err)
	}
	var gotOrder int
	if err := conn.QueryRowContext(ctx, `SELECT step_order FROM cancel_flow_steps WHERE id = $1`, step.ID).Scan(&gotOrder); err != nil {
		t.Fatalf("read step_order: %v", err)
	}
	if gotOrder != 1 {
		t.Errorf("cross-tenant update mutated the step: step_order = %d, want 1", gotOrder)
	}

	// Attacker delete -> no rows, sql.ErrNoRows, and the row survives.
	if err := repo.DeleteStep(ctx, step.ID, attacker); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("cross-tenant DeleteStep: want sql.ErrNoRows, got %v", err)
	}
	var count int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM cancel_flow_steps WHERE id = $1`, step.ID).Scan(&count); err != nil {
		t.Fatalf("count step: %v", err)
	}
	if count != 1 {
		t.Fatalf("cross-tenant delete removed the step: count = %d, want 1", count)
	}

	// Owner update succeeds and persists.
	ownerUpd := &domain.CancelFlowStep{ID: step.ID, StepOrder: 5, StepType: step.StepType, Config: []byte(`{}`)}
	if err := repo.UpdateStep(ctx, ownerUpd, owner); err != nil {
		t.Fatalf("owner UpdateStep: %v", err)
	}
	if err := conn.QueryRowContext(ctx, `SELECT step_order FROM cancel_flow_steps WHERE id = $1`, step.ID).Scan(&gotOrder); err != nil {
		t.Fatalf("read step_order after owner update: %v", err)
	}
	if gotOrder != 5 {
		t.Errorf("owner update did not persist: step_order = %d, want 5", gotOrder)
	}

	// Owner delete succeeds.
	if err := repo.DeleteStep(ctx, step.ID, owner); err != nil {
		t.Fatalf("owner DeleteStep: %v", err)
	}
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM cancel_flow_steps WHERE id = $1`, step.ID).Scan(&count); err != nil {
		t.Fatalf("count step after owner delete: %v", err)
	}
	if count != 0 {
		t.Errorf("owner delete did not remove the step: count = %d, want 0", count)
	}
}
