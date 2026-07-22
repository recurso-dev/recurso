package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

type SubscriptionRepository struct {
	db *sql.DB
}

func NewSubscriptionRepository(db *sql.DB) port.SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

// subscriptionColumns is the single canonical column list for loading a full
// subscription row — the union of everything the individual queries used to
// select. Every full-row SELECT/RETURNING uses it (optionally prefixed for a
// JOIN alias) and every scan goes through scanSubscription, so a new column is
// added in exactly one place and the sites can never drift apart.
const subscriptionColumnList = `id, tenant_id, customer_id, plan_id, status, ` +
	`current_period_start, current_period_end, billing_anchor, ` +
	`billing_anchor_type, billing_anchor_day, payment_terms, ` +
	`cancel_at_period_end, reference_id, razorpay_subscription_id, stripe_subscription_id, ` +
	`trial_end, commitment_amount, created_at, updated_at, resume_at`

// subscriptionColumns returns the canonical column list, optionally prefixed
// (e.g. "s." for aliased JOIN queries).
func subscriptionColumns(prefix string) string {
	if prefix == "" {
		return subscriptionColumnList
	}
	parts := strings.Split(subscriptionColumnList, ", ")
	for i, p := range parts {
		parts[i] = prefix + p
	}
	return strings.Join(parts, ", ")
}

// rowScanner is satisfied by both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanSubscription scans one full subscription row (columns in subscriptionColumnList
// order), handling the nullable columns. The single scan point for the whole repo.
func scanSubscription(row rowScanner) (*domain.Subscription, error) {
	sub := &domain.Subscription{}
	var razorpayID, stripeID, refID, anchorType, paymentTerms sql.NullString
	var billingAnchor, trialEnd, resumeAt sql.NullTime
	var anchorDay sql.NullInt64
	err := row.Scan(
		&sub.ID, &sub.TenantID, &sub.CustomerID, &sub.PlanID, &sub.Status,
		&sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &billingAnchor,
		&anchorType, &anchorDay, &paymentTerms,
		&sub.CancelAtPeriodEnd, &refID, &razorpayID, &stripeID,
		&trialEnd, &sub.CommitmentAmount, &sub.CreatedAt, &sub.UpdatedAt, &resumeAt,
	)
	if err != nil {
		return nil, err
	}
	if billingAnchor.Valid {
		sub.BillingAnchor = billingAnchor.Time
	}
	if trialEnd.Valid {
		sub.TrialEnd = &trialEnd.Time
	}
	if resumeAt.Valid {
		sub.ResumeAt = &resumeAt.Time
	}
	sub.BillingAnchorType = anchorType.String
	sub.BillingAnchorDay = int(anchorDay.Int64)
	sub.PaymentTerms = paymentTerms.String
	sub.ReferenceID = refID.String
	sub.RazorpaySubscriptionID = razorpayID.String
	sub.StripeSubscriptionID = stripeID.String
	return sub, nil
}

func (r *SubscriptionRepository) Create(ctx context.Context, sub *domain.Subscription) error {
	query := `
		INSERT INTO subscriptions (
			id, tenant_id, customer_id, plan_id, status,
			current_period_start, current_period_end, billing_anchor,
			cancel_at_period_end, reference_id, razorpay_subscription_id, stripe_subscription_id,
			trial_end, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`
	_, err := r.db.ExecContext(ctx, query,
		sub.ID, sub.TenantID, sub.CustomerID, sub.PlanID, sub.Status,
		sub.CurrentPeriodStart, sub.CurrentPeriodEnd, sub.BillingAnchor,
		sub.CancelAtPeriodEnd, sub.ReferenceID, sub.RazorpaySubscriptionID, sub.StripeSubscriptionID,
		sub.TrialEnd, sub.CreatedAt, sub.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert subscription: %w", err)
	}
	return nil
}

// CreateWithTx creates a subscription within an existing transaction for atomic operations.
func (r *SubscriptionRepository) CreateWithTx(ctx context.Context, tx *sql.Tx, sub *domain.Subscription) error {
	query := `
		INSERT INTO subscriptions (
			id, tenant_id, customer_id, plan_id, status,
			current_period_start, current_period_end, billing_anchor,
			cancel_at_period_end, reference_id, razorpay_subscription_id, stripe_subscription_id,
			trial_end, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`
	_, err := tx.ExecContext(ctx, query,
		sub.ID, sub.TenantID, sub.CustomerID, sub.PlanID, sub.Status,
		sub.CurrentPeriodStart, sub.CurrentPeriodEnd, sub.BillingAnchor,
		sub.CancelAtPeriodEnd, sub.ReferenceID, sub.RazorpaySubscriptionID, sub.StripeSubscriptionID,
		sub.TrialEnd, sub.CreatedAt, sub.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert subscription in tx: %w", err)
	}
	return nil
}

func (r *SubscriptionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Subscription, error) {
	tenantID, ok := ctx.Value(domain.TenantIDKey).(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("tenant_id missing from context")
	}

	query := `SELECT ` + subscriptionColumns("") + ` FROM subscriptions WHERE id = $1 AND tenant_id = $2`
	sub, err := scanSubscription(r.db.QueryRowContext(ctx, query, id, tenantID))
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}
	return sub, nil
}

// GetByStripeSubscriptionID intentionally has no tenant_id filter: it exists
// for the Stripe webhook handler, which must resolve the owning tenant from
// the subscription itself. A unique index on stripe_subscription_id
// (migration 000047) guarantees at most one match. Do not call this from
// tenant-scoped request paths — use GetByID with a tenant_id instead.
func (r *SubscriptionRepository) GetByStripeSubscriptionID(ctx context.Context, stripeSubID string) (*domain.Subscription, error) {
	query := `SELECT ` + subscriptionColumns("") + ` FROM subscriptions WHERE stripe_subscription_id = $1`
	sub, err := scanSubscription(r.db.QueryRowContext(ctx, query, stripeSubID))
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription by stripe ID: %w", err)
	}
	return sub, nil
}

// GetActiveSubscriptions returns the tenant's active subscriptions. It MUST be
// tenant-scoped — an unscoped variant would leak (and mis-total) other tenants'
// subscriptions into per-tenant analytics like MRR.
// CountActiveByCustomer returns customer_id -> active-subscription count for the
// tenant. Customers with none are simply absent from the map.
func (r *SubscriptionRepository) CountActiveByCustomer(ctx context.Context, tenantID uuid.UUID) (map[uuid.UUID]int, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT customer_id, COUNT(*) FROM subscriptions
		 WHERE tenant_id = $1 AND status = 'active'
		 GROUP BY customer_id`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("count active subscriptions by customer: %w", err)
	}
	defer func() { _ = rows.Close() }()

	counts := make(map[uuid.UUID]int)
	for rows.Next() {
		var customerID uuid.UUID
		var n int
		if err := rows.Scan(&customerID, &n); err != nil {
			return nil, err
		}
		counts[customerID] = n
	}
	return counts, rows.Err()
}

func (r *SubscriptionRepository) GetActiveSubscriptions(ctx context.Context, tenantID uuid.UUID) ([]*domain.Subscription, error) {
	query := `SELECT ` + subscriptionColumns("") + ` FROM subscriptions WHERE status = 'active' AND tenant_id = $1`
	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query active subscriptions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var subs []*domain.Subscription
	for rows.Next() {
		sub, err := scanSubscription(rows)
		if err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, nil
}

func (r *SubscriptionRepository) List(ctx context.Context, tenantID uuid.UUID, filter domain.SubscriptionFilter) ([]*domain.Subscription, error) {
	query := `
		SELECT ` + subscriptionColumns("s.") + `
		FROM subscriptions s
		LEFT JOIN customers c ON s.customer_id = c.id
		WHERE s.tenant_id = $1
	`
	args := []interface{}{tenantID}
	argIdx := 2

	if filter.Status != "" {
		query += fmt.Sprintf(" AND s.status = $%d", argIdx)
		args = append(args, filter.Status)
		argIdx++
	}

	if filter.Search != "" {
		query += fmt.Sprintf(" AND (c.name ILIKE $%d OR c.email ILIKE $%d OR s.id::text ILIKE $%d)", argIdx, argIdx, argIdx)
		searchPattern := "%" + filter.Search + "%"
		args = append(args, searchPattern)
		argIdx++
	}

	if filter.CustomerID != uuid.Nil {
		query += fmt.Sprintf(" AND s.customer_id = $%d", argIdx)
		args = append(args, filter.CustomerID)
		argIdx++
	}

	query += " ORDER BY s.created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, filter.Limit)
		argIdx++
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var subs []*domain.Subscription
	for rows.Next() {
		sub, err := scanSubscription(rows)
		if err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, nil
}

// Update updates a subscription
func (r *SubscriptionRepository) Update(ctx context.Context, sub *domain.Subscription) error {
	// plan_id MUST be in this SET: UpdateSubscription changes plans by
	// mutating the struct and calling Update — omitting it made plan changes
	// silently no-op while the proration invoice was still issued.
	query := `
		UPDATE subscriptions SET
			plan_id = $1,
			status = $2,
			current_period_start = $3,
			current_period_end = $4,
			cancel_at_period_end = $5,
			canceled_at = $6,
			cancellation_reason = $7,
			cancellation_feedback = $8,
			razorpay_subscription_id = $9,
			stripe_subscription_id = $10,
			trial_end = $11,
			updated_at = $12
		WHERE id = $13 AND tenant_id = $14
	`
	_, err := r.db.ExecContext(ctx, query, subUpdateArgs(sub)...)
	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}
	return nil
}

// UpdateWithTx is Update inside a caller-provided transaction, so a plan change
// (invoice + subscription) can be committed atomically (ENG-150).
// ActivateTrialWithTx atomically transitions a subscription from trialing to
// active and returns whether THIS caller performed the transition. The
// `AND status = 'trialing'` guard means only one of several concurrent trial-
// conversion runners flips the row; the losers match zero rows and get false,
// so only the winner goes on to create the first invoice (ENG-161). Mirrors the
// atomic MarkPaid pattern used for invoice settlement.
func (r *SubscriptionRepository) ActivateTrialWithTx(ctx context.Context, tx *sql.Tx, sub *domain.Subscription) (bool, error) {
	res, err := tx.ExecContext(ctx,
		`UPDATE subscriptions
		 SET status = $1, current_period_start = $2, current_period_end = $3, updated_at = $4
		 WHERE id = $5 AND tenant_id = $6 AND status = 'trialing'`,
		sub.Status, sub.CurrentPeriodStart, sub.CurrentPeriodEnd, sub.UpdatedAt, sub.ID, sub.TenantID)
	if err != nil {
		return false, fmt.Errorf("failed to activate trial (tx): %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to read activate-trial result: %w", err)
	}
	return n == 1, nil
}

func (r *SubscriptionRepository) UpdateWithTx(ctx context.Context, tx *sql.Tx, sub *domain.Subscription) error {
	query := `
		UPDATE subscriptions SET
			plan_id = $1, status = $2, current_period_start = $3, current_period_end = $4,
			cancel_at_period_end = $5, canceled_at = $6, cancellation_reason = $7,
			cancellation_feedback = $8, razorpay_subscription_id = $9, stripe_subscription_id = $10,
			trial_end = $11, updated_at = $12
		WHERE id = $13 AND tenant_id = $14
	`
	if _, err := tx.ExecContext(ctx, query, subUpdateArgs(sub)...); err != nil {
		return fmt.Errorf("failed to update subscription (tx): %w", err)
	}
	return nil
}

func subUpdateArgs(sub *domain.Subscription) []interface{} {
	return []interface{}{
		sub.PlanID, sub.Status, sub.CurrentPeriodStart, sub.CurrentPeriodEnd,
		sub.CancelAtPeriodEnd, sub.CanceledAt, sub.CancellationReason, sub.CancellationFeedback,
		sub.RazorpaySubscriptionID, sub.StripeSubscriptionID, sub.TrialEnd, sub.UpdatedAt,
		sub.ID, sub.TenantID,
	}
}

// GetSubscriptionsDueTomorrow returns active subscriptions that renew tomorrow
// and haven't had a pre-charge notification sent yet
func (r *SubscriptionRepository) GetSubscriptionsDueTomorrow(ctx context.Context) ([]SubscriptionWithCustomer, error) {
	query := `
		SELECT 
			s.id, s.tenant_id, s.customer_id, s.plan_id, s.status,
			s.current_period_end,
			c.name as customer_name, c.email as customer_email,
			p.name as plan_name, pr.amount as amount, pr.currency
		FROM subscriptions s
		JOIN customers c ON s.customer_id = c.id
		JOIN plans p ON s.plan_id = p.id
		JOIN prices pr ON pr.plan_id = p.id
		LEFT JOIN precharge_notifications pn ON pn.subscription_id = s.id 
			AND pn.scheduled_charge_date = DATE(s.current_period_end)
		WHERE s.status = 'active'
			AND s.current_period_end >= CURRENT_TIMESTAMP
			AND s.current_period_end < CURRENT_TIMESTAMP + INTERVAL '25 hours'
			AND pn.id IS NULL
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query subscriptions due tomorrow: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var subs []SubscriptionWithCustomer
	for rows.Next() {
		var sub SubscriptionWithCustomer
		// current_period_end is timestamptz (scan into time.Time, not string);
		// customer name is nullable.
		var nextBilling time.Time
		var customerName sql.NullString
		if err := rows.Scan(
			&sub.ID, &sub.TenantID, &sub.CustomerID, &sub.PlanID, &sub.Status,
			&nextBilling,
			&customerName, &sub.CustomerEmail,
			&sub.PlanName, &sub.Amount, &sub.Currency,
		); err != nil {
			return nil, err
		}
		sub.NextBillingDate = nextBilling.Format("January 2, 2006")
		sub.CustomerName = customerName.String
		subs = append(subs, sub)
	}
	return subs, nil
}

// MarkPreChargeNotificationSent records that a pre-charge notification was sent
func (r *SubscriptionRepository) MarkPreChargeNotificationSent(ctx context.Context, subscriptionID uuid.UUID, chargeDate string) error {
	// First get the subscription to get tenant_id and customer_id
	var tenantID, customerID uuid.UUID
	var amount int64
	var currency string

	// Pricing lives in the prices table, not plans (plans has no price/currency
	// column — the old query errored on every call, so the notification was
	// never marked sent and the customer was re-emailed every tick).
	err := r.db.QueryRowContext(ctx, `
		SELECT s.tenant_id, s.customer_id, pr.amount, pr.currency
		FROM subscriptions s
		JOIN prices pr ON pr.plan_id = s.plan_id
		WHERE s.id = $1
		LIMIT 1
	`, subscriptionID).Scan(&tenantID, &customerID, &amount, &currency)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	query := `
		INSERT INTO precharge_notifications (
			tenant_id, subscription_id, customer_id,
			scheduled_charge_date, amount, currency,
			notification_sent_at, notification_type
		)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), 'email')
		ON CONFLICT DO NOTHING
	`
	_, err = r.db.ExecContext(ctx, query,
		tenantID, subscriptionID, customerID,
		chargeDate, amount, currency,
	)
	if err != nil {
		return fmt.Errorf("failed to mark pre-charge notification sent: %w", err)
	}
	return nil
}

// GetExpiredTrials returns trialing subscriptions whose trial_end has passed.
// Cross-tenant by design: the trial scheduler runs globally (like the dunning
// and pre-charge jobs) and resolves the owning tenant from each row.
func (r *SubscriptionRepository) GetExpiredTrials(ctx context.Context) ([]*domain.Subscription, error) {
	query := `
		SELECT ` + subscriptionColumns("") + `
		FROM subscriptions
		WHERE status = 'trialing'
			AND trial_end IS NOT NULL
			AND trial_end <= CURRENT_TIMESTAMP
		ORDER BY trial_end ASC
		LIMIT 100
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query expired trials: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var subs []*domain.Subscription
	for rows.Next() {
		sub, err := scanSubscription(rows)
		if err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, rows.Err()
}

// SetCommitment sets the subscription's per-period minimum (Lago-parity
// B2). Returns sql.ErrNoRows for an unknown/foreign subscription.
func (r *SubscriptionRepository) SetCommitment(ctx context.Context, tenantID, subscriptionID uuid.UUID, amount int64) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE subscriptions SET commitment_amount = $3, updated_at = NOW() WHERE tenant_id = $1 AND id = $2`,
		tenantID, subscriptionID, amount)
	if err != nil {
		return fmt.Errorf("failed to set commitment: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ClaimDueForRenewal atomically claims up to limit ACTIVE, locally-billed
// subscriptions whose billing period has ended (Lago-parity A1). Locally
// billed = no UPI mandate and no gateway-managed cycle — those renew through
// their own flows. The claim leases renewal_claimed_at forward so exactly one
// runner processes a subscription per lease window even without Redis;
// FOR UPDATE SKIP LOCKED keeps concurrent claimers from blocking each other.
// A successful renewal advances current_period_end (undue); a failed one
// simply lets the lease lapse and retries on a later tick.
func (r *SubscriptionRepository) ClaimDueForRenewal(ctx context.Context, lease time.Duration, limit int) ([]*domain.Subscription, error) {
	query := `
		UPDATE subscriptions SET renewal_claimed_at = NOW()
		WHERE id IN (
			SELECT id FROM subscriptions
			WHERE status = 'active'
				AND current_period_end <= NOW()
				AND mandate_id IS NULL
				AND COALESCE(razorpay_subscription_id, '') = ''
				AND COALESCE(stripe_subscription_id, '') = ''
				AND (renewal_claimed_at IS NULL OR renewal_claimed_at < NOW() - $1 * INTERVAL '1 second')
			ORDER BY current_period_end
			LIMIT $2
			FOR UPDATE SKIP LOCKED
		)
		RETURNING ` + subscriptionColumns("") + `
	`
	rows, err := r.db.QueryContext(ctx, query, int64(lease.Seconds()), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to claim due renewals: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var subs []*domain.Subscription
	for rows.Next() {
		sub, err := scanSubscription(rows)
		if err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, rows.Err()
}

// SetResumeAt records (or clears, when resumeAt is nil) the scheduled
// auto-resume time for a subscription (issue #111). It is a targeted write that
// touches only resume_at — deliberately NOT routed through the full-row Update,
// so a caller that hasn't loaded resume_at can never clobber it (the ENG-144
// class of bug). tenant_id is scoped in the WHERE (defense-in-depth).
func (r *SubscriptionRepository) SetResumeAt(ctx context.Context, tenantID, subID uuid.UUID, resumeAt *time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE subscriptions SET resume_at = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3`,
		resumeAt, subID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to set resume_at: %w", err)
	}
	return nil
}

// ClaimDueForResume atomically leases up to `limit` paused subscriptions whose
// scheduled resume_at has elapsed, for the calling scheduler instance. It pushes
// resume_at forward by `lease` so a second instance (the distributed lock is a
// no-op without Redis) can't claim the same rows this tick (ADR-003);
// FOR UPDATE SKIP LOCKED keeps concurrent claimers disjoint. The caller resumes
// each (which sets status active and clears resume_at); if it dies mid-resume
// the lease lapses and the row is retried on a later tick.
func (r *SubscriptionRepository) ClaimDueForResume(ctx context.Context, lease time.Duration, limit int) ([]*domain.Subscription, error) {
	query := `
		UPDATE subscriptions SET resume_at = NOW() + $1 * INTERVAL '1 second'
		WHERE id IN (
			SELECT id FROM subscriptions
			WHERE status = 'paused' AND resume_at IS NOT NULL AND resume_at <= NOW()
			ORDER BY resume_at
			LIMIT $2
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, tenant_id, customer_id, plan_id, status,
			current_period_start, current_period_end
	`
	rows, err := r.db.QueryContext(ctx, query, int64(lease.Seconds()), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to claim subscriptions due for resume: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var subs []*domain.Subscription
	for rows.Next() {
		sub := &domain.Subscription{}
		if err := rows.Scan(
			&sub.ID, &sub.TenantID, &sub.CustomerID, &sub.PlanID, &sub.Status,
			&sub.CurrentPeriodStart, &sub.CurrentPeriodEnd,
		); err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, rows.Err()
}

// TrialEndingNotice carries the data needed to email a trial-ending reminder.
type TrialEndingNotice struct {
	SubscriptionID uuid.UUID
	TenantID       uuid.UUID
	CustomerID     uuid.UUID
	CustomerName   string
	CustomerEmail  string
	PlanName       string
	Amount         int64
	Currency       string
	TrialEnd       time.Time
}

// GetTrialsEndingWithin returns trialing subscriptions whose trial_end falls
// inside (now, now+within] and that have not yet had a reminder sent.
func (r *SubscriptionRepository) GetTrialsEndingWithin(ctx context.Context, within time.Duration) ([]TrialEndingNotice, error) {
	query := `
		SELECT
			s.id, s.tenant_id, s.customer_id, s.trial_end,
			c.name, c.email,
			p.name, pr.amount, pr.currency
		FROM subscriptions s
		JOIN customers c ON s.customer_id = c.id
		JOIN plans p ON s.plan_id = p.id
		JOIN prices pr ON pr.plan_id = p.id
		WHERE s.status = 'trialing'
			AND s.trial_reminder_sent = FALSE
			AND s.trial_end IS NOT NULL
			AND s.trial_end > CURRENT_TIMESTAMP
			AND s.trial_end <= CURRENT_TIMESTAMP + make_interval(secs => $1)
		ORDER BY s.trial_end ASC
		LIMIT 100
	`
	rows, err := r.db.QueryContext(ctx, query, within.Seconds())
	if err != nil {
		return nil, fmt.Errorf("failed to query trials ending soon: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var notices []TrialEndingNotice
	for rows.Next() {
		var n TrialEndingNotice
		var name sql.NullString
		if err := rows.Scan(
			&n.SubscriptionID, &n.TenantID, &n.CustomerID, &n.TrialEnd,
			&name, &n.CustomerEmail,
			&n.PlanName, &n.Amount, &n.Currency,
		); err != nil {
			return nil, err
		}
		n.CustomerName = name.String
		notices = append(notices, n)
	}
	return notices, rows.Err()
}

// MarkTrialReminderSent flags a subscription so its trial-ending reminder is not
// sent twice.
func (r *SubscriptionRepository) MarkTrialReminderSent(ctx context.Context, subscriptionID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE subscriptions SET trial_reminder_sent = TRUE WHERE id = $1`, subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to mark trial reminder sent: %w", err)
	}
	return nil
}

// SubscriptionWithCustomer contains subscription info with customer details for notifications
type SubscriptionWithCustomer struct {
	ID                 uuid.UUID
	TenantID           uuid.UUID
	CustomerID         uuid.UUID
	PlanID             uuid.UUID
	Status             string
	NextBillingDate    string
	CustomerName       string
	CustomerEmail      string
	PlanName           string
	Amount             int64
	Currency           string
	PaymentMethodLast4 string
}
