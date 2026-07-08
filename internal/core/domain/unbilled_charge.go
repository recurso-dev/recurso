package domain

import (
	"time"

	"github.com/google/uuid"
)

type UnbilledChargeStatus string

const (
	UnbilledChargeStatusPending  UnbilledChargeStatus = "pending"
	UnbilledChargeStatusInvoiced UnbilledChargeStatus = "invoiced"
	UnbilledChargeStatusCanceled UnbilledChargeStatus = "canceled"
)

type UnbilledCharge struct {
	ID             uuid.UUID `json:"id"`
	SubscriptionID uuid.UUID `json:"subscription_id"`
	Amount         int64     `json:"amount"`   // In cents
	Currency       string    `json:"currency"` // ISO 4217 code
	Description    string    `json:"description"`
	// HSNCode is the HSN/SAC code this charge is taxed at when it is folded onto
	// an invoice as its own line item. Empty falls back to the tenant SAC (then
	// the 998314 default) at tax-resolution time.
	HSNCode     string               `json:"hsn_code,omitempty"`
	Status      UnbilledChargeStatus `json:"status"`
	PeriodStart *time.Time           `json:"period_start,omitempty"`
	PeriodEnd   *time.Time           `json:"period_end,omitempty"`
	CreatedAt   time.Time            `json:"created_at"`
}
