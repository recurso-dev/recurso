package recurso

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// BillingAddress is a customer's billing address.
type BillingAddress struct {
	Line1   string `json:"line1"`
	City    string `json:"city"`
	State   string `json:"state"`
	Zip     string `json:"zip"`
	Country string `json:"country"`
}

// Customer is a billed customer.
type Customer struct {
	ID              string          `json:"id"`
	TenantID        string          `json:"tenant_id"`
	Email           string          `json:"email"`
	Name            string          `json:"name"`
	Phone           string          `json:"phone"`
	TaxID           string          `json:"tax_id"`
	BillingAddress  *BillingAddress `json:"billing_address,omitempty"`
	LedgerAccountID string          `json:"ledger_account_id"`
	GSTIN           string          `json:"gstin"`
	TaxType         string          `json:"tax_type"`
	PlaceOfSupply   string          `json:"place_of_supply"`
	ReferralCode    string          `json:"referral_code"`
	RiskScore       int             `json:"risk_score"`
	CardBrand       string          `json:"card_brand"`
	CardLast4       string          `json:"card_last4"`
	CardExpMonth    int             `json:"card_exp_month"`
	CardExpYear     int             `json:"card_exp_year"`
	CreatedAt       time.Time       `json:"created_at"`
}

// CustomerCreateParams is the body for creating a customer.
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

// CustomerListParams filters the customer list.
type CustomerListParams struct {
	Q       string
	Country string
	Status  string
	Limit   int
	Page    int
}

// PaymentMethodParams updates a customer's stored card.
type PaymentMethodParams struct {
	CardBrand    string `json:"card_brand"`
	CardLast4    string `json:"card_last4"`
	CardExpMonth int    `json:"card_exp_month"`
	CardExpYear  int    `json:"card_exp_year"`
}

// StatusResponse is the {"status": ...} acknowledgement some endpoints return.
type StatusResponse struct {
	Status string `json:"status"`
}

// ChurnFeatures are the inputs to a customer's churn score.
type ChurnFeatures struct {
	DaysSinceSignup    int     `json:"days_since_signup"`
	TotalInvoices      int     `json:"total_invoices"`
	FailedInvoices90d  int     `json:"failed_invoices_90d"`
	PaymentFailureRate float64 `json:"payment_failure_rate"`
	AvgDaysToPay       float64 `json:"avg_days_to_pay"`
	PlanDowngrades     int     `json:"plan_downgrades"`
	MonthsActive       int     `json:"months_active"`
	CurrentMRR         int64   `json:"current_mrr"`
	UsageTrend         float64 `json:"usage_trend"`
}

// ChurnScore is a customer's churn-risk score.
type ChurnScore struct {
	CustomerID   string        `json:"customer_id"`
	Score        int           `json:"score"`
	RiskLevel    string        `json:"risk_level"`
	Features     ChurnFeatures `json:"features"`
	ModelVersion string        `json:"model_version"`
}

// Consent is a recorded customer consent (RBI compliance).
type Consent struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	CustomerID     string    `json:"customer_id"`
	SubscriptionID string    `json:"subscription_id"`
	ConsentType    string    `json:"consent_type"`
	Granted        bool      `json:"granted"`
	GrantedAt      time.Time `json:"granted_at"`
	RevokedAt      time.Time `json:"revoked_at"`
	IPAddress      string    `json:"ip_address"`
	UserAgent      string    `json:"user_agent"`
	ConsentText    string    `json:"consent_text"`
	Version        string    `json:"version"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// CustomersService groups the customer endpoints.
type CustomersService struct{ client *Client }

// Create creates a customer.
func (s *CustomersService) Create(ctx context.Context, params *CustomerCreateParams) (*Customer, error) {
	var out Customer
	if err := s.client.do(ctx, http.MethodPost, "/customers", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// List returns the tenant's customers.
func (s *CustomersService) List(ctx context.Context, params *CustomerListParams) ([]Customer, error) {
	path := "/customers"
	if params != nil {
		path = newQuery().
			str("q", params.Q).
			str("country", params.Country).
			str("status", params.Status).
			int("limit", params.Limit).
			int("page", params.Page).
			apply(path)
	}
	return getData[[]Customer](ctx, s.client, http.MethodGet, path, nil)
}

// UpdatePaymentMethod replaces the customer's card on file.
func (s *CustomersService) UpdatePaymentMethod(ctx context.Context, id string, params *PaymentMethodParams) (*StatusResponse, error) {
	var out StatusResponse
	if err := s.client.do(ctx, http.MethodPut, fmt.Sprintf("/customers/%s/payment-method", id), params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Churn returns the customer's churn-risk score and features.
func (s *CustomersService) Churn(ctx context.Context, id string) (*ChurnScore, error) {
	out, err := getData[ChurnScore](ctx, s.client, http.MethodGet, fmt.Sprintf("/customers/%s/churn", id), nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Consents lists the customer's consent records.
func (s *CustomersService) Consents(ctx context.Context, id string) ([]Consent, error) {
	return getData[[]Consent](ctx, s.client, http.MethodGet, fmt.Sprintf("/customers/%s/consents", id), nil)
}
