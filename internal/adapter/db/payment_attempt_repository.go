package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// PaymentAttemptRepository persists the async settlement lifecycle of a payment
// (Inc 3 / ACH), keyed for webhook advancement by PaymentIntent id.
type PaymentAttemptRepository struct {
	db *sql.DB
}

func NewPaymentAttemptRepository(db *sql.DB) *PaymentAttemptRepository {
	return &PaymentAttemptRepository{db: db}
}

const paymentAttemptColumns = `id, tenant_id, invoice_id, gateway, method, gateway_payment_intent_id, status, failure_code, amount, created_at, updated_at, settled_at`

func scanPaymentAttempt(row interface{ Scan(...any) error }) (*domain.PaymentAttempt, error) {
	a := &domain.PaymentAttempt{}
	var status string
	if err := row.Scan(&a.ID, &a.TenantID, &a.InvoiceID, &a.Gateway, &a.Method,
		&a.GatewayPaymentIntentID, &status, &a.FailureCode, &a.Amount,
		&a.CreatedAt, &a.UpdatedAt, &a.SettledAt); err != nil {
		return nil, err
	}
	a.Status = domain.PaymentAttemptStatus(status)
	return a, nil
}

// Create inserts a new attempt (defaults id + status).
func (r *PaymentAttemptRepository) Create(ctx context.Context, a *domain.PaymentAttempt) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	if a.Status == "" {
		a.Status = domain.PaymentAttemptInitiated
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO payment_attempts (id, tenant_id, invoice_id, gateway, method, gateway_payment_intent_id, status, failure_code, amount)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		a.ID, a.TenantID, a.InvoiceID, a.Gateway, a.Method, a.GatewayPaymentIntentID, string(a.Status), a.FailureCode, a.Amount)
	if err != nil {
		return fmt.Errorf("failed to create payment attempt: %w", err)
	}
	return nil
}

// ListByInvoice returns an invoice's payment attempts, oldest first — the
// retry/settlement history the invoice page's Payments section shows (an ACH
// debit's initiated → processing → succeeded, or a card's failed → succeeded).
// Tenant-scoped. Read-only.
func (r *PaymentAttemptRepository) ListByInvoice(ctx context.Context, tenantID, invoiceID uuid.UUID) ([]*domain.PaymentAttempt, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+paymentAttemptColumns+` FROM payment_attempts
		 WHERE tenant_id = $1 AND invoice_id = $2
		 ORDER BY created_at, id`, tenantID, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list payment attempts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []*domain.PaymentAttempt
	for rows.Next() {
		a, err := scanPaymentAttempt(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan payment attempt: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// List returns the tenant's payment attempts newest-first, paginated, with an
// optional status filter and each attempt's invoice number joined — the
// payments log. Returns the page plus the unfiltered-by-page total. Read-only.
func (r *PaymentAttemptRepository) List(ctx context.Context, tenantID uuid.UUID, status string, limit, offset int) ([]domain.PaymentAttemptListItem, int, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT COUNT(*) OVER() AS total,
		       pa.id, pa.tenant_id, pa.invoice_id, pa.gateway, pa.method,
		       pa.gateway_payment_intent_id, pa.status, pa.failure_code, pa.amount,
		       pa.created_at, pa.updated_at, pa.settled_at,
		       COALESCE(i.invoice_number, ''), COALESCE(i.currency, '')
		FROM payment_attempts pa
		LEFT JOIN invoices i ON i.id = pa.invoice_id
		WHERE pa.tenant_id = $1 AND ($2 = '' OR pa.status = $2)
		ORDER BY pa.created_at DESC, pa.id DESC
		LIMIT $3 OFFSET $4`,
		tenantID, status, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list payment attempts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := []domain.PaymentAttemptListItem{}
	total := 0
	for rows.Next() {
		var it domain.PaymentAttemptListItem
		var st string
		if err := rows.Scan(&total,
			&it.ID, &it.TenantID, &it.InvoiceID, &it.Gateway, &it.Method,
			&it.GatewayPaymentIntentID, &st, &it.FailureCode, &it.Amount,
			&it.CreatedAt, &it.UpdatedAt, &it.SettledAt, &it.InvoiceNumber, &it.Currency); err != nil {
			return nil, 0, fmt.Errorf("failed to scan payment attempt: %w", err)
		}
		it.Status = domain.PaymentAttemptStatus(st)
		items = append(items, it)
	}
	return items, total, rows.Err()
}

// GetByPaymentIntentID resolves the attempt a webhook is about, or (nil, nil).
func (r *PaymentAttemptRepository) GetByPaymentIntentID(ctx context.Context, paymentIntentID string) (*domain.PaymentAttempt, error) {
	if paymentIntentID == "" {
		return nil, nil
	}
	a, err := scanPaymentAttempt(r.db.QueryRowContext(ctx,
		`SELECT `+paymentAttemptColumns+` FROM payment_attempts WHERE gateway_payment_intent_id = $1`, paymentIntentID))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get payment attempt: %w", err)
	}
	return a, nil
}

// UpdateStatusByPaymentIntent advances an attempt's status (+ failure_code /
// settled_at) keyed on its PaymentIntent id — the webhook's idempotent handle.
func (r *PaymentAttemptRepository) UpdateStatusByPaymentIntent(ctx context.Context, paymentIntentID string, status domain.PaymentAttemptStatus, failureCode string, settledAt *time.Time) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE payment_attempts SET status = $1, failure_code = $2, settled_at = $3, updated_at = NOW()
		 WHERE gateway_payment_intent_id = $4`,
		string(status), failureCode, settledAt, paymentIntentID)
	if err != nil {
		return fmt.Errorf("failed to update payment attempt: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("no payment attempt for intent %s", paymentIntentID)
	}
	return nil
}

// HasInFlightForInvoice reports whether an invoice has an initiated/processing
// attempt — dunning uses this to skip a settling ACH (Inc 3b).
func (r *PaymentAttemptRepository) HasInFlightForInvoice(ctx context.Context, invoiceID uuid.UUID) (bool, error) {
	var n int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM payment_attempts WHERE invoice_id = $1 AND status IN ('initiated','processing')`,
		invoiceID).Scan(&n); err != nil {
		return false, fmt.Errorf("failed to check in-flight attempts: %w", err)
	}
	return n > 0, nil
}
