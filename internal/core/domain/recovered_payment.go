package domain

import (
	"time"

	"github.com/google/uuid"
)

// RecoveredPayment records revenue recovered by the retry/dunning engine: an
// invoice that transitioned to paid after at least one failed payment attempt.
// One row per invoice (invoice_id is unique), written at the moment of payment.
type RecoveredPayment struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	TenantID      uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	InvoiceID     uuid.UUID  `json:"invoice_id" db:"invoice_id"`
	Amount        int64      `json:"amount" db:"amount"`
	Currency      string     `json:"currency" db:"currency"`
	Attempts      int        `json:"attempts" db:"attempts"`
	Strategy      string     `json:"strategy" db:"strategy"`
	CampaignID    *uuid.UUID `json:"campaign_id,omitempty" db:"campaign_id"`
	DaysToRecover int        `json:"days_to_recover" db:"days_to_recover"`
	RecoveredAt   time.Time  `json:"recovered_at" db:"recovered_at"`
}

// RecoveryTotals aggregates recovered revenue for a tenant.
type RecoveryTotals struct {
	RecoveredAmountTotal map[string]int64 `json:"recovered_amount_total"` // by currency, minor units
	RecoveredCount       int              `json:"recovered_count"`
	AvgAttempts          float64          `json:"avg_attempts"`
	AvgDaysToRecover     float64          `json:"avg_days_to_recover"`
}

// RecoveryMonthBucket is one month/currency cell of the recovered-revenue series.
type RecoveryMonthBucket struct {
	Month    string `json:"month"` // "YYYY-MM"
	Currency string `json:"currency"`
	Amount   int64  `json:"amount"` // minor units
	Count    int    `json:"count"`
}
