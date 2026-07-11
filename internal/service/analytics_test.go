package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
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

// --- billing-interval normalization (money correctness) ---

func TestMonthlyMinorUnits(t *testing.T) {
	cases := []struct {
		name   string
		amount int64
		unit   domain.IntervalUnit
		count  int
		want   int64
	}{
		{"monthly", 1000, domain.IntervalMonth, 1, 1000},
		{"unset treated as monthly", 500, "", 0, 500},
		{"annual -> /12", 12000, domain.IntervalYear, 1, 1000},
		{"biennial -> /24", 24000, domain.IntervalYear, 2, 1000},
		{"quarterly -> /3", 3000, domain.IntervalMonth, 3, 1000},
		{"weekly -> *52/12", 700, domain.IntervalWeek, 1, 3044},
		{"daily -> *365.25/12", 100, domain.IntervalDay, 1, 3044},
		{"unknown unit falls back to month-like", 800, domain.IntervalUnit("fortnight"), 1, 800},
	}
	for _, tc := range cases {
		if got := monthlyMinorUnits(tc.amount, tc.unit, tc.count); got != tc.want {
			t.Errorf("%s: monthlyMinorUnits(%d,%q,%d) = %d, want %d", tc.name, tc.amount, tc.unit, tc.count, got, tc.want)
		}
	}
}

// TestGetMRR_NormalizesBillingInterval proves an annual and a quarterly plan
// contribute their monthly-equivalent MRR (not their full billed amount), and
// that ARR is the normalized MRR × 12.
func TestGetMRR_NormalizesBillingInterval(t *testing.T) {
	annual := &domain.Plan{
		ID: uuid.New(), IntervalUnit: domain.IntervalYear, IntervalCount: 1,
		Prices: []domain.Price{{Currency: "USD", Amount: 12000}}, // 12000/yr -> 1000/mo
	}
	quarterly := &domain.Plan{
		ID: uuid.New(), IntervalUnit: domain.IntervalMonth, IntervalCount: 3,
		Prices: []domain.Price{{Currency: "USD", Amount: 3000}}, // 3000/qtr -> 1000/mo
	}
	subRepo, planRepo := mrrFixture(annual, quarterly)
	svc := NewAnalyticsService(subRepo, nil, planRepo, nil)
	svc.SetFX(&mockFXForMRR{source: "live", asOf: time.Now()}, nil, "USD")

	got, err := svc.GetMRR(context.Background(), uuid.Nil)
	if err != nil {
		t.Fatalf("GetMRR: %v", err)
	}
	if got.NormalizedMRR != 2000 {
		t.Errorf("NormalizedMRR = %d, want 2000 (1000 annual-normalized + 1000 quarterly-normalized)", got.NormalizedMRR)
	}
	if got.ARR != 24000 {
		t.Errorf("ARR = %d, want 24000 (2000 MRR × 12)", got.ARR)
	}
}

// --- MRR snapshot + waterfall ---

// fakeMRRSnapshotStore is an in-memory MRRSnapshotStore for tests.
type fakeMRRSnapshotStore struct{ rows []domain.MRRSnapshot }

func (f *fakeMRRSnapshotStore) UpsertSnapshots(_ context.Context, snaps []domain.MRRSnapshot) error {
	for _, s := range snaps {
		replaced := false
		for i, r := range f.rows {
			if r.TenantID == s.TenantID && r.SubscriptionID == s.SubscriptionID && r.SnapshotDate.Equal(s.SnapshotDate) {
				f.rows[i] = s
				replaced = true
				break
			}
		}
		if !replaced {
			f.rows = append(f.rows, s)
		}
	}
	return nil
}

func (f *fakeMRRSnapshotStore) ResolveSnapshotDate(_ context.Context, tenantID uuid.UUID, onOrBefore time.Time) (time.Time, bool, error) {
	var best time.Time
	found := false
	for _, r := range f.rows {
		if r.TenantID == tenantID && !r.SnapshotDate.After(onOrBefore) && (!found || r.SnapshotDate.After(best)) {
			best, found = r.SnapshotDate, true
		}
	}
	return best, found, nil
}

func (f *fakeMRRSnapshotStore) GetSnapshotsOn(_ context.Context, tenantID uuid.UUID, date time.Time) ([]domain.MRRSnapshot, error) {
	var out []domain.MRRSnapshot
	for _, r := range f.rows {
		if r.TenantID == tenantID && r.SnapshotDate.Equal(date) {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeMRRSnapshotStore) SubscriptionIDsSeenBefore(_ context.Context, tenantID uuid.UUID, date time.Time) (map[uuid.UUID]bool, error) {
	m := map[uuid.UUID]bool{}
	for _, r := range f.rows {
		if r.TenantID == tenantID && r.SnapshotDate.Before(date) {
			m[r.SubscriptionID] = true
		}
	}
	return m, nil
}

func snap(tenant, sub uuid.UUID, date time.Time, amount int64) domain.MRRSnapshot {
	return domain.MRRSnapshot{TenantID: tenant, SubscriptionID: sub, SnapshotDate: date, MRRAmount: amount, Currency: "USD"}
}

// TestGetMRRWaterfall_Components covers every movement type in one period.
func TestGetMRRWaterfall_Components(t *testing.T) {
	tenant := uuid.New()
	d0 := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC) // before the window (for reactivation)
	d1 := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC) // start
	d2 := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC) // end

	subA, subB, subC, subD, subE, subF := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	store := &fakeMRRSnapshotStore{rows: []domain.MRRSnapshot{
		snap(tenant, subA, d1, 1000), snap(tenant, subA, d2, 1000), // unchanged
		snap(tenant, subB, d1, 1000), snap(tenant, subB, d2, 1500), // expansion +500
		snap(tenant, subC, d1, 1000), snap(tenant, subC, d2, 600), // contraction -400
		snap(tenant, subD, d1, 1000),                             // churned -1000 (absent at d2)
		snap(tenant, subE, d2, 2000),                             // new +2000 (never seen before)
		snap(tenant, subF, d0, 800), snap(tenant, subF, d2, 800), // reactivation +800 (seen at d0, gone at d1)
	}}

	svc := NewAnalyticsService(nil, nil, nil, nil)
	svc.SetFX(&mockFXForMRR{source: "live", asOf: time.Now()}, nil, "USD")
	svc.SetSnapshotStore(store)

	wf, err := svc.GetMRRWaterfall(context.Background(), tenant, d1, d2)
	if err != nil {
		t.Fatalf("GetMRRWaterfall: %v", err)
	}
	checks := []struct {
		name string
		got  int64
		want int64
	}{
		{"StartingMRR", wf.StartingMRR, 4000},
		{"EndingMRR", wf.EndingMRR, 5900},
		{"New", wf.New, 2000},
		{"Expansion", wf.Expansion, 500},
		{"Contraction", wf.Contraction, 400},
		{"Churned", wf.Churned, 1000},
		{"Reactivation", wf.Reactivation, 800},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.name, c.got, c.want)
		}
	}
	if !wf.HasStartHistory {
		t.Error("HasStartHistory = false, want true (a snapshot exists at the start)")
	}
	// GDR = (4000 - 400 - 1000)/4000 = 65%; NDR = (4000 + 500 - 400 - 1000)/4000 = 77.5%.
	if wf.GrossDollarRetention < 64.99 || wf.GrossDollarRetention > 65.01 {
		t.Errorf("GrossDollarRetention = %.2f, want 65.0", wf.GrossDollarRetention)
	}
	if wf.NetDollarRetention < 77.49 || wf.NetDollarRetention > 77.51 {
		t.Errorf("NetDollarRetention = %.2f, want 77.5", wf.NetDollarRetention)
	}
	// The waterfall identity must close.
	if id := wf.StartingMRR + wf.New + wf.Expansion + wf.Reactivation - wf.Contraction - wf.Churned; id != wf.EndingMRR {
		t.Errorf("identity broke: %d != EndingMRR %d", id, wf.EndingMRR)
	}
}

// TestGetMRRWaterfall_NoStartHistory: with no snapshot at/before the start,
// everything present at the end is New and HasStartHistory is false.
func TestGetMRRWaterfall_NoStartHistory(t *testing.T) {
	tenant := uuid.New()
	d2 := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	sub := uuid.New()
	store := &fakeMRRSnapshotStore{rows: []domain.MRRSnapshot{snap(tenant, sub, d2, 3000)}}

	svc := NewAnalyticsService(nil, nil, nil, nil)
	svc.SetFX(&mockFXForMRR{source: "live"}, nil, "USD")
	svc.SetSnapshotStore(store)

	wf, err := svc.GetMRRWaterfall(context.Background(), tenant, d2.AddDate(0, -1, 0), d2)
	if err != nil {
		t.Fatalf("GetMRRWaterfall: %v", err)
	}
	if wf.HasStartHistory {
		t.Error("HasStartHistory = true, want false (no snapshot before end)")
	}
	if wf.New != 3000 || wf.StartingMRR != 0 || wf.EndingMRR != 3000 {
		t.Errorf("got New=%d Starting=%d Ending=%d, want 3000/0/3000", wf.New, wf.StartingMRR, wf.EndingMRR)
	}
}

// TestCaptureMRRSnapshot writes one snapshot per active subscription, at the
// monthly-normalized amount (annual plan → /12).
func TestCaptureMRRSnapshot(t *testing.T) {
	annual := &domain.Plan{ID: uuid.New(), IntervalUnit: domain.IntervalYear, IntervalCount: 1,
		Prices: []domain.Price{{Currency: "USD", Amount: 12000}}} // → 1000/mo
	monthly := &domain.Plan{ID: uuid.New(), IntervalUnit: domain.IntervalMonth, IntervalCount: 1,
		Prices: []domain.Price{{Currency: "USD", Amount: 1000}}} // → 1000/mo
	subRepo, planRepo := mrrFixture(annual, monthly)
	store := &fakeMRRSnapshotStore{}

	svc := NewAnalyticsService(subRepo, nil, planRepo, nil)
	svc.SetFX(&mockFXForMRR{source: "live"}, nil, "USD")
	svc.SetSnapshotStore(store)

	n, err := svc.CaptureMRRSnapshot(context.Background(), uuid.New(), time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("CaptureMRRSnapshot: %v", err)
	}
	if n != 2 || len(store.rows) != 2 {
		t.Fatalf("captured %d snapshots (store has %d), want 2", n, len(store.rows))
	}
	var total int64
	for _, r := range store.rows {
		total += r.MRRAmount
		if r.SnapshotDate.Hour() != 0 {
			t.Errorf("snapshot date not normalized to day: %v", r.SnapshotDate)
		}
	}
	if total != 2000 {
		t.Errorf("captured MRR total = %d, want 2000 (both plans normalize to 1000/mo)", total)
	}
}

// --- invoice aging ---

type fakeInvoiceAgingStore struct{ rows []domain.InvoiceAgingRow }

func (f *fakeInvoiceAgingStore) GetInvoiceAgingRows(_ context.Context, _ uuid.UUID) ([]domain.InvoiceAgingRow, error) {
	return f.rows, nil
}

// TestGetInvoiceAging_NormalizesAndOrders: buckets come back all-present and
// ordered, foreign-currency amounts convert to the reporting currency, and the
// totals sum across buckets.
func TestGetInvoiceAging_NormalizesAndOrders(t *testing.T) {
	store := &fakeInvoiceAgingStore{rows: []domain.InvoiceAgingRow{
		{Currency: "USD", Bucket: "current", Count: 2, Amount: 10000},
		{Currency: "EUR", Bucket: "1-30", Count: 1, Amount: 9200}, // × 1.25 = 11500
		{Currency: "USD", Bucket: "90+", Count: 1, Amount: 5000},
	}}
	svc := NewAnalyticsService(nil, nil, nil, nil)
	svc.SetFX(&mockFXForMRR{rates: map[string]float64{"EUR:USD": 1.25}, source: "live"}, nil, "USD")
	svc.SetInvoiceAgingStore(store)

	rep, err := svc.GetInvoiceAging(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("GetInvoiceAging: %v", err)
	}
	order := []string{"current", "1-30", "31-60", "61-90", "90+"}
	if len(rep.Buckets) != len(order) {
		t.Fatalf("got %d buckets, want %d", len(rep.Buckets), len(order))
	}
	byLabel := map[string]domain.InvoiceAgingBucket{}
	for i, b := range rep.Buckets {
		if b.Label != order[i] {
			t.Errorf("bucket[%d] = %q, want %q", i, b.Label, order[i])
		}
		byLabel[b.Label] = b
	}
	if byLabel["current"].Amount != 10000 || byLabel["current"].Count != 2 {
		t.Errorf("current = %+v, want amt 10000 count 2", byLabel["current"])
	}
	if byLabel["1-30"].Amount != 11500 {
		t.Errorf("1-30 amount = %d, want 11500 (EUR 9200 × 1.25)", byLabel["1-30"].Amount)
	}
	if byLabel["31-60"].Amount != 0 || byLabel["61-90"].Amount != 0 {
		t.Errorf("empty buckets non-zero: 31-60=%d 61-90=%d", byLabel["31-60"].Amount, byLabel["61-90"].Amount)
	}
	if byLabel["90+"].Amount != 5000 {
		t.Errorf("90+ amount = %d, want 5000", byLabel["90+"].Amount)
	}
	if rep.TotalOutstanding != 26500 || rep.TotalCount != 4 {
		t.Errorf("totals = %d / %d, want 26500 / 4", rep.TotalOutstanding, rep.TotalCount)
	}
	if rep.ReportingCurrency != "USD" {
		t.Errorf("reporting currency = %q, want USD", rep.ReportingCurrency)
	}
}

// --- unit economics (ARPA / ARPU / LTV) ---

// TestGetUnitEconomics: 3 subscriptions across 2 customers → ARPA = MRR/2,
// ARPU = MRR/3. LTV needs history, so with no snapshot store it's unavailable.
func TestGetUnitEconomics(t *testing.T) {
	c1, c2 := uuid.New(), uuid.New()
	p1 := &domain.Plan{ID: uuid.New(), Prices: []domain.Price{{Currency: "USD", Amount: 1000}}}
	p2 := &domain.Plan{ID: uuid.New(), Prices: []domain.Price{{Currency: "USD", Amount: 2000}}}
	p3 := &domain.Plan{ID: uuid.New(), Prices: []domain.Price{{Currency: "USD", Amount: 3000}}}
	planRepo := &mockPlanRepoForMRR{plans: map[uuid.UUID]*domain.Plan{p1.ID: p1, p2.ID: p2, p3.ID: p3}}
	subRepo := &mockSubRepoForMRR{active: []*domain.Subscription{
		{ID: uuid.New(), CustomerID: c1, PlanID: p1.ID},
		{ID: uuid.New(), CustomerID: c1, PlanID: p2.ID},
		{ID: uuid.New(), CustomerID: c2, PlanID: p3.ID},
	}}

	svc := NewAnalyticsService(subRepo, nil, planRepo, nil)
	svc.SetFX(&mockFXForMRR{source: "live"}, nil, "USD")

	ue, err := svc.GetUnitEconomics(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("GetUnitEconomics: %v", err)
	}
	if ue.MRR != 6000 {
		t.Errorf("MRR = %d, want 6000", ue.MRR)
	}
	if ue.ActiveSubscriptions != 3 || ue.ActiveCustomers != 2 {
		t.Errorf("counts = %d subs / %d customers, want 3 / 2", ue.ActiveSubscriptions, ue.ActiveCustomers)
	}
	if ue.ARPA != 3000 {
		t.Errorf("ARPA = %d, want 3000 (6000/2)", ue.ARPA)
	}
	if ue.ARPU != 2000 {
		t.Errorf("ARPU = %d, want 2000 (6000/3)", ue.ARPU)
	}
	if ue.HasLTV || ue.LTV != 0 {
		t.Errorf("LTV should be unavailable with no history, got HasLTV=%v LTV=%d", ue.HasLTV, ue.LTV)
	}
}
