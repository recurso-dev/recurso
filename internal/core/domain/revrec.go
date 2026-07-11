package domain

import (
	"time"

	"github.com/google/uuid"
)

// RevenueSchedule defines how revenue from an invoice is allocated over time
type RevenueSchedule struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	TenantID       uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	InvoiceID      uuid.UUID  `json:"invoice_id" db:"invoice_id"`
	SubscriptionID *uuid.UUID `json:"subscription_id,omitempty" db:"subscription_id"`
	TotalAmount    int64      `json:"total_amount" db:"total_amount"` // In cents
	Currency       string     `json:"currency" db:"currency"`
	StartDate      time.Time  `json:"start_date" db:"start_date"`
	EndDate        time.Time  `json:"end_date" db:"end_date"`
	Status         string     `json:"status" db:"status"` // 'active', 'canceled'
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// RecognitionEvent represents a single point in time when a portion of revenue is recognized
type RecognitionEvent struct {
	ID                uuid.UUID  `json:"id" db:"id"`
	RevenueScheduleID uuid.UUID  `json:"revenue_schedule_id" db:"revenue_schedule_id"`
	TenantID          uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	Amount            int64      `json:"amount" db:"amount"` // In cents
	RecognitionDate   time.Time  `json:"recognition_date" db:"recognition_date"`
	Status            string     `json:"status" db:"status"` // 'pending', 'recognized', 'failed'
	LedgerTxID        *uuid.UUID `json:"ledger_tx_id,omitempty" db:"ledger_tx_id"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
}

const (
	RevRecStatusActive   = "active"
	RevRecStatusCanceled = "canceled"

	RecognitionStatusPending    = "pending"
	RecognitionStatusRecognized = "recognized"
	RecognitionStatusFailed     = "failed"
	// RecognitionStatusCanceled marks a future event voided by an unwind
	// (cancel/refund mid-period, ENG-147) so the worker never recognizes it.
	RecognitionStatusCanceled = "canceled"
)
