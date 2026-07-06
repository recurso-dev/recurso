package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

type CreditNoteRepository struct {
	db *sqlx.DB
}

func NewCreditNoteRepository(db *sqlx.DB) *CreditNoteRepository {
	return &CreditNoteRepository{db: db}
}

func (r *CreditNoteRepository) Create(ctx context.Context, creditNote *domain.CreditNote) error {
	query := `
		INSERT INTO credit_notes (
			tenant_id, customer_id, invoice_id, reference, amount, balance,
			currency, status, reason, type, refund_status, refund_id,
			refund_message, created_at, updated_at
		) VALUES (
			:tenant_id, :customer_id, :invoice_id, :reference, :amount, :balance,
			:currency, :status, :reason, :type, :refund_status, :refund_id,
			:refund_message, :created_at, :updated_at
		) RETURNING id`

	rows, err := r.db.NamedQueryContext(ctx, query, creditNote)
	if err != nil {
		return fmt.Errorf("failed to create credit note: %w", err)
	}
	defer func() { _ = rows.Close() }()

	if rows.Next() {
		return rows.Scan(&creditNote.ID)
	}
	return fmt.Errorf("failed to return id from create credit note")
}

// UpdateRefund persists the outcome of a gateway refund attempt on a
// refund-type credit note.
func (r *CreditNoteRepository) UpdateRefund(ctx context.Context, id uuid.UUID, status domain.CreditNoteRefundStatus, refundID *string, message string) error {
	query := `
		UPDATE credit_notes
		SET refund_status = $1, refund_id = $2, refund_message = $3, updated_at = NOW()
		WHERE id = $4`

	res, err := r.db.ExecContext(ctx, query, status, refundID, message, id)
	if err != nil {
		return fmt.Errorf("failed to update credit note refund state: %w", err)
	}
	if rows, err := res.RowsAffected(); err == nil && rows == 0 {
		return fmt.Errorf("credit note %s not found", id)
	}
	return nil
}

// SumActiveRefundsForInvoice returns the total amount of refund-type credit
// notes already issued against an invoice (excluding voided notes and failed
// refunds). Used for the over-refund guard.
func (r *CreditNoteRepository) SumActiveRefundsForInvoice(ctx context.Context, invoiceID uuid.UUID) (int64, error) {
	query := `
		SELECT COALESCE(SUM(amount), 0)
		FROM credit_notes
		WHERE invoice_id = $1
		  AND type = $2
		  AND status <> $3
		  AND refund_status IN ($4, $5, $6)`

	var total int64
	err := r.db.GetContext(ctx, &total, query,
		invoiceID,
		domain.CreditNoteTypeRefund,
		domain.CreditNoteStatusVoid,
		domain.RefundStatusPending, domain.RefundStatusProcessed, domain.RefundStatusManualRequired,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to sum refunds for invoice: %w", err)
	}
	return total, nil
}

// GetByRefundID resolves the credit note that owns a gateway refund id
// (Stripe re_*, Razorpay rfnd_*). Used by the refund webhook consumers.
// Returns (nil, nil) when no credit note tracks that refund id — callers
// treat those events as ignorable rather than an error.
func (r *CreditNoteRepository) GetByRefundID(ctx context.Context, refundID string) (*domain.CreditNote, error) {
	var cn domain.CreditNote
	err := r.db.GetContext(ctx, &cn, `SELECT * FROM credit_notes WHERE refund_id = $1`, refundID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get credit note by refund id: %w", err)
	}
	return &cn, nil
}

func (r *CreditNoteRepository) List(ctx context.Context, tenantID uuid.UUID, filter domain.CreditNoteFilter) ([]*domain.CreditNote, error) {
	query := `SELECT * FROM credit_notes WHERE tenant_id = $1`
	args := []interface{}{tenantID}
	argIdx := 2

	if filter.CustomerID != nil {
		query += fmt.Sprintf(" AND customer_id = $%d", argIdx)
		args = append(args, *filter.CustomerID)
		argIdx++
	}

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *filter.Status)
	}

	query += ` ORDER BY created_at DESC`

	var creditNotes []*domain.CreditNote
	err := r.db.SelectContext(ctx, &creditNotes, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list credit notes: %w", err)
	}

	// Fetch related customers if needed (For MVP we might skip JOINs in Repo and do simple fetches or allow Join later)
	// For listing in grid, we usually want Customer Name.
	// Since we are not doing a Join here, the Service layer or Client has to handle it, or we rely on 'Customer' field being nil.
	// Let's keep it simple.

	return creditNotes, nil
}
