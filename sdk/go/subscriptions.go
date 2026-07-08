package recurso

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Subscription is a customer's recurring subscription.
type Subscription struct {
	ID                     string    `json:"id"`
	TenantID               string    `json:"tenant_id"`
	CustomerID             string    `json:"customer_id"`
	PlanID                 string    `json:"plan_id"`
	Status                 string    `json:"status"`
	CurrentPeriodStart     time.Time `json:"current_period_start"`
	CurrentPeriodEnd       time.Time `json:"current_period_end"`
	CancelAtPeriodEnd      bool      `json:"cancel_at_period_end"`
	CanceledAt             time.Time `json:"canceled_at"`
	CancellationReason     string    `json:"cancellation_reason"`
	CancellationFeedback   string    `json:"cancellation_feedback"`
	BillingAnchor          time.Time `json:"billing_anchor"`
	BillingAnchorType      string    `json:"billing_anchor_type"`
	BillingAnchorDay       int       `json:"billing_anchor_day"`
	PaymentTerms           string    `json:"payment_terms"`
	CouponID               string    `json:"coupon_id"`
	ReferenceID            string    `json:"reference_id"`
	MandateID              string    `json:"mandate_id"`
	RazorpaySubscriptionID string    `json:"razorpay_subscription_id"`
	StripeSubscriptionID   string    `json:"stripe_subscription_id"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

// SubscriptionCreateParams is the body for creating a subscription.
type SubscriptionCreateParams struct {
	CustomerID        string `json:"customer_id"`
	PlanID            string `json:"plan_id"`
	CouponCode        string `json:"coupon_code,omitempty"`
	StartDate         string `json:"start_date,omitempty"`
	BillingAnchorType string `json:"billing_anchor_type,omitempty"`
	PaymentTerms      string `json:"payment_terms,omitempty"`
}

// SubscriptionListParams filters the subscription list.
type SubscriptionListParams struct {
	Status string
	Q      string
	Limit  int
	Page   int
}

// SubscriptionUpdateParams changes a subscription's plan (upgrade/downgrade).
type SubscriptionUpdateParams struct {
	PlanID string `json:"plan_id"`
}

// SubscriptionCancelParams controls how a subscription is canceled. When
// Immediately is nil/false the cancellation takes effect at period end.
type SubscriptionCancelParams struct {
	CancelAtPeriodEnd *bool  `json:"cancel_at_period_end,omitempty"`
	Immediately       *bool  `json:"immediately,omitempty"`
	Reason            string `json:"reason"`
	Feedback          string `json:"feedback,omitempty"`
	RevokeConsent     *bool  `json:"revoke_consent,omitempty"`
}

// CancelResult is returned by Cancel.
type CancelResult struct {
	ID                 string    `json:"id"`
	Status             string    `json:"status"`
	CancelAtPeriodEnd  bool      `json:"cancel_at_period_end"`
	CancelledAt        time.Time `json:"cancelled_at"`
	CurrentPeriodEnd   time.Time `json:"current_period_end"`
	CancellationReason string    `json:"cancellation_reason"`
	Message            string    `json:"message"`
}

// ReactivateResult is returned by Reactivate.
type ReactivateResult struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// UnbilledCharge is an ad-hoc charge queued onto a subscription's next invoice.
type UnbilledCharge struct {
	ID             string    `json:"id"`
	SubscriptionID string    `json:"subscription_id"`
	Amount         int64     `json:"amount"`
	Currency       string    `json:"currency"`
	Description    string    `json:"description"`
	HSNCode        string    `json:"hsn_code"`
	Status         string    `json:"status"`
	PeriodStart    time.Time `json:"period_start"`
	PeriodEnd      time.Time `json:"period_end"`
	CreatedAt      time.Time `json:"created_at"`
}

// ChargeCreateParams adds an unbilled charge.
type ChargeCreateParams struct {
	Amount      int64  `json:"amount"`
	Currency    string `json:"currency"`
	Description string `json:"description"`
	HSNCode     string `json:"hsn_code,omitempty"`
}

// AdvanceParams bills a subscription ahead by the given number of periods.
type AdvanceParams struct {
	Periods int `json:"periods"`
}

// SubscriptionAddon is a plan attached to a subscription as a priced add-on.
type SubscriptionAddon struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	SubscriptionID string    `json:"subscription_id"`
	PlanID         string    `json:"plan_id"`
	Quantity       int       `json:"quantity"`
	CreatedAt      time.Time `json:"created_at"`
}

// AddonCreateParams attaches an add-on plan to a subscription.
type AddonCreateParams struct {
	PlanID   string `json:"plan_id"`
	Quantity int    `json:"quantity"`
}

// PlanChangePreview is the read-only proration breakdown for a plan change.
type PlanChangePreview struct {
	SubscriptionID    string    `json:"subscription_id"`
	CurrentPlanID     string    `json:"current_plan_id"`
	NewPlanID         string    `json:"new_plan_id"`
	Currency          string    `json:"currency"`
	CreditAmount      int64     `json:"credit_amount"`
	ChargeAmount      int64     `json:"charge_amount"`
	NetAmount         int64     `json:"net_amount"`
	TaxAmount         int64     `json:"tax_amount"`
	TotalAmount       int64     `json:"total_amount"`
	EffectiveDate     time.Time `json:"effective_date"`
	NextInvoiceAmount int64     `json:"next_invoice_amount"`
	IsUpgrade         bool      `json:"is_upgrade"`
}

// SubscriptionDimensionUsage is one dimension's usage for a subscription.
type SubscriptionDimensionUsage struct {
	Dimension        string `json:"dimension"`
	PeriodQuantity   int64  `json:"period_quantity"`
	LifetimeQuantity int64  `json:"lifetime_quantity"`
	LimitValue       *int64 `json:"limit_value"`
	Remaining        *int64 `json:"remaining"`
}

// SubscriptionUsage is the current-period usage report for a subscription.
type SubscriptionUsage struct {
	SubscriptionID     string                       `json:"subscription_id"`
	CustomerID         string                       `json:"customer_id"`
	CurrentPeriodStart time.Time                    `json:"current_period_start"`
	CurrentPeriodEnd   time.Time                    `json:"current_period_end"`
	Dimensions         []SubscriptionDimensionUsage `json:"dimensions"`
}

// SubscriptionsService groups the subscription lifecycle endpoints.
type SubscriptionsService struct{ client *Client }

// Create creates a subscription and generates its first invoice.
func (s *SubscriptionsService) Create(ctx context.Context, params *SubscriptionCreateParams) (*Subscription, error) {
	var out Subscription
	if err := s.client.do(ctx, http.MethodPost, "/subscriptions", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// List returns the tenant's subscriptions.
func (s *SubscriptionsService) List(ctx context.Context, params *SubscriptionListParams) ([]Subscription, error) {
	path := "/subscriptions"
	if params != nil {
		path = newQuery().
			str("status", params.Status).
			str("q", params.Q).
			int("limit", params.Limit).
			int("page", params.Page).
			apply(path)
	}
	return getData[[]Subscription](ctx, s.client, http.MethodGet, path, nil)
}

// Update changes the subscription's plan.
func (s *SubscriptionsService) Update(ctx context.Context, id string, params *SubscriptionUpdateParams) (*Subscription, error) {
	var out Subscription
	if err := s.client.do(ctx, http.MethodPut, "/subscriptions/"+id, params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// PreviewChange previews the proration for switching to newPlanID. Nothing is
// charged or persisted.
func (s *SubscriptionsService) PreviewChange(ctx context.Context, id, newPlanID string) (*PlanChangePreview, error) {
	path := newQuery().str("plan_id", newPlanID).apply(fmt.Sprintf("/subscriptions/%s/preview-change", id))
	var out PlanChangePreview
	if err := s.client.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Cancel cancels a subscription (at period end by default).
func (s *SubscriptionsService) Cancel(ctx context.Context, id string, params *SubscriptionCancelParams) (*CancelResult, error) {
	var out CancelResult
	if err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/subscriptions/%s/cancel", id), params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Reactivate reactivates a cancelled subscription.
func (s *SubscriptionsService) Reactivate(ctx context.Context, id string) (*ReactivateResult, error) {
	var out ReactivateResult
	if err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/subscriptions/%s/reactivate", id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Pause pauses a subscription.
func (s *SubscriptionsService) Pause(ctx context.Context, id string) (*Subscription, error) {
	out, err := getData[Subscription](ctx, s.client, http.MethodPost, fmt.Sprintf("/subscriptions/%s/pause", id), nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Resume resumes a paused subscription.
func (s *SubscriptionsService) Resume(ctx context.Context, id string) (*Subscription, error) {
	out, err := getData[Subscription](ctx, s.client, http.MethodPost, fmt.Sprintf("/subscriptions/%s/resume", id), nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Charges lists a subscription's pending unbilled charges.
func (s *SubscriptionsService) Charges(ctx context.Context, id string) ([]UnbilledCharge, error) {
	return getData[[]UnbilledCharge](ctx, s.client, http.MethodGet, fmt.Sprintf("/subscriptions/%s/charges", id), nil)
}

// AddCharge records an ad-hoc charge for the subscription's next invoice.
func (s *SubscriptionsService) AddCharge(ctx context.Context, id string, params *ChargeCreateParams) (*UnbilledCharge, error) {
	var out UnbilledCharge
	if err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/subscriptions/%s/charges", id), params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Advance bills the subscription ahead for the given number of periods.
func (s *SubscriptionsService) Advance(ctx context.Context, id string, params *AdvanceParams) (*Invoice, error) {
	var out Invoice
	if err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/subscriptions/%s/advance", id), params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Usage returns the subscription's current-period usage report.
func (s *SubscriptionsService) Usage(ctx context.Context, id string) (*SubscriptionUsage, error) {
	var out SubscriptionUsage
	if err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/subscriptions/%s/usage", id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// AddAddon attaches an add-on plan to a subscription.
func (s *SubscriptionsService) AddAddon(ctx context.Context, id string, params *AddonCreateParams) (*SubscriptionAddon, error) {
	var out SubscriptionAddon
	if err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/subscriptions/%s/addons", id), params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListAddons lists a subscription's add-ons.
func (s *SubscriptionsService) ListAddons(ctx context.Context, id string) ([]SubscriptionAddon, error) {
	return getData[[]SubscriptionAddon](ctx, s.client, http.MethodGet, fmt.Sprintf("/subscriptions/%s/addons", id), nil)
}

// RemoveAddon detaches an add-on from a subscription.
func (s *SubscriptionsService) RemoveAddon(ctx context.Context, id, addonID string) error {
	return s.client.do(ctx, http.MethodDelete, fmt.Sprintf("/subscriptions/%s/addons/%s", id, addonID), nil, nil)
}
