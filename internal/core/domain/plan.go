package domain

import (
	"time"

	"github.com/google/uuid"
)

type IntervalUnit string

const (
	IntervalDay   IntervalUnit = "day"
	IntervalWeek  IntervalUnit = "week"
	IntervalMonth IntervalUnit = "month"
	IntervalYear  IntervalUnit = "year"
)

type Plan struct {
	ID            uuid.UUID    `json:"id"`
	TenantID      uuid.UUID    `json:"tenant_id"`
	Name          string       `json:"name"`
	Code          string       `json:"code"`
	IntervalUnit  IntervalUnit `json:"interval_unit"`
	IntervalCount int          `json:"interval_count"`
	Active        bool         `json:"active"`
	// HSNCode is the plan's HSN/SAC code. Each invoice line for this plan is
	// taxed at this code's GST rate. Empty means "use the tenant SAC" (then the
	// 998314 default) at tax-resolution time — i.e. Phase-1 behaviour unchanged.
	HSNCode   string    `json:"hsn_code" db:"hsn_code"`
	CreatedAt time.Time `json:"created_at"`

	// Relationships
	Prices []Price `json:"prices,omitempty"`
}

type Price struct {
	ID        uuid.UUID `json:"id"`
	PlanID    uuid.UUID `json:"plan_id"`
	Currency  string    `json:"currency"` // ISO 3-letter code
	Amount    int64     `json:"amount"`   // Lowest unit
	Type      string    `json:"type"`     // 'recurring' or 'one_time'
	CreatedAt time.Time `json:"created_at"`
}

type PlanFilter struct {
	Search string
	Limit  int
	Offset int
}
