package domain

import (
	"time"

	"github.com/google/uuid"
)

// SubscriptionChange is one recorded transition of a subscription — captured by
// a trigger so none is missed. ChangeType is "status" (a lifecycle transition;
// values are status strings) or "plan" (a plan switch; values are plan ids,
// resolved to names in the UI). The first status row has a null FromValue.
type SubscriptionChange struct {
	ID             uuid.UUID `json:"id"`
	SubscriptionID uuid.UUID `json:"subscription_id"`
	ChangeType     string    `json:"change_type"`
	FromValue      *string   `json:"from_value,omitempty"`
	ToValue        *string   `json:"to_value,omitempty"`
	ChangedAt      time.Time `json:"changed_at"`
}
