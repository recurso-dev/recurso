package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// InvoiceStatusHistoryRepository reads the append-only invoice status timeline
// captured by the invoice_status_history trigger. Read-only — writes happen in
// the database, never from Go.
type InvoiceStatusHistoryRepository struct {
	db *sql.DB
}

func NewInvoiceStatusHistoryRepository(db *sql.DB) *InvoiceStatusHistoryRepository {
	return &InvoiceStatusHistoryRepository{db: db}
}

// ListByInvoice returns an invoice's status transitions oldest-first,
// tenant-scoped. from_status is null on the creation row.
func (r *InvoiceStatusHistoryRepository) ListByInvoice(ctx context.Context, tenantID, invoiceID uuid.UUID) ([]domain.InvoiceStatusChange, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, invoice_id, from_status, to_status, changed_at
		 FROM invoice_status_history
		 WHERE invoice_id = $1 AND tenant_id = $2
		 ORDER BY changed_at, id`, invoiceID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list invoice status history: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := []domain.InvoiceStatusChange{}
	for rows.Next() {
		var c domain.InvoiceStatusChange
		var from sql.NullString
		if err := rows.Scan(&c.ID, &c.InvoiceID, &from, &c.ToStatus, &c.ChangedAt); err != nil {
			return nil, fmt.Errorf("failed to scan invoice status change: %w", err)
		}
		if from.Valid {
			c.FromStatus = &from.String
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
