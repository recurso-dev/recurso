package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrDisputeNotFound is returned when no matching (open) dispute exists for the
// tenant, e.g. when resolving an already-resolved or non-existent dispute.
var ErrDisputeNotFound = errors.New("dispute not found")

// DisputeStatus represents the lifecycle state of an invoice dispute.
type DisputeStatus string

const (
	DisputeStatusOpen     DisputeStatus = "open"
	DisputeStatusResolved DisputeStatus = "resolved"
)

// InvoiceDispute is a customer-raised query/dispute against one of their own
// invoices. v1 is deliberately lightweight: a customer opens a dispute with a
// short reason, and an admin resolves it with an optional note.
type InvoiceDispute struct {
	ID         uuid.UUID     `json:"id" db:"id"`
	TenantID   uuid.UUID     `json:"tenant_id" db:"tenant_id"`
	InvoiceID  uuid.UUID     `json:"invoice_id" db:"invoice_id"`
	CustomerID uuid.UUID     `json:"customer_id" db:"customer_id"`
	Reason     string        `json:"reason" db:"reason"`
	Status     DisputeStatus `json:"status" db:"status"`
	Note       *string       `json:"note" db:"note"`
	CreatedAt  time.Time     `json:"created_at" db:"created_at"`
	ResolvedAt *time.Time    `json:"resolved_at" db:"resolved_at"`
}
