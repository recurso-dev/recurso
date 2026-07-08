package recurso

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Referral links a referrer to a referred customer.
type Referral struct {
	ID           string     `json:"id"`
	TenantID     string     `json:"tenant_id"`
	ReferrerID   string     `json:"referrer_id"`
	ReferredID   string     `json:"referred_id"`
	Code         string     `json:"code"`
	Status       string     `json:"status"`
	RewardAmount int64      `json:"reward_amount"`
	Currency     string     `json:"currency"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	QualifiedAt  *time.Time `json:"qualified_at"`
}

// ReferralCreateParams creates a referral. RewardAmount is in the smallest unit
// (defaults to 500 server-side when zero).
type ReferralCreateParams struct {
	ReferrerID   string `json:"referrer_id"`
	ReferredID   string `json:"referred_id"`
	RewardAmount int64  `json:"reward_amount,omitempty"`
	Currency     string `json:"currency,omitempty"`
}

// ReferralListParams paginates the referral list.
type ReferralListParams struct {
	Page    int
	PerPage int
}

// ReferralCode is a customer's referral code.
type ReferralCode struct {
	Code string `json:"code"`
}

// ReferralsService groups the referral-program endpoints.
type ReferralsService struct{ client *Client }

// Create links a referrer to a referred customer.
func (s *ReferralsService) Create(ctx context.Context, params *ReferralCreateParams) (*Referral, error) {
	out, err := getData[Referral](ctx, s.client, http.MethodPost, "/referrals", params)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// List returns the tenant's referrals.
func (s *ReferralsService) List(ctx context.Context, params *ReferralListParams) ([]Referral, error) {
	path := "/referrals"
	if params != nil {
		path = newQuery().int("page", params.Page).int("per_page", params.PerPage).apply(path)
	}
	return getData[[]Referral](ctx, s.client, http.MethodGet, path, nil)
}

// GenerateCode generates (or fetches) a customer's referral code.
func (s *ReferralsService) GenerateCode(ctx context.Context, customerID string) (*ReferralCode, error) {
	body := map[string]string{"customer_id": customerID}
	out, err := getData[ReferralCode](ctx, s.client, http.MethodPost, "/referrals/generate-code", body)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Qualify marks a referral as qualified and issues the reward credit.
func (s *ReferralsService) Qualify(ctx context.Context, id string) (*Referral, error) {
	out, err := getData[Referral](ctx, s.client, http.MethodPost, fmt.Sprintf("/referrals/%s/qualify", id), nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
