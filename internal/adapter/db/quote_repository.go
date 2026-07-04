package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// QuoteRepository implements port.QuoteRepository
type QuoteRepository struct {
	db *sql.DB
}

func NewQuoteRepository(db *sql.DB) *QuoteRepository {
	return &QuoteRepository{db: db}
}

func (r *QuoteRepository) Create(ctx context.Context, quote *domain.Quote) error {
	lineItemsJSON, err := json.Marshal(quote.LineItems)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO quotes (
			id, tenant_id, customer_id, quote_number, status,
			line_items, subtotal, tax_amount, discount_amount, total, currency,
			valid_until, notes, terms, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW(), NOW())
	`
	_, err = r.db.ExecContext(ctx, query,
		quote.ID, quote.TenantID, quote.CustomerID, quote.QuoteNumber, quote.Status,
		lineItemsJSON, quote.Subtotal, quote.TaxAmount, quote.DiscountAmount, quote.Total, quote.Currency,
		quote.ValidUntil, quote.Notes, quote.Terms,
	)
	return err
}

func (r *QuoteRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Quote, error) {
	query := `
		SELECT id, tenant_id, customer_id, quote_number, status,
			line_items, subtotal, tax_amount, discount_amount, total, currency,
			valid_until, notes, terms, invoice_id, accepted_at, declined_at,
			created_at, updated_at
		FROM quotes WHERE id = $1
	`
	var quote domain.Quote
	var lineItemsJSON []byte
	var notes, terms sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&quote.ID, &quote.TenantID, &quote.CustomerID, &quote.QuoteNumber, &quote.Status,
		&lineItemsJSON, &quote.Subtotal, &quote.TaxAmount, &quote.DiscountAmount, &quote.Total, &quote.Currency,
		&quote.ValidUntil, &notes, &terms, &quote.InvoiceID, &quote.AcceptedAt, &quote.DeclinedAt,
		&quote.CreatedAt, &quote.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	quote.Notes = notes.String
	quote.Terms = terms.String

	if err := json.Unmarshal(lineItemsJSON, &quote.LineItems); err != nil {
		return nil, err
	}

	return &quote, nil
}

func (r *QuoteRepository) Update(ctx context.Context, quote *domain.Quote) error {
	lineItemsJSON, err := json.Marshal(quote.LineItems)
	if err != nil {
		return err
	}

	query := `
		UPDATE quotes SET
			status = $2, line_items = $3, subtotal = $4, tax_amount = $5,
			discount_amount = $6, total = $7, valid_until = $8, notes = $9,
			terms = $10, invoice_id = $11, accepted_at = $12, declined_at = $13,
			updated_at = NOW()
		WHERE id = $1
	`
	_, err = r.db.ExecContext(ctx, query,
		quote.ID, quote.Status, lineItemsJSON, quote.Subtotal, quote.TaxAmount,
		quote.DiscountAmount, quote.Total, quote.ValidUntil, quote.Notes,
		quote.Terms, quote.InvoiceID, quote.AcceptedAt, quote.DeclinedAt,
	)
	return err
}

func (r *QuoteRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM quotes WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *QuoteRepository) List(ctx context.Context, tenantID uuid.UUID, filter domain.QuoteFilter) ([]*domain.Quote, error) {
	query := `
		SELECT id, tenant_id, customer_id, quote_number, status,
			line_items, subtotal, tax_amount, discount_amount, total, currency,
			valid_until, notes, terms, invoice_id, accepted_at, declined_at,
			created_at, updated_at
		FROM quotes WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}
	argIdx := 2

	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, filter.Status)
		argIdx++
	}

	if filter.CustomerID != "" {
		query += fmt.Sprintf(" AND customer_id = $%d", argIdx)
		args = append(args, filter.CustomerID)
		argIdx++
	}

	if filter.Search != "" {
		query += fmt.Sprintf(" AND (quote_number ILIKE $%d OR notes ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	query += " ORDER BY created_at DESC"

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

	var quotes []*domain.Quote
	for rows.Next() {
		var quote domain.Quote
		var lineItemsJSON []byte
		var notes, terms sql.NullString

		if err := rows.Scan(
			&quote.ID, &quote.TenantID, &quote.CustomerID, &quote.QuoteNumber, &quote.Status,
			&lineItemsJSON, &quote.Subtotal, &quote.TaxAmount, &quote.DiscountAmount, &quote.Total, &quote.Currency,
			&quote.ValidUntil, &notes, &terms, &quote.InvoiceID, &quote.AcceptedAt, &quote.DeclinedAt,
			&quote.CreatedAt, &quote.UpdatedAt,
		); err != nil {
			return nil, err
		}

		quote.Notes = notes.String
		quote.Terms = terms.String

		if err := json.Unmarshal(lineItemsJSON, &quote.LineItems); err != nil {
			return nil, err
		}

		quotes = append(quotes, &quote)
	}

	return quotes, nil
}

func (r *QuoteRepository) GetNextQuoteNumber(ctx context.Context, tenantID uuid.UUID) (string, error) {
	var count int
	query := `SELECT COUNT(*) FROM quotes WHERE tenant_id = $1`
	if err := r.db.QueryRowContext(ctx, query, tenantID).Scan(&count); err != nil {
		return "", err
	}
	return fmt.Sprintf("QT-%05d", count+1), nil
}
