package domain

import (
	"time"

	"github.com/google/uuid"
)

type UsageEvent struct {
	ID             uuid.UUID `json:"id"`
	SubscriptionID uuid.UUID `json:"subscription_id"`
	CustomerID     uuid.UUID `json:"customer_id"`
	Dimension      string    `json:"dimension"` // e.g., "api_calls", "storage_gb"
	Quantity       int64     `json:"quantity"`
	Timestamp      time.Time `json:"timestamp"`
}
