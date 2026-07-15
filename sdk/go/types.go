package recurso

// This file defines the request and response types for the API surface. Field
// names and JSON tags mirror the OpenAPI schema (the same source the Node and
// Python SDKs are generated from). Monetary amounts are int64 minor units.

// Params is a free-form JSON body for endpoints whose request shape is small or
// flexible. Prefer a typed *…Params struct where one is provided.
type Params = map[string]any

// ListParams are the common query parameters accepted by list endpoints.
type ListParams struct {
	Page   int
	Limit  int
	Q      string
	Status string
}

// --- Resources ---

// Price is a plan's price in one currency.
type Price struct {
	ID        string `json:"id,omitempty"`
	PlanID    string `json:"plan_id,omitempty"`
	Currency  string `json:"currency,omitempty"`
	Amount    int64  `json:"amount,omitempty"`
	Type      string `json:"type,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

// Plan is a subscription plan.
type Plan struct {
	ID            string  `json:"id,omitempty"`
	TenantID      string  `json:"tenant_id,omitempty"`
	Name          string  `json:"name,omitempty"`
	Code          string  `json:"code,omitempty"`
	IntervalUnit  string  `json:"interval_unit,omitempty"`
	IntervalCount int     `json:"interval_count,omitempty"`
	Active        bool    `json:"active,omitempty"`
	HSNCode       string  `json:"hsn_code,omitempty"`
	CreatedAt     string  `json:"created_at,omitempty"`
	Prices        []Price `json:"prices,omitempty"`
}

// BillingAddress is a customer's postal address.
type BillingAddress struct {
	Line1   string `json:"line1,omitempty"`
	City    string `json:"city,omitempty"`
	State   string `json:"state,omitempty"`
	Zip     string `json:"zip,omitempty"`
	Country string `json:"country,omitempty"`
}

// Customer is a billing customer.
type Customer struct {
	ID             string          `json:"id,omitempty"`
	TenantID       string          `json:"tenant_id,omitempty"`
	Email          string          `json:"email,omitempty"`
	Name           string          `json:"name,omitempty"`
	Phone          string          `json:"phone,omitempty"`
	TaxID          string          `json:"tax_id,omitempty"`
	BillingAddress *BillingAddress `json:"billing_address,omitempty"`
	GSTIN          string          `json:"gstin,omitempty"`
	TaxType        string          `json:"tax_type,omitempty"`
	PlaceOfSupply  string          `json:"place_of_supply,omitempty"`
	ReferralCode   string          `json:"referral_code,omitempty"`
	RiskScore      int             `json:"risk_score,omitempty"`
	CardBrand      string          `json:"card_brand,omitempty"`
	CardLast4      string          `json:"card_last4,omitempty"`
	CreatedAt      string          `json:"created_at,omitempty"`
}

// Subscription is a customer's subscription to a plan.
type Subscription struct {
	ID                 string `json:"id,omitempty"`
	TenantID           string `json:"tenant_id,omitempty"`
	CustomerID         string `json:"customer_id,omitempty"`
	PlanID             string `json:"plan_id,omitempty"`
	Status             string `json:"status,omitempty"`
	CurrentPeriodStart string `json:"current_period_start,omitempty"`
	CurrentPeriodEnd   string `json:"current_period_end,omitempty"`
	CancelAtPeriodEnd  bool   `json:"cancel_at_period_end,omitempty"`
	CanceledAt         string `json:"canceled_at,omitempty"`
	CancellationReason string `json:"cancellation_reason,omitempty"`
	BillingAnchor      string `json:"billing_anchor,omitempty"`
	BillingAnchorType  string `json:"billing_anchor_type,omitempty"`
	PaymentTerms       string `json:"payment_terms,omitempty"`
	CouponID           string `json:"coupon_id,omitempty"`
	ReferenceID        string `json:"reference_id,omitempty"`
	MandateID          string `json:"mandate_id,omitempty"`
	CreatedAt          string `json:"created_at,omitempty"`
	UpdatedAt          string `json:"updated_at,omitempty"`
}

// Invoice is a billing invoice. Amounts are int64 minor units.
type Invoice struct {
	ID             string `json:"id,omitempty"`
	TenantID       string `json:"tenant_id,omitempty"`
	SubscriptionID string `json:"subscription_id,omitempty"`
	CustomerID     string `json:"customer_id,omitempty"`
	InvoiceNumber  string `json:"invoice_number,omitempty"`
	Status         string `json:"status,omitempty"`
	Currency       string `json:"currency,omitempty"`
	Subtotal       int64  `json:"subtotal,omitempty"`
	TaxAmount      int64  `json:"tax_amount,omitempty"`
	Total          int64  `json:"total,omitempty"`
	AmountPaid     int64  `json:"amount_paid,omitempty"`
	EInvoiceStatus string `json:"e_invoice_status,omitempty"`
	IRN            string `json:"irn,omitempty"`
	CreatedAt      string `json:"created_at,omitempty"`
	DueDate        string `json:"due_date,omitempty"`
	PaidAt         string `json:"paid_at,omitempty"`
}

// Coupon is a discount coupon.
type Coupon struct {
	ID            string `json:"id,omitempty"`
	Code          string `json:"code,omitempty"`
	DiscountType  string `json:"discount_type,omitempty"`
	DiscountValue int64  `json:"discount_value,omitempty"`
	Duration      string `json:"duration,omitempty"`
	CreatedAt     string `json:"created_at,omitempty"`
}

// CreditNote is a credit note against an invoice.
type CreditNote struct {
	ID         string `json:"id,omitempty"`
	CustomerID string `json:"customer_id,omitempty"`
	InvoiceID  string `json:"invoice_id,omitempty"`
	Amount     int64  `json:"amount,omitempty"`
	Currency   string `json:"currency,omitempty"`
	Status     string `json:"status,omitempty"`
	Reason     string `json:"reason,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
}

// Quote is a sales quote.
type Quote struct {
	ID         string `json:"id,omitempty"`
	CustomerID string `json:"customer_id,omitempty"`
	Status     string `json:"status,omitempty"`
	Total      int64  `json:"total,omitempty"`
	Currency   string `json:"currency,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
}

// WebhookEndpoint is a registered webhook delivery target.
type WebhookEndpoint struct {
	ID        string   `json:"id,omitempty"`
	TenantID  string   `json:"tenant_id,omitempty"`
	URL       string   `json:"url,omitempty"`
	Events    []string `json:"events,omitempty"`
	Status    string   `json:"status,omitempty"`
	CreatedAt string   `json:"created_at,omitempty"`
}

// Event is a platform event.
type Event struct {
	ID         string         `json:"id,omitempty"`
	TenantID   string         `json:"tenant_id,omitempty"`
	Type       string         `json:"type,omitempty"`
	ObjectType string         `json:"object_type,omitempty"`
	ObjectID   string         `json:"object_id,omitempty"`
	Data       map[string]any `json:"data,omitempty"`
	CreatedAt  string         `json:"created_at,omitempty"`
}

// EventDelivery is one delivery attempt of an event to an endpoint.
type EventDelivery struct {
	ID                string `json:"id,omitempty"`
	EventID           string `json:"event_id,omitempty"`
	WebhookEndpointID string `json:"webhook_endpoint_id,omitempty"`
	StatusCode        int    `json:"status_code,omitempty"`
	Attempt           int    `json:"attempt,omitempty"`
	NextRetryAt       string `json:"next_retry_at,omitempty"`
	DeliveredAt       string `json:"delivered_at,omitempty"`
}

// Mandate is a recurring-payment mandate (e.g. UPI autopay).
type Mandate struct {
	ID             string `json:"id,omitempty"`
	TenantID       string `json:"tenant_id,omitempty"`
	CustomerID     string `json:"customer_id,omitempty"`
	SubscriptionID string `json:"subscription_id,omitempty"`
	Status         string `json:"status,omitempty"`
	MaxAmount      int64  `json:"max_amount,omitempty"`
	Frequency      string `json:"frequency,omitempty"`
	CreatedAt      string `json:"created_at,omitempty"`
}

// Gift is a gifted subscription.
type Gift struct {
	ID        string `json:"id,omitempty"`
	Code      string `json:"code,omitempty"`
	Status    string `json:"status,omitempty"`
	PlanID    string `json:"plan_id,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

// Referral is a customer referral.
type Referral struct {
	ID         string `json:"id,omitempty"`
	ReferrerID string `json:"referrer_id,omitempty"`
	ReferredID string `json:"referred_id,omitempty"`
	Status     string `json:"status,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
}

// Entitlement is a plan or customer feature grant.
type Entitlement struct {
	FeatureKey string `json:"feature_key,omitempty"`
	Kind       string `json:"kind,omitempty"`
	BoolValue  bool   `json:"bool_value,omitempty"`
	LimitValue int64  `json:"limit_value,omitempty"`
}

// LedgerAccount is a double-entry ledger account.
type LedgerAccount struct {
	ID            string `json:"id,omitempty"`
	TenantID      string `json:"tenant_id,omitempty"`
	Name          string `json:"name,omitempty"`
	Type          string `json:"type,omitempty"`
	Currency      string `json:"currency,omitempty"`
	Balance       int64  `json:"balance,omitempty"`
	DebitsPosted  int64  `json:"debits_posted,omitempty"`
	CreditsPosted int64  `json:"credits_posted,omitempty"`
}

// LedgerTransaction is a posted double-entry transaction.
type LedgerTransaction struct {
	ID              string `json:"id,omitempty"`
	DebitAccountID  string `json:"debit_account_id,omitempty"`
	CreditAccountID string `json:"credit_account_id,omitempty"`
	Amount          int64  `json:"amount,omitempty"`
	Currency        string `json:"currency,omitempty"`
	Description     string `json:"description,omitempty"`
	CreatedAt       string `json:"created_at,omitempty"`
}

// MRRMetrics is FX-normalized monthly recurring revenue with provenance.
type MRRMetrics struct {
	MRR               int64          `json:"mrr,omitempty"`
	NormalizedMRR     int64          `json:"normalized_mrr,omitempty"`
	ReportingCurrency string         `json:"reporting_currency,omitempty"`
	Breakdown         []any          `json:"breakdown,omitempty"`
	FX                map[string]any `json:"fx,omitempty"`
}

// --- Request params ---

// PlanCreateParams is the body for Plans.Create.
type PlanCreateParams struct {
	Name          string `json:"name"`
	Code          string `json:"code"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	IntervalUnit  string `json:"interval_unit"`
	IntervalCount int    `json:"interval_count,omitempty"`
}

// CustomerCreateParams is the body for Customers.Create.
type CustomerCreateParams struct {
	Email         string `json:"email"`
	Name          string `json:"name"`
	Phone         string `json:"phone,omitempty"`
	TaxID         string `json:"tax_id,omitempty"`
	GSTIN         string `json:"gstin,omitempty"`
	TaxType       string `json:"tax_type,omitempty"`
	PlaceOfSupply string `json:"place_of_supply,omitempty"`
	Line1         string `json:"line1,omitempty"`
	City          string `json:"city,omitempty"`
	State         string `json:"state,omitempty"`
	Zip           string `json:"zip,omitempty"`
	Country       string `json:"country,omitempty"`
}

// SubscriptionCreateParams is the body for Subscriptions.Create.
type SubscriptionCreateParams struct {
	CustomerID        string `json:"customer_id"`
	PlanID            string `json:"plan_id"`
	CouponCode        string `json:"coupon_code,omitempty"`
	StartDate         string `json:"start_date,omitempty"`
	BillingAnchorType string `json:"billing_anchor_type,omitempty"`
	PaymentTerms      string `json:"payment_terms,omitempty"`
}

// CouponCreateParams is the body for Coupons.Create.
type CouponCreateParams struct {
	Code          string `json:"code"`
	DiscountType  string `json:"discount_type"`
	DiscountValue int64  `json:"discount_value"`
	Duration      string `json:"duration"`
}

// UsageEventParams is the body for Usage.Record.
type UsageEventParams struct {
	SubscriptionID string `json:"subscription_id"`
	CustomerID     string `json:"customer_id"`
	Dimension      string `json:"dimension"`
	Quantity       int64  `json:"quantity"`
}

// UsageQueryParams are the query parameters for Usage.Query.
type UsageQueryParams struct {
	SubscriptionID string
	CustomerID     string
	Dimension      string
	From           string
	To             string
	Granularity    string
}

// WebhookCreateParams is the body for Webhooks.Create.
type WebhookCreateParams struct {
	URL        string   `json:"url"`
	EventTypes []string `json:"event_types,omitempty"`
}
