package recurso

import (
	"context"
	"net/url"
	"strconv"
)

// listQuery converts ListParams into URL query values.
func (p *ListParams) values() url.Values {
	v := url.Values{}
	if p == nil {
		return v
	}
	if p.Page > 0 {
		v.Set("page", strconv.Itoa(p.Page))
	}
	if p.Limit > 0 {
		v.Set("limit", strconv.Itoa(p.Limit))
	}
	if p.Q != "" {
		v.Set("q", p.Q)
	}
	if p.Status != "" {
		v.Set("status", p.Status)
	}
	return v
}

// --- Account ---

type AccountService struct{ client *Client }

// Get returns the authenticated tenant's account.
func (s *AccountService) Get(ctx context.Context) (*Tenant, error) {
	return doResource[Tenant](ctx, s.client, "GET", "/account", nil, nil)
}

// Update updates the account.
func (s *AccountService) Update(ctx context.Context, body Params) (*Tenant, error) {
	return doResource[Tenant](ctx, s.client, "PUT", "/account", nil, body)
}

// Tenant is the authenticated account.
type Tenant struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Email     string `json:"email,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

// --- Customers ---

type CustomersService struct{ client *Client }

func (s *CustomersService) Create(ctx context.Context, p *CustomerCreateParams) (*Customer, error) {
	if p.Country == "" {
		p.Country = "US"
	}
	return doResource[Customer](ctx, s.client, "POST", "/customers", nil, p)
}

func (s *CustomersService) List(ctx context.Context, p *ListParams) ([]Customer, error) {
	return doList[Customer](ctx, s.client, "GET", "/customers", p.values(), nil)
}

func (s *CustomersService) UpdatePaymentMethod(ctx context.Context, id string, body Params) (*Customer, error) {
	return doResource[Customer](ctx, s.client, "PUT", "/customers/"+id+"/payment-method", nil, body)
}

// Churn returns the customer's churn score.
func (s *CustomersService) Churn(ctx context.Context, id string) (*ChurnScoreResult, error) {
	return doResource[ChurnScoreResult](ctx, s.client, "GET", "/customers/"+id+"/churn", nil, nil)
}

// Consents lists the customer's recorded consents.
func (s *CustomersService) Consents(ctx context.Context, id string) ([]Consent, error) {
	return doList[Consent](ctx, s.client, "GET", "/customers/"+id+"/consents", nil, nil)
}

// ChurnScoreResult is a customer's churn prediction.
type ChurnScoreResult struct {
	CustomerID string  `json:"customer_id,omitempty"`
	Score      float64 `json:"score,omitempty"`
	RiskLevel  string  `json:"risk_level,omitempty"`
}

// Consent is a recorded customer consent.
type Consent struct {
	ID          string `json:"id,omitempty"`
	CustomerID  string `json:"customer_id,omitempty"`
	ConsentType string `json:"consent_type,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
}

// --- Plans ---

type PlansService struct{ client *Client }

func (s *PlansService) Create(ctx context.Context, p *PlanCreateParams) (*Plan, error) {
	if p.IntervalCount == 0 {
		p.IntervalCount = 1
	}
	return doResource[Plan](ctx, s.client, "POST", "/plans", nil, p)
}

func (s *PlansService) List(ctx context.Context, p *ListParams) ([]Plan, error) {
	return doList[Plan](ctx, s.client, "GET", "/plans", p.values(), nil)
}

// --- Subscriptions ---

type SubscriptionsService struct{ client *Client }

func (s *SubscriptionsService) Create(ctx context.Context, p *SubscriptionCreateParams) (*Subscription, error) {
	return doResource[Subscription](ctx, s.client, "POST", "/subscriptions", nil, p)
}

func (s *SubscriptionsService) List(ctx context.Context, p *ListParams) ([]Subscription, error) {
	return doList[Subscription](ctx, s.client, "GET", "/subscriptions", p.values(), nil)
}

func (s *SubscriptionsService) Update(ctx context.Context, id string, body Params) (*Subscription, error) {
	return doResource[Subscription](ctx, s.client, "PUT", "/subscriptions/"+id, nil, body)
}

func (s *SubscriptionsService) Cancel(ctx context.Context, id string, body Params) (*Subscription, error) {
	return doResource[Subscription](ctx, s.client, "POST", "/subscriptions/"+id+"/cancel", nil, body)
}

func (s *SubscriptionsService) Pause(ctx context.Context, id string, body Params) (*Subscription, error) {
	return doResource[Subscription](ctx, s.client, "POST", "/subscriptions/"+id+"/pause", nil, body)
}

func (s *SubscriptionsService) Resume(ctx context.Context, id string) (*Subscription, error) {
	return doResource[Subscription](ctx, s.client, "POST", "/subscriptions/"+id+"/resume", nil, nil)
}

func (s *SubscriptionsService) Reactivate(ctx context.Context, id string) (*Subscription, error) {
	return doResource[Subscription](ctx, s.client, "POST", "/subscriptions/"+id+"/reactivate", nil, nil)
}

// Advance bills N future periods immediately (advance invoicing).
func (s *SubscriptionsService) Advance(ctx context.Context, id string, body Params) (*Invoice, error) {
	return doResource[Invoice](ctx, s.client, "POST", "/subscriptions/"+id+"/advance", nil, body)
}

// Charges lists the subscription's unbilled charges.
func (s *SubscriptionsService) Charges(ctx context.Context, id string) ([]UnbilledCharge, error) {
	return doList[UnbilledCharge](ctx, s.client, "GET", "/subscriptions/"+id+"/charges", nil, nil)
}

func (s *SubscriptionsService) AddCharge(ctx context.Context, id string, body Params) (*UnbilledCharge, error) {
	return doResource[UnbilledCharge](ctx, s.client, "POST", "/subscriptions/"+id+"/charges", nil, body)
}

// Usage returns current-period usage per dimension with entitlement limits.
func (s *SubscriptionsService) Usage(ctx context.Context, id string) (*SubscriptionUsage, error) {
	return doResource[SubscriptionUsage](ctx, s.client, "GET", "/subscriptions/"+id+"/usage", nil, nil)
}

// UnbilledCharge is a one-off charge awaiting the next invoice.
type UnbilledCharge struct {
	ID             string `json:"id,omitempty"`
	SubscriptionID string `json:"subscription_id,omitempty"`
	Amount         int64  `json:"amount,omitempty"`
	Description    string `json:"description,omitempty"`
	Status         string `json:"status,omitempty"`
	CreatedAt      string `json:"created_at,omitempty"`
}

// SubscriptionUsage is a subscription's usage summary.
type SubscriptionUsage struct {
	SubscriptionID string `json:"subscription_id,omitempty"`
	Dimensions     []any  `json:"dimensions,omitempty"`
}

// --- Invoices ---

type InvoicesService struct{ client *Client }

func (s *InvoicesService) List(ctx context.Context, p *ListParams) ([]Invoice, error) {
	return doList[Invoice](ctx, s.client, "GET", "/invoices", p.values(), nil)
}

// PDFURL returns the public PDF download URL for an invoice.
func (s *InvoicesService) PDFURL(id string) string {
	return s.client.baseURL + "/invoices/" + id + "/pdf"
}

func (s *InvoicesService) EInvoiceStatus(ctx context.Context, id string) (*Invoice, error) {
	return doResource[Invoice](ctx, s.client, "GET", "/invoices/"+id+"/einvoice", nil, nil)
}

func (s *InvoicesService) RetryEInvoice(ctx context.Context, id string) (*Invoice, error) {
	return doResource[Invoice](ctx, s.client, "POST", "/invoices/"+id+"/einvoice/retry", nil, nil)
}

func (s *InvoicesService) CancelEInvoice(ctx context.Context, id string, body Params) (*Invoice, error) {
	return doResource[Invoice](ctx, s.client, "POST", "/invoices/"+id+"/einvoice/cancel", nil, body)
}

// --- Coupons ---

type CouponsService struct{ client *Client }

func (s *CouponsService) Create(ctx context.Context, p *CouponCreateParams) (*Coupon, error) {
	return doResource[Coupon](ctx, s.client, "POST", "/coupons", nil, p)
}

func (s *CouponsService) List(ctx context.Context, p *ListParams) ([]Coupon, error) {
	return doList[Coupon](ctx, s.client, "GET", "/coupons", p.values(), nil)
}

// --- Usage platform ---

type UsageService struct{ client *Client }

// Record records a metered usage event against a subscription.
func (s *UsageService) Record(ctx context.Context, p *UsageEventParams) (*UnbilledCharge, error) {
	return doResource[UnbilledCharge](ctx, s.client, "POST", "/usage/events", nil, p)
}

// Query returns time-windowed usage buckets.
func (s *UsageService) Query(ctx context.Context, p *UsageQueryParams) (map[string]any, error) {
	v := url.Values{}
	if p != nil {
		for k, val := range map[string]string{
			"subscription_id": p.SubscriptionID, "customer_id": p.CustomerID,
			"dimension": p.Dimension, "from": p.From, "to": p.To, "granularity": p.Granularity,
		} {
			if val != "" {
				v.Set(k, val)
			}
		}
	}
	raw, err := s.client.do(ctx, "GET", "/usage", v, nil)
	if err != nil {
		return nil, err
	}
	return decodeMap(raw)
}

// Dimensions returns the tenant's usage dimension catalog.
func (s *UsageService) Dimensions(ctx context.Context) ([]any, error) {
	return doAnyList(ctx, s.client, "GET", "/usage/dimensions")
}

// --- Credit notes ---

type CreditNotesService struct{ client *Client }

func (s *CreditNotesService) Create(ctx context.Context, body Params) (*CreditNote, error) {
	return doResource[CreditNote](ctx, s.client, "POST", "/credit-notes", nil, body)
}

func (s *CreditNotesService) List(ctx context.Context, p *ListParams) ([]CreditNote, error) {
	return doList[CreditNote](ctx, s.client, "GET", "/credit-notes", p.values(), nil)
}

// --- Quotes ---

type QuotesService struct{ client *Client }

func (s *QuotesService) Create(ctx context.Context, body Params) (*Quote, error) {
	return doResource[Quote](ctx, s.client, "POST", "/quotes", nil, body)
}

func (s *QuotesService) List(ctx context.Context, p *ListParams) ([]Quote, error) {
	return doList[Quote](ctx, s.client, "GET", "/quotes", p.values(), nil)
}

func (s *QuotesService) Get(ctx context.Context, id string) (*Quote, error) {
	return doResource[Quote](ctx, s.client, "GET", "/quotes/"+id, nil, nil)
}

func (s *QuotesService) Update(ctx context.Context, id string, body Params) (*Quote, error) {
	return doResource[Quote](ctx, s.client, "PUT", "/quotes/"+id, nil, body)
}

func (s *QuotesService) Send(ctx context.Context, id string) (*Quote, error) {
	return doResource[Quote](ctx, s.client, "POST", "/quotes/"+id+"/send", nil, nil)
}

func (s *QuotesService) Accept(ctx context.Context, id string) (*Quote, error) {
	return doResource[Quote](ctx, s.client, "POST", "/quotes/"+id+"/accept", nil, nil)
}

func (s *QuotesService) Decline(ctx context.Context, id string) (*Quote, error) {
	return doResource[Quote](ctx, s.client, "POST", "/quotes/"+id+"/decline", nil, nil)
}

// Convert converts an accepted quote into a subscription.
func (s *QuotesService) Convert(ctx context.Context, id string) (*Quote, error) {
	return doResource[Quote](ctx, s.client, "POST", "/quotes/"+id+"/convert", nil, nil)
}

func (s *QuotesService) Delete(ctx context.Context, id string) error {
	_, err := s.client.do(ctx, "DELETE", "/quotes/"+id, nil, nil)
	return err
}

// --- Webhooks ---

type WebhooksService struct{ client *Client }

func (s *WebhooksService) Create(ctx context.Context, p *WebhookCreateParams) (*WebhookEndpoint, error) {
	return doResource[WebhookEndpoint](ctx, s.client, "POST", "/webhooks", nil, p)
}

func (s *WebhooksService) List(ctx context.Context) ([]WebhookEndpoint, error) {
	return doList[WebhookEndpoint](ctx, s.client, "GET", "/webhooks", nil, nil)
}

func (s *WebhooksService) Delete(ctx context.Context, id string) error {
	_, err := s.client.do(ctx, "DELETE", "/webhooks/"+id, nil, nil)
	return err
}

// Deliveries lists recent delivery attempts to an endpoint.
func (s *WebhooksService) Deliveries(ctx context.Context, id string) ([]EventDelivery, error) {
	return doList[EventDelivery](ctx, s.client, "GET", "/webhooks/"+id+"/deliveries", nil, nil)
}

// --- Events ---

type EventsService struct{ client *Client }

func (s *EventsService) List(ctx context.Context, p *ListParams) ([]Event, error) {
	return doList[Event](ctx, s.client, "GET", "/events", p.values(), nil)
}

func (s *EventsService) Types(ctx context.Context) ([]any, error) {
	return doAnyList(ctx, s.client, "GET", "/events/types")
}

func (s *EventsService) Deliveries(ctx context.Context, id string) ([]EventDelivery, error) {
	return doList[EventDelivery](ctx, s.client, "GET", "/events/"+id+"/deliveries", nil, nil)
}

// Redeliver re-enqueues delivery of an event to all subscribed endpoints.
func (s *EventsService) Redeliver(ctx context.Context, id string) (map[string]any, error) {
	raw, err := s.client.do(ctx, "POST", "/events/"+id+"/redeliver", nil, nil)
	if err != nil {
		return nil, err
	}
	return decodeMap(raw)
}

// --- Mandates ---

type MandatesService struct{ client *Client }

func (s *MandatesService) Create(ctx context.Context, body Params) (*Mandate, error) {
	return doResource[Mandate](ctx, s.client, "POST", "/mandates", nil, body)
}

func (s *MandatesService) List(ctx context.Context, p *ListParams) ([]Mandate, error) {
	return doList[Mandate](ctx, s.client, "GET", "/mandates", p.values(), nil)
}

func (s *MandatesService) Get(ctx context.Context, id string) (*Mandate, error) {
	return doResource[Mandate](ctx, s.client, "GET", "/mandates/"+id, nil, nil)
}

func (s *MandatesService) Revoke(ctx context.Context, id string) (*Mandate, error) {
	return doResource[Mandate](ctx, s.client, "POST", "/mandates/"+id+"/revoke", nil, nil)
}

// --- Gifts ---

type GiftsService struct{ client *Client }

func (s *GiftsService) Purchase(ctx context.Context, body Params) (*Gift, error) {
	return doResource[Gift](ctx, s.client, "POST", "/gifts/purchase", nil, body)
}

func (s *GiftsService) Redeem(ctx context.Context, body Params) (*Gift, error) {
	return doResource[Gift](ctx, s.client, "POST", "/gifts/redeem", nil, body)
}

func (s *GiftsService) List(ctx context.Context, p *ListParams) ([]Gift, error) {
	return doList[Gift](ctx, s.client, "GET", "/gifts", p.values(), nil)
}

// --- Referrals ---

type ReferralsService struct{ client *Client }

func (s *ReferralsService) Create(ctx context.Context, body Params) (*Referral, error) {
	return doResource[Referral](ctx, s.client, "POST", "/referrals", nil, body)
}

func (s *ReferralsService) List(ctx context.Context, p *ListParams) ([]Referral, error) {
	return doList[Referral](ctx, s.client, "GET", "/referrals", p.values(), nil)
}

func (s *ReferralsService) GenerateCode(ctx context.Context, body Params) (map[string]any, error) {
	raw, err := s.client.do(ctx, "POST", "/referrals/generate-code", nil, body)
	if err != nil {
		return nil, err
	}
	return decodeMap(raw)
}

func (s *ReferralsService) Qualify(ctx context.Context, id string) (*Referral, error) {
	return doResource[Referral](ctx, s.client, "POST", "/referrals/"+id+"/qualify", nil, nil)
}

// --- Entitlements ---

type EntitlementsService struct{ client *Client }

// SetForPlan replaces a plan's full entitlement set (PUT semantics).
func (s *EntitlementsService) SetForPlan(ctx context.Context, planID string, list []Entitlement) ([]Entitlement, error) {
	return doList[Entitlement](ctx, s.client, "PUT", "/plans/"+planID+"/entitlements", nil, list)
}

func (s *EntitlementsService) GetForPlan(ctx context.Context, planID string) ([]Entitlement, error) {
	return doList[Entitlement](ctx, s.client, "GET", "/plans/"+planID+"/entitlements", nil, nil)
}

// ForCustomer returns a customer's effective entitlements.
func (s *EntitlementsService) ForCustomer(ctx context.Context, customerID string) ([]Entitlement, error) {
	return doList[Entitlement](ctx, s.client, "GET", "/customers/"+customerID+"/entitlements", nil, nil)
}

// Check is a fast single-feature entitlement check.
func (s *EntitlementsService) Check(ctx context.Context, customerID, feature string) (map[string]any, error) {
	v := url.Values{"customer_id": {customerID}, "feature": {feature}}
	raw, err := s.client.do(ctx, "GET", "/entitlements/check", v, nil)
	if err != nil {
		return nil, err
	}
	return decodeMap(raw)
}

// --- Analytics ---

type AnalyticsService struct{ client *Client }

// MRR returns FX-normalized monthly recurring revenue with provenance.
func (s *AnalyticsService) MRR(ctx context.Context) (*MRRMetrics, error) {
	return doResource[MRRMetrics](ctx, s.client, "GET", "/analytics/mrr", nil, nil)
}

// --- Ledger ---

type LedgerService struct{ client *Client }

func (s *LedgerService) Accounts(ctx context.Context) ([]LedgerAccount, error) {
	return doList[LedgerAccount](ctx, s.client, "GET", "/ledger/accounts", nil, nil)
}

// LedgerEntriesParams filters ledger entries.
type LedgerEntriesParams struct{ AccountID string }

func (s *LedgerService) Entries(ctx context.Context, p *LedgerEntriesParams) ([]LedgerTransaction, error) {
	v := url.Values{}
	if p != nil && p.AccountID != "" {
		v.Set("account_id", p.AccountID)
	}
	return doList[LedgerTransaction](ctx, s.client, "GET", "/ledger/entries", v, nil)
}
