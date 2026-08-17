package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

func TestComputeCloudCharge(t *testing.T) {
	cases := []struct {
		name             string
		tracked, collect int64
		wantCharge       int64
		wantReasonHas    string
	}{
		{"under free tier → $0", 8_000_00, 7_000_00, 0, "free tier"},
		{"exactly $10k tracked → still free", 10_000_00, 50_000_00, 0, "free tier"},
		{"over tier, 0.4% wins", 18_000_00, 15_000_00, 60_00, "0.4%"},  // 0.4% of $15,000 = $60
		{"over tier, cap wins", 40_000_00, 30_000_00, 99_00, "cap"},    // 0.4% of $30,000 = $120 → capped $99
		{"over tier, exactly at cap", 20_000_00, 24_750_00, 99_00, ""}, // 0.4% of $24,750 = $99 (boundary, not over)
		{"over tier but zero collected → $0", 12_000_00, 0, 0, "0.4%"}, // over the free tier, but nothing collected
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, reason := domain.ComputeCloudCharge(c.tracked, c.collect)
			if got != c.wantCharge {
				t.Fatalf("charge = %d, want %d (reason %q)", got, c.wantCharge, reason)
			}
			if c.wantReasonHas != "" && !contains(reason, c.wantReasonHas) {
				t.Fatalf("reason = %q, want to contain %q", reason, c.wantReasonHas)
			}
		})
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// --- preview service fakes ---

type fakeUsageReader struct{ rows []*domain.CloudTenantUsage }

func (f *fakeUsageReader) ListByPeriod(_ context.Context, _ time.Time) ([]*domain.CloudTenantUsage, error) {
	return f.rows, nil
}

type fakeChargeRepo struct{ saved []*domain.CloudChargePreview }

func (f *fakeChargeRepo) UpsertPreview(_ context.Context, rows []*domain.CloudChargePreview) error {
	f.saved = rows
	return nil
}

func TestPreviewPeriod_AppliesPricingPerTenant(t *testing.T) {
	tA, tB := uuid.New(), uuid.New()
	usage := &fakeUsageReader{rows: []*domain.CloudTenantUsage{
		// Tenant A (USD): over the free tier, 0.4% of $30k = $120 → capped $99.
		{TenantID: tA, Currency: "USD", TrackedRevenueMinor: 40_000_00, CollectedVolumeMinor: 30_000_00},
		// Tenant B (USD): under the free tier → $0.
		{TenantID: tB, Currency: "USD", TrackedRevenueMinor: 5_000_00, CollectedVolumeMinor: 4_000_00},
	}}
	repo := &fakeChargeRepo{}
	svc := NewCloudChargeService(usage, repo, nil) // no FX; all USD

	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	previews, err := svc.PreviewPeriod(context.Background(), start, end)
	if err != nil {
		t.Fatalf("PreviewPeriod: %v", err)
	}
	if len(previews) != 2 || len(repo.saved) != 2 {
		t.Fatalf("expected 2 previews stored, got %d/%d", len(previews), len(repo.saved))
	}
	byTenant := map[uuid.UUID]*domain.CloudChargePreview{}
	for _, p := range previews {
		byTenant[p.TenantID] = p
		if p.Currency != "USD" || p.PeriodStart != start {
			t.Fatalf("preview meta wrong: %+v", p)
		}
	}
	if byTenant[tA].WouldChargeMinor != 99_00 {
		t.Fatalf("tenant A should be capped at $99, got %d", byTenant[tA].WouldChargeMinor)
	}
	if byTenant[tB].WouldChargeMinor != 0 {
		t.Fatalf("tenant B is under the free tier, should be $0, got %d", byTenant[tB].WouldChargeMinor)
	}
}

// fakeFX converts INR→USD at a fixed 1/80 rate. Implements the full
// port.ExchangeRateProvider; the normalizer only calls GetRate, so Convert and
// ListRates are unused stubs here.
type fakeFX struct{}

func (fakeFX) GetRate(_ context.Context, from, to string) (float64, error) {
	if from == to {
		return 1, nil
	}
	if from == "INR" && to == "USD" {
		return 1.0 / 80.0, nil
	}
	return 0, nil
}
func (fakeFX) Convert(_ context.Context, amount int64, _, _ string) (int64, float64, error) {
	return amount, 1, nil
}
func (fakeFX) ListRates(_ context.Context, _ string) ([]port.ExchangeRate, error) { return nil, nil }

func TestPreviewPeriod_NormalizesForeignCurrencyViaFX(t *testing.T) {
	tenant := uuid.New()
	usage := &fakeUsageReader{rows: []*domain.CloudTenantUsage{
		// Tracked ₹960,000 (96_000_000 minor) /80 = $12,000 → over the $10k free tier.
		// Collected ₹8,000,000 (800_000_000 minor) /80 = $100,000 → 0.4% = $400 → capped $99.
		{TenantID: tenant, Currency: "INR", TrackedRevenueMinor: 96_000_000, CollectedVolumeMinor: 800_000_000},
	}}
	repo := &fakeChargeRepo{}
	svc := NewCloudChargeService(usage, repo, nil)
	svc.SetFX(fakeFX{}, nil, "USD")

	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	previews, err := svc.PreviewPeriod(context.Background(), start, start.AddDate(0, 1, 0))
	if err != nil {
		t.Fatalf("PreviewPeriod: %v", err)
	}
	if len(previews) != 1 {
		t.Fatalf("expected 1 preview, got %d", len(previews))
	}
	p := previews[0]
	if p.Currency != "USD" {
		t.Fatalf("preview should be in USD, got %s", p.Currency)
	}
	// $12,000 tracked (> free tier), $100,000 collected → 0.4% = $400 → capped $99.
	if p.WouldChargeMinor != 99_00 {
		t.Fatalf("expected $99 cap after FX, got %d (tracked=%d collected=%d)", p.WouldChargeMinor, p.TrackedRevenueMinor, p.CollectedVolumeMinor)
	}
}
