package db

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestClaimStep_ExactlyOneConcurrentWinner is the PHASE2 #2 guard: when many
// SubmitStep requests hit the same session step concurrently, only ONE wins the
// atomic claim. Without it, all readers pass the in-memory status/step checks
// and each applies the retention offer (trial extension / pause / plan switch) —
// double-applying the side effect. The conditional UPDATE serializes them.
func TestClaimStep_ExactlyOneConcurrentWinner(t *testing.T) {
	dbx := openCancelFlowTestDB(t)
	defer func() { _ = dbx.Close() }()
	conn := dbx.DB
	repo := NewCancelFlowRepository(conn)
	ctx := context.Background()

	tenantID := seedCancelFlowTenant(t, conn)
	flow := &domain.CancelFlow{
		ID: uuid.New(), TenantID: tenantID, Name: "Retention",
		IsActive: true, IsDefault: true, CooldownDays: 30,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := repo.CreateFlow(ctx, flow); err != nil {
		t.Fatalf("create flow: %v", err)
	}
	sessionID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO cancel_flow_sessions (id, tenant_id, customer_id, subscription_id, flow_id, status, current_step_index)
		 VALUES ($1,$2,$3,$4,$5,'in_progress',0)`,
		sessionID, tenantID, uuid.New(), uuid.New(), flow.ID); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	const n = 12
	var wg sync.WaitGroup
	won := make([]bool, n)
	errs := make([]error, n)
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start // release all goroutines together to maximize contention
			won[i], errs[i] = repo.ClaimStep(ctx, sessionID, tenantID, 0)
		}(i)
	}
	close(start)
	wg.Wait()

	winners := 0
	for i := 0; i < n; i++ {
		if errs[i] != nil {
			t.Fatalf("ClaimStep error: %v", errs[i])
		}
		if won[i] {
			winners++
		}
	}
	if winners != 1 {
		t.Errorf("concurrent claim winners = %d, want exactly 1 (offer must apply at most once)", winners)
	}

	// The step advanced exactly once, not n times.
	var step int
	if err := conn.QueryRowContext(ctx, `SELECT current_step_index FROM cancel_flow_sessions WHERE id=$1`, sessionID).Scan(&step); err != nil {
		t.Fatalf("read current_step_index: %v", err)
	}
	if step != 1 {
		t.Errorf("current_step_index = %d, want 1 (claimed exactly once)", step)
	}

	// A subsequent claim from a DIFFERENT tenant must not win (defense-in-depth).
	other := seedCancelFlowTenant(t, conn)
	if claimed, err := repo.ClaimStep(ctx, sessionID, other, 1); err != nil || claimed {
		t.Errorf("cross-tenant ClaimStep = (%v, %v), want (false, nil)", claimed, err)
	}
}
