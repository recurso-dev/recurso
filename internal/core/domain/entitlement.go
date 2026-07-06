package domain

import (
	"time"

	"github.com/google/uuid"
)

// EntitlementKind is the type of a plan-level feature grant.
type EntitlementKind string

const (
	// EntitlementKindBoolean is an on/off feature flag (bool_value).
	EntitlementKindBoolean EntitlementKind = "boolean"
	// EntitlementKindLimit is a numeric cap (limit_value), e.g. seats or API calls.
	EntitlementKindLimit EntitlementKind = "limit"
)

// Entitlement is a plan-level feature grant. Every customer holding an
// ACTIVE or TRIALING subscription to the plan receives the grant.
type Entitlement struct {
	ID         uuid.UUID       `json:"id"`
	TenantID   uuid.UUID       `json:"tenant_id"`
	PlanID     uuid.UUID       `json:"plan_id"`
	FeatureKey string          `json:"feature_key"`
	Kind       EntitlementKind `json:"kind"`
	BoolValue  *bool           `json:"bool_value,omitempty"`
	LimitValue *int64          `json:"limit_value,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

// EffectiveEntitlement is one entry of a customer's resolved entitlement
// set: the union over the plans of their ACTIVE and TRIALING subscriptions.
//
// Resolution semantics (documented choice):
//   - boolean: Value is true if ANY contributing plan grants true.
//   - limit:   Value is the MAX limit_value across contributing plans —
//     the most generous plan wins (union-of-grants, never an
//     intersection/min, so an upgrade can only add capability).
//   - mixed kinds for the same feature_key across plans resolve to
//     'limit' (a cap is more specific than a flag); the max cap is kept.
//
// PlanIDs lists every plan that defines the feature key, sorted for
// deterministic output.
type EffectiveEntitlement struct {
	FeatureKey string          `json:"feature_key"`
	Kind       EntitlementKind `json:"kind"`
	Value      any             `json:"value"`
	PlanIDs    []uuid.UUID     `json:"plan_ids"`
}

// EntitlementCheck is the fast-path answer for a single feature check.
type EntitlementCheck struct {
	FeatureKey string `json:"feature_key"`
	Granted    bool   `json:"granted"`
	LimitValue *int64 `json:"limit_value"`
}
