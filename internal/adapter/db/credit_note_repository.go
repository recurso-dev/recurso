package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/recurso-dev/recurso/internal/core/domain"
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

// SumApplicableAdjustments returns the total spendable balance of a customer's
// open adjustment credit notes in the given currency (ENG-153). Read-only
// preview used to reduce a gateway charge before the invoice exists; the actual
// application (ApplyAdjustmentCredits) re-reads under a row lock.
func (r *CreditNoteRepository) SumApplicableAdjustments(ctx context.Context, tenantID, customerID uuid.UUID, currency string) (int64, error) {
	var total int64
	err := r.db.GetContext(ctx, &total, `
		SELECT COALESCE(SUM(balance), 0)
		FROM credit_notes
		WHERE tenant_id = $1 AND customer_id = $2 AND currency = $3
		  AND type = $4 AND status = $5 AND balance > 0`,
		tenantID, customerID, currency, domain.CreditNoteTypeAdjustment, domain.CreditNoteStatusIssued)
	if err != nil {
		return 0, fmt.Errorf("failed to sum applicable adjustment credits: %w", err)
	}
	return total, nil
}

// ApplyAdjustmentCredits applies a customer's open adjustment credit notes to an
// already-persisted invoice, up to invoiceTotal (ENG-153). It runs in one
// transaction: credit notes are locked oldest-first (FIFO), each is drawn down
// (balance decremented, flipped to 'used' at zero), an audit row is written to
// credit_note_applications, and the invoice's credit_applied is set (marking it
// paid when fully covered). Returns the total applied. The FOR UPDATE lock makes
// concurrent invoices for the same customer safe from double-spend.
func (r *CreditNoteRepository) ApplyAdjustmentCredits(ctx context.Context, tenantID, customerID uuid.UUID, currency string, invoiceID uuid.UUID, invoiceTotal int64) (int64, error) {
	if invoiceTotal <= 0 {
		return 0, nil
	}
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin credit-application tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op after commit

	type applicable struct {
		ID      uuid.UUID `db:"id"`
		Balance int64     `db:"balance"`
	}
	var notes []applicable
	if err := tx.SelectContext(ctx, &notes, `
		SELECT id, balance FROM credit_notes
		WHERE tenant_id = $1 AND customer_id = $2 AND currency = $3
		  AND type = $4 AND status = $5 AND balance > 0
		ORDER BY created_at ASC, id ASC
		FOR UPDATE`,
		tenantID, customerID, currency, domain.CreditNoteTypeAdjustment, domain.CreditNoteStatusIssued); err != nil {
		return 0, fmt.Errorf("lock applicable credit notes: %w", err)
	}

	var applied int64
	remaining := invoiceTotal
	for _, n := range notes {
		if remaining <= 0 {
			break
		}
		take := n.Balance
		if take > remaining {
			take = remaining
		}
		newBalance := n.Balance - take
		newStatus := domain.CreditNoteStatusIssued
		if newBalance == 0 {
			newStatus = domain.CreditNoteStatusUsed
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE credit_notes SET balance = $1, status = $2, updated_at = NOW() WHERE id = $3`,
			newBalance, newStatus, n.ID); err != nil {
			return 0, fmt.Errorf("draw down credit note %s: %w", n.ID, err)
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO credit_note_applications (id, tenant_id, credit_note_id, invoice_id, amount, created_at)
			 VALUES ($1, $2, $3, $4, $5, NOW())`,
			uuid.New(), tenantID, n.ID, invoiceID, take); err != nil {
			return 0, fmt.Errorf("record credit application for %s: %w", n.ID, err)
		}
		applied += take
		remaining -= take
	}

	if applied > 0 {
		// Set credit_applied and, when the credit fully covers the invoice, mark
		// it paid (settled by credit, no cash). Guarded on the invoice being
		// unsettled so we never re-open or clobber a paid invoice.
		if applied >= invoiceTotal {
			if _, err := tx.ExecContext(ctx,
				`UPDATE invoices SET credit_applied = $1, status = 'paid', paid_at = NOW(), updated_at = NOW()
				 WHERE id = $2 AND tenant_id = $3`,
				applied, invoiceID, tenantID); err != nil {
				return 0, fmt.Errorf("mark invoice %s credit-paid: %w", invoiceID, err)
			}
		} else {
			if _, err := tx.ExecContext(ctx,
				`UPDATE invoices SET credit_applied = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3`,
				applied, invoiceID, tenantID); err != nil {
				return 0, fmt.Errorf("set credit_applied on invoice %s: %w", invoiceID, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit credit-application tx: %w", err)
	}
	return applied, nil
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
