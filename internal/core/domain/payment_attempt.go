package domain

import (
	"time"

	"github.com/google/uuid"
)

// PaymentAttemptStatus is a payment's async settlement lifecycle. Cards jump
// straight to succeeded/failed; ACH moves initiated → processing → succeeded,
// and can even go succeeded → returned days later.
type PaymentAttemptStatus string

const (
	PaymentAttemptInitiated  PaymentAttemptStatus = "initiated"
	PaymentAttemptProcessing PaymentAttemptStatus = "processing"
	PaymentAttemptSucceeded  PaymentAttemptStatus = "succeeded"
	PaymentAttemptFailed     PaymentAttemptStatus = "failed"
	PaymentAttemptReturned   PaymentAttemptStatus = "returned" // settled, then reversed by the bank
)

// PaymentAttempt carries a payment's async settlement out-of-band from the
// invoice status (which has no in-flight state). Primarily for ACH, where a
// debit is `processing` for days and can be `returned` after it settled. The
// invoice stays `open` until an attempt reaches `succeeded`. (Inc 3.)
type PaymentAttempt struct {
	ID                     uuid.UUID            `json:"id" db:"id"`
	TenantID               uuid.UUID            `json:"tenant_id" db:"tenant_id"`
	InvoiceID              uuid.UUID            `json:"invoice_id" db:"invoice_id"`
	Gateway                string               `json:"gateway" db:"gateway"`
	Method                 string               `json:"method" db:"method"`
	GatewayPaymentIntentID string               `json:"gateway_payment_intent_id" db:"gateway_payment_intent_id"`
	Status                 PaymentAttemptStatus `json:"status" db:"status"`
	FailureCode            string               `json:"failure_code" db:"failure_code"`
	Amount                 int64                `json:"amount" db:"amount"`
	CreatedAt              time.Time            `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time            `json:"updated_at" db:"updated_at"`
	SettledAt              *time.Time           `json:"settled_at,omitempty" db:"settled_at"`
}

// InFlight reports whether the attempt is still settling. Dunning must not
// re-charge an invoice that has an in-flight attempt.
func (a *PaymentAttempt) InFlight() bool {
	return a.Status == PaymentAttemptInitiated || a.Status == PaymentAttemptProcessing
}
