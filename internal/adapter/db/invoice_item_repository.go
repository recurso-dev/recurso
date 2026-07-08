package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

// InvoiceItemRepository is the SQL-backed store for itemized invoice lines.
type InvoiceItemRepository struct {
	db *sql.DB
}

// NewInvoiceItemRepository constructs an InvoiceItemRepository.
func NewInvoiceItemRepository(db *sql.DB) *InvoiceItemRepository {
	return &InvoiceItemRepository{db: db}
}

const invoiceItemInsert = `
	INSERT INTO invoice_items (
		id, invoice_id, description, hsn_code, quantity, unit_amount, amount,
		tax_rate, cgst_amount, sgst_amount, igst_amount, taxable_amount, created_at
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, COALESCE($13, NOW()))
`

// execInsert runs the bulk insert against any execer (either *sql.DB or *sql.Tx).
type execer interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

func insertInvoiceItems(ctx context.Context, ex execer, items []*domain.InvoiceItem) error {
	for _, it := range items {
		if it == nil {
			continue
		}
		if it.ID == uuid.Nil {
			it.ID = uuid.New()
		}
		var createdAt interface{}
		if !it.CreatedAt.IsZero() {
			createdAt = it.CreatedAt
		}
		if _, err := ex.ExecContext(ctx, invoiceItemInsert,
			it.ID, it.InvoiceID, it.Description, it.HSNCode, it.Quantity, it.UnitAmount, it.Amount,
			it.TaxRate, it.CGSTAmount, it.SGSTAmount, it.IGSTAmount, it.TaxableAmount, createdAt,
		); err != nil {
			return fmt.Errorf("failed to insert invoice item: %w", err)
		}
	}
	return nil
}

// Create bulk-inserts line items inside a fresh transaction.
func (r *InvoiceItemRepository) Create(ctx context.Context, items []*domain.InvoiceItem) error {
	if len(items) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin invoice items tx: %w", err)
	}
	if err := insertInvoiceItems(ctx, tx, items); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit invoice items tx: %w", err)
	}
	return nil
}

// CreateWithTx bulk-inserts line items on an existing transaction so they land
// atomically with the parent invoice.
func (r *InvoiceItemRepository) CreateWithTx(ctx context.Context, tx *sql.Tx, items []*domain.InvoiceItem) error {
	return insertInvoiceItems(ctx, tx, items)
}

const invoiceItemSelect = `
	SELECT id, invoice_id, description, hsn_code, quantity, unit_amount, amount,
	       tax_rate, cgst_amount, sgst_amount, igst_amount, taxable_amount, created_at
	FROM invoice_items
`

func scanInvoiceItems(rows *sql.Rows) ([]domain.InvoiceItem, error) {
	defer func() { _ = rows.Close() }()
	var items []domain.InvoiceItem
	for rows.Next() {
		var it domain.InvoiceItem
		if err := rows.Scan(
			&it.ID, &it.InvoiceID, &it.Description, &it.HSNCode, &it.Quantity, &it.UnitAmount, &it.Amount,
			&it.TaxRate, &it.CGSTAmount, &it.SGSTAmount, &it.IGSTAmount, &it.TaxableAmount, &it.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan invoice item: %w", err)
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// ListByInvoiceID returns the line items for a single invoice.
func (r *InvoiceItemRepository) ListByInvoiceID(ctx context.Context, invoiceID uuid.UUID) ([]domain.InvoiceItem, error) {
	rows, err := r.db.QueryContext(ctx, invoiceItemSelect+" WHERE invoice_id = $1 ORDER BY created_at, id", invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query invoice items: %w", err)
	}
	return scanInvoiceItems(rows)
}

// ListByInvoiceIDs batch-loads line items for many invoices, grouped by
// invoice id, so list endpoints avoid an N+1 query.
func (r *InvoiceItemRepository) ListByInvoiceIDs(ctx context.Context, invoiceIDs []uuid.UUID) (map[uuid.UUID][]domain.InvoiceItem, error) {
	out := make(map[uuid.UUID][]domain.InvoiceItem)
	if len(invoiceIDs) == 0 {
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx, invoiceItemSelect+" WHERE invoice_id = ANY($1) ORDER BY created_at, id", pq.Array(invoiceIDs))
	if err != nil {
		return nil, fmt.Errorf("failed to query invoice items batch: %w", err)
	}
	items, err := scanInvoiceItems(rows)
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		out[it.InvoiceID] = append(out[it.InvoiceID], it)
	}
	return out, nil
}

var _ port.InvoiceItemRepository = (*InvoiceItemRepository)(nil)
