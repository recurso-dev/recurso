package domain

import (
	"time"

	"github.com/google/uuid"
)

// SubscriptionFinancialSummary is one subscription's financial position: its
// recurring value (MRR + list price), when/what it bills next, and its
// invoice-derived outstanding position. It exists so the subscription object
// page can answer "what is happening financially?" without the client
// re-deriving MRR (which must match the tenant-wide definition) or aggregating a
// paginated invoice list.
//
// MRR uses the SAME definition as the tenant-wide GetMRR / CaptureMRRSnapshot:
// the plan list price normalized to a per-month figure, counted ONLY when the
// subscription is active (0 otherwise). It deliberately excludes coupons,
// trials, add-ons, metered usage and commitment true-ups — none of which are in
// the recurring MRR figure elsewhere either.
type SubscriptionFinancialSummary struct {
	SubscriptionID uuid.UUID `json:"subscription_id"`
	Status         string    `json:"status"`
	// Currency is the subscription's own (its plan price currency). A subscription
	// is single-currency, so MRR/recurring_amount are in this currency; the
	// outstanding position is still reported per-currency for shape-fidelity with
	// the customer summary.
	Currency string `json:"currency"`
	// MRR is the monthly-normalized recurring value in Currency (minor units).
	// 0 for any non-active subscription, by the same rule GetMRR uses.
	MRR int64 `json:"mrr"`
	// RecurringAmount is the plan's list price charged once per interval (minor
	// units); IntervalUnit/IntervalCount describe that period.
	RecurringAmount int64  `json:"recurring_amount"`
	IntervalUnit    string `json:"interval_unit"`
	IntervalCount   int    `json:"interval_count"`

	CurrentPeriodStart time.Time `json:"current_period_start"`
	CurrentPeriodEnd   time.Time `json:"current_period_end"`

	// NextInvoiceDate is when the subscription next bills — set only when it will
	// actually renew (active/trialing and not cancel-at-period-end). Nil when the
	// subscription will not produce another invoice (paused/canceled/etc.).
	NextInvoiceDate *time.Time `json:"next_invoice_date,omitempty"`
	// NextInvoiceBaseAmount is the plan LIST price that will recur (minor units).
	// It is the base only: it excludes tax, coupon discounts, add-ons, metered
	// usage and commitment true-ups, which have no single deterministic source
	// today (documented gap). Never present this as the total amount due.
	NextInvoiceBaseAmount int64 `json:"next_invoice_base_amount"`

	// Outstanding is the invoice-derived position for THIS subscription's
	// invoices, one row per currency (outstanding, past-due + count, billed,
	// paid). Empty when the subscription has issued no invoices.
	Outstanding []CustomerFinancialSummaryCurrency `json:"outstanding"`

	// CouponID / DiscountActive expose the current discount context (they do NOT
	// affect MRR, matching the engine's MRR definition).
	CouponID       *uuid.UUID `json:"coupon_id,omitempty"`
	DiscountActive bool       `json:"discount_active"`
}
