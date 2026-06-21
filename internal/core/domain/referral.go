package domain

import (
	"time"

	"github.com/google/uuid"
)

type ReferralStatus string

const (
	ReferralStatusPending   ReferralStatus = "pending"
	ReferralStatusQualified ReferralStatus = "qualified"
	ReferralStatusRewarded  ReferralStatus = "rewarded"
)

type Referral struct {
	ID           uuid.UUID      `json:"id" db:"id"`
	TenantID     uuid.UUID      `json:"tenant_id" db:"tenant_id"`
	ReferrerID   uuid.UUID      `json:"referrer_id" db:"referrer_id"`
	ReferredID   uuid.UUID      `json:"referred_id" db:"referred_id"`
	Code         string         `json:"code" db:"code"`
	Status       ReferralStatus `json:"status" db:"status"`
	RewardAmount int64          `json:"reward_amount" db:"reward_amount"`
	Currency     string         `json:"currency" db:"currency"`
	CreatedAt    time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at" db:"updated_at"`
	QualifiedAt  *time.Time     `json:"qualified_at" db:"qualified_at"`
}
