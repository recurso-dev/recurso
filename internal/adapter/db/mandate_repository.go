package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

type MandateRepository struct {
	db *sql.DB
}

func NewMandateRepository(db *sql.DB) *MandateRepository {
	return &MandateRepository{db: db}
}

func (r *MandateRepository) Create(ctx context.Context, mandate *domain.Mandate) error {
	query := `INSERT INTO mandates (id, tenant_id, customer_id, subscription_id, mandate_type, payment_method, vpa,
		razorpay_token_id, razorpay_subscription_id, razorpay_customer_id, max_amount, frequency, status, next_debit_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`
	_, err := r.db.ExecContext(ctx, query,
		mandate.ID, mandate.TenantID, mandate.CustomerID, mandate.SubscriptionID,
		mandate.MandateType, mandate.PaymentMethod, mandate.VPA,
		mandate.RazorpayTokenID, mandate.RazorpaySubscriptionID, mandate.RazorpayCustomerID,
		mandate.MaxAmount, mandate.Frequency, mandate.Status,
		mandate.NextDebitAt, mandate.CreatedAt, mandate.UpdatedAt,
	)
	return err
}

func (r *MandateRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Mandate, error) {
	query := `SELECT id, tenant_id, customer_id, subscription_id, mandate_type, payment_method, vpa,
		razorpay_token_id, razorpay_subscription_id, razorpay_customer_id, max_amount, frequency, status,
		authorized_at, activated_at, revoked_at, last_debit_at, next_debit_at,
		pre_debit_notified, created_at, updated_at
		FROM mandates WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)
	return r.scanMandate(row)
}

func (r *MandateRepository) GetByRazorpayTokenID(ctx context.Context, tokenID string) (*domain.Mandate, error) {
	query := `SELECT id, tenant_id, customer_id, subscription_id, mandate_type, payment_method, vpa,
		razorpay_token_id, razorpay_subscription_id, razorpay_customer_id, max_amount, frequency, status,
		authorized_at, activated_at, revoked_at, last_debit_at, next_debit_at,
		pre_debit_notified, created_at, updated_at
		FROM mandates WHERE razorpay_token_id = $1`
	row := r.db.QueryRowContext(ctx, query, tokenID)
	return r.scanMandate(row)
}

func (r *MandateRepository) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Mandate, error) {
	query := `SELECT id, tenant_id, customer_id, subscription_id, mandate_type, payment_method, vpa,
		razorpay_token_id, razorpay_subscription_id, razorpay_customer_id, max_amount, frequency, status,
		authorized_at, activated_at, revoked_at, last_debit_at, next_debit_at,
		pre_debit_notified, created_at, updated_at
		FROM mandates WHERE tenant_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var mandates []*domain.Mandate
	for rows.Next() {
		m, err := r.scanMandateRow(rows)
		if err != nil {
			return nil, err
		}
		mandates = append(mandates, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return mandates, nil
}

func (r *MandateRepository) Update(ctx context.Context, mandate *domain.Mandate) error {
	query := `UPDATE mandates SET status = $1, razorpay_token_id = $2, razorpay_subscription_id = $3,
		razorpay_customer_id = $4, authorized_at = $5, activated_at = $6, revoked_at = $7, last_debit_at = $8,
		next_debit_at = $9, pre_debit_notified = $10, updated_at = $11, subscription_id = $12
		WHERE id = $13`
	_, err := r.db.ExecContext(ctx, query,
		mandate.Status, mandate.RazorpayTokenID, mandate.RazorpaySubscriptionID,
		mandate.RazorpayCustomerID, mandate.AuthorizedAt, mandate.ActivatedAt, mandate.RevokedAt,
		mandate.LastDebitAt, mandate.NextDebitAt, mandate.PreDebitNotified,
		time.Now(), mandate.SubscriptionID, mandate.ID,
	)
	return err
}

func (r *MandateRepository) GetDueForPreNotification(ctx context.Context) ([]*domain.Mandate, error) {
	query := `SELECT id, tenant_id, customer_id, subscription_id, mandate_type, payment_method, vpa,
		razorpay_token_id, razorpay_subscription_id, razorpay_customer_id, max_amount, frequency, status,
		authorized_at, activated_at, revoked_at, last_debit_at, next_debit_at,
		pre_debit_notified, created_at, updated_at
		FROM mandates
		WHERE status = 'active'
		AND pre_debit_notified = FALSE
		AND next_debit_at IS NOT NULL
		AND next_debit_at <= NOW() + INTERVAL '24 hours'`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var mandates []*domain.Mandate
	for rows.Next() {
		m, err := r.scanMandateRow(rows)
		if err != nil {
			return nil, err
		}
		mandates = append(mandates, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return mandates, nil
}

func (r *MandateRepository) GetReadyForDebit(ctx context.Context) ([]*domain.Mandate, error) {
	query := `SELECT id, tenant_id, customer_id, subscription_id, mandate_type, payment_method, vpa,
		razorpay_token_id, razorpay_subscription_id, razorpay_customer_id, max_amount, frequency, status,
		authorized_at, activated_at, revoked_at, last_debit_at, next_debit_at,
		pre_debit_notified, created_at, updated_at
		FROM mandates
		WHERE status = 'active'
		AND pre_debit_notified = TRUE
		AND next_debit_at IS NOT NULL
		AND next_debit_at <= NOW()`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var mandates []*domain.Mandate
	for rows.Next() {
		m, err := r.scanMandateRow(rows)
		if err != nil {
			return nil, err
		}
		mandates = append(mandates, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return mandates, nil
}

func (r *MandateRepository) scanMandate(row *sql.Row) (*domain.Mandate, error) {
	var m domain.Mandate
	err := row.Scan(
		&m.ID, &m.TenantID, &m.CustomerID, &m.SubscriptionID,
		&m.MandateType, &m.PaymentMethod, &m.VPA,
		&m.RazorpayTokenID, &m.RazorpaySubscriptionID, &m.RazorpayCustomerID,
		&m.MaxAmount, &m.Frequency, &m.Status,
		&m.AuthorizedAt, &m.ActivatedAt, &m.RevokedAt,
		&m.LastDebitAt, &m.NextDebitAt,
		&m.PreDebitNotified, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *MandateRepository) scanMandateRow(rows *sql.Rows) (*domain.Mandate, error) {
	var m domain.Mandate
	err := rows.Scan(
		&m.ID, &m.TenantID, &m.CustomerID, &m.SubscriptionID,
		&m.MandateType, &m.PaymentMethod, &m.VPA,
		&m.RazorpayTokenID, &m.RazorpaySubscriptionID, &m.RazorpayCustomerID,
		&m.MaxAmount, &m.Frequency, &m.Status,
		&m.AuthorizedAt, &m.ActivatedAt, &m.RevokedAt,
		&m.LastDebitAt, &m.NextDebitAt,
		&m.PreDebitNotified, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &m, nil
}
