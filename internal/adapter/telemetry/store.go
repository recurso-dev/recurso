package telemetry

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// Store persists the anonymous instance identity and the check-once
// milestone flags. The Postgres implementation keeps everything in the
// single-row telemetry_instance table; tests use an in-memory fake.
type Store interface {
	// EnsureInstance returns the instance ID and current milestone flags,
	// creating the single telemetry_instance row (with a fresh random UUID)
	// on first call. No row exists until telemetry is enabled.
	EnsureInstance(ctx context.Context) (uuid.UUID, map[string]bool, error)
	// MarkMilestone sets the named milestone flag and reports whether this
	// call flipped it from false to true. Only the flipping caller sends the
	// event, so a milestone fires at most once per instance — across
	// restarts and across replicas sharing the database.
	MarkMilestone(ctx context.Context, name string) (bool, error)
}

// milestoneColumns whitelists the milestone-name -> column mapping; names
// are never interpolated into SQL directly.
var milestoneColumns = map[string]string{
	milestoneFirstPlan:     "milestone_first_plan",
	milestoneFirstCustomer: "milestone_first_customer",
	milestoneFirstInvoice:  "milestone_first_invoice",
	milestoneFirstPayment:  "milestone_first_payment",
}

// PostgresStore backs Store with the telemetry_instance table
// (migration 000059).
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore wraps db in a PostgresStore.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// EnsureInstance inserts the singleton row with a fresh random UUID if it
// does not exist yet, then reads the identity and milestone flags back.
func (s *PostgresStore) EnsureInstance(ctx context.Context) (uuid.UUID, map[string]bool, error) {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO telemetry_instance (singleton, instance_id) VALUES (TRUE, $1)
		 ON CONFLICT (singleton) DO NOTHING`, uuid.New())
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("telemetry: ensure instance row: %w", err)
	}

	var id uuid.UUID
	var plan, customer, invoice, payment bool
	err = s.db.QueryRowContext(ctx,
		`SELECT instance_id, milestone_first_plan, milestone_first_customer,
		        milestone_first_invoice, milestone_first_payment
		   FROM telemetry_instance WHERE singleton`).
		Scan(&id, &plan, &customer, &invoice, &payment)
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("telemetry: read instance row: %w", err)
	}
	return id, map[string]bool{
		milestoneFirstPlan:     plan,
		milestoneFirstCustomer: customer,
		milestoneFirstInvoice:  invoice,
		milestoneFirstPayment:  payment,
	}, nil
}

// MarkMilestone flips the flag only if it is currently false; the row count
// tells us whether this call won the flip.
func (s *PostgresStore) MarkMilestone(ctx context.Context, name string) (bool, error) {
	col, ok := milestoneColumns[name]
	if !ok {
		return false, fmt.Errorf("telemetry: unknown milestone %q", name)
	}
	res, err := s.db.ExecContext(ctx,
		fmt.Sprintf(`UPDATE telemetry_instance SET %s = TRUE WHERE singleton AND NOT %s`, col, col))
	if err != nil {
		return false, fmt.Errorf("telemetry: mark milestone %s: %w", name, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("telemetry: mark milestone %s: %w", name, err)
	}
	return n == 1, nil
}

// postgresCounts returns a CountsFunc with the exact totals the heartbeat
// buckets before sending.
func postgresCounts(db *sql.DB) CountsFunc {
	return func(ctx context.Context) (int64, int64, error) {
		var tenants, subscriptions int64
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM tenants`).Scan(&tenants); err != nil {
			return 0, 0, err
		}
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM subscriptions`).Scan(&subscriptions); err != nil {
			return 0, 0, err
		}
		return tenants, subscriptions, nil
	}
}
