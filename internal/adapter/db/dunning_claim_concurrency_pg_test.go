package db_test

import (
	"context"
	"database/sql"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/recurso-dev/recurso/internal/adapter/db"
)

// The dunning scheduler was the one money-path sweep without an atomic claim
// (ADR-003): two instances reading the same overdue set both emailed the
// customer and both bumped retry_count. ClaimOverdueForDunning must partition
// the eligible rows across concurrent claimers and lease them out of later
// sweeps.
func TestClaimOverdueForDunning_ExclusiveAndLeased_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed dunning claim test")
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

	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := conn.ExecContext(ctx, q, args...); err != nil {
			t.Fatalf("exec %q: %v", q, err)
		}
	}

	tenantID := uuid.New()
	exec(`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "Dun-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com")
	customerID := uuid.New()
	exec(`INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1,$2,$3,$4,NOW())`,
		customerID, tenantID, customerID.String()[:8]+"@t.com", uuid.New())

	// insertInvoice creates one invoice with the given dunning knobs. nextRetry
	// "" means NULL (never retried).
	insertInvoice := func(status string, overdue bool, nextRetry string, managedBy *string, paused bool) uuid.UUID {
		id := uuid.New()
		due := "NOW() - INTERVAL '3 day'"
		if !overdue {
			due = "NOW() + INTERVAL '3 day'"
		}
		retry := "NULL"
		if nextRetry != "" {
			retry = nextRetry
		}
		exec(`INSERT INTO invoices (id, tenant_id, customer_id, currency, subtotal, total, amount_paid, status, invoice_number, next_retry_at, dunning_managed_by, dunning_paused, created_at, due_date)
			VALUES ($1,$2,$3,'USD',10000,10000,0,$4,$5, `+retry+`, $6, $7, NOW(), `+due+`)`,
			id, tenantID, customerID, status, "DUN-"+id.String()[:8], managedBy, paused)
		return id
	}
	sched := "scheduler"
	worker := "worker"

	const n = 12
	eligible := map[uuid.UUID]bool{}
	for i := 0; i < n; i++ {
		if i%2 == 0 {
			eligible[insertInvoice("past_due", true, "", &sched, false)] = true
		} else {
			eligible[insertInvoice("open", true, "NOW() - INTERVAL '1 minute'", nil, false)] = true
		}
	}
	// Excluded rows: paused, worker-managed, retry in the future, not yet due, paid.
	insertInvoice("past_due", true, "", &sched, true)
	insertInvoice("past_due", true, "", &worker, false)
	insertInvoice("past_due", true, "NOW() + INTERVAL '1 hour'", &sched, false)
	insertInvoice("past_due", false, "", &sched, false)
	insertInvoice("paid", true, "", &sched, false)

	repo := db.NewInvoiceRepository(conn)

	const workers = 4
	var ready, wg sync.WaitGroup
	var mu sync.Mutex
	claimCount := map[uuid.UUID]int{}
	claimTenant := map[uuid.UUID]uuid.UUID{}
	start := make(chan struct{})
	ready.Add(workers)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ready.Done()
			<-start
			invs, err := repo.ClaimOverdueForDunning(ctx, time.Hour, 1000)
			if err != nil {
				t.Errorf("ClaimOverdueForDunning: %v", err)
				return
			}
			mu.Lock()
			for _, inv := range invs {
				claimCount[inv.ID]++
				claimTenant[inv.ID] = inv.TenantID
				if inv.CustomerEmail == "" {
					t.Errorf("claimed invoice %s carries no customer email — the join is broken", inv.ID)
				}
			}
			mu.Unlock()
		}()
	}
	ready.Wait()
	close(start)
	wg.Wait()

	// The sweep is deliberately tenant-agnostic, so the shared test database
	// may hand back other tests' overdue rows too; only this tenant's rows are
	// judged for eligibility. The double-claim check stays global.
	claimed := 0
	for id, c := range claimCount {
		if c > 1 {
			t.Fatalf("invoice %s was claimed %d times — concurrent sweeps would dun it twice", id, c)
		}
		if claimTenant[id] != tenantID {
			continue
		}
		if !eligible[id] {
			t.Errorf("a non-eligible invoice %s was claimed", id)
		} else {
			claimed++
		}
	}
	if claimed != n {
		t.Errorf("claimed %d of %d eligible invoices across workers, want all", claimed, n)
	}

	// The lease holds: a later sweep in the same window finds nothing, and the
	// plain overdue read no longer sees the leased rows either.
	again, err := repo.ClaimOverdueForDunning(ctx, time.Hour, 1000)
	if err != nil {
		t.Fatalf("second claim: %v", err)
	}
	for _, inv := range again {
		if eligible[inv.ID] {
			t.Fatalf("invoice %s was claimable again inside its lease", inv.ID)
		}
	}
}
