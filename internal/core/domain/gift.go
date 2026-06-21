package domain

import (
	"time"

	"github.com/google/uuid"
)

type GiftStatus string

const (
	GiftStatusPurchased GiftStatus = "purchased"
	GiftStatusRedeemed  GiftStatus = "redeemed"
)

type Gift struct {
	ID                   uuid.UUID  `json:"id" db:"id"`
	TenantID             uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	Code                 string     `json:"code" db:"code"`
	PlanID               uuid.UUID  `json:"plan_id" db:"plan_id"`
	BuyerCustomerID      uuid.UUID  `json:"buyer_customer_id" db:"buyer_customer_id"`
	RecipientEmail       string     `json:"recipient_email" db:"recipient_email"`
	Status               GiftStatus `json:"status" db:"status"`
	RedeemedByCustomerID *uuid.UUID `json:"redeemed_by_customer_id" db:"redeemed_by_customer_id"`
	RedeemedAt           *time.Time `json:"redeemed_at" db:"redeemed_at"`
	DurationMonths       int        `json:"duration_months" db:"duration_months"` // How long the gift lasts (e.g. 12 months)
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at" db:"updated_at"`
}
