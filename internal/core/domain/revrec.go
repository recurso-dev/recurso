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
	// RecognitionStatusProcessing is the worker's claim (F2): flipping
	// pending -> processing atomically hands each due event to exactly one
	// worker, so a concurrent runner can never re-post it — or mis-mark an
	// already-recognized event as failed when its duplicate posting errors.
	RecognitionStatusProcessing = "processing"
	// RecognitionStatusCanceled marks a future event voided by an unwind
	// (cancel/refund mid-period, ENG-147) so the worker never recognizes it.
	RecognitionStatusCanceled = "canceled"
)

// DeferredRevenueReport is a point-in-time rollforward of deferred (unearned)
// revenue: what was recognized in the requested period, the deferred balance
// still on the books, when that balance is scheduled to release, and how it
// splits by currency.
type DeferredRevenueReport struct {
	Month            int                         `json:"month"`
	Year             int                         `json:"year"`
	RecognizedAmount int64                       `json:"recognized_amount"` // recognized in month/year, minor units
	DeferredBalance  int64                       `json:"deferred_balance"`  // all still-pending recognition, minor units (summed across currencies — see ByCurrency)
	Upcoming         []DeferredRecognitionBucket `json:"upcoming"`          // the still-pending balance grouped by the month it will recognize
	ByCurrency       []DeferredCurrencyBalance   `json:"by_currency"`       // deferred balance split by the schedule's currency
}

// DeferredRecognitionBucket is one month of scheduled future recognition.
type DeferredRecognitionBucket struct {
	Month  int   `json:"month"`
	Year   int   `json:"year"`
	Amount int64 `json:"amount"` // minor units, summed across currencies
}

// DeferredCurrencyBalance is the still-deferred balance for one currency.
type DeferredCurrencyBalance struct {
	Currency string `json:"currency"`
	Deferred int64  `json:"deferred"` // minor units, native currency
}

// RevenueWaterfallBucket is one month on the recognition curve: revenue already
// recognized that month, and revenue still scheduled (pending) to recognize.
type RevenueWaterfallBucket struct {
	Year       int   `json:"year"`
	Month      int   `json:"month"`
	Recognized int64 `json:"recognized"` // status=recognized, minor units
	Scheduled  int64 `json:"scheduled"`  // status=pending, minor units
}

// RevenueWaterfall is a tenant's full recognized-plus-scheduled revenue curve,
// month by month — the classic rev-rec waterfall an auditor plots to see how
// deferred revenue releases over time (past recognitions and future schedule
// in one series).
type RevenueWaterfall struct {
	TenantID        uuid.UUID                `json:"tenant_id"`
	Buckets         []RevenueWaterfallBucket `json:"buckets"`
	TotalRecognized int64                    `json:"total_recognized"`
	TotalScheduled  int64                    `json:"total_scheduled"`
}
