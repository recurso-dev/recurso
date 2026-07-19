package domain

import (
	"time"

	"github.com/google/uuid"
)

type DiscountType string
type DurationType string

const (
	DiscountTypePercent DiscountType = "percent"
	DiscountTypeAmount  DiscountType = "amount"

	DurationForever   DurationType = "forever"
	DurationOnce      DurationType = "once"
	DurationRepeating DurationType = "repeating"
)

type Coupon struct {
	ID             uuid.UUID    `json:"id"`
	TenantID       uuid.UUID    `json:"tenant_id"`
	Code           string       `json:"code"`
	DiscountType   DiscountType `json:"discount_type"`
	DiscountValue  int64        `json:"discount_value"`
	Duration       DurationType `json:"duration"`
	DurationMonths *int         `json:"duration_months,omitempty"`
	// Active gates redemption: deactivated coupons are rejected when creating
	// a subscription, but existing subscriptions keep their applied discount.
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
