package db

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// TestListActiveProgressiveSubscriptionIDs_Postgres proves the sweep's candidate
// query returns exactly the active subscriptions that have a progressive
// threshold — excluding non-progressive (null threshold) and non-active ones.
func TestListActiveProgressiveSubscriptionIDs_Postgres(t *testing.T) {
	conn := openProgressiveTestDB(t)
	repo := NewProgressiveBillingRepository(conn)
	ctx := context.Background()

	run := uuid.NewString()[:8]
	tenantID := uuid.New()
	must(t, conn, `INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "Sweep-"+run, "sweep-"+run+"@t.com")
	custID := uuid.New()
	must(t, conn, `INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1,$2,$3,$4,NOW())`,
		custID, tenantID, custID.String()[:8]+"@t.com", uuid.New())
	planID := uuid.New()
	must(t, conn, `INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active) VALUES ($1,$2,'Pro',$3,'month',1,TRUE)`,
		planID, tenantID, "pro-"+run)

	mkSub := func(status string, threshold interface{}) uuid.UUID {
		id := uuid.New()
		must(t, conn, `INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, status, current_period_start, current_period_end, billing_anchor, created_at, updated_at, progressive_billing_threshold)
			VALUES ($1,$2,$3,$4,$5,NOW(),NOW()+INTERVAL '1 month',NOW(),NOW(),NOW(),$6)`,
			id, tenantID, custID, planID, status, threshold)
		return id
	}

	wantIn := mkSub("active", int64(50000)) // active + threshold -> included
	mkSub("active", nil)                    // active, no threshold -> excluded
	mkSub("canceled", int64(50000))         // has threshold but canceled -> excluded
	mkSub("paused", int64(50000))           // has threshold but not active -> excluded

	ids, err := repo.ListActiveProgressiveSubscriptionIDs(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	// Filter to the ones this test created (the shared DB may hold others).
	found := false
	for _, id := range ids {
		if id == wantIn {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected the active+threshold subscription %s in the sweep set, got %v", wantIn, ids)
	}
	// The excluded ones must not appear.
	for _, id := range ids {
		if id != wantIn {
			// verify it isn't one of ours that should have been excluded by
			// re-checking it's genuinely active+threshold (belongs to another test)
			var status string
			var th *int64
			must0(t, conn.QueryRowContext(ctx,
				`SELECT status, progressive_billing_threshold FROM subscriptions WHERE id=$1`, id).Scan(&status, &th))
			if status != "active" || th == nil {
				t.Fatalf("sweep returned a non-active or non-progressive subscription %s (status=%s, threshold=%v)", id, status, th)
			}
		}
	}
}

func must0(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("query: %v", err)
	}
}
