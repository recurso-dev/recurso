package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// --- Mocks ---

type mockUsageRepo struct {
	port.UsageRepository

	// RecordEvent
	recorded []*domain.UsageEvent

	// QueryUsage
	queryBuckets []domain.UsageBucket
	queryErr     error
	queryTenant  uuid.UUID
	queryFilter  domain.UsageQueryFilter
	queryCalls   int

	// GetSubscriptionUsageByDimension
	subUsage       []domain.SubscriptionDimensionUsage
	subUsageErr    error
	subUsageTenant uuid.UUID
	subUsageSubID  uuid.UUID
	subUsageStart  time.Time
	subUsageEnd    time.Time

	// ListDimensions
	dims       []domain.UsageDimension
	dimsErr    error
	dimsTenant uuid.UUID
}

func (m *mockUsageRepo) RecordEvent(_ context.Context, event *domain.UsageEvent) error {
	m.recorded = append(m.recorded, event)
	return nil
}

func (m *mockUsageRepo) QueryUsage(_ context.Context, tenantID uuid.UUID, filter domain.UsageQueryFilter) ([]domain.UsageBucket, error) {
	m.queryCalls++
	m.queryTenant = tenantID
	m.queryFilter = filter
	return m.queryBuckets, m.queryErr
}

func (m *mockUsageRepo) GetSubscriptionUsageByDimension(_ context.Context, tenantID, subID uuid.UUID, start, end time.Time) ([]domain.SubscriptionDimensionUsage, error) {
	m.subUsageTenant = tenantID
	m.subUsageSubID = subID
	m.subUsageStart = start
	m.subUsageEnd = end
	return m.subUsage, m.subUsageErr
}

func (m *mockUsageRepo) ListDimensions(_ context.Context, tenantID uuid.UUID) ([]domain.UsageDimension, error) {
	m.dimsTenant = tenantID
	return m.dims, m.dimsErr
}

type mockUsageSubRepo struct {
	port.SubscriptionRepository
	subs map[uuid.UUID]*domain.Subscription
}

func (m *mockUsageSubRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Subscription, error) {
	return m.subs[id], nil // nil = not found (repo convention)
}

type mockUsageChecker struct {
	checks   map[string]*domain.EntitlementCheck // by feature key
	err      error
	calls    int
	tenants  []uuid.UUID
	custs    []uuid.UUID
	features []string
}

func (m *mockUsageChecker) CheckFeature(_ context.Context, tenantID, customerID uuid.UUID, featureKey string) (*domain.EntitlementCheck, error) {
	m.calls++
	m.tenants = append(m.tenants, tenantID)
	m.custs = append(m.custs, customerID)
	m.features = append(m.features, featureKey)
	if m.err != nil {
		return nil, m.err
	}
	if c, ok := m.checks[featureKey]; ok {
		return c, nil
	}
	return &domain.EntitlementCheck{FeatureKey: featureKey}, nil
}

// --- Fixture ---

type usageFixture struct {
	svc      *UsageService
	usage    *mockUsageRepo
	subs     *mockUsageSubRepo
	checker  *mockUsageChecker
	tenantID uuid.UUID
	now      time.Time
}

func newUsageFixture() *usageFixture {
	f := &usageFixture{
		usage:    &mockUsageRepo{},
		subs:     &mockUsageSubRepo{subs: map[uuid.UUID]*domain.Subscription{}},
		checker:  &mockUsageChecker{checks: map[string]*domain.EntitlementCheck{}},
		tenantID: uuid.New(),
		now:      time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC),
	}
	f.svc = NewUsageService(f.usage, f.subs, f.checker)
	f.svc.now = func() time.Time { return f.now }
	return f
}

func (f *usageFixture) addSubscription(tenantID uuid.UUID) *domain.Subscription {
	sub := &domain.Subscription{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		CustomerID:         uuid.New(),
		Status:             domain.SubscriptionStatusActive,
		CurrentPeriodStart: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		CurrentPeriodEnd:   time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
	}
	f.subs.subs[sub.ID] = sub
	return sub
}

// --- QueryUsage: windowing ---

func TestQueryUsageDefaultsWindowAndGranularity(t *testing.T) {
	f := newUsageFixture()
	custID := uuid.New()

	buckets, resolved, err := f.svc.QueryUsage(context.Background(), f.tenantID, UsageQueryParams{CustomerID: &custID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantFrom := f.now.Add(-30 * 24 * time.Hour)
	if !f.usage.queryFilter.To.Equal(f.now) || !f.usage.queryFilter.From.Equal(wantFrom) {
		t.Errorf("default window = [%v, %v), want [%v, %v)", f.usage.queryFilter.From, f.usage.queryFilter.To, wantFrom, f.now)
	}
	if f.usage.queryFilter.Granularity != domain.UsageGranularityDay {
		t.Errorf("default granularity = %q, want %q", f.usage.queryFilter.Granularity, domain.UsageGranularityDay)
	}
	if !resolved.From.Equal(wantFrom) || !resolved.To.Equal(f.now) || resolved.Granularity != domain.UsageGranularityDay {
		t.Errorf("resolved echo = %+v, want from=%v to=%v granularity=day", resolved, wantFrom, f.now)
	}
	if buckets == nil || len(buckets) != 0 {
		t.Errorf("nil repo result should normalize to empty slice, got %#v", buckets)
	}
}

func TestQueryUsageExplicitWindowAndFilters(t *testing.T) {
	f := newUsageFixture()
	subID := uuid.New()
	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	want := []domain.UsageBucket{
		{Period: from, Dimension: "api_calls", Quantity: 100},
		{Period: from.AddDate(0, 0, 1), Dimension: "api_calls", Quantity: 50},
	}
	f.usage.queryBuckets = want

	got, _, err := f.svc.QueryUsage(context.Background(), f.tenantID, UsageQueryParams{
		SubscriptionID: &subID,
		Dimension:      "api_calls",
		From:           &from,
		To:             &to,
		Granularity:    domain.UsageGranularityMonth,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fl := f.usage.queryFilter
	if fl.SubscriptionID == nil || *fl.SubscriptionID != subID {
		t.Errorf("subscription_id not forwarded: %v", fl.SubscriptionID)
	}
	if fl.Dimension != "api_calls" || fl.Granularity != domain.UsageGranularityMonth {
		t.Errorf("dimension/granularity not forwarded: %+v", fl)
	}
	if !fl.From.Equal(from) || !fl.To.Equal(to) {
		t.Errorf("window = [%v, %v), want [%v, %v)", fl.From, fl.To, from, to)
	}
	if f.usage.queryTenant != f.tenantID {
		t.Errorf("tenant = %v, want %v", f.usage.queryTenant, f.tenantID)
	}
	if len(got) != 2 || got[0].Quantity != 100 || got[1].Quantity != 50 {
		t.Errorf("buckets not passed through: %#v", got)
	}
}

func TestQueryUsageFromOnlyDefaultsToNow(t *testing.T) {
	f := newUsageFixture()
	custID := uuid.New()
	from := f.now.Add(-90 * 24 * time.Hour)

	_, resolved, err := f.svc.QueryUsage(context.Background(), f.tenantID, UsageQueryParams{CustomerID: &custID, From: &from})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resolved.From.Equal(from) || !resolved.To.Equal(f.now) {
		t.Errorf("window = [%v, %v), want [%v, %v)", resolved.From, resolved.To, from, f.now)
	}
}

func TestQueryUsageValidation(t *testing.T) {
	f := newUsageFixture()
	custID := uuid.New()
	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	toBeforeFrom := from.Add(-time.Hour)

	cases := []struct {
		name   string
		params UsageQueryParams
	}{
		{"neither subscription_id nor customer_id", UsageQueryParams{}},
		{"bad granularity", UsageQueryParams{CustomerID: &custID, Granularity: "week"}},
		{"from not before to", UsageQueryParams{CustomerID: &custID, From: &from, To: &toBeforeFrom}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := f.svc.QueryUsage(context.Background(), f.tenantID, tc.params)
			var valErr UsageValidationError
			if !errors.As(err, &valErr) {
				t.Fatalf("expected UsageValidationError, got %v", err)
			}
		})
	}
	if f.usage.queryCalls != 0 {
		t.Errorf("repo must not be queried on validation failure, got %d calls", f.usage.queryCalls)
	}
}

// --- GetSubscriptionUsage: period vs lifetime + entitlement join ---

func TestGetSubscriptionUsagePeriodAndLifetime(t *testing.T) {
	f := newUsageFixture()
	sub := f.addSubscription(f.tenantID)
	f.usage.subUsage = []domain.SubscriptionDimensionUsage{
		{Dimension: "api_calls", PeriodQuantity: 4231, LifetimeQuantity: 98000},
	}

	got, err := f.svc.GetSubscriptionUsage(context.Background(), f.tenantID, sub.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Repo must be asked for the subscription's CURRENT billing period.
	if !f.usage.subUsageStart.Equal(sub.CurrentPeriodStart) || !f.usage.subUsageEnd.Equal(sub.CurrentPeriodEnd) {
		t.Errorf("period window = [%v, %v), want [%v, %v)", f.usage.subUsageStart, f.usage.subUsageEnd, sub.CurrentPeriodStart, sub.CurrentPeriodEnd)
	}
	if f.usage.subUsageTenant != f.tenantID || f.usage.subUsageSubID != sub.ID {
		t.Errorf("repo scoping: tenant=%v sub=%v", f.usage.subUsageTenant, f.usage.subUsageSubID)
	}

	if got.SubscriptionID != sub.ID || got.CustomerID != sub.CustomerID {
		t.Errorf("envelope ids: %+v", got)
	}
	if !got.CurrentPeriodStart.Equal(sub.CurrentPeriodStart) || !got.CurrentPeriodEnd.Equal(sub.CurrentPeriodEnd) {
		t.Errorf("envelope period: %+v", got)
	}
	if len(got.Dimensions) != 1 || got.Dimensions[0].PeriodQuantity != 4231 || got.Dimensions[0].LifetimeQuantity != 98000 {
		t.Errorf("dimensions: %#v", got.Dimensions)
	}
}

func TestGetSubscriptionUsageEntitlementLimitJoin(t *testing.T) {
	f := newUsageFixture()
	sub := f.addSubscription(f.tenantID)
	f.usage.subUsage = []domain.SubscriptionDimensionUsage{
		{Dimension: "api_calls", PeriodQuantity: 4231, LifetimeQuantity: 98000},
		{Dimension: "storage_gb", PeriodQuantity: 12, LifetimeQuantity: 12},
		{Dimension: "seats used", PeriodQuantity: 3, LifetimeQuantity: 3}, // not a valid feature key
	}
	f.checker.checks["api_calls"] = &domain.EntitlementCheck{FeatureKey: "api_calls", Granted: true, LimitValue: int64Ptr(10000)}
	// storage_gb: boolean grant → no limit
	f.checker.checks["storage_gb"] = &domain.EntitlementCheck{FeatureKey: "storage_gb", Granted: true}

	got, err := f.svc.GetSubscriptionUsage(context.Background(), f.tenantID, sub.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	api := got.Dimensions[0]
	if api.LimitValue == nil || *api.LimitValue != 10000 {
		t.Fatalf("api_calls limit = %v, want 10000", api.LimitValue)
	}
	if api.Remaining == nil || *api.Remaining != 10000-4231 {
		t.Errorf("api_calls remaining = %v, want %d", api.Remaining, 10000-4231)
	}

	storage := got.Dimensions[1]
	if storage.LimitValue != nil || storage.Remaining != nil {
		t.Errorf("boolean grant must not produce a limit: %+v", storage)
	}

	seats := got.Dimensions[2]
	if seats.LimitValue != nil || seats.Remaining != nil {
		t.Errorf("invalid feature-key dimension must not produce a limit: %+v", seats)
	}
	for _, feat := range f.checker.features {
		if feat == "seats used" {
			t.Errorf("checker must not be consulted for invalid feature keys")
		}
	}
	if f.checker.calls != 2 {
		t.Errorf("checker calls = %d, want 2", f.checker.calls)
	}
	// The check must run against the subscription's customer and tenant.
	for i := range f.checker.tenants {
		if f.checker.tenants[i] != f.tenantID || f.checker.custs[i] != sub.CustomerID {
			t.Errorf("check %d scoped to tenant=%v cust=%v, want tenant=%v cust=%v", i, f.checker.tenants[i], f.checker.custs[i], f.tenantID, sub.CustomerID)
		}
	}
}

func TestGetSubscriptionUsageOverLimitRemainingIsNegative(t *testing.T) {
	f := newUsageFixture()
	sub := f.addSubscription(f.tenantID)
	f.usage.subUsage = []domain.SubscriptionDimensionUsage{
		{Dimension: "api_calls", PeriodQuantity: 12000, LifetimeQuantity: 12000},
	}
	f.checker.checks["api_calls"] = &domain.EntitlementCheck{FeatureKey: "api_calls", Granted: true, LimitValue: int64Ptr(10000)}

	got, err := f.svc.GetSubscriptionUsage(context.Background(), f.tenantID, sub.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Dimensions[0].Remaining == nil || *got.Dimensions[0].Remaining != -2000 {
		t.Errorf("remaining = %v, want -2000", got.Dimensions[0].Remaining)
	}
}

func TestGetSubscriptionUsageEmptyUsage(t *testing.T) {
	f := newUsageFixture()
	sub := f.addSubscription(f.tenantID)

	got, err := f.svc.GetSubscriptionUsage(context.Background(), f.tenantID, sub.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Dimensions == nil || len(got.Dimensions) != 0 {
		t.Errorf("expected empty (non-nil) dimensions, got %#v", got.Dimensions)
	}
	if f.checker.calls != 0 {
		t.Errorf("no dimensions → no entitlement checks, got %d", f.checker.calls)
	}
}

// --- Tenant isolation ---

func TestGetSubscriptionUsageTenantIsolation(t *testing.T) {
	f := newUsageFixture()
	otherTenantSub := f.addSubscription(uuid.New()) // belongs to another tenant

	_, err := f.svc.GetSubscriptionUsage(context.Background(), f.tenantID, otherTenantSub.ID)
	if !errors.Is(err, ErrUsageSubscriptionNotFound) {
		t.Fatalf("cross-tenant read must 404, got %v", err)
	}

	_, err = f.svc.GetSubscriptionUsage(context.Background(), f.tenantID, uuid.New())
	if !errors.Is(err, ErrUsageSubscriptionNotFound) {
		t.Fatalf("unknown subscription must 404, got %v", err)
	}
}

// --- Dimension catalog ---

func TestListDimensions(t *testing.T) {
	f := newUsageFixture()
	first := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	last := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	f.usage.dims = []domain.UsageDimension{
		{Dimension: "api_calls", EventCount: 420, FirstSeen: first, LastSeen: last},
	}

	got, err := f.svc.ListDimensions(context.Background(), f.tenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.usage.dimsTenant != f.tenantID {
		t.Errorf("tenant = %v, want %v", f.usage.dimsTenant, f.tenantID)
	}
	if len(got) != 1 || got[0].Dimension != "api_calls" || got[0].EventCount != 420 {
		t.Errorf("catalog: %#v", got)
	}
}

func TestListDimensionsEmpty(t *testing.T) {
	f := newUsageFixture()
	got, err := f.svc.ListDimensions(context.Background(), f.tenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Errorf("expected empty (non-nil) slice, got %#v", got)
	}
}

func TestRecordEventTenantAndCustomerGuards(t *testing.T) {
	f := newUsageFixture()
	tenantA, tenantB := uuid.New(), uuid.New()
	custID, subID := uuid.New(), uuid.New()
	f.subs.subs[subID] = &domain.Subscription{ID: subID, TenantID: tenantA, CustomerID: custID}

	event := func(cust uuid.UUID) *domain.UsageEvent {
		return &domain.UsageEvent{ID: uuid.New(), SubscriptionID: subID, CustomerID: cust, Dimension: "api_calls", Quantity: 5}
	}

	// Cross-tenant write rejected as not-found (no existence leak).
	if err := f.svc.RecordEvent(context.Background(), tenantB, event(custID)); err != ErrUsageSubscriptionNotFound {
		t.Fatalf("cross-tenant record: got %v, want ErrUsageSubscriptionNotFound", err)
	}
	// Mismatched customer rejected.
	if err := f.svc.RecordEvent(context.Background(), tenantA, event(uuid.New())); err != ErrUsageCustomerMismatch {
		t.Fatalf("customer mismatch: got %v, want ErrUsageCustomerMismatch", err)
	}
	// Correct tenant + customer passes through.
	if err := f.svc.RecordEvent(context.Background(), tenantA, event(custID)); err != nil {
		t.Fatalf("valid record: unexpected error %v", err)
	}
}

// TestRecordEventRejectsNonPositiveQuantity proves the ENG-165 H2 guard: a zero
// or negative usage quantity is refused, so it cannot offset legitimate metered
// usage at aggregation time. (binding:"required" only rejects 0 at the edge.)
func TestRecordEventRejectsNonPositiveQuantity(t *testing.T) {
	f := newUsageFixture()
	tenantA := uuid.New()
	custID, subID := uuid.New(), uuid.New()
	f.subs.subs[subID] = &domain.Subscription{ID: subID, TenantID: tenantA, CustomerID: custID}

	for _, q := range []int64{0, -1, -1000} {
		ev := &domain.UsageEvent{ID: uuid.New(), SubscriptionID: subID, CustomerID: custID, Dimension: "api_calls", Quantity: q}
		var valErr UsageValidationError
		if err := f.svc.RecordEvent(context.Background(), tenantA, ev); !errors.As(err, &valErr) {
			t.Errorf("RecordEvent(quantity=%d): err = %v, want UsageValidationError", q, err)
		}
	}
}
