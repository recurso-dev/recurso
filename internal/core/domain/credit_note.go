package domain

import (
	"time"

	"github.com/google/uuid"
)

type CreditNoteStatus string

const (
	CreditNoteStatusIssued   CreditNoteStatus = "issued"
	CreditNoteStatusUsed     CreditNoteStatus = "used"
	CreditNoteStatusVoid     CreditNoteStatus = "void"
	CreditNoteStatusPending  CreditNoteStatus = "pending_approval"
	CreditNoteStatusRejected CreditNoteStatus = "rejected"
)

// CreditNoteType distinguishes a plain balance adjustment (spendable credit)
// from a refund that returns money to the customer via the payment gateway.
type CreditNoteType string

const (
	CreditNoteTypeAdjustment CreditNoteType = "adjustment"
	CreditNoteTypeRefund     CreditNoteType = "refund"
)

// CreditNoteRefundStatus tracks the state of the gateway refund attached to a
// refund-type credit note. Adjustments always stay at "none".
type CreditNoteRefundStatus string

const (
	// RefundStatusNone — no refund involved (adjustment credit notes).
	RefundStatusNone CreditNoteRefundStatus = "none"
	// RefundStatusPending — refund initiated at the gateway, not yet settled.
	RefundStatusPending CreditNoteRefundStatus = "pending"
	// RefundStatusProcessed — gateway confirmed the refund.
	RefundStatusProcessed CreditNoteRefundStatus = "processed"
	// RefundStatusFailed — the gateway refund call failed; needs operator action.
	RefundStatusFailed CreditNoteRefundStatus = "refund_failed"
	// RefundStatusManualRequired — the invoice has no gateway payment id on
	// record (offline / pre-tracking payment), so no API refund was attempted.
	RefundStatusManualRequired CreditNoteRefundStatus = "manual_required"
)

type CreditNote struct {
	ID         uuid.UUID        `json:"id" db:"id"`
	TenantID   uuid.UUID        `json:"tenant_id" db:"tenant_id"`
	CustomerID uuid.UUID        `json:"customer_id" db:"customer_id"`
	InvoiceID  *uuid.UUID       `json:"invoice_id,omitempty" db:"invoice_id"`
	Reference  *string          `json:"reference,omitempty" db:"reference"`
	Amount     int64            `json:"amount" db:"amount"`
	Balance    int64            `json:"balance" db:"balance"`
	Currency   string           `json:"currency" db:"currency"`
	Status     CreditNoteStatus `json:"status" db:"status"`
	Reason     string           `json:"reason" db:"reason"`
	CreatedAt  time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at" db:"updated_at"`

	// Audit tracking
	CreatedBy  *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
	ApprovedBy *uuid.UUID `json:"approved_by,omitempty" db:"approved_by"`
	ApprovedAt *time.Time `json:"approved_at,omitempty" db:"approved_at"`

	// Refund tracking (type == "refund" only)
	Type          CreditNoteType         `json:"type" db:"type"`
	RefundStatus  CreditNoteRefundStatus `json:"refund_status" db:"refund_status"`
	RefundID      *string                `json:"refund_id,omitempty" db:"refund_id"`
	RefundMessage string                 `json:"refund_message,omitempty" db:"refund_message"`

	// Relations
	Customer *Customer `json:"customer,omitempty" db:"-"`
}

type CreateCreditNoteRequest struct {
	CustomerID uuid.UUID  `json:"customer_id" binding:"required"`
	InvoiceID  *uuid.UUID `json:"invoice_id"`
	Amount     int64      `json:"amount" binding:"required,gt=0"`
	Currency   string     `json:"currency" binding:"required"`
	Reason     string     `json:"reason"`
	// Type defaults to "adjustment"; "refund" triggers a gateway refund
	// against the (paid) invoice referenced by InvoiceID.
	Type string `json:"type" binding:"omitempty,oneof=adjustment refund"`
}

type CreditNoteFilter struct {
	CustomerID *uuid.UUID
	Status     *CreditNoteStatus
}
