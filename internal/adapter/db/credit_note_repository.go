package db

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/recur-so/recurso/internal/core/domain"
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
			currency, status, reason, created_at, updated_at
		) VALUES (
			:tenant_id, :customer_id, :invoice_id, :reference, :amount, :balance,
			:currency, :status, :reason, :created_at, :updated_at
		) RETURNING id`

	rows, err := r.db.NamedQueryContext(ctx, query, creditNote)
	if err != nil {
		return fmt.Errorf("failed to create credit note: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		return rows.Scan(&creditNote.ID)
	}
	return fmt.Errorf("failed to return id from create credit note")
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
		argIdx++
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
