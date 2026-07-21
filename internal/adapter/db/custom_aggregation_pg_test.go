package db

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestStreamEventsForMetric_Postgres proves the streaming primitive the custom
// aggregation folds over: it returns each period event's quantity and its
// string properties, in occurrence order, and scopes to the subscription,
// dimension, and [start,end) window. Property JSON decodes back to a map;
// property-less events yield a nil map.
func TestStreamEventsForMetric_Postgres(t *testing.T) {
	conn := openProgressiveTestDB(t)
	repo := NewUsageRepository(conn)
	ctx := context.Background()

	run := uuid.NewString()[:8]
	tenantID := uuid.New()
	must(t, conn, `INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "Cust-"+run, "cust-"+run+"@t.com")
	custID := uuid.New()
	must(t, conn, `INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1,$2,$3,$4,NOW())`,
		custID, tenantID, custID.String()[:8]+"@t.com", uuid.New())
	planID := uuid.New()
	must(t, conn, `INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active) VALUES ($1,$2,'Pro',$3,'month',1,TRUE)`,
		planID, tenantID, "pro-"+run)
	subID := uuid.New()
	must(t, conn, `INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, status, current_period_start, current_period_end, billing_anchor, created_at, updated_at)
		VALUES ($1,$2,$3,$4,'active',NOW(),NOW()+INTERVAL '1 month',NOW(),NOW(),NOW())`,
		subID, tenantID, custID, planID)

	dim := "storage_" + run
	start := time.Now().Add(-time.Hour)
	end := time.Now().Add(time.Hour)
	// Three in-window events (two with props, one without) + one out-of-window.
	must(t, conn, `INSERT INTO usage_events (id, subscription_id, customer_id, dimension, quantity, timestamp, properties) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		uuid.New(), subID, custID, dim, 1, time.Now().Add(-30*time.Minute), `{"bytes":"2500000"}`)
	must(t, conn, `INSERT INTO usage_events (id, subscription_id, customer_id, dimension, quantity, timestamp, properties) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		uuid.New(), subID, custID, dim, 1, time.Now().Add(-20*time.Minute), `{"bytes":"3000000"}`)
	must(t, conn, `INSERT INTO usage_events (id, subscription_id, customer_id, dimension, quantity, timestamp) VALUES ($1,$2,$3,$4,$5,$6)`,
		uuid.New(), subID, custID, dim, 7, time.Now().Add(-10*time.Minute))
	must(t, conn, `INSERT INTO usage_events (id, subscription_id, customer_id, dimension, quantity, timestamp, properties) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		uuid.New(), subID, custID, dim, 1, time.Now().Add(48*time.Hour), `{"bytes":"9999999"}`) // out of window

	type seen struct {
		qty   int64
		ts    time.Time
		props map[string]string
	}
	var got []seen
	if err := repo.StreamEventsForMetric(ctx, subID, dim, start, end, func(q int64, ts time.Time, p map[string]string) error {
		got = append(got, seen{q, ts, p})
		return nil
	}); err != nil {
		t.Fatalf("stream: %v", err)
	}
	// Timestamps must be non-zero and non-decreasing (occurrence order).
	for i := 1; i < len(got); i++ {
		if got[i].ts.Before(got[i-1].ts) {
			t.Fatalf("events not in timestamp order: %v before %v", got[i].ts, got[i-1].ts)
		}
	}

	if len(got) != 3 {
		t.Fatalf("want 3 in-window events, got %d", len(got))
	}
	// Occurrence order: bytes 2.5M, 3.0M, then the prop-less qty-7 event.
	if got[0].props["bytes"] != "2500000" || got[1].props["bytes"] != "3000000" {
		t.Fatalf("events out of order or props wrong: %+v", got)
	}
	if got[2].props != nil {
		t.Fatalf("property-less event should yield nil props, got %+v", got[2].props)
	}
	if got[2].qty != 7 {
		t.Fatalf("third event quantity want 7, got %d", got[2].qty)
	}
}

// TestSumQuantityBefore_Postgres proves the carry-forward primitive weighted_sum
// uses: Σ quantity of events strictly before a cutoff, scoped to the
// subscription and dimension. Negative deltas net in.
func TestSumQuantityBefore_Postgres(t *testing.T) {
	conn := openProgressiveTestDB(t)
	repo := NewUsageRepository(conn)
	ctx := context.Background()

	run := uuid.NewString()[:8]
	tenantID := uuid.New()
	must(t, conn, `INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "WS-"+run, "ws-"+run+"@t.com")
	custID := uuid.New()
	must(t, conn, `INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1,$2,$3,$4,NOW())`,
		custID, tenantID, custID.String()[:8]+"@t.com", uuid.New())
	planID := uuid.New()
	must(t, conn, `INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active) VALUES ($1,$2,'Pro',$3,'month',1,TRUE)`,
		planID, tenantID, "pro-"+run)
	subID := uuid.New()
	must(t, conn, `INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, status, current_period_start, current_period_end, billing_anchor, created_at, updated_at)
		VALUES ($1,$2,$3,$4,'active',NOW(),NOW()+INTERVAL '1 month',NOW(),NOW(),NOW())`,
		subID, tenantID, custID, planID)

	dim := "seats_" + run
	cutoff := time.Now()
	// Before cutoff: +5, +3, -2 => net 6. After cutoff: +10 (excluded).
	for _, e := range []struct {
		qty int64
		at  time.Time
	}{
		{5, cutoff.Add(-72 * time.Hour)},
		{3, cutoff.Add(-48 * time.Hour)},
		{-2, cutoff.Add(-24 * time.Hour)},
		{10, cutoff.Add(24 * time.Hour)},
	} {
		must(t, conn, `INSERT INTO usage_events (id, subscription_id, customer_id, dimension, quantity, timestamp) VALUES ($1,$2,$3,$4,$5,$6)`,
			uuid.New(), subID, custID, dim, e.qty, e.at)
	}

	got, err := repo.SumQuantityBefore(ctx, subID, dim, cutoff)
	if err != nil {
		t.Fatalf("sum before: %v", err)
	}
	if got != 6 {
		t.Fatalf("carry-forward level want 6 (5+3-2, excluding the +10 after cutoff), got %d", got)
	}
}
