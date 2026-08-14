package domain

import (
	"time"

	"github.com/google/uuid"
)

// InvoiceStatusChange is one recorded transition of an invoice's status —
// captured by a trigger at the source of truth, so no transition is missed. The
// first row of an invoice's history has a null FromStatus (its creation state).
type InvoiceStatusChange struct {
	ID         uuid.UUID `json:"id"`
	InvoiceID  uuid.UUID `json:"invoice_id"`
	FromStatus *string   `json:"from_status,omitempty"`
	ToStatus   string    `json:"to_status"`
	ChangedAt  time.Time `json:"changed_at"`
}
