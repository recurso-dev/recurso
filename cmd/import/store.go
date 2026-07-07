package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// store abstracts the data-access operations that the idempotent match,
// -update, and -cancel-missing paths rely on. The production implementation
// (dbStore) issues tenant-scoped SQL; tests inject a fake so the sync logic is
// exercised without a database. Only provider-safe columns are ever written —
// nothing here re-links an entity to a different customer/plan, so a match can
// never turn into a duplicate.
type store interface {
	// CustomerIDByEmail resolves an existing customer by tenant + lower(email).
	CustomerIDByEmail(ctx context.Context, tenantID uuid.UUID, email string) (id uuid.UUID, found bool, err error)
	// UpdateCustomer writes the provider-safe customer fields (name, country,
	// tax_id, gstin, place_of_supply). Empty inputs leave the stored value
	// untouched so a sparse import never blanks existing data.
	UpdateCustomer(ctx context.Context, tenantID, id uuid.UUID, in CustomerInput) error

	// PlanByCode resolves an existing plan by tenant + code.
	PlanByCode(ctx context.Context, tenantID uuid.UUID, code string) (*existingPlan, error)
	// UpdatePlan writes the safe plan fields (name, and the price amount for the
	// input currency). It never changes code, interval, or currency.
	UpdatePlan(ctx context.Context, tenantID uuid.UUID, plan existingPlan, in PlanInput) error

	// SubscriptionIDByExternalID resolves an existing subscription by tenant +
	// reference_id (the source-system external_id / idempotency key).
	SubscriptionIDByExternalID(ctx context.Context, tenantID uuid.UUID, externalID string) (id uuid.UUID, found bool, err error)
	// UpdateSubscription writes only status + period. It never re-links the
	// subscription to a different customer or plan.
	UpdateSubscription(ctx context.Context, tenantID, id uuid.UUID, in SubscriptionInput) error

	// ListSubscriptions returns every not-yet-canceled subscription for the
	// tenant, used by cancel-sync to find records absent from the import file.
	ListSubscriptions(ctx context.Context, tenantID uuid.UUID) ([]existingSubscription, error)
	// CancelSubscription schedules a period-end cancellation. The WHERE clause
	// hard-guards against ever touching a dashboard-created subscription (one
	// with no reference_id).
	CancelSubscription(ctx context.Context, tenantID, id uuid.UUID, reason string) error
}

// existingPlan is the minimal view of a stored plan the importer needs to
// match and update it.
type existingPlan struct {
	ID   uuid.UUID
	Name string
}

// existingSubscription is the minimal view of a stored subscription used by
// cancel-sync. ReferenceID is empty for dashboard-created subscriptions.
type existingSubscription struct {
	ID          uuid.UUID
	ReferenceID string
	Status      string
}

// cancelSyncReason is recorded on subscriptions canceled because they were
// absent from an authoritative import.
const cancelSyncReason = "cancel_sync: absent from authoritative import"

// dbStore is the production store backed by Postgres.
type dbStore struct {
	db *sqlx.DB
}

func (s *dbStore) CustomerIDByEmail(ctx context.Context, tenantID uuid.UUID, email string) (uuid.UUID, bool, error) {
	var id uuid.UUID
	err := s.db.QueryRowContext(ctx,
		`SELECT id FROM customers WHERE tenant_id = $1 AND lower(email) = $2`,
		tenantID, strings.ToLower(email)).Scan(&id)
	if err == sql.ErrNoRows {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("lookup customer %s: %w", email, err)
	}
	return id, true, nil
}

func (s *dbStore) UpdateCustomer(ctx context.Context, tenantID, id uuid.UUID, in CustomerInput) error {
	// COALESCE(NULLIF(...,''), col) keeps the stored value when the import
	// omits a field, so a partial export never blanks existing data.
	query := `
		UPDATE customers SET
			name            = COALESCE(NULLIF($1, ''), name),
			country         = COALESCE(NULLIF($2, ''), country),
			tax_id          = COALESCE(NULLIF($3, ''), tax_id),
			gstin           = COALESCE(NULLIF($4, ''), gstin),
			place_of_supply = COALESCE(NULLIF($5, ''), place_of_supply),
			updated_at      = NOW()
		WHERE id = $6 AND tenant_id = $7
	`
	_, err := s.db.ExecContext(ctx, query,
		in.Name, in.Country, in.TaxID, in.GSTIN, in.PlaceOfSupply, id, tenantID)
	if err != nil {
		return fmt.Errorf("update customer %s: %w", in.Email, err)
	}
	return nil
}

func (s *dbStore) PlanByCode(ctx context.Context, tenantID uuid.UUID, code string) (*existingPlan, error) {
	var p existingPlan
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name FROM plans WHERE tenant_id = $1 AND code = $2`,
		tenantID, code).Scan(&p.ID, &p.Name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lookup plan %s: %w", code, err)
	}
	return &p, nil
}

func (s *dbStore) UpdatePlan(ctx context.Context, tenantID uuid.UUID, plan existingPlan, in PlanInput) error {
	if _, err := s.db.ExecContext(ctx,
		`UPDATE plans SET name = COALESCE(NULLIF($1, ''), name) WHERE id = $2 AND tenant_id = $3`,
		in.Name, plan.ID, tenantID); err != nil {
		return fmt.Errorf("update plan %s: %w", in.Code, err)
	}
	// Amount is only applied to the price row matching the import currency; a
	// missing currency match updates nothing (never creates or re-currencies).
	if _, err := s.db.ExecContext(ctx,
		`UPDATE prices SET amount = $1 WHERE plan_id = $2 AND currency = $3`,
		in.Amount, plan.ID, strings.ToUpper(in.Currency)); err != nil {
		return fmt.Errorf("update plan %s price: %w", in.Code, err)
	}
	return nil
}

func (s *dbStore) SubscriptionIDByExternalID(ctx context.Context, tenantID uuid.UUID, externalID string) (uuid.UUID, bool, error) {
	var id uuid.UUID
	err := s.db.QueryRowContext(ctx,
		`SELECT id FROM subscriptions WHERE tenant_id = $1 AND reference_id = $2`,
		tenantID, externalID).Scan(&id)
	if err == sql.ErrNoRows {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("lookup subscription %s: %w", externalID, err)
	}
	return id, true, nil
}

func (s *dbStore) UpdateSubscription(ctx context.Context, tenantID, id uuid.UUID, in SubscriptionInput) error {
	start, _ := time.Parse(time.RFC3339, in.CurrentPeriodStart) // validated earlier
	end, _ := time.Parse(time.RFC3339, in.CurrentPeriodEnd)
	query := `
		UPDATE subscriptions SET
			status               = $1,
			current_period_start = $2,
			current_period_end   = $3,
			updated_at           = NOW()
		WHERE id = $4 AND tenant_id = $5
	`
	if _, err := s.db.ExecContext(ctx, query, in.Status, start, end, id, tenantID); err != nil {
		return fmt.Errorf("update subscription %s: %w", in.ExternalID, err)
	}
	return nil
}

func (s *dbStore) ListSubscriptions(ctx context.Context, tenantID uuid.UUID) ([]existingSubscription, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, COALESCE(reference_id, ''), status
		 FROM subscriptions
		 WHERE tenant_id = $1 AND status <> 'canceled'`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list subscriptions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []existingSubscription
	for rows.Next() {
		var e existingSubscription
		if err := rows.Scan(&e.ID, &e.ReferenceID, &e.Status); err != nil {
			return nil, fmt.Errorf("scan subscription: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *dbStore) CancelSubscription(ctx context.Context, tenantID, id uuid.UUID, reason string) error {
	// reference_id guard: a defense-in-depth backstop so this can never cancel a
	// dashboard-created subscription even if the caller's filtering regresses.
	query := `
		UPDATE subscriptions SET
			cancel_at_period_end = TRUE,
			canceled_at          = COALESCE(canceled_at, NOW()),
			cancellation_reason  = $1,
			updated_at           = NOW()
		WHERE id = $2 AND tenant_id = $3
		  AND reference_id IS NOT NULL AND reference_id <> ''
	`
	res, err := s.db.ExecContext(ctx, query, reason, id, tenantID)
	if err != nil {
		return fmt.Errorf("cancel subscription %s: %w", id, err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("cancel subscription %s: no import-origin row matched (refusing to cancel)", id)
	}
	return nil
}
