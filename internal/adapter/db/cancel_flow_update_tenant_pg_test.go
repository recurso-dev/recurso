package db

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestCancelFlowUpdate_TenantIsolation proves the defense-in-depth tenant
// scoping added to UpdateFlow and UpdateSession. Both writes are gated by
// tenant_id in the WHERE, so a write carrying another tenant's id affects zero
// rows and leaves the owning tenant's row untouched, while the owner's own
// write still persists. This is belt-and-suspenders behind the service/handler
// guards: the DB itself refuses a cross-tenant mutation.
func TestCancelFlowUpdate_TenantIsolation(t *testing.T) {
	dbx := openCancelFlowTestDB(t)
	defer func() { _ = dbx.Close() }()
	conn := dbx.DB
	repo := NewCancelFlowRepository(conn)
	ctx := context.Background()

	owner := seedCancelFlowTenant(t, conn)
	attacker := seedCancelFlowTenant(t, conn)

	// --- UpdateFlow ---
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

	// Attacker updates the owner's flow id but under their own tenant -> no match.
	tamperedFlow := &domain.CancelFlow{
		ID:           flow.ID,
		TenantID:     attacker,
		Name:         "HACKED",
		IsActive:     false,
		IsDefault:    false,
		CooldownDays: 0,
		UpdatedAt:    time.Now().UTC(),
	}
	if err := repo.UpdateFlow(ctx, tamperedFlow); err != nil {
		t.Fatalf("cross-tenant UpdateFlow returned error (expected silent no-op): %v", err)
	}
	var gotName string
	if err := conn.QueryRowContext(ctx, `SELECT name FROM cancel_flows WHERE id = $1`, flow.ID).Scan(&gotName); err != nil {
		t.Fatalf("read flow name: %v", err)
	}
	if gotName != "Retention" {
		t.Errorf("cross-tenant UpdateFlow mutated the row: name = %q, want %q", gotName, "Retention")
	}

	// Owner's own update persists.
	flow.Name = "Retention v2"
	if err := repo.UpdateFlow(ctx, flow); err != nil {
		t.Fatalf("owner UpdateFlow: %v", err)
	}
	if err := conn.QueryRowContext(ctx, `SELECT name FROM cancel_flows WHERE id = $1`, flow.ID).Scan(&gotName); err != nil {
		t.Fatalf("read flow name after owner update: %v", err)
	}
	if gotName != "Retention v2" {
		t.Errorf("owner UpdateFlow did not persist: name = %q, want %q", gotName, "Retention v2")
	}

	// --- UpdateSession ---
	session := &domain.CancelFlowSession{
		ID:               uuid.New(),
		TenantID:         owner,
		CustomerID:       uuid.New(),
		SubscriptionID:   uuid.New(),
		FlowID:           flow.ID,
		Status:           domain.SessionStatusInProgress,
		CurrentStepIndex: 0,
		OfferPresented:   []byte("{}"),
		StartedAt:        time.Now().UTC(),
	}
	if err := repo.CreateSession(ctx, session); err != nil {
		t.Fatalf("create session: %v", err)
	}

	// Attacker completes/cancels the owner's session under their own tenant -> no match.
	tamperedSession := &domain.CancelFlowSession{
		ID:               session.ID,
		TenantID:         attacker,
		Status:           domain.SessionStatusCancelled,
		CurrentStepIndex: 99,
		OfferPresented:   []byte("{}"),
		StartedAt:        session.StartedAt,
	}
	if err := repo.UpdateSession(ctx, tamperedSession); err != nil {
		t.Fatalf("cross-tenant UpdateSession returned error (expected silent no-op): %v", err)
	}
	var gotStatus string
	var gotIdx int
	if err := conn.QueryRowContext(ctx, `SELECT status, current_step_index FROM cancel_flow_sessions WHERE id = $1`, session.ID).Scan(&gotStatus, &gotIdx); err != nil {
		t.Fatalf("read session: %v", err)
	}
	if gotStatus != string(domain.SessionStatusInProgress) || gotIdx != 0 {
		t.Errorf("cross-tenant UpdateSession mutated the row: status=%q idx=%d, want in_progress/0", gotStatus, gotIdx)
	}

	// Owner's own update persists.
	session.Status = domain.SessionStatusCompleted
	session.CurrentStepIndex = 3
	if err := repo.UpdateSession(ctx, session); err != nil {
		t.Fatalf("owner UpdateSession: %v", err)
	}
	if err := conn.QueryRowContext(ctx, `SELECT status, current_step_index FROM cancel_flow_sessions WHERE id = $1`, session.ID).Scan(&gotStatus, &gotIdx); err != nil {
		t.Fatalf("read session after owner update: %v", err)
	}
	if gotStatus != string(domain.SessionStatusCompleted) || gotIdx != 3 {
		t.Errorf("owner UpdateSession did not persist: status=%q idx=%d, want completed/3", gotStatus, gotIdx)
	}
}
