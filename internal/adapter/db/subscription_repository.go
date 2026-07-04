package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
	"github.com/recur-so/recurso/internal/core/port"
)

type SubscriptionRepository struct {
	db *sql.DB
}

func NewSubscriptionRepository(db *sql.DB) port.SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

func (r *SubscriptionRepository) Create(ctx context.Context, sub *domain.Subscription) error {
	query := `
		INSERT INTO subscriptions (
			id, tenant_id, customer_id, plan_id, status,
			current_period_start, current_period_end, billing_anchor,
			cancel_at_period_end, reference_id, razorpay_subscription_id, stripe_subscription_id,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	_, err := r.db.ExecContext(ctx, query,
		sub.ID, sub.TenantID, sub.CustomerID, sub.PlanID, sub.Status,
		sub.CurrentPeriodStart, sub.CurrentPeriodEnd, sub.BillingAnchor,
		sub.CancelAtPeriodEnd, sub.ReferenceID, sub.RazorpaySubscriptionID, sub.StripeSubscriptionID,
		sub.CreatedAt, sub.UpdatedAt,
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
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	_, err := tx.ExecContext(ctx, query,
		sub.ID, sub.TenantID, sub.CustomerID, sub.PlanID, sub.Status,
		sub.CurrentPeriodStart, sub.CurrentPeriodEnd, sub.BillingAnchor,
		sub.CancelAtPeriodEnd, sub.ReferenceID, sub.RazorpaySubscriptionID, sub.StripeSubscriptionID,
		sub.CreatedAt, sub.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert subscription in tx: %w", err)
	}
	return nil
}

func (r *SubscriptionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Subscription, error) {
	tenantID, ok := ctx.Value("tenant_id").(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("tenant_id missing from context")
	}

	sub := &domain.Subscription{}
	query := `
		SELECT 
			id, tenant_id, customer_id, plan_id, status,
			current_period_start, current_period_end, billing_anchor,
			cancel_at_period_end, reference_id, razorpay_subscription_id, stripe_subscription_id,
			created_at, updated_at
		FROM subscriptions WHERE id = $1 AND tenant_id = $2
	`
	var razorpayID, stripeID, refID sql.NullString
	var billingAnchor sql.NullTime
	err := r.db.QueryRowContext(ctx, query, id, tenantID).Scan(
		&sub.ID, &sub.TenantID, &sub.CustomerID, &sub.PlanID, &sub.Status,
		&sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &billingAnchor,
		&sub.CancelAtPeriodEnd, &refID, &razorpayID, &stripeID,
		&sub.CreatedAt, &sub.UpdatedAt,
	)
	if billingAnchor.Valid {
		sub.BillingAnchor = billingAnchor.Time
	}
	sub.ReferenceID = refID.String
	sub.RazorpaySubscriptionID = razorpayID.String
	sub.StripeSubscriptionID = stripeID.String
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
	sub := &domain.Subscription{}
	query := `
		SELECT
			id, tenant_id, customer_id, plan_id, status,
			current_period_start, current_period_end, billing_anchor,
			cancel_at_period_end, reference_id, razorpay_subscription_id, stripe_subscription_id,
			created_at, updated_at
		FROM subscriptions WHERE stripe_subscription_id = $1
	`
	var razorpayID, stripeID, refID sql.NullString
	var billingAnchor sql.NullTime
	err := r.db.QueryRowContext(ctx, query, stripeSubID).Scan(
		&sub.ID, &sub.TenantID, &sub.CustomerID, &sub.PlanID, &sub.Status,
		&sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &billingAnchor,
		&sub.CancelAtPeriodEnd, &refID, &razorpayID, &stripeID,
		&sub.CreatedAt, &sub.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription by stripe ID: %w", err)
	}
	if billingAnchor.Valid {
		sub.BillingAnchor = billingAnchor.Time
	}
	sub.ReferenceID = refID.String
	sub.RazorpaySubscriptionID = razorpayID.String
	sub.StripeSubscriptionID = stripeID.String
	return sub, nil
}

func (r *SubscriptionRepository) GetActiveSubscriptions(ctx context.Context) ([]*domain.Subscription, error) {
	query := `
		SELECT 
			id, tenant_id, customer_id, plan_id, status,
			current_period_start, current_period_end, billing_anchor,
			cancel_at_period_end, razorpay_subscription_id, stripe_subscription_id,
			created_at, updated_at
		FROM subscriptions 
		WHERE status = 'active'
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []*domain.Subscription
	for rows.Next() {
		sub := &domain.Subscription{}
		var razorpayID, stripeID sql.NullString
		var billingAnchor sql.NullTime
		if err := rows.Scan(
			&sub.ID, &sub.TenantID, &sub.CustomerID, &sub.PlanID, &sub.Status,
			&sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &billingAnchor,
			&sub.CancelAtPeriodEnd, &razorpayID, &stripeID,
			&sub.CreatedAt, &sub.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if billingAnchor.Valid {
			sub.BillingAnchor = billingAnchor.Time
		}
		sub.RazorpaySubscriptionID = razorpayID.String
		sub.StripeSubscriptionID = stripeID.String
		subs = append(subs, sub)
	}
	return subs, nil
}

func (r *SubscriptionRepository) List(ctx context.Context, tenantID uuid.UUID, filter domain.SubscriptionFilter) ([]*domain.Subscription, error) {
	query := `
		SELECT 
			s.id, s.tenant_id, s.customer_id, s.plan_id, s.status,
			s.current_period_start, s.current_period_end, s.billing_anchor,
			s.cancel_at_period_end, s.reference_id, s.razorpay_subscription_id, s.stripe_subscription_id,
			s.created_at, s.updated_at
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
		argIdx++
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []*domain.Subscription
	for rows.Next() {
		sub := &domain.Subscription{}
		var razorpayID, stripeID, refID sql.NullString
		var billingAnchor sql.NullTime
		if err := rows.Scan(
			&sub.ID, &sub.TenantID, &sub.CustomerID, &sub.PlanID, &sub.Status,
			&sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &billingAnchor,
			&sub.CancelAtPeriodEnd, &refID, &razorpayID, &stripeID,
			&sub.CreatedAt, &sub.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if billingAnchor.Valid {
			sub.BillingAnchor = billingAnchor.Time
		}
		sub.ReferenceID = refID.String
		sub.RazorpaySubscriptionID = razorpayID.String
		sub.StripeSubscriptionID = stripeID.String
		subs = append(subs, sub)
	}
	return subs, nil
}

// Update updates a subscription
func (r *SubscriptionRepository) Update(ctx context.Context, sub *domain.Subscription) error {
	query := `
		UPDATE subscriptions SET
			status = $1,
			current_period_start = $2,
			current_period_end = $3,
			cancel_at_period_end = $4,
			canceled_at = $5,
			cancellation_reason = $6,
			cancellation_feedback = $7,
			razorpay_subscription_id = $8,
			stripe_subscription_id = $9,
			updated_at = $10
		WHERE id = $11 AND tenant_id = $12
	`
	_, err := r.db.ExecContext(ctx, query,
		sub.Status,
		sub.CurrentPeriodStart,
		sub.CurrentPeriodEnd,
		sub.CancelAtPeriodEnd,
		sub.CanceledAt,
		sub.CancellationReason,
		sub.CancellationFeedback,
		sub.RazorpaySubscriptionID,
		sub.StripeSubscriptionID,
		sub.UpdatedAt,
		sub.ID,
		sub.TenantID,
	)
	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}
	return nil
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
	defer rows.Close()

	var subs []SubscriptionWithCustomer
	for rows.Next() {
		var sub SubscriptionWithCustomer
		if err := rows.Scan(
			&sub.ID, &sub.TenantID, &sub.CustomerID, &sub.PlanID, &sub.Status,
			&sub.NextBillingDate,
			&sub.CustomerName, &sub.CustomerEmail,
			&sub.PlanName, &sub.Amount, &sub.Currency,
		); err != nil {
			return nil, err
		}
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

	err := r.db.QueryRowContext(ctx, `
		SELECT s.tenant_id, s.customer_id, p.price, p.currency
		FROM subscriptions s
		JOIN plans p ON s.plan_id = p.id
		WHERE s.id = $1
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
