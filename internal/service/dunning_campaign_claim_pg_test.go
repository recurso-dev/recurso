package service

import (
	"context"
	"database/sql"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/adapter/db"
)

// TestClaimDueExecutions_ExclusiveAndLeased_Postgres proves the ENG-199 guard:
// two concurrent runners claiming the same due set never both get the same
// execution (no duplicate dunning email/SMS), and a claimed row is leased
// forward so an immediate re-claim sees nothing.
func TestClaimDueExecutions_ExclusiveAndLeased_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed claim test")
	}
	if err := db.RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = conn.Close() }()
	ctx := context.Background()

	tenantID := uuid.New()
	mustExec(t, conn, `INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "DCClaim-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com")
	campaignID := uuid.New()
	mustExec(t, conn, `INSERT INTO dunning_campaigns (id, tenant_id, name, is_active, trigger_event, created_at, updated_at)
		VALUES ($1,$2,'Recovery',TRUE,'payment_failed',NOW(),NOW())`, campaignID, tenantID)

	const n = 8
	for i := 0; i < n; i++ {
		mustExec(t, conn, `INSERT INTO dunning_campaign_executions (id, tenant_id, invoice_id, campaign_id, current_step_index, status, started_at, next_step_at)
			VALUES ($1,$2,$3,$4,0,'active',NOW(), NOW() - INTERVAL '1 minute')`,
			uuid.New(), tenantID, uuid.New(), campaignID)
	}

	repo := db.NewDunningCampaignRepository(conn)
	now := time.Now().UTC()
	lease := now.Add(15 * time.Minute)

	// Two runners race to claim the same due set. FOR UPDATE SKIP LOCKED + the
	// lease UPDATE must give each execution to exactly one runner.
	var wg sync.WaitGroup
	var mu sync.Mutex
	claimed := map[uuid.UUID]int{}
	for r := 0; r < 2; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			execs, err := repo.ClaimDueExecutions(ctx, now, lease, n)
			if err != nil {
				t.Errorf("ClaimDueExecutions: %v", err)
				return
			}
			mu.Lock()
			for _, e := range execs {
				claimed[e.ID]++
			}
			mu.Unlock()
		}()
	}
	wg.Wait()

	if len(claimed) != n {
		t.Fatalf("claimed %d distinct executions, want %d (some were lost)", len(claimed), n)
	}
	for id, c := range claimed {
		if c != 1 {
			t.Errorf("execution %s claimed %d times, want exactly 1 (double-processing)", id, c)
		}
	}

	// The claimed rows are leased past `now`, so an immediate re-claim sees none.
	again, err := repo.ClaimDueExecutions(ctx, now, lease, n)
	if err != nil {
		t.Fatalf("re-claim: %v", err)
	}
	if len(again) != 0 {
		t.Errorf("re-claim returned %d rows, want 0 (lease should hide claimed rows)", len(again))
	}
}
