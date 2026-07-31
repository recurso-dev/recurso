package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// PlatformRepository serves cross-tenant, operator-only aggregates (the founder
// metrics view). It is deliberately NOT reachable through any tenant-scoped
// handler — only the FOUNDER_TOKEN endpoint calls it.
type PlatformRepository struct {
	db *sql.DB
}

func NewPlatformRepository(db *sql.DB) *PlatformRepository {
	return &PlatformRepository{db: db}
}

// PlatformMetrics aggregates the managed-cloud funnel across all tenants:
// signups, activation (>=1 customer created), trials expiring soon, and the
// billing-status / plan-tier breakdowns, plus the newest signups.
func (r *PlatformRepository) PlatformMetrics(ctx context.Context) (*domain.PlatformMetrics, error) {
	m := &domain.PlatformMetrics{
		ByBillingStatus: map[string]int{},
		ByPlanTier:      map[string]int{},
		GeneratedAt:     time.Now().UTC(),
	}

	// Totals + windowed signups + activation + trials-expiring, in one round-trip.
	err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '7 days'),
			COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '30 days'),
			COUNT(*) FILTER (WHERE billing_status = 'trialing'
			                 AND trial_ends_at IS NOT NULL
			                 AND trial_ends_at BETWEEN NOW() AND NOW() + INTERVAL '7 days'),
			(SELECT COUNT(*) FROM tenants t WHERE EXISTS (SELECT 1 FROM customers c WHERE c.tenant_id = t.id))
		FROM tenants
	`).Scan(&m.TotalTenants, &m.SignupsLast7d, &m.SignupsLast30d, &m.TrialsExpiring7d, &m.ActivatedTenants)
	if err != nil {
		return nil, err
	}

	if err := scanTenantCounts(ctx, r.db, `SELECT billing_status, COUNT(*) FROM tenants GROUP BY billing_status`, m.ByBillingStatus); err != nil {
		return nil, err
	}
	if err := scanTenantCounts(ctx, r.db, `SELECT plan_tier, COUNT(*) FROM tenants GROUP BY plan_tier`, m.ByPlanTier); err != nil {
		return nil, err
	}

	// Newest 15 signups, each flagged activated (has >=1 customer).
	rows, err := r.db.QueryContext(ctx, `
		SELECT COALESCE(t.name, ''), COALESCE(t.email, ''), t.plan_tier, t.billing_status,
		       t.trial_ends_at, t.created_at,
		       EXISTS (SELECT 1 FROM customers c WHERE c.tenant_id = t.id)
		FROM tenants t
		ORDER BY t.created_at DESC
		LIMIT 15
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var s domain.PlatformSignup
		var trialEnds sql.NullTime
		if err := rows.Scan(&s.Name, &s.Email, &s.PlanTier, &s.BillingStatus, &trialEnds, &s.CreatedAt, &s.Activated); err != nil {
			return nil, err
		}
		if trialEnds.Valid {
			s.TrialEndsAt = &trialEnds.Time
		}
		m.RecentSignups = append(m.RecentSignups, s)
	}
	return m, rows.Err()
}

func scanTenantCounts(ctx context.Context, db *sql.DB, query string, into map[string]int) error {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var key string
		var n int
		if err := rows.Scan(&key, &n); err != nil {
			return err
		}
		into[key] = n
	}
	return rows.Err()
}
