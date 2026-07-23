package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// MRRSnapshotRepository is the Postgres store for per-subscription MRR history.
type MRRSnapshotRepository struct {
	db *sql.DB
}

func NewMRRSnapshotRepository(db *sql.DB) *MRRSnapshotRepository {
	return &MRRSnapshotRepository{db: db}
}

// UpsertSnapshots writes a batch of snapshots for one date, replacing any
// existing rows for the same (tenant, subscription, date). Safe to re-run for a
// date (the capture job is idempotent).
func (r *MRRSnapshotRepository) UpsertSnapshots(ctx context.Context, snaps []domain.MRRSnapshot) error {
	if len(snaps) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString(`INSERT INTO mrr_snapshots
		(tenant_id, subscription_id, snapshot_date, mrr_amount, currency, customer_id, plan_id, entity_id) VALUES `)
	const cols = 8
	args := make([]interface{}, 0, len(snaps)*cols)
	for i, s := range snaps {
		if i > 0 {
			b.WriteString(", ")
		}
		n := i * cols
		fmt.Fprintf(&b, "($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)", n+1, n+2, n+3, n+4, n+5, n+6, n+7, n+8)
		args = append(args, s.TenantID, s.SubscriptionID, s.SnapshotDate, s.MRRAmount, s.Currency, s.CustomerID, s.PlanID, s.EntityID)
	}
	b.WriteString(`
		ON CONFLICT (tenant_id, subscription_id, snapshot_date) DO UPDATE SET
			mrr_amount  = EXCLUDED.mrr_amount,
			currency    = EXCLUDED.currency,
			customer_id = EXCLUDED.customer_id,
			plan_id     = EXCLUDED.plan_id,
			entity_id   = EXCLUDED.entity_id`)
	_, err := r.db.ExecContext(ctx, b.String(), args...)
	return err
}

// ResolveSnapshotDate returns the most recent snapshot_date on or before the
// given date for a tenant, and whether any exists. A period boundary rarely
// lands exactly on a snapshot day, so callers resolve it to the nearest prior
// captured day.
func (r *MRRSnapshotRepository) ResolveSnapshotDate(ctx context.Context, tenantID uuid.UUID, entityID *uuid.UUID, onOrBefore time.Time) (time.Time, bool, error) {
	var d sql.NullTime
	q := `SELECT MAX(snapshot_date) FROM mrr_snapshots WHERE tenant_id = $1 AND snapshot_date <= $2`
	args := []interface{}{tenantID, onOrBefore}
	if entityID != nil {
		q += ` AND entity_id = $3`
		args = append(args, *entityID)
	}
	err := r.db.QueryRowContext(ctx, q, args...).Scan(&d)
	if err != nil {
		return time.Time{}, false, err
	}
	if !d.Valid {
		return time.Time{}, false, nil
	}
	return d.Time, true, nil
}

// GetSnapshotsOn returns every subscription's snapshot for a tenant on an exact
// snapshot date (use ResolveSnapshotDate to map a period boundary to one).
func (r *MRRSnapshotRepository) GetSnapshotsOn(ctx context.Context, tenantID uuid.UUID, entityID *uuid.UUID, date time.Time) ([]domain.MRRSnapshot, error) {
	q := `SELECT tenant_id, subscription_id, snapshot_date, mrr_amount, currency, customer_id, plan_id, entity_id
		   FROM mrr_snapshots WHERE tenant_id = $1 AND snapshot_date = $2`
	args := []interface{}{tenantID, date}
	if entityID != nil {
		q += ` AND entity_id = $3`
		args = append(args, *entityID)
	}
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []domain.MRRSnapshot
	for rows.Next() {
		var s domain.MRRSnapshot
		var cust, plan, entity uuid.NullUUID
		if err := rows.Scan(&s.TenantID, &s.SubscriptionID, &s.SnapshotDate, &s.MRRAmount, &s.Currency, &cust, &plan, &entity); err != nil {
			return nil, err
		}
		if cust.Valid {
			s.CustomerID = &cust.UUID
		}
		if plan.Valid {
			s.PlanID = &plan.UUID
		}
		if entity.Valid {
			s.EntityID = &entity.UUID
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// SubscriptionIDsSeenBefore returns the set of subscription IDs that had any
// snapshot strictly before the given date — used to tell a reactivation (seen
// before, gone at start, back at end) from a genuinely new subscription.
func (r *MRRSnapshotRepository) SubscriptionIDsSeenBefore(ctx context.Context, tenantID uuid.UUID, entityID *uuid.UUID, date time.Time) (map[uuid.UUID]bool, error) {
	q := `SELECT DISTINCT subscription_id FROM mrr_snapshots WHERE tenant_id = $1 AND snapshot_date < $2`
	args := []interface{}{tenantID, date}
	if entityID != nil {
		q += ` AND entity_id = $3`
		args = append(args, *entityID)
	}
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	seen := make(map[uuid.UUID]bool)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		seen[id] = true
	}
	return seen, rows.Err()
}
