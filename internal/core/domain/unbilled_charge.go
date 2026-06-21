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
	ID             uuid.UUID            `json:"id"`
	SubscriptionID uuid.UUID            `json:"subscription_id"`
	Amount         int64                `json:"amount"`   // In cents
	Currency       string               `json:"currency"` // ISO 4217 code
	Description    string               `json:"description"`
	Status         UnbilledChargeStatus `json:"status"`
	PeriodStart    *time.Time           `json:"period_start,omitempty"`
	PeriodEnd      *time.Time           `json:"period_end,omitempty"`
	CreatedAt      time.Time            `json:"created_at"`
}
