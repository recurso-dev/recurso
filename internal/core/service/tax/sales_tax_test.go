package tax

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/recurso-dev/recurso/internal/core/port"
)

// fakeSalesTaxProvider counts lookups and returns a fixed result or error.
type fakeSalesTaxProvider struct {
	calls  int
	lastQ  *SalesTaxQuery
	result *SalesTaxResult
	err    error
}

func (f *fakeSalesTaxProvider) Name() string { return "faketax" }

func (f *fakeSalesTaxProvider) LookupSalesTax(ctx context.Context, q *SalesTaxQuery) (*SalesTaxResult, error) {
	f.calls++
	f.lastQ = q
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

func usRequest(amount int64) *port.TaxRequest {
	return &port.TaxRequest{
		Amount:       amount,
		Currency:     "USD",
		BuyerState:   "CA",
		BuyerZip:     "90002",
		BuyerCountry: "US",
	}
}

func TestUSSalesTax_NoProvider_StubBehavior(t *testing.T) {
	e := NewUSSalesTaxEngine("CA")

	calc, err := e.CalculateTax(context.Background(), usRequest(10000))
	if err != nil {
		t.Fatalf("CalculateTax: %v", err)
	}
	if calc.TotalTax != 0 || calc.TaxRate != 0 {
		t.Errorf("stub tax = %d @ %v, want 0 @ 0", calc.TotalTax, calc.TaxRate)
	}
	if calc.TaxType != "sales_tax_stub" {
		t.Errorf("TaxType = %q, want 'sales_tax_stub'", calc.TaxType)
	}
	if e.HasProvider() {
		t.Error("HasProvider() = true for stub engine")
	}

	rate, err := e.GetApplicableRate(context.Background(), usRequest(10000))
	if err != nil || rate != 0 {
		t.Errorf("GetApplicableRate = %v, %v; want 0, nil", rate, err)
	}
}

func TestUSSalesTax_WithProvider_RealRate(t *testing.T) {
	fake := &fakeSalesTaxProvider{result: &SalesTaxResult{
		Rate:         0.0865,
		TaxAmount:    865,
		Jurisdiction: "US/CA/LOS ANGELES",
		HasNexus:     true,
	}}
	e := NewUSSalesTaxEngineWithProvider("ca", fake)

	calc, err := e.CalculateTax(context.Background(), usRequest(10000))
	if err != nil {
		t.Fatalf("CalculateTax: %v", err)
	}
	if calc.TotalTax != 865 {
		t.Errorf("TotalTax = %d, want 865", calc.TotalTax)
	}
	if calc.TaxRate != 0.0865 {
		t.Errorf("TaxRate = %v, want 0.0865", calc.TaxRate)
	}
	if calc.TaxType != "sales_tax" {
		t.Errorf("TaxType = %q, want 'sales_tax' (live provider)", calc.TaxType)
	}
	if calc.Note == "" || !contains(calc.Note, "faketax") {
		t.Errorf("Note = %q, want provider name in note", calc.Note)
	}

	// Query mapping: buyer state/zip from the TaxRequest, seller state from
	// the engine, from-country pinned to US.
	q := fake.lastQ
	if q.ToState != "CA" || q.ToZip != "90002" || q.ToCountry != "US" {
		t.Errorf("to = %s/%s/%s, want US/CA/90002", q.ToCountry, q.ToState, q.ToZip)
	}
	if q.FromState != "CA" || q.FromCountry != "US" {
		t.Errorf("from = %s/%s, want US/CA", q.FromCountry, q.FromState)
	}
	if q.Amount != 10000 {
		t.Errorf("query amount = %d, want 10000", q.Amount)
	}
}

func TestUSSalesTax_ProviderError_Surfaced(t *testing.T) {
	fake := &fakeSalesTaxProvider{err: errors.New("boom")}
	e := NewUSSalesTaxEngineWithProvider("CA", fake)

	calc, err := e.CalculateTax(context.Background(), usRequest(10000))
	if err == nil {
		t.Fatal("expected error from provider-backed engine, got nil")
	}
	if calc != nil {
		t.Errorf("calc = %+v, want nil on error", calc)
	}
	if !contains(err.Error(), "faketax") {
		t.Errorf("error %q should name the provider", err)
	}
}

func TestUSSalesTax_NoNexus_NotedHonestly(t *testing.T) {
	fake := &fakeSalesTaxProvider{result: &SalesTaxResult{Rate: 0, TaxAmount: 0, HasNexus: false}}
	e := NewUSSalesTaxEngineWithProvider("CA", fake)

	calc, err := e.CalculateTax(context.Background(), usRequest(10000))
	if err != nil {
		t.Fatalf("CalculateTax: %v", err)
	}
	if calc.TotalTax != 0 {
		t.Errorf("TotalTax = %d, want 0 without nexus", calc.TotalTax)
	}
	if !contains(calc.Note, "nexus") {
		t.Errorf("Note = %q, want a no-nexus explanation", calc.Note)
	}
}

func TestCachedSalesTaxProvider_SecondLookupServedFromCache(t *testing.T) {
	fake := &fakeSalesTaxProvider{result: &SalesTaxResult{
		Rate: 0.10, TaxAmount: 1000, Jurisdiction: "US/CA", HasNexus: true,
	}}
	cached := NewCachedSalesTaxProvider(fake, DefaultSalesTaxRateTTL)

	q1 := &SalesTaxQuery{ToState: "CA", ToZip: "90002", Amount: 10000, ToCountry: "US"}
	if _, err := cached.LookupSalesTax(context.Background(), q1); err != nil {
		t.Fatalf("first lookup: %v", err)
	}

	// Same (state, zip), different amount: no second provider call, tax
	// recomputed locally from the cached rate.
	q2 := &SalesTaxQuery{ToState: "CA", ToZip: "90002", Amount: 5000, ToCountry: "US"}
	res, err := cached.LookupSalesTax(context.Background(), q2)
	if err != nil {
		t.Fatalf("second lookup: %v", err)
	}
	if fake.calls != 1 {
		t.Errorf("provider calls = %d, want 1 (cache hit)", fake.calls)
	}
	if res.TaxAmount != 500 {
		t.Errorf("cached TaxAmount = %d, want 500 (5000 * 0.10)", res.TaxAmount)
	}
	if res.Rate != 0.10 || res.Jurisdiction != "US/CA" || !res.HasNexus {
		t.Errorf("cached result = %+v, want rate/jurisdiction/nexus preserved", res)
	}

	// A different zip misses the cache.
	q3 := &SalesTaxQuery{ToState: "CA", ToZip: "94105", Amount: 10000, ToCountry: "US"}
	if _, err := cached.LookupSalesTax(context.Background(), q3); err != nil {
		t.Fatalf("third lookup: %v", err)
	}
	if fake.calls != 2 {
		t.Errorf("provider calls = %d, want 2 (different zip)", fake.calls)
	}
}

func TestCachedSalesTaxProvider_EvictsExpiredEntries(t *testing.T) {
	fake := &fakeSalesTaxProvider{result: &SalesTaxResult{Rate: 0.05, HasNexus: true}}
	cached := NewCachedSalesTaxProvider(fake, time.Hour)
	now := time.Now()
	cached.now = func() time.Time { return now }

	// Populate five distinct jurisdictions.
	for _, zip := range []string{"10001", "10002", "10003", "10004", "10005"} {
		_, _ = cached.LookupSalesTax(context.Background(), &SalesTaxQuery{ToState: "NY", ToZip: zip, Amount: 10000, ToCountry: "US"})
	}
	if got := len(cached.entries); got != 5 {
		t.Fatalf("cached entries = %d, want 5", got)
	}

	// Advance past the TTL so all five are stale; the next write (a fresh
	// lookup) must sweep them instead of growing the map to six.
	now = now.Add(2 * time.Hour)
	_, _ = cached.LookupSalesTax(context.Background(), &SalesTaxQuery{ToState: "NY", ToZip: "20001", Amount: 10000, ToCountry: "US"})
	if got := len(cached.entries); got != 1 {
		t.Errorf("cached entries after eviction = %d, want 1 (5 stale swept + 1 fresh)", got)
	}
}

func TestCachedSalesTaxProvider_TTLExpiryRefetches(t *testing.T) {
	fake := &fakeSalesTaxProvider{result: &SalesTaxResult{Rate: 0.08, TaxAmount: 800, HasNexus: true}}
	cached := NewCachedSalesTaxProvider(fake, time.Hour)

	now := time.Now()
	cached.now = func() time.Time { return now }

	q := &SalesTaxQuery{ToState: "NY", ToZip: "10001", Amount: 10000, ToCountry: "US"}
	_, _ = cached.LookupSalesTax(context.Background(), q)
	_, _ = cached.LookupSalesTax(context.Background(), q)
	if fake.calls != 1 {
		t.Fatalf("provider calls = %d, want 1 before expiry", fake.calls)
	}

	now = now.Add(2 * time.Hour) // past the 1h TTL
	_, _ = cached.LookupSalesTax(context.Background(), q)
	if fake.calls != 2 {
		t.Errorf("provider calls = %d, want 2 after TTL expiry", fake.calls)
	}
}

func TestCachedSalesTaxProvider_ErrorsNotCached(t *testing.T) {
	fake := &fakeSalesTaxProvider{err: errors.New("down")}
	cached := NewCachedSalesTaxProvider(fake, time.Hour)

	q := &SalesTaxQuery{ToState: "TX", ToZip: "75001", Amount: 10000, ToCountry: "US"}
	if _, err := cached.LookupSalesTax(context.Background(), q); err == nil {
		t.Fatal("expected error passthrough")
	}

	// Provider recovers: the next call must reach it (errors were not cached).
	fake.err = nil
	fake.result = &SalesTaxResult{Rate: 0.0825, TaxAmount: 825, HasNexus: true}
	res, err := cached.LookupSalesTax(context.Background(), q)
	if err != nil || res.Rate != 0.0825 {
		t.Errorf("recovered lookup = %+v, %v; want rate 0.0825, nil", res, err)
	}
	if fake.calls != 2 {
		t.Errorf("provider calls = %d, want 2", fake.calls)
	}
}

func TestFactory_USWithProvider_WiresEngine(t *testing.T) {
	fake := &fakeSalesTaxProvider{result: &SalesTaxResult{Rate: 0.05, TaxAmount: 500, HasNexus: true}}

	engine := NewTaxEngineWithSalesTaxProvider("US", "CA", fake)
	us, ok := engine.(*USSalesTaxEngine)
	if !ok {
		t.Fatalf("engine = %T, want *USSalesTaxEngine", engine)
	}
	if !us.HasProvider() || us.ProviderName() != "faketax" {
		t.Errorf("provider not wired through factory: has=%v name=%q", us.HasProvider(), us.ProviderName())
	}

	// The provider must not leak into non-US engines.
	if _, ok := NewTaxEngineWithSalesTaxProvider("IN", "TN", fake).(*GSTEngine); !ok {
		t.Error("IN seller must still get the GST engine")
	}

	// Legacy constructor stays the stub.
	if NewTaxEngine("US", "CA").(*USSalesTaxEngine).HasProvider() {
		t.Error("NewTaxEngine must not wire a provider")
	}
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }

// TestUSSalesTax_ExemptPassedThroughAndNoted proves D2's core: an exempt request
// is passed THROUGH to the provider (not short-circuited), the provider's zero
// tax is used, the type stays "sales_tax" (so the sale still counts for
// reporting/nexus), and the note records the exemption.
func TestUSSalesTax_ExemptPassedThroughAndNoted(t *testing.T) {
	fake := &fakeSalesTaxProvider{result: &SalesTaxResult{Rate: 0, TaxAmount: 0, Jurisdiction: "US/CA", HasNexus: true}}
	e := NewUSSalesTaxEngineWithProvider("CA", fake)

	req := usRequest(10000)
	req.TaxExempt = true
	req.TaxExemptionNumber = "RESALE-123"
	req.TaxExemptionCode = "A"

	calc, err := e.CalculateTax(context.Background(), req)
	if err != nil {
		t.Fatalf("CalculateTax: %v", err)
	}
	if fake.calls != 1 {
		t.Fatalf("provider calls = %d, want 1 (must call through, not short-circuit)", fake.calls)
	}
	if fake.lastQ == nil || !fake.lastQ.Exempt || fake.lastQ.ExemptionNo != "RESALE-123" || fake.lastQ.EntityUseCode != "A" {
		t.Fatalf("exemption not passed to provider query: %+v", fake.lastQ)
	}
	if calc.TotalTax != 0 {
		t.Errorf("exempt tax = %d, want 0", calc.TotalTax)
	}
	if calc.TaxType != "sales_tax_exempt" {
		t.Errorf("TaxType = %q, want sales_tax_exempt (distinct, for audit)", calc.TaxType)
	}
	if !strings.Contains(calc.Note, "exempt sale") || !strings.Contains(calc.Note, "RESALE-123") || !strings.Contains(calc.Note, "code A") {
		t.Errorf("note missing exemption detail: %q", calc.Note)
	}
}

// TestCachedSalesTaxProvider_ExemptBypassesCache proves an exempt (buyer-specific)
// lookup never reads from nor writes to the shared (state, zip) rate cache — so an
// exempt 0-rate can't leak to other buyers, and a cached jurisdiction rate is
// never applied to an exempt buyer.
func TestCachedSalesTaxProvider_ExemptBypassesCache(t *testing.T) {
	fake := &fakeSalesTaxProvider{result: &SalesTaxResult{Rate: 0.10, TaxAmount: 1000, Jurisdiction: "US/CA", HasNexus: true}}
	cached := NewCachedSalesTaxProvider(fake, DefaultSalesTaxRateTTL)
	ctx := context.Background()
	ex := func() *SalesTaxQuery {
		return &SalesTaxQuery{ToState: "CA", ToZip: "90002", Amount: 10000, ToCountry: "US", Exempt: true, EntityUseCode: "A"}
	}
	nonex := func() *SalesTaxQuery {
		return &SalesTaxQuery{ToState: "CA", ToZip: "90002", Amount: 10000, ToCountry: "US"}
	}

	// Two exempt lookups both hit the provider (never cached).
	_, _ = cached.LookupSalesTax(ctx, ex())
	_, _ = cached.LookupSalesTax(ctx, ex())
	if fake.calls != 2 {
		t.Fatalf("exempt lookups were cached: calls=%d, want 2", fake.calls)
	}

	// The exempt lookups did not populate the cache: a non-exempt lookup at the
	// same (state, zip) still calls the provider...
	_, _ = cached.LookupSalesTax(ctx, nonex())
	if fake.calls != 3 {
		t.Fatalf("non-exempt after exempt should miss cache: calls=%d, want 3", fake.calls)
	}
	// ...and now caches, so a repeat non-exempt is served from cache.
	_, _ = cached.LookupSalesTax(ctx, nonex())
	if fake.calls != 3 {
		t.Fatalf("non-exempt should be cached: calls=%d, want 3", fake.calls)
	}

	// But an exempt lookup is never served from that cached rate.
	_, _ = cached.LookupSalesTax(ctx, ex())
	if fake.calls != 4 {
		t.Fatalf("exempt served from cache: calls=%d, want 4", fake.calls)
	}
}

// The (state, zip) rate cache must not serve a with-nexus result to a
// without-nexus query (or vice versa): asserted nexus changes the answer.
func TestSalesTaxCacheKey_NexusDistinct(t *testing.T) {
	base := &SalesTaxQuery{ToState: "TX", ToZip: "78701"}
	withNexus := &SalesTaxQuery{ToState: "TX", ToZip: "78701", NexusStates: []string{"TX"}}
	if salesTaxCacheKey(base) == salesTaxCacheKey(withNexus) {
		t.Fatal("cache key must differ when nexus is asserted")
	}
	// Order-insensitive: the same declared set is the same key.
	a := &SalesTaxQuery{ToState: "TX", ToZip: "78701", NexusStates: []string{"CA", "TX"}}
	b := &SalesTaxQuery{ToState: "TX", ToZip: "78701", NexusStates: []string{"TX", "CA"}}
	if salesTaxCacheKey(a) != salesTaxCacheKey(b) {
		t.Fatal("cache key must be order-insensitive over nexus states")
	}
}

// TestSalesTaxCacheKey_SellerOriginDistinct proves the seller's origin
// (from-state/zip) is part of the cache key, so two tenants selling into the
// SAME destination from DIFFERENT seller states — which matters for
// origin-sourced intra-state sales — never serve each other's cached rate.
func TestSalesTaxCacheKey_SellerOriginDistinct(t *testing.T) {
	caSeller := &SalesTaxQuery{FromState: "CA", FromZip: "90001", ToState: "CA", ToZip: "90001"}
	txSeller := &SalesTaxQuery{FromState: "TX", FromZip: "73301", ToState: "CA", ToZip: "90001"}
	if salesTaxCacheKey(caSeller) == salesTaxCacheKey(txSeller) {
		t.Fatal("cache key must differ by seller origin state (origin-sourced sales price on the seller's location)")
	}
	// Same seller + same destination → same key (still cacheable).
	again := &SalesTaxQuery{FromState: "CA", FromZip: "90001", ToState: "CA", ToZip: "90001"}
	if salesTaxCacheKey(caSeller) != salesTaxCacheKey(again) {
		t.Fatal("identical seller+destination must share a cache key")
	}
}

// TestSalesTaxCacheKey_StreetDistinct is the correctness reason the street line
// belongs in the key. A US ZIP can span several tax jurisdictions, so once a
// provider resolves to the rooftop, two addresses in the same ZIP can carry
// genuinely different rates. Keying on ZIP alone would serve one buyer's rate
// to the other.
func TestSalesTaxCacheKey_StreetDistinct(t *testing.T) {
	a := &SalesTaxQuery{ToState: "CA", ToZip: "92618", ToStreet: "200 Spectrum Center Dr"}
	b := &SalesTaxQuery{ToState: "CA", ToZip: "92618", ToStreet: "1 Civic Center Plaza"}
	if salesTaxCacheKey(a) == salesTaxCacheKey(b) {
		t.Fatal("two streets in one ZIP must not share a cache key")
	}

	// The same street is still cacheable, insensitively to case and spacing.
	same := &SalesTaxQuery{ToState: "CA", ToZip: "92618", ToStreet: "  200  spectrum center dr "}
	if salesTaxCacheKey(a) != salesTaxCacheKey(same) {
		t.Fatal("the same street must share a cache key regardless of case/spacing")
	}
}

// TestSalesTaxCacheKey_EmptyStreetUnchangedFromZipOnly pins that adding the
// street field did not change behaviour for ZIP-only deployments: a query with
// no street still collides with an identical one, so caching is exactly as
// before.
func TestSalesTaxCacheKey_EmptyStreetUnchangedFromZipOnly(t *testing.T) {
	a := &SalesTaxQuery{FromState: "CA", FromZip: "90001", ToState: "CA", ToZip: "92618"}
	b := &SalesTaxQuery{FromState: "CA", FromZip: "90001", ToState: "CA", ToZip: "92618"}
	if salesTaxCacheKey(a) != salesTaxCacheKey(b) {
		t.Fatal("ZIP-only queries must still share a cache key")
	}
	withStreet := &SalesTaxQuery{FromState: "CA", FromZip: "90001", ToState: "CA", ToZip: "92618", ToStreet: "200 Spectrum Center Dr"}
	if salesTaxCacheKey(a) == salesTaxCacheKey(withStreet) {
		t.Fatal("a street-qualified query must not reuse the ZIP-level cache entry")
	}
}
