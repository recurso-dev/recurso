package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

// --- Mock SubscriptionRepository (minimal for MRR tests) ---
type mockSubRepoForMRR struct {
	active []*domain.Subscription
	byList map[uuid.UUID][]*domain.Subscription // tenantID -> subs (org consolidated MRR)
}

func (m *mockSubRepoForMRR) GetActiveSubscriptions(ctx context.Context, _ uuid.UUID) ([]*domain.Subscription, error) {
	return m.active, nil
}
func (m *mockSubRepoForMRR) Create(ctx context.Context, s *domain.Subscription) error { return nil }
func (m *mockSubRepoForMRR) GetByID(ctx context.Context, id uuid.UUID) (*domain.Subscription, error) {
	return nil, nil
}
func (m *mockSubRepoForMRR) GetByStripeSubscriptionID(ctx context.Context, stripeSubID string) (*domain.Subscription, error) {
	return nil, nil
}
func (m *mockSubRepoForMRR) List(ctx context.Context, tenantID uuid.UUID, filter domain.SubscriptionFilter) ([]*domain.Subscription, error) {
	return m.byList[tenantID], nil
}
func (m *mockSubRepoForMRR) Update(ctx context.Context, s *domain.Subscription) error { return nil }
func (m *mockSubRepoForMRR) CountActiveByCustomer(ctx context.Context, tenantID uuid.UUID) (map[uuid.UUID]int, error) {
	return nil, nil
}

// --- Mock PlanRepository (minimal for MRR tests) ---
type mockPlanRepoForMRR struct {
	plans map[uuid.UUID]*domain.Plan
}

func (m *mockPlanRepoForMRR) GetByID(ctx context.Context, id uuid.UUID) (*domain.Plan, error) {
	if p, ok := m.plans[id]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("plan not found")
}
func (m *mockPlanRepoForMRR) Create(ctx context.Context, p *domain.Plan) error { return nil }
func (m *mockPlanRepoForMRR) GetByCode(ctx context.Context, tenantID uuid.UUID, code string) (*domain.Plan, error) {
	return nil, nil
}
func (m *mockPlanRepoForMRR) List(ctx context.Context, tenantID uuid.UUID, filter domain.PlanFilter) ([]*domain.Plan, error) {
	return nil, nil
}

// --- Mock ExchangeRateProvider with fixed rates ---
type mockFXForMRR struct {
	rates  map[string]float64 // "FROM:TO" -> rate
	err    error              // when set, all lookups fail (simulates OXR outage)
	source string
	asOf   time.Time
	calls  int
}

func (m *mockFXForMRR) GetRate(ctx context.Context, from, to string) (float64, error) {
	m.calls++
	if from == to {
		return 1.0, nil
	}
	if m.err != nil {
		return 0, m.err
	}
	if rate, ok := m.rates[from+":"+to]; ok {
		return rate, nil
	}
	return 0, fmt.Errorf("exchange rate not available: %s -> %s", from, to)
}

func (m *mockFXForMRR) Convert(ctx context.Context, amount int64, from, to string) (int64, float64, error) {
	rate, err := m.GetRate(ctx, from, to)
	if err != nil {
		return 0, 0, err
	}
	return int64(float64(amount) * rate), rate, nil
}

func (m *mockFXForMRR) ListRates(ctx context.Context, baseCurrency string) ([]port.ExchangeRate, error) {
	return nil, nil
}

func (m *mockFXForMRR) RateMetadata() port.RateMetadata {
	return port.RateMetadata{Source: m.source, AsOf: m.asOf}
}

// --- Mock TenantLookup ---
type mockTenantLookupForMRR struct {
	tenants map[uuid.UUID]*domain.Tenant
}

func (m *mockTenantLookupForMRR) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	if t, ok := m.tenants[id]; ok {
		return t, nil
	}
	return nil, fmt.Errorf("tenant not found")
}

// --- Mock OrganizationRepository (minimal for consolidated MRR tests) ---
type mockOrgRepoForMRR struct {
	tenants []*domain.Tenant
}

func (m *mockOrgRepoForMRR) ListTenants(ctx context.Context, orgID uuid.UUID) ([]*domain.Tenant, error) {
	return m.tenants, nil
}
func (m *mockOrgRepoForMRR) Create(ctx context.Context, org *domain.Organization) error { return nil }
func (m *mockOrgRepoForMRR) GetByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	return nil, nil
}
func (m *mockOrgRepoForMRR) Update(ctx context.Context, org *domain.Organization) error { return nil }
func (m *mockOrgRepoForMRR) Delete(ctx context.Context, id uuid.UUID) error             { return nil }
func (m *mockOrgRepoForMRR) List(ctx context.Context) ([]*domain.Organization, error) {
	return nil, nil
}
func (m *mockOrgRepoForMRR) AddTenant(ctx context.Context, orgID, tenantID uuid.UUID) error {
	return nil
}
func (m *mockOrgRepoForMRR) RemoveTenant(ctx context.Context, orgID, tenantID uuid.UUID) error {
	return nil
}

// --- Helpers ---

func mrrPlan(currency string, amount int64) *domain.Plan {
	return &domain.Plan{
		ID:     uuid.New(),
		Prices: []domain.Price{{Currency: currency, Amount: amount}},
	}
}

func mrrFixture(plans ...*domain.Plan) (*mockSubRepoForMRR, *mockPlanRepoForMRR) {
	subRepo := &mockSubRepoForMRR{}
	planRepo := &mockPlanRepoForMRR{plans: make(map[uuid.UUID]*domain.Plan)}
	for _, p := range plans {
		planRepo.plans[p.ID] = p
		subRepo.active = append(subRepo.active, &domain.Subscription{ID: uuid.New(), PlanID: p.ID})
	}
	return subRepo, planRepo
}

// --- Tests ---

func TestGetMRR_SingleCurrency_NoConversion(t *testing.T) {
	planA := mrrPlan("USD", 1000)
	planB := mrrPlan("USD", 2500)
	subRepo, planRepo := mrrFixture(planA, planB)
	fxp := &mockFXForMRR{source: "live", asOf: time.Now()}

	svc := NewAnalyticsService(subRepo, nil, planRepo, nil)
	svc.SetFX(fxp, nil, "USD")

	got, err := svc.GetMRR(context.Background(), uuid.Nil)
	if err != nil {
		t.Fatalf("GetMRR: %v", err)
	}
	if got.NormalizedMRR != 3500 {
		t.Errorf("NormalizedMRR = %d, want 3500", got.NormalizedMRR)
	}
	if got.Amount != 3500 || got.MRR != 3500 {
		t.Errorf("backward-compat Amount/MRR = %d/%d, want 3500", got.Amount, got.MRR)
	}
	if got.ReportingCurrency != "USD" || got.Currency != "USD" {
		t.Errorf("reporting currency = %s/%s, want USD", got.ReportingCurrency, got.Currency)
	}
	if len(got.Breakdown) != 1 || got.Breakdown[0].Rate != 1.0 || got.Breakdown[0].Subscriptions != 2 {
		t.Errorf("unexpected breakdown: %+v", got.Breakdown)
	}
	if fxp.calls != 0 {
		t.Errorf("same-currency conversion should not hit the FX provider, got %d calls", fxp.calls)
	}
}

func TestGetMRR_MixedCurrency_ConversionMath(t *testing.T) {
	usdPlan := mrrPlan("USD", 10000) // $100.00
	eurPlan := mrrPlan("EUR", 9200)  // €92.00
	subRepo, planRepo := mrrFixture(usdPlan, eurPlan)

	asOf := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	fxp := &mockFXForMRR{
		rates:  map[string]float64{"EUR:USD": 1.25},
		source: "live",
		asOf:   asOf,
	}

	svc := NewAnalyticsService(subRepo, nil, planRepo, nil)
	svc.SetFX(fxp, nil, "USD")

	got, err := svc.GetMRR(context.Background(), uuid.Nil)
	if err != nil {
		t.Fatalf("GetMRR: %v", err)
	}
	// 9200 * 1.25 = 11500; total = 10000 + 11500 = 21500
	if got.NormalizedMRR != 21500 {
		t.Errorf("NormalizedMRR = %d, want 21500", got.NormalizedMRR)
	}
	if len(got.Breakdown) != 2 {
		t.Fatalf("breakdown length = %d, want 2", len(got.Breakdown))
	}
	// Sorted by currency: EUR then USD.
	eur, usd := got.Breakdown[0], got.Breakdown[1]
	if eur.Currency != "EUR" || eur.Amount != 9200 || eur.ConvertedAmount != 11500 || eur.Rate != 1.25 {
		t.Errorf("EUR entry = %+v", eur)
	}
	if usd.Currency != "USD" || usd.Amount != 10000 || usd.ConvertedAmount != 10000 || usd.Rate != 1.0 {
		t.Errorf("USD entry = %+v", usd)
	}
	if got.FX == nil {
		t.Fatal("FX snapshot missing")
	}
	if got.FX.Source != "live" {
		t.Errorf("FX.Source = %q, want live", got.FX.Source)
	}
	if !got.FX.AsOf.Equal(asOf) {
		t.Errorf("FX.AsOf = %v, want %v", got.FX.AsOf, asOf)
	}
	if got.FX.Rates["EUR"] != 1.25 || got.FX.Rates["USD"] != 1.0 {
		t.Errorf("FX.Rates = %v", got.FX.Rates)
	}
}

func TestGetMRR_RoundsHalfAwayFromZero(t *testing.T) {
	// 1001 * 0.915 = 915.915 -> 916 (int64 truncation would give 915)
	plan := mrrPlan("EUR", 1001)
	subRepo, planRepo := mrrFixture(plan)
	fxp := &mockFXForMRR{rates: map[string]float64{"EUR:USD": 0.915}, source: "live"}

	svc := NewAnalyticsService(subRepo, nil, planRepo, nil)
	svc.SetFX(fxp, nil, "USD")

	got, err := svc.GetMRR(context.Background(), uuid.Nil)
	if err != nil {
		t.Fatalf("GetMRR: %v", err)
	}
	if got.NormalizedMRR != 916 {
		t.Errorf("NormalizedMRR = %d, want 916 (rounded)", got.NormalizedMRR)
	}
}

func TestGetMRR_FallbackSourceFlagged(t *testing.T) {
	plan := mrrPlan("EUR", 9200)
	subRepo, planRepo := mrrFixture(plan)

	primary := &mockFXForMRR{err: fmt.Errorf("OXR unreachable"), source: "live"}
	fallbackAsOf := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	fallback := &mockFXForMRR{
		rates:  map[string]float64{"EUR:USD": 1.10},
		source: "static-fallback",
		asOf:   fallbackAsOf,
	}

	svc := NewAnalyticsService(subRepo, nil, planRepo, nil)
	svc.SetFX(primary, fallback, "USD")

	got, err := svc.GetMRR(context.Background(), uuid.Nil)
	if err != nil {
		t.Fatalf("GetMRR: %v", err)
	}
	if got.NormalizedMRR != 10120 { // 9200 * 1.10
		t.Errorf("NormalizedMRR = %d, want 10120", got.NormalizedMRR)
	}
	if got.FX == nil || got.FX.Source != "static-fallback" {
		t.Fatalf("FX source not flagged as static-fallback: %+v", got.FX)
	}
	if !got.FX.AsOf.Equal(fallbackAsOf) {
		t.Errorf("FX.AsOf = %v, want fallback's %v", got.FX.AsOf, fallbackAsOf)
	}
}

func TestGetMRR_UnconvertibleCurrency_ExcludedAndFlagged(t *testing.T) {
	usdPlan := mrrPlan("USD", 5000)
	xyzPlan := mrrPlan("XYZ", 7000)
	subRepo, planRepo := mrrFixture(usdPlan, xyzPlan)
	fxp := &mockFXForMRR{source: "live"}

	svc := NewAnalyticsService(subRepo, nil, planRepo, nil)
	svc.SetFX(fxp, nil, "USD")

	got, err := svc.GetMRR(context.Background(), uuid.Nil)
	if err != nil {
		t.Fatalf("GetMRR: %v", err)
	}
	if got.NormalizedMRR != 5000 {
		t.Errorf("NormalizedMRR = %d, want 5000 (XYZ excluded)", got.NormalizedMRR)
	}
	var xyz *MRRCurrencyBreakdown
	for i := range got.Breakdown {
		if got.Breakdown[i].Currency == "XYZ" {
			xyz = &got.Breakdown[i]
		}
	}
	if xyz == nil || xyz.Error == "" {
		t.Errorf("XYZ entry should carry a conversion error: %+v", got.Breakdown)
	}
}

func TestGetMRR_ZeroSubscriptions(t *testing.T) {
	subRepo := &mockSubRepoForMRR{}
	planRepo := &mockPlanRepoForMRR{plans: map[uuid.UUID]*domain.Plan{}}
	fxp := &mockFXForMRR{source: "live"}

	svc := NewAnalyticsService(subRepo, nil, planRepo, nil)
	svc.SetFX(fxp, nil, "USD")

	got, err := svc.GetMRR(context.Background(), uuid.Nil)
	if err != nil {
		t.Fatalf("GetMRR: %v", err)
	}
	if got.NormalizedMRR != 0 || got.Amount != 0 {
		t.Errorf("expected zero MRR, got %+v", got)
	}
	if got.Breakdown == nil || len(got.Breakdown) != 0 {
		t.Errorf("expected empty (non-nil) breakdown, got %+v", got.Breakdown)
	}
	if got.ReportingCurrency != "USD" {
		t.Errorf("ReportingCurrency = %q, want USD", got.ReportingCurrency)
	}
}

func TestGetMRR_TenantBaseCurrencyPreferred(t *testing.T) {
	plan := mrrPlan("USD", 10000)
	subRepo, planRepo := mrrFixture(plan)
	fxp := &mockFXForMRR{rates: map[string]float64{"USD:EUR": 0.92}, source: "live"}

	tenantID := uuid.New()
	lookup := &mockTenantLookupForMRR{tenants: map[uuid.UUID]*domain.Tenant{
		tenantID: {ID: tenantID, BaseCurrency: "EUR"},
	}}

	svc := NewAnalyticsService(subRepo, nil, planRepo, nil)
	svc.SetFX(fxp, nil, "USD")
	svc.SetTenantLookup(lookup)

	got, err := svc.GetMRR(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("GetMRR: %v", err)
	}
	if got.ReportingCurrency != "EUR" {
		t.Errorf("ReportingCurrency = %q, want EUR (tenant base currency)", got.ReportingCurrency)
	}
	if got.NormalizedMRR != 9200 { // 10000 * 0.92
		t.Errorf("NormalizedMRR = %d, want 9200", got.NormalizedMRR)
	}
}

func TestGetConsolidatedMRR_Normalized(t *testing.T) {
	usdPlan := mrrPlan("USD", 10000)
	eurPlan := mrrPlan("EUR", 9200)

	tenantA := &domain.Tenant{ID: uuid.New(), Name: "A"}
	tenantB := &domain.Tenant{ID: uuid.New(), Name: "B"}

	subRepo := &mockSubRepoForMRR{byList: map[uuid.UUID][]*domain.Subscription{
		tenantA.ID: {{ID: uuid.New(), PlanID: usdPlan.ID}},
		tenantB.ID: {{ID: uuid.New(), PlanID: eurPlan.ID}},
	}}
	planRepo := &mockPlanRepoForMRR{plans: map[uuid.UUID]*domain.Plan{
		usdPlan.ID: usdPlan,
		eurPlan.ID: eurPlan,
	}}
	orgRepo := &mockOrgRepoForMRR{tenants: []*domain.Tenant{tenantA, tenantB}}
	fxp := &mockFXForMRR{rates: map[string]float64{"EUR:USD": 1.25}, source: "live"}

	svc := NewOrganizationService(orgRepo, subRepo, planRepo)
	svc.SetFX(fxp, nil, "USD")

	got, err := svc.GetConsolidatedMRR(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("GetConsolidatedMRR: %v", err)
	}
	if got.NormalizedMRR != 21500 { // 10000 + 9200*1.25
		t.Errorf("NormalizedMRR = %d, want 21500", got.NormalizedMRR)
	}
	if got.ReportingCurrency != "USD" {
		t.Errorf("ReportingCurrency = %q, want USD", got.ReportingCurrency)
	}
	if len(got.ByCurrency) != 2 {
		t.Fatalf("ByCurrency length = %d, want 2", len(got.ByCurrency))
	}
	// Sorted: EUR then USD.
	eur := got.ByCurrency[0]
	if eur.Currency != "EUR" || eur.TotalMRR != 9200 || eur.ConvertedMRR != 11500 || eur.Rate != 1.25 {
		t.Errorf("EUR entry = %+v", eur)
	}
	if got.FX == nil || got.FX.Source != "live" {
		t.Errorf("FX snapshot = %+v, want live source", got.FX)
	}
}

func TestGetConsolidatedMRR_ZeroTenants(t *testing.T) {
	svc := NewOrganizationService(&mockOrgRepoForMRR{}, &mockSubRepoForMRR{}, &mockPlanRepoForMRR{})
	svc.SetFX(&mockFXForMRR{source: "live"}, nil, "USD")

	got, err := svc.GetConsolidatedMRR(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("GetConsolidatedMRR: %v", err)
	}
	if got.NormalizedMRR != 0 || len(got.ByCurrency) != 0 {
		t.Errorf("expected empty metrics, got %+v", got)
	}
}
