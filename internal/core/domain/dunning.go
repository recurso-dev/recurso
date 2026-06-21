package domain

import (
	"time"

	"github.com/google/uuid"
)

// DunningAction represents a specific retry interval "arm" in the Bandit
type DunningAction struct {
	ID       string        `json:"id"`
	Interval time.Duration `json:"interval"`
}

// DunningContext represents the features used to select an action
type DunningContext struct {
	Currency     string `json:"currency"`
	ErrorCode    string `json:"error_code"`
	AttemptCount int    `json:"attempt_count"`
}

// Key returns a string representation of the context for DB lookup
func (c DunningContext) Key() string {
	return c.Currency + ":" + c.ErrorCode
}

// DunningWeight persists the learned value (Expected Reward) for an action in a context
type DunningWeight struct {
	ContextKey    string    `json:"context_key" db:"context_key"`
	ActionID      string    `json:"action_id" db:"action_id"`
	AverageReward float64   `json:"average_reward" db:"average_reward"` // Success rate (0.0 to 1.0)
	SampleCount   int64     `json:"sample_count" db:"sample_count"`     // Total attempts made
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// DunningHistory tracks individual retry attempts and their outcomes
type DunningHistory struct {
	ID             uuid.UUID `json:"id" db:"id"`
	TenantID       uuid.UUID `json:"tenant_id" db:"tenant_id"`
	InvoiceID      uuid.UUID `json:"invoice_id" db:"invoice_id"`
	ContextKey     string    `json:"context_key" db:"context_key"`
	ActionID       string    `json:"action_id" db:"action_id"`
	RetryInterval  int64     `json:"retry_interval" db:"retry_interval"` // Seconds
	Outcome        string    `json:"outcome" db:"outcome"`               // "success" or "failure"
	Reward         float64   `json:"reward" db:"reward"`                 // 1.0 for success, 0.0 for failure
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// Standard Dunning Actions
var (
	Action1Hour  = DunningAction{ID: "1h", Interval: 1 * time.Hour}
	Action24Hour = DunningAction{ID: "24h", Interval: 24 * time.Hour}
	Action3Day   = DunningAction{ID: "3d", Interval: 72 * time.Hour}
	Action7Day   = DunningAction{ID: "7d", Interval: 168 * time.Hour}

	DefaultDunningActions = []DunningAction{
		Action1Hour,
		Action24Hour,
		Action3Day,
		Action7Day,
	}
)
