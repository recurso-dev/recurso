package domain

import (
	"time"

	"github.com/google/uuid"
)

type CreditNoteStatus string

const (
	CreditNoteStatusIssued CreditNoteStatus = "issued"
	CreditNoteStatusUsed   CreditNoteStatus = "used"
	CreditNoteStatusVoid   CreditNoteStatus = "void"
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

	// Relations
	Customer *Customer `json:"customer,omitempty" db:"-"`
}

type CreateCreditNoteRequest struct {
	CustomerID uuid.UUID  `json:"customer_id" binding:"required"`
	InvoiceID  *uuid.UUID `json:"invoice_id"`
	Amount     int64      `json:"amount" binding:"required,gt=0"`
	Currency   string     `json:"currency" binding:"required"`
	Reason     string     `json:"reason"`
}

type CreditNoteFilter struct {
	CustomerID *uuid.UUID
	Status     *CreditNoteStatus
}
