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
	TrialEnd               *time.Time         `json:"trial_end,omitempty" db:"trial_end"`             // set while status = trialing; nil for non-trial subs
	CancelAtPeriodEnd      bool               `json:"cancel_at_period_end" db:"cancel_at_period_end"` // P43
	CanceledAt             *time.Time         `json:"canceled_at,omitempty" db:"canceled_at"`
	CancellationReason     string             `json:"cancellation_reason,omitempty" db:"cancellation_reason"`
	CancellationFeedback   string             `json:"cancellation_feedback,omitempty" db:"cancellation_feedback"`
	BillingAnchor          time.Time          `json:"billing_anchor"`
	BillingAnchorType      string             `json:"billing_anchor_type"`                      // P15
	BillingAnchorDay       int                `json:"billing_anchor_day"`                       // P15
	PaymentTerms           string             `json:"payment_terms"`                            // P15
	CouponID               *uuid.UUID         `json:"coupon_id,omitempty"`                      // P7
	ReferenceID            string             `json:"reference_id,omitempty" db:"reference_id"` // P43
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
		return s.addIntervalAnchored(startDate, intervalUnit, intervalCount)
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

	return s.addIntervalAnchored(startDate, intervalUnit, intervalCount)
}

// addIntervalAnchored is AddInterval plus month-end anchor restoration: a
// subscription anchored on the 31st that was clamped to Feb 28 returns to the
// 31st in March instead of staying "sticky" at 28. Restoration happens only
// when advancing FROM a clamped month-end date toward a later anchor day, so
// cycles that legitimately run mid-month (e.g. after a trial conversion) keep
// their current day.
func (s *Subscription) addIntervalAnchored(t time.Time, unit string, count int) time.Time {
	next := AddInterval(t, unit, count)
	if unit != "month" && unit != "year" && unit != "" {
		return next
	}
	anchorDay := s.BillingAnchorDay
	if anchorDay <= 0 && !s.BillingAnchor.IsZero() {
		anchorDay = s.BillingAnchor.Day()
	}
	y, m, d := t.Date()
	if anchorDay <= d || d != daysInMonth(y, m) {
		return next
	}
	ny, nm, _ := next.Date()
	if last := daysInMonth(ny, nm); anchorDay > last {
		anchorDay = last
	}
	return time.Date(ny, nm, anchorDay, next.Hour(), next.Minute(), next.Second(), next.Nanosecond(), next.Location())
}

// AddInterval adds a billing interval to a time. Month/year math CLAMPS the day
// to the target month's last valid day rather than overflowing: Go's time.AddDate
// normalizes Jan 31 + 1 month to Mar 3, which skips February and permanently
// drifts the billing cycle off its anchor. Exported for use in advance invoicing.
func AddInterval(t time.Time, unit string, count int) time.Time {
	switch unit {
	case "day":
		return t.AddDate(0, 0, count)
	case "week":
		return t.AddDate(0, 0, count*7)
	case "month":
		return addMonthsClamped(t, count)
	case "year":
		return addMonthsClamped(t, count*12)
	default:
		return addMonthsClamped(t, 1) // Default to 1 month
	}
}

// addMonthsClamped adds n months, clamping the day to the target month's last
// day so it never rolls into the following month (Jan 31 + 1 => Feb 28/29, not
// Mar 3). Time-of-day and location are preserved.
func addMonthsClamped(t time.Time, n int) time.Time {
	y, m, d := t.Date()
	// The 1st of any month never overflows, so advancing from it lands on the
	// correct target year/month; then clamp the original day to that month.
	target := time.Date(y, m, 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location()).AddDate(0, n, 0)
	ty, tm, _ := target.Date()
	if last := daysInMonth(ty, tm); d > last {
		d = last
	}
	return time.Date(ty, tm, d, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}

// daysInMonth returns the number of days in the given month, handling leap years.
// Day 0 of the next month is the last day of this month.
func daysInMonth(year int, m time.Month) int {
	return time.Date(year, m+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

type SubscriptionFilter struct {
	Status     string // "active", "trialing", etc.
	CustomerID uuid.UUID
	Search     string
	Limit      int
	Offset     int
}
