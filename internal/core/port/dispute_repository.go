package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// DisputeRepository persists customer-raised invoice disputes.
type DisputeRepository interface {
	Create(ctx context.Context, d *domain.InvoiceDispute) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.InvoiceDispute, error)
	// GetOpenByInvoiceID returns the single open dispute for an invoice, or nil
	// if none exists. Used to enforce one-open-dispute-per-invoice semantics.
	GetOpenByInvoiceID(ctx context.Context, invoiceID uuid.UUID) (*domain.InvoiceDispute, error)
	// UpdateReason updates the reason of an existing (open) dispute.
	UpdateReason(ctx context.Context, id uuid.UUID, reason string) error
	// ListByCustomerID returns all disputes raised by a customer (newest first).
	ListByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*domain.InvoiceDispute, error)
	// ListByTenant returns tenant-scoped disputes, optionally filtered by status.
	ListByTenant(ctx context.Context, tenantID uuid.UUID, status string) ([]*domain.InvoiceDispute, error)
	// Resolve marks a dispute resolved with an optional note. It is scoped by
	// tenant and only affects open disputes; returns ErrDisputeNotFound when no
	// matching open dispute exists for the tenant.
	Resolve(ctx context.Context, tenantID, id uuid.UUID, note string) error
}
