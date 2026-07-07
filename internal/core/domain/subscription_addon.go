package domain

import (
	"time"

	"github.com/google/uuid"
)

// SubscriptionAddon is an existing plan attached to a subscription with a
// quantity (Multi-product catalog v1, Lane 2). The subscription's base plan is
// unchanged; the add-on is billed as an extra line — its plan price × quantity,
// taxed independently — on the subscription's next recurring invoice.
type SubscriptionAddon struct {
	ID             uuid.UUID `json:"id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	SubscriptionID uuid.UUID `json:"subscription_id"`
	PlanID         uuid.UUID `json:"plan_id"`
	Quantity       int       `json:"quantity"`
	CreatedAt      time.Time `json:"created_at"`
}
