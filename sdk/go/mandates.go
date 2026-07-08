package recurso

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Mandate is a UPI Autopay mandate.
type Mandate struct {
	ID                     string     `json:"id"`
	TenantID               string     `json:"tenant_id"`
	CustomerID             string     `json:"customer_id"`
	SubscriptionID         string     `json:"subscription_id"`
	MandateType            string     `json:"mandate_type"`
	PaymentMethod          string     `json:"payment_method"`
	VPA                    string     `json:"vpa"`
	RazorpayTokenID        string     `json:"razorpay_token_id"`
	RazorpaySubscriptionID string     `json:"razorpay_subscription_id"`
	RazorpayCustomerID     string     `json:"razorpay_customer_id"`
	MaxAmount              int64      `json:"max_amount"`
	Frequency              string     `json:"frequency"`
	Status                 string     `json:"status"`
	AuthorizedAt           *time.Time `json:"authorized_at"`
	ActivatedAt            *time.Time `json:"activated_at"`
	RevokedAt              *time.Time `json:"revoked_at"`
	LastDebitAt            *time.Time `json:"last_debit_at"`
	NextDebitAt            *time.Time `json:"next_debit_at"`
	PreDebitNotified       bool       `json:"pre_debit_notified"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

// MandateCreateParams registers a UPI Autopay mandate. MaxAmount is in the
// currency's smallest unit.
type MandateCreateParams struct {
	CustomerID     string `json:"customer_id"`
	SubscriptionID string `json:"subscription_id,omitempty"`
	VPA            string `json:"vpa"`
	MaxAmount      int64  `json:"max_amount"`
	Frequency      string `json:"frequency"`
}

// MandateCreateResult is returned by Create: the mandate plus the customer
// authorization URL.
type MandateCreateResult struct {
	Mandate Mandate `json:"mandate"`
	AuthURL string  `json:"auth_url"`
}

// MandatesService groups the UPI Autopay mandate endpoints.
type MandatesService struct{ client *Client }

// Create registers a mandate and returns an authorization URL for the customer.
func (s *MandatesService) Create(ctx context.Context, params *MandateCreateParams) (*MandateCreateResult, error) {
	var out MandateCreateResult
	if err := s.client.do(ctx, http.MethodPost, "/mandates", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// List returns the tenant's mandates.
func (s *MandatesService) List(ctx context.Context) ([]Mandate, error) {
	return getData[[]Mandate](ctx, s.client, http.MethodGet, "/mandates", nil)
}

// Get retrieves a mandate by ID.
func (s *MandatesService) Get(ctx context.Context, id string) (*Mandate, error) {
	out, err := getData[Mandate](ctx, s.client, http.MethodGet, "/mandates/"+id, nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Revoke revokes a mandate.
func (s *MandatesService) Revoke(ctx context.Context, id string) (*StatusResponse, error) {
	var out StatusResponse
	if err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/mandates/%s/revoke", id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
