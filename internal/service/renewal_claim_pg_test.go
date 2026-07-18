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

// TestClaimDueForRenewal_ExclusiveAndFiltered_Postgres proves the Lago-parity
// A1 guard: two concurrent billing-cycle runners never both claim the same due
// subscription (no duplicate renewal invoices), a claimed row is leased so an
// immediate re-claim sees nothing, and mandate-/gateway-managed subscriptions
// are never claimed at all.
func TestClaimDueForRenewal_ExclusiveAndFiltered_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed renewal claim test")
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
		tenantID, "Renew-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com")
	customerID := uuid.New()
	mustExec(t, conn, `INSERT INTO customers (id, tenant_id, name, email, ledger_account_id, created_at) VALUES ($1,$2,$3,$4,$5,NOW())`,
		customerID, tenantID, "Renew Cust", "rn-"+customerID.String()[:8]+"@example.com", uuid.New())
	planID := uuid.New()
	mustExec(t, conn, `INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active, created_at) VALUES ($1,$2,'Renewal Plan',$3,'month',1,true,NOW())`,
		planID, tenantID, "renew-"+planID.String()[:8])

	insertSub := func(mandate *uuid.UUID, razorpayID, stripeID, status string, due bool) uuid.UUID {
		id := uuid.New()
		end := "NOW() - INTERVAL '1 minute'"
		if !due {
			end = "NOW() + INTERVAL '10 days'"
		}
		mustExec(t, conn, `INSERT INTO subscriptions
			(id, tenant_id, customer_id, plan_id, status, current_period_start, current_period_end, billing_anchor, razorpay_subscription_id, stripe_subscription_id, mandate_id, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5, NOW() - INTERVAL '1 month', `+end+`, NOW() - INTERVAL '1 month', $6, $7, $8, NOW(), NOW())`,
			id, tenantID, customerID, planID, status, razorpayID, stripeID, mandate)
		return id
	}

	const n = 6
	dueIDs := map[uuid.UUID]bool{}
	for i := 0; i < n; i++ {
		dueIDs[insertSub(nil, "", "", "active", true)] = true
	}
	// Excluded rows: gateway-managed cycles, a mandate, wrong status, not due.
	mandateID := uuid.New()
	mustExec(t, conn, `INSERT INTO mandates (id, tenant_id, customer_id, subscription_id, status, max_amount, frequency, created_at, updated_at)
		VALUES ($1,$2,$3,$4,'active',100000,'monthly',NOW(),NOW())`,
		mandateID, tenantID, customerID, insertSub(nil, "", "", "active", true))
	// (mandate row above references a sub; make a sub that carries mandate_id itself)
	excluded := []uuid.UUID{
		insertSub(&mandateID, "", "", "active", true),
		insertSub(nil, "rzp_sub_1", "", "active", true),
		insertSub(nil, "", "sub_stripe_1", "active", true),
		insertSub(nil, "", "", "canceled", true),
		insertSub(nil, "", "", "active", false),
	}

	repo := db.NewSubscriptionRepository(conn).(*db.SubscriptionRepository)

	var wg sync.WaitGroup
	var mu sync.Mutex
	claimed := map[uuid.UUID]int{}
	for r := 0; r < 2; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			subs, err := repo.ClaimDueForRenewal(ctx, 15*time.Minute, 100)
			if err != nil {
				t.Errorf("ClaimDueForRenewal: %v", err)
				return
			}
			mu.Lock()
			for _, s := range subs {
				claimed[s.ID]++
			}
			mu.Unlock()
		}()
	}
	wg.Wait()

	for id, c := range claimed {
		if c != 1 {
			t.Errorf("subscription %s claimed %d times, want exactly 1 (duplicate renewal invoice)", id, c)
		}
	}
	for id := range dueIDs {
		if claimed[id] != 1 {
			t.Errorf("due subscription %s not claimed", id)
		}
	}
	for _, id := range excluded {
		if claimed[id] != 0 {
			t.Errorf("excluded subscription %s was claimed (mandate/gateway/status/undue filter broken)", id)
		}
	}

	again, err := repo.ClaimDueForRenewal(ctx, 15*time.Minute, 100)
	if err != nil {
		t.Fatalf("re-claim: %v", err)
	}
	if len(again) != 0 {
		t.Errorf("re-claim returned %d rows, want 0 (lease should hide claimed rows)", len(again))
	}
}
