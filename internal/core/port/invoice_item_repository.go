package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// InvoiceItemRepository persists and reads itemized invoice line items.
// Items are tenant-scoped transitively through their parent invoice and are
// deleted with it (ON DELETE CASCADE).
//
// Bulk creation runs inside the invoice's own transaction: the concrete db
// implementation additionally exposes CreateWithTx(*sql.Tx, ...) so line items
// land atomically with the invoice row. The interface stays sql-free.
type InvoiceItemRepository interface {
	// Create bulk-inserts the given line items in a single transaction.
	Create(ctx context.Context, items []*domain.InvoiceItem) error
	// ListByInvoiceID returns the line items for an invoice, ordered by creation.
	ListByInvoiceID(ctx context.Context, invoiceID uuid.UUID) ([]domain.InvoiceItem, error)
}
