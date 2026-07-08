package recurso

import (
	"context"
	"fmt"
	"net/http"
)

// Entitlement is a feature grant on a plan.
type Entitlement struct {
	ID         string `json:"id"`
	PlanID     string `json:"plan_id"`
	FeatureKey string `json:"feature_key"`
	Kind       string `json:"kind"`
	BoolValue  *bool  `json:"bool_value"`
	LimitValue *int64 `json:"limit_value"`
}

// EntitlementInput sets a single feature grant when replacing a plan's set.
type EntitlementInput struct {
	FeatureKey string `json:"feature_key"`
	Kind       string `json:"kind"`
	BoolValue  *bool  `json:"bool_value,omitempty"`
	LimitValue *int64 `json:"limit_value,omitempty"`
}

// CustomerEntitlement is an effective entitlement resolved across a customer's
// active plans, with the granting plan IDs.
type CustomerEntitlement struct {
	FeatureKey string   `json:"feature_key"`
	Kind       string   `json:"kind"`
	BoolValue  *bool    `json:"bool_value"`
	LimitValue *int64   `json:"limit_value"`
	PlanIDs    []string `json:"plan_ids"`
}

// EntitlementCheck is the answer to a single-feature entitlement check.
type EntitlementCheck struct {
	FeatureKey string `json:"feature_key"`
	Granted    bool   `json:"granted"`
	LimitValue *int64 `json:"limit_value"`
}

// EntitlementsService groups the entitlement endpoints.
type EntitlementsService struct{ client *Client }

// GetForPlan lists a plan's entitlements.
func (s *EntitlementsService) GetForPlan(ctx context.Context, planID string) ([]Entitlement, error) {
	return getData[[]Entitlement](ctx, s.client, http.MethodGet, fmt.Sprintf("/plans/%s/entitlements", planID), nil)
}

// SetForPlan replaces a plan's full entitlement set (feature keys absent from
// the list are removed).
func (s *EntitlementsService) SetForPlan(ctx context.Context, planID string, entitlements []EntitlementInput) ([]Entitlement, error) {
	return getData[[]Entitlement](ctx, s.client, http.MethodPut, fmt.Sprintf("/plans/%s/entitlements", planID), entitlements)
}

// ForCustomer returns a customer's effective entitlements.
func (s *EntitlementsService) ForCustomer(ctx context.Context, customerID string) ([]CustomerEntitlement, error) {
	return getData[[]CustomerEntitlement](ctx, s.client, http.MethodGet, fmt.Sprintf("/customers/%s/entitlements", customerID), nil)
}

// Check performs a fast single-feature entitlement check for a customer.
func (s *EntitlementsService) Check(ctx context.Context, customerID, feature string) (*EntitlementCheck, error) {
	path := newQuery().str("customer_id", customerID).str("feature", feature).apply("/entitlements/check")
	var out EntitlementCheck
	if err := s.client.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
