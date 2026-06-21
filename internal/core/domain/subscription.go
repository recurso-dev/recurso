package domain

import (
	"time"

	"github.com/google/uuid"
)

type SubscriptionStatus string

const (
	SubscriptionStatusTrialing SubscriptionStatus = "trialing"
	SubscriptionStatusActive   SubscriptionStatus = "active"
	SubscriptionStatusPastDue  SubscriptionStatus = "past_due"
	SubscriptionStatusPaused   SubscriptionStatus = "paused"
	SubscriptionStatusCanceled SubscriptionStatus = "canceled"
	SubscriptionStatusUnpaid   SubscriptionStatus = "unpaid"
)

type Subscription struct {
	ID                     uuid.UUID          `json:"id"`
	TenantID               uuid.UUID          `json:"tenant_id"`
	CustomerID             uuid.UUID          `json:"customer_id"`
	PlanID                 uuid.UUID          `json:"plan_id"`
	Status                 SubscriptionStatus `json:"status"`
	CurrentPeriodStart     time.Time          `json:"current_period_start" db:"current_period_start"`
	CurrentPeriodEnd       time.Time          `json:"current_period_end" db:"current_period_end"`
	CancelAtPeriodEnd      bool               `json:"cancel_at_period_end" db:"cancel_at_period_end"` // P43
	CanceledAt             *time.Time         `json:"canceled_at,omitempty" db:"canceled_at"`
	CancellationReason     string             `json:"cancellation_reason,omitempty" db:"cancellation_reason"`
	CancellationFeedback   string             `json:"cancellation_feedback,omitempty" db:"cancellation_feedback"`
	BillingAnchor          time.Time          `json:"billing_anchor"`
	BillingAnchorType      string             `json:"billing_anchor_type"`                                    // P15
	BillingAnchorDay       int                `json:"billing_anchor_day"`                                     // P15
	PaymentTerms           string             `json:"payment_terms"`                                          // P15
	CouponID               *uuid.UUID         `json:"coupon_id,omitempty"`                                    // P7
	ReferenceID            string             `json:"reference_id,omitempty" db:"reference_id"`               // P43
	MandateID              *uuid.UUID         `json:"mandate_id,omitempty" db:"mandate_id"`
	RazorpaySubscriptionID string             `json:"razorpay_subscription_id" db:"razorpay_subscription_id"` // P24
	StripeSubscriptionID   string             `json:"stripe_subscription_id" db:"stripe_subscription_id"`     // P26
	CreatedAt              time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time          `json:"updated_at" db:"updated_at"`
}

// CalculateNextBillingDate calculates the end date of the next period based on the anchor.
// interval: "month", "year"
func (s *Subscription) CalculateNextBillingDate(intervalUnit string, intervalCount int) time.Time {
	startDate := s.CurrentPeriodEnd
	if startDate.IsZero() {
		startDate = time.Now()
	}

	// Default: Acquisition based (just add interval)
	if s.BillingAnchorType == "acquisition" || s.BillingAnchorType == "" {
		return AddInterval(startDate, intervalUnit, intervalCount)
	}

	// Calendar Billing: First of Month
	if s.BillingAnchorType == "first_of_month" {
		// If we are already aligned (day is 1), just add interval
		if startDate.Day() == 1 {
			return AddInterval(startDate, intervalUnit, intervalCount)
		}

		// Otherwise, prorate to the 1st of next month
		// Find 1st of next month
		year, month, _ := startDate.Date()
		firstOfNextMonth := time.Date(year, month, 1, 0, 0, 0, 0, startDate.Location()).AddDate(0, 1, 0)
		return firstOfNextMonth
	}

	return AddInterval(startDate, intervalUnit, intervalCount)
}

// AddInterval adds a billing interval to a time. Exported for use in advance invoicing.
func AddInterval(t time.Time, unit string, count int) time.Time {
	switch unit {
	case "day":
		return t.AddDate(0, 0, count)
	case "week":
		return t.AddDate(0, 0, count*7)
	case "month":
		return t.AddDate(0, count, 0)
	case "year":
		return t.AddDate(count, 0, 0)
	default:
		return t.AddDate(0, 1, 0) // Default to 1 month
	}
}

type SubscriptionFilter struct {
	Status     string // "active", "trialing", etc.
	CustomerID uuid.UUID
	Search     string
	Limit      int
	Offset     int
}
