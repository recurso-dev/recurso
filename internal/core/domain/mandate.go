package domain

import (
	"time"

	"github.com/google/uuid"
)

type MandateStatus string

const (
	MandateStatusCreated    MandateStatus = "created"
	MandateStatusAuthorized MandateStatus = "authorized"
	MandateStatusActive     MandateStatus = "active"
	MandateStatusPaused     MandateStatus = "paused"
	MandateStatusRevoked    MandateStatus = "revoked"
)

type Mandate struct {
	ID                     uuid.UUID     `json:"id" db:"id"`
	TenantID               uuid.UUID     `json:"tenant_id" db:"tenant_id"`
	CustomerID             uuid.UUID     `json:"customer_id" db:"customer_id"`
	SubscriptionID         *uuid.UUID    `json:"subscription_id,omitempty" db:"subscription_id"`
	MandateType            string        `json:"mandate_type" db:"mandate_type"`
	PaymentMethod          string        `json:"payment_method" db:"payment_method"`
	VPA                    string        `json:"vpa,omitempty" db:"vpa"`
	RazorpayTokenID        string        `json:"razorpay_token_id,omitempty" db:"razorpay_token_id"`
	RazorpaySubscriptionID string        `json:"razorpay_subscription_id,omitempty" db:"razorpay_subscription_id"`
	RazorpayCustomerID     string        `json:"razorpay_customer_id,omitempty" db:"razorpay_customer_id"`
	MaxAmount              int64         `json:"max_amount" db:"max_amount"`
	Frequency              string        `json:"frequency" db:"frequency"`
	Status                 MandateStatus `json:"status" db:"status"`
	AuthorizedAt           *time.Time    `json:"authorized_at,omitempty" db:"authorized_at"`
	ActivatedAt            *time.Time    `json:"activated_at,omitempty" db:"activated_at"`
	RevokedAt              *time.Time    `json:"revoked_at,omitempty" db:"revoked_at"`
	LastDebitAt            *time.Time    `json:"last_debit_at,omitempty" db:"last_debit_at"`
	NextDebitAt            *time.Time    `json:"next_debit_at,omitempty" db:"next_debit_at"`
	PreDebitNotified       bool          `json:"pre_debit_notified" db:"pre_debit_notified"`
	CreatedAt              time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time     `json:"updated_at" db:"updated_at"`
}
