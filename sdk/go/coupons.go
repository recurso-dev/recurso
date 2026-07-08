package recurso

import (
	"context"
	"net/http"
	"time"
)

// Coupon is a discount code.
type Coupon struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	Code           string    `json:"code"`
	DiscountType   string    `json:"discount_type"`
	DiscountValue  int64     `json:"discount_value"`
	Duration       string    `json:"duration"`
	DurationMonths *int      `json:"duration_months"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// CouponCreateParams is the body for creating a coupon. DiscountValue is a
// percentage (0-100) when DiscountType is "percent", otherwise an amount in the
// currency's smallest unit.
type CouponCreateParams struct {
	Code           string `json:"code"`
	DiscountType   string `json:"discount_type"`
	DiscountValue  int64  `json:"discount_value"`
	Duration       string `json:"duration"`
	DurationMonths int    `json:"duration_months,omitempty"`
}

// CouponsService groups the coupon endpoints.
type CouponsService struct{ client *Client }

// Create creates a coupon.
func (s *CouponsService) Create(ctx context.Context, params *CouponCreateParams) (*Coupon, error) {
	var out Coupon
	if err := s.client.do(ctx, http.MethodPost, "/coupons", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// List returns the tenant's coupons.
func (s *CouponsService) List(ctx context.Context) ([]Coupon, error) {
	return getData[[]Coupon](ctx, s.client, http.MethodGet, "/coupons", nil)
}
