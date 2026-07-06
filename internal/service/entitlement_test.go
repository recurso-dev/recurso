package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

// --- Mocks ---

type mockEntitlementPlanRepo struct {
	port.PlanRepository
	plans map[uuid.UUID]*domain.Plan
}

func (m *mockEntitlementPlanRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Plan, error) {
	return m.plans[id], nil // nil = not found (repo convention)
}

type mockEntitlementCustomerRepo struct {
	port.CustomerRepository
	customers map[uuid.UUID]*domain.Customer
}

func (m *mockEntitlementCustomerRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Customer, error) {
	return m.customers[id], nil
}

type mockEntitlementSubRepo struct {
	port.SubscriptionRepository
	subs []*domain.Subscription
}

func (m *mockEntitlementSubRepo) List(_ context.Context, tenantID uuid.UUID, filter domain.SubscriptionFilter) ([]*domain.Subscription, error) {
	var out []*domain.Subscription
	for _, s := range m.subs {
		if s.TenantID == tenantID && s.CustomerID == filter.CustomerID {
			out = append(out, s)
		}
	}
	return out, nil
}

type mockEntitlementRepo struct {
	byPlan   map[uuid.UUID][]domain.Entitlement
	replaced map[uuid.UUID][]domain.Entitlement
	check    *domain.EntitlementCheck
	checkErr error
	// captured CheckFeature args
	checkTenant   uuid.UUID
	checkCustomer uuid.UUID
	checkFeature  string
	checkCalls    int
}

func (m *mockEntitlementRepo) ReplaceForPlan(_ context.Context, _, planID uuid.UUID, ents []domain.Entitlement) error {
	if m.replaced == nil {
		m.replaced = map[uuid.UUID][]domain.Entitlement{}
	}
	m.replaced[planID] = ents
	if m.byPlan == nil {
		m.byPlan = map[uuid.UUID][]domain.Entitlement{}
	}
	m.byPlan[planID] = ents
	return nil
}

func (m *mockEntitlementRepo) ListByPlan(_ context.Context, _, planID uuid.UUID) ([]domain.Entitlement, error) {
	return m.byPlan[planID], nil
}

func (m *mockEntitlementRepo) ListByPlanIDs(_ context.Context, _ uuid.UUID, planIDs []uuid.UUID) ([]domain.Entitlement, error) {
	var out []domain.Entitlement
	for _, id := range planIDs {
		out = append(out, m.byPlan[id]...)
	}
	return out, nil
}

func (m *mockEntitlementRepo) CheckFeature(_ context.Context, tenantID, customerID uuid.UUID, featureKey string) (*domain.EntitlementCheck, error) {
	m.checkCalls++
	m.checkTenant = tenantID
	m.checkCustomer = customerID
	m.checkFeature = featureKey
	if m.checkErr != nil {
		return nil, m.checkErr
	}
	if m.check != nil {
		return m.check, nil
	}
	return &domain.EntitlementCheck{FeatureKey: featureKey}, nil
}

// --- Helpers ---

func boolPtr(b bool) *bool    { return &b }
func int64Ptr(v int64) *int64 { return &v }

type entitlementFixture struct {
	svc      *EntitlementService
	ents     *mockEntitlementRepo
	plans    *mockEntitlementPlanRepo
	subs     *mockEntitlementSubRepo
	tenantID uuid.UUID
	customer *domain.Customer
}

func newEntitlementFixture() *entitlementFixture {
	tenantID := uuid.New()
	customer := &domain.Customer{ID: uuid.New(), TenantID: tenantID}

	ents := &mockEntitlementRepo{byPlan: map[uuid.UUID][]domain.Entitlement{}}
	plans := &mockEntitlementPlanRepo{plans: map[uuid.UUID]*domain.Plan{}}
	custs := &mockEntitlementCustomerRepo{customers: map[uuid.UUID]*domain.Customer{customer.ID: customer}}
	subs := &mockEntitlementSubRepo{}

	return &entitlementFixture{
		svc:      NewEntitlementService(ents, plans, custs, subs),
		ents:     ents,
		plans:    plans,
		subs:     subs,
		tenantID: tenantID,
		customer: customer,
	}
}

func (f *entitlementFixture) addPlan() uuid.UUID {
	id := uuid.New()
	f.plans.plans[id] = &domain.Plan{ID: id, TenantID: f.tenantID}
	return id
}

func (f *entitlementFixture) addSubscription(planID uuid.UUID, status domain.SubscriptionStatus) {
	f.subs.subs = append(f.subs.subs, &domain.Subscription{
		ID:         uuid.New(),
		TenantID:   f.tenantID,
		CustomerID: f.customer.ID,
		PlanID:     planID,
		Status:     status,
	})
}

func (f *entitlementFixture) grant(planID uuid.UUID, key string, kind domain.EntitlementKind, boolVal *bool, limitVal *int64) {
	f.ents.byPlan[planID] = append(f.ents.byPlan[planID], domain.Entitlement{
		ID:         uuid.New(),
		TenantID:   f.tenantID,
		PlanID:     planID,
		FeatureKey: key,
		Kind:       kind,
		BoolValue:  boolVal,
		LimitValue: limitVal,
	})
}

// --- Union semantics ---

func TestGetCustomerEntitlements_BooleanAnyTrueWins(t *testing.T) {
	f := newEntitlementFixture()
	planA, planB := f.addPlan(), f.addPlan()
	f.addSubscription(planA, domain.SubscriptionStatusActive)
	f.addSubscription(planB, domain.SubscriptionStatusTrialing)

	f.grant(planA, "sso", domain.EntitlementKindBoolean, boolPtr(false), nil)
	f.grant(planB, "sso", domain.EntitlementKindBoolean, boolPtr(true), nil)

	eff, err := f.svc.GetCustomerEntitlements(context.Background(), f.tenantID, f.customer.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(eff) != 1 {
		t.Fatalf("expected 1 effective entitlement, got %d", len(eff))
	}
	if eff[0].FeatureKey != "sso" || eff[0].Kind != domain.EntitlementKindBoolean {
		t.Errorf("unexpected entry: %+v", eff[0])
	}
	if v, ok := eff[0].Value.(bool); !ok || !v {
		t.Errorf("boolean union should be true when any plan grants true, got %v", eff[0].Value)
	}
	if len(eff[0].PlanIDs) != 2 {
		t.Errorf("expected both plans as sources, got %v", eff[0].PlanIDs)
	}
}

func TestGetCustomerEntitlements_LimitMaxWins(t *testing.T) {
	f := newEntitlementFixture()
	planA, planB := f.addPlan(), f.addPlan()
	f.addSubscription(planA, domain.SubscriptionStatusActive)
	f.addSubscription(planB, domain.SubscriptionStatusActive)

	f.grant(planA, "api_calls", domain.EntitlementKindLimit, nil, int64Ptr(1000))
	f.grant(planB, "api_calls", domain.EntitlementKindLimit, nil, int64Ptr(50000))

	eff, err := f.svc.GetCustomerEntitlements(context.Background(), f.tenantID, f.customer.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(eff) != 1 {
		t.Fatalf("expected 1 effective entitlement, got %d", len(eff))
	}
	if v, ok := eff[0].Value.(int64); !ok || v != 50000 {
		t.Errorf("limit union should be MAX (50000), got %v", eff[0].Value)
	}
	if eff[0].Kind != domain.EntitlementKindLimit {
		t.Errorf("kind = %q, want limit", eff[0].Kind)
	}
}

func TestGetCustomerEntitlements_IgnoresNonActiveSubscriptions(t *testing.T) {
	f := newEntitlementFixture()
	planA, planB := f.addPlan(), f.addPlan()
	f.addSubscription(planA, domain.SubscriptionStatusCanceled)
	f.addSubscription(planB, domain.SubscriptionStatusPastDue)

	f.grant(planA, "sso", domain.EntitlementKindBoolean, boolPtr(true), nil)
	f.grant(planB, "api_calls", domain.EntitlementKindLimit, nil, int64Ptr(100))

	eff, err := f.svc.GetCustomerEntitlements(context.Background(), f.tenantID, f.customer.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(eff) != 0 {
		t.Errorf("canceled/past_due subscriptions must not grant entitlements, got %+v", eff)
	}
}

func TestGetCustomerEntitlements_EmptyWithoutSubscriptions(t *testing.T) {
	f := newEntitlementFixture()

	eff, err := f.svc.GetCustomerEntitlements(context.Background(), f.tenantID, f.customer.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if eff == nil || len(eff) != 0 {
		t.Errorf("expected empty (non-nil) set, got %#v", eff)
	}
}

func TestGetCustomerEntitlements_CustomerNotFound(t *testing.T) {
	f := newEntitlementFixture()

	_, err := f.svc.GetCustomerEntitlements(context.Background(), f.tenantID, uuid.New())
	if !errors.Is(err, ErrEntitlementCustomerNotFound) {
		t.Errorf("expected ErrEntitlementCustomerNotFound, got %v", err)
	}
}

func TestGetCustomerEntitlements_TenantIsolation(t *testing.T) {
	f := newEntitlementFixture()
	// Customer exists but belongs to a different tenant.
	otherTenant := uuid.New()

	_, err := f.svc.GetCustomerEntitlements(context.Background(), otherTenant, f.customer.ID)
	if !errors.Is(err, ErrEntitlementCustomerNotFound) {
		t.Errorf("cross-tenant customer must be rejected as not found, got %v", err)
	}
}

// --- PUT replace semantics + validation ---

func TestSetPlanEntitlements_ReplaceRemovesAbsentKeys(t *testing.T) {
	f := newEntitlementFixture()
	planID := f.addPlan()

	first := []EntitlementInput{
		{FeatureKey: "sso", Kind: "boolean", BoolValue: boolPtr(true)},
		{FeatureKey: "api_calls", Kind: "limit", LimitValue: int64Ptr(1000)},
	}
	if _, err := f.svc.SetPlanEntitlements(context.Background(), f.tenantID, planID, first); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	second := []EntitlementInput{
		{FeatureKey: "api_calls", Kind: "limit", LimitValue: int64Ptr(2000)},
	}
	if _, err := f.svc.SetPlanEntitlements(context.Background(), f.tenantID, planID, second); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stored := f.ents.replaced[planID]
	if len(stored) != 1 {
		t.Fatalf("replace must pass the full new set (absent keys removed), got %d rows", len(stored))
	}
	if stored[0].FeatureKey != "api_calls" || *stored[0].LimitValue != 2000 {
		t.Errorf("unexpected stored entitlement: %+v", stored[0])
	}
}

func TestSetPlanEntitlements_EmptySetClearsAll(t *testing.T) {
	f := newEntitlementFixture()
	planID := f.addPlan()

	if _, err := f.svc.SetPlanEntitlements(context.Background(), f.tenantID, planID, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, ok := f.ents.replaced[planID]; !ok || len(got) != 0 {
		t.Errorf("empty PUT should clear the set via replace, got %v (called=%v)", got, ok)
	}
}

func TestSetPlanEntitlements_TenantIsolation(t *testing.T) {
	f := newEntitlementFixture()
	planID := f.addPlan()
	// Plan lookup succeeds, but the plan belongs to f.tenantID, not the caller.
	otherTenant := uuid.New()

	_, err := f.svc.SetPlanEntitlements(context.Background(), otherTenant, planID, []EntitlementInput{
		{FeatureKey: "sso", Kind: "boolean", BoolValue: boolPtr(true)},
	})
	if !errors.Is(err, ErrEntitlementPlanNotFound) {
		t.Errorf("plan from another tenant must be rejected, got %v", err)
	}
	if len(f.ents.replaced) != 0 {
		t.Error("replace must not be called for a cross-tenant plan")
	}
}

func TestSetPlanEntitlements_UnknownPlan(t *testing.T) {
	f := newEntitlementFixture()

	_, err := f.svc.SetPlanEntitlements(context.Background(), f.tenantID, uuid.New(), nil)
	if !errors.Is(err, ErrEntitlementPlanNotFound) {
		t.Errorf("expected ErrEntitlementPlanNotFound, got %v", err)
	}
}

func TestSetPlanEntitlements_Validation(t *testing.T) {
	f := newEntitlementFixture()
	planID := f.addPlan()

	cases := []struct {
		name  string
		input EntitlementInput
	}{
		{"missing feature_key", EntitlementInput{Kind: "boolean", BoolValue: boolPtr(true)}},
		{"bad feature_key chars", EntitlementInput{FeatureKey: "has spaces", Kind: "boolean", BoolValue: boolPtr(true)}},
		{"bad kind", EntitlementInput{FeatureKey: "sso", Kind: "flag", BoolValue: boolPtr(true)}},
		{"boolean missing bool_value", EntitlementInput{FeatureKey: "sso", Kind: "boolean"}},
		{"boolean with limit_value", EntitlementInput{FeatureKey: "sso", Kind: "boolean", BoolValue: boolPtr(true), LimitValue: int64Ptr(1)}},
		{"limit missing limit_value", EntitlementInput{FeatureKey: "api", Kind: "limit"}},
		{"limit negative", EntitlementInput{FeatureKey: "api", Kind: "limit", LimitValue: int64Ptr(-1)}},
		{"limit with bool_value", EntitlementInput{FeatureKey: "api", Kind: "limit", LimitValue: int64Ptr(1), BoolValue: boolPtr(true)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := f.svc.SetPlanEntitlements(context.Background(), f.tenantID, planID, []EntitlementInput{tc.input})
			var valErr EntitlementValidationError
			if !errors.As(err, &valErr) {
				t.Errorf("expected validation error, got %v", err)
			}
		})
	}

	// Duplicate keys within one payload.
	_, err := f.svc.SetPlanEntitlements(context.Background(), f.tenantID, planID, []EntitlementInput{
		{FeatureKey: "sso", Kind: "boolean", BoolValue: boolPtr(true)},
		{FeatureKey: "sso", Kind: "boolean", BoolValue: boolPtr(false)},
	})
	var valErr EntitlementValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("duplicate feature_key must fail validation, got %v", err)
	}
}

// --- Check fast path ---

func TestCheckFeature_DelegatesSingleRepoCall(t *testing.T) {
	f := newEntitlementFixture()
	limit := int64(500)
	f.ents.check = &domain.EntitlementCheck{FeatureKey: "api_calls", Granted: true, LimitValue: &limit}

	got, err := f.svc.CheckFeature(context.Background(), f.tenantID, f.customer.ID, "api_calls")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Granted || got.LimitValue == nil || *got.LimitValue != 500 {
		t.Errorf("unexpected check result: %+v", got)
	}
	if f.ents.checkCalls != 1 {
		t.Errorf("hot path must be exactly one repo call, got %d", f.ents.checkCalls)
	}
	if f.ents.checkTenant != f.tenantID || f.ents.checkCustomer != f.customer.ID || f.ents.checkFeature != "api_calls" {
		t.Error("check must pass tenant, customer, and feature through unchanged")
	}
}

func TestCheckFeature_AbsentGrantIsFalseWithNilLimit(t *testing.T) {
	f := newEntitlementFixture()
	f.ents.check = &domain.EntitlementCheck{FeatureKey: "nope", Granted: false, LimitValue: nil}

	got, err := f.svc.CheckFeature(context.Background(), f.tenantID, f.customer.ID, "nope")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Granted || got.LimitValue != nil {
		t.Errorf("absent grant must be granted=false, limit=nil, got %+v", got)
	}
}

func TestCheckFeature_Validation(t *testing.T) {
	f := newEntitlementFixture()

	var valErr EntitlementValidationError
	if _, err := f.svc.CheckFeature(context.Background(), f.tenantID, uuid.Nil, "sso"); !errors.As(err, &valErr) {
		t.Errorf("nil customer_id must fail validation, got %v", err)
	}
	if _, err := f.svc.CheckFeature(context.Background(), f.tenantID, f.customer.ID, ""); !errors.As(err, &valErr) {
		t.Errorf("empty feature must fail validation, got %v", err)
	}
	if f.ents.checkCalls != 0 {
		t.Error("repo must not be hit on validation failure")
	}
}

// --- Mixed kinds (documented resolution: limit wins, max cap kept) ---

func TestGetCustomerEntitlements_MixedKindsResolveToLimit(t *testing.T) {
	f := newEntitlementFixture()
	planA, planB := f.addPlan(), f.addPlan()
	f.addSubscription(planA, domain.SubscriptionStatusActive)
	f.addSubscription(planB, domain.SubscriptionStatusActive)

	f.grant(planA, "exports", domain.EntitlementKindBoolean, boolPtr(true), nil)
	f.grant(planB, "exports", domain.EntitlementKindLimit, nil, int64Ptr(10))

	eff, err := f.svc.GetCustomerEntitlements(context.Background(), f.tenantID, f.customer.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(eff) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(eff))
	}
	if eff[0].Kind != domain.EntitlementKindLimit {
		t.Errorf("mixed kinds must resolve to limit, got %q", eff[0].Kind)
	}
	if v, ok := eff[0].Value.(int64); !ok || v != 10 {
		t.Errorf("expected max limit 10, got %v", eff[0].Value)
	}
}
