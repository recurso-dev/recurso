package service

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

// Entitlement Engine v1.
//
// Plans carry feature grants (boolean flags or numeric limits); a
// customer's effective entitlements are the UNION over the plans of their
// ACTIVE and TRIALING subscriptions:
//
//   - boolean: granted if ANY plan grants true (any-true wins).
//   - limit:   MAX limit_value across plans — the most generous plan wins.
//     Union-of-grants (never min/intersection) so adding a plan
//     can only ever add capability, matching customer intuition
//     when they hold multiple subscriptions.
//   - mixed kinds for one feature_key across plans resolve to 'limit'
//     with the max cap (a cap is the more specific grant).
//   - no active/trialing subscriptions -> empty set.

// EntitlementError is a sentinel error category for entitlement flows.
type EntitlementError string

func (e EntitlementError) Error() string { return string(e) }

var (
	ErrEntitlementPlanNotFound     = EntitlementError("plan not found")
	ErrEntitlementCustomerNotFound = EntitlementError("customer not found")
)

// EntitlementValidationError marks invalid caller input (maps to HTTP 400).
type EntitlementValidationError string

func (e EntitlementValidationError) Error() string { return string(e) }

// featureKeyRe constrains feature keys to a safe machine-identifier shape.
var featureKeyRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]*$`)

const maxFeatureKeyLen = 128

type EntitlementService struct {
	entitlements  port.EntitlementRepository
	plans         port.PlanRepository
	customers     port.CustomerRepository
	subscriptions port.SubscriptionRepository
}

func NewEntitlementService(
	entitlements port.EntitlementRepository,
	plans port.PlanRepository,
	customers port.CustomerRepository,
	subscriptions port.SubscriptionRepository,
) *EntitlementService {
	return &EntitlementService{
		entitlements:  entitlements,
		plans:         plans,
		customers:     customers,
		subscriptions: subscriptions,
	}
}

// EntitlementInput is one entry of a plan's desired entitlement set.
type EntitlementInput struct {
	FeatureKey string `json:"feature_key"`
	Kind       string `json:"kind"`
	BoolValue  *bool  `json:"bool_value,omitempty"`
	LimitValue *int64 `json:"limit_value,omitempty"`
}

// SetPlanEntitlements replaces the plan's full entitlement set (PUT
// semantics: entries absent from inputs are removed).
func (s *EntitlementService) SetPlanEntitlements(ctx context.Context, tenantID, planID uuid.UUID, inputs []EntitlementInput) ([]domain.Entitlement, error) {
	if err := s.requirePlan(ctx, tenantID, planID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	seen := make(map[string]bool, len(inputs))
	ents := make([]domain.Entitlement, 0, len(inputs))

	for _, in := range inputs {
		if err := validateEntitlementInput(in); err != nil {
			return nil, err
		}
		if seen[in.FeatureKey] {
			return nil, EntitlementValidationError(fmt.Sprintf("duplicate feature_key %q", in.FeatureKey))
		}
		seen[in.FeatureKey] = true

		ents = append(ents, domain.Entitlement{
			ID:         uuid.New(),
			TenantID:   tenantID,
			PlanID:     planID,
			FeatureKey: in.FeatureKey,
			Kind:       domain.EntitlementKind(in.Kind),
			BoolValue:  in.BoolValue,
			LimitValue: in.LimitValue,
			CreatedAt:  now,
			UpdatedAt:  now,
		})
	}

	if err := s.entitlements.ReplaceForPlan(ctx, tenantID, planID, ents); err != nil {
		return nil, err
	}
	return ents, nil
}

// GetPlanEntitlements lists a plan's entitlement rows.
func (s *EntitlementService) GetPlanEntitlements(ctx context.Context, tenantID, planID uuid.UUID) ([]domain.Entitlement, error) {
	if err := s.requirePlan(ctx, tenantID, planID); err != nil {
		return nil, err
	}
	return s.entitlements.ListByPlan(ctx, tenantID, planID)
}

// GetCustomerEntitlements resolves the customer's effective entitlement
// set (see the union semantics documented at the top of this file).
func (s *EntitlementService) GetCustomerEntitlements(ctx context.Context, tenantID, customerID uuid.UUID) ([]domain.EffectiveEntitlement, error) {
	customer, err := s.customers.GetByID(ctx, customerID)
	if err != nil {
		return nil, err
	}
	if customer == nil || customer.TenantID != tenantID {
		return nil, ErrEntitlementCustomerNotFound
	}

	subs, err := s.subscriptions.List(ctx, tenantID, domain.SubscriptionFilter{CustomerID: customerID})
	if err != nil {
		return nil, err
	}

	planIDSet := make(map[uuid.UUID]bool)
	var planIDs []uuid.UUID
	for _, sub := range subs {
		if sub.Status != domain.SubscriptionStatusActive && sub.Status != domain.SubscriptionStatusTrialing {
			continue
		}
		if !planIDSet[sub.PlanID] {
			planIDSet[sub.PlanID] = true
			planIDs = append(planIDs, sub.PlanID)
		}
	}
	if len(planIDs) == 0 {
		return []domain.EffectiveEntitlement{}, nil
	}

	ents, err := s.entitlements.ListByPlanIDs(ctx, tenantID, planIDs)
	if err != nil {
		return nil, err
	}
	return resolveEffective(ents), nil
}

// CheckFeature is the hot path: one indexed query, no N+1. A missing
// customer, no subscriptions, or an ungranted feature all resolve to
// granted=false with a nil limit.
func (s *EntitlementService) CheckFeature(ctx context.Context, tenantID, customerID uuid.UUID, featureKey string) (*domain.EntitlementCheck, error) {
	if customerID == uuid.Nil {
		return nil, EntitlementValidationError("customer_id is required")
	}
	if featureKey == "" {
		return nil, EntitlementValidationError("feature is required")
	}
	return s.entitlements.CheckFeature(ctx, tenantID, customerID, featureKey)
}

// requirePlan enforces tenant isolation: the plan must exist and belong to
// the caller's tenant (the repo already scopes by the ctx tenant; the
// explicit TenantID comparison is defense in depth).
func (s *EntitlementService) requirePlan(ctx context.Context, tenantID, planID uuid.UUID) error {
	plan, err := s.plans.GetByID(ctx, planID)
	if err != nil {
		return err
	}
	if plan == nil || plan.TenantID != tenantID {
		return ErrEntitlementPlanNotFound
	}
	return nil
}

func validateEntitlementInput(in EntitlementInput) error {
	if in.FeatureKey == "" {
		return EntitlementValidationError("feature_key is required")
	}
	if len(in.FeatureKey) > maxFeatureKeyLen {
		return EntitlementValidationError(fmt.Sprintf("feature_key %q exceeds %d characters", in.FeatureKey, maxFeatureKeyLen))
	}
	if !featureKeyRe.MatchString(in.FeatureKey) {
		return EntitlementValidationError(fmt.Sprintf("feature_key %q must match %s", in.FeatureKey, featureKeyRe.String()))
	}

	switch domain.EntitlementKind(in.Kind) {
	case domain.EntitlementKindBoolean:
		if in.BoolValue == nil {
			return EntitlementValidationError(fmt.Sprintf("feature_key %q: bool_value is required for kind 'boolean'", in.FeatureKey))
		}
		if in.LimitValue != nil {
			return EntitlementValidationError(fmt.Sprintf("feature_key %q: limit_value must be omitted for kind 'boolean'", in.FeatureKey))
		}
	case domain.EntitlementKindLimit:
		if in.LimitValue == nil {
			return EntitlementValidationError(fmt.Sprintf("feature_key %q: limit_value is required for kind 'limit'", in.FeatureKey))
		}
		if *in.LimitValue < 0 {
			return EntitlementValidationError(fmt.Sprintf("feature_key %q: limit_value must be >= 0", in.FeatureKey))
		}
		if in.BoolValue != nil {
			return EntitlementValidationError(fmt.Sprintf("feature_key %q: bool_value must be omitted for kind 'limit'", in.FeatureKey))
		}
	default:
		return EntitlementValidationError(fmt.Sprintf("feature_key %q: kind must be 'boolean' or 'limit'", in.FeatureKey))
	}
	return nil
}

// resolveEffective folds plan entitlement rows into the effective set.
func resolveEffective(ents []domain.Entitlement) []domain.EffectiveEntitlement {
	type acc struct {
		anyTrue  bool
		hasLimit bool
		maxLimit int64
		planIDs  map[uuid.UUID]bool
	}

	byKey := make(map[string]*acc)
	for _, e := range ents {
		a := byKey[e.FeatureKey]
		if a == nil {
			a = &acc{planIDs: make(map[uuid.UUID]bool)}
			byKey[e.FeatureKey] = a
		}
		a.planIDs[e.PlanID] = true

		switch e.Kind {
		case domain.EntitlementKindBoolean:
			if e.BoolValue != nil && *e.BoolValue {
				a.anyTrue = true
			}
		case domain.EntitlementKindLimit:
			if e.LimitValue != nil {
				if !a.hasLimit || *e.LimitValue > a.maxLimit {
					a.maxLimit = *e.LimitValue
				}
				a.hasLimit = true
			}
		}
	}

	out := make([]domain.EffectiveEntitlement, 0, len(byKey))
	for key, a := range byKey {
		eff := domain.EffectiveEntitlement{FeatureKey: key}

		planIDs := make([]uuid.UUID, 0, len(a.planIDs))
		for id := range a.planIDs {
			planIDs = append(planIDs, id)
		}
		sort.Slice(planIDs, func(i, j int) bool { return planIDs[i].String() < planIDs[j].String() })
		eff.PlanIDs = planIDs

		if a.hasLimit {
			eff.Kind = domain.EntitlementKindLimit
			eff.Value = a.maxLimit
		} else {
			eff.Kind = domain.EntitlementKindBoolean
			eff.Value = a.anyTrue
		}
		out = append(out, eff)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].FeatureKey < out[j].FeatureKey })
	return out
}
