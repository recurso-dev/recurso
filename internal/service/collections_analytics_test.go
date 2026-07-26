package service

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// stubCollectionsAgg feeds the funnel + failure breakdown fixed invoice-side rows.
type stubCollectionsAgg struct {
	atRisk   []domain.CollectionsAtRiskRow
	failures []domain.CollectionsFailureRow
}

func (s *stubCollectionsAgg) GetCollectionsAtRisk(_ context.Context, _ uuid.UUID) ([]domain.CollectionsAtRiskRow, error) {
	return s.atRisk, nil
}
func (s *stubCollectionsAgg) GetCollectionsFailureBreakdown(_ context.Context, _ uuid.UUID) ([]domain.CollectionsFailureRow, error) {
	return s.failures, nil
}

// The funnel sums past_due vs uncollectible from the invoice side, recovered from
// the recovered-payments repo, and computes recovery rate over concluded cases
// (recovered vs written-off) — in-flight past_due does not dilute the rate.
func TestGetCollectionsFunnel(t *testing.T) {
	repo := newMockRecoveredPaymentRepo()
	repo.totals = &domain.RecoveryTotals{
		RecoveredAmountTotal: map[string]int64{"USD": 30000},
		RecoveredCount:       6,
	}
	svc := NewDunningRecoveryService(repo, "")
	svc.SetCollectionsAggregator(&stubCollectionsAgg{
		atRisk: []domain.CollectionsAtRiskRow{
			{Status: "past_due", Currency: "USD", Count: 4, Amount: 40000},
			{Status: "uncollectible", Currency: "USD", Count: 2, Amount: 15000},
		},
	})

	funnel, err := svc.GetCollectionsFunnel(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("GetCollectionsFunnel: %v", err)
	}
	if funnel.PastDue.Count != 4 || funnel.PastDue.Amount != 40000 {
		t.Errorf("past_due = %+v, want {4, 40000}", funnel.PastDue)
	}
	if funnel.Uncollectible.Count != 2 || funnel.Uncollectible.Amount != 15000 {
		t.Errorf("uncollectible = %+v, want {2, 15000}", funnel.Uncollectible)
	}
	if funnel.Recovered.Count != 6 || funnel.Recovered.Amount != 30000 {
		t.Errorf("recovered = %+v, want {6, 30000}", funnel.Recovered)
	}
	// recovered 6 / (recovered 6 + uncollectible 2) = 0.75
	if funnel.RecoveryRate < 0.749 || funnel.RecoveryRate > 0.751 {
		t.Errorf("recovery_rate = %f, want 0.75", funnel.RecoveryRate)
	}
	if funnel.ReportingCurrency != "USD" {
		t.Errorf("reporting currency = %q, want USD", funnel.ReportingCurrency)
	}
}

// With no concluded cases (nothing recovered, nothing written off), the rate is
// 0 rather than a divide-by-zero.
func TestGetCollectionsFunnel_NoConcludedCases(t *testing.T) {
	svc := NewDunningRecoveryService(newMockRecoveredPaymentRepo(), "")
	svc.SetCollectionsAggregator(&stubCollectionsAgg{
		atRisk: []domain.CollectionsAtRiskRow{{Status: "past_due", Currency: "USD", Count: 3, Amount: 9000}},
	})
	funnel, err := svc.GetCollectionsFunnel(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("GetCollectionsFunnel: %v", err)
	}
	if funnel.RecoveryRate != 0 {
		t.Errorf("recovery_rate = %f, want 0 (no concluded cases)", funnel.RecoveryRate)
	}
}

// A nil aggregator (not wired) yields an empty, valid funnel — never an error.
func TestGetCollectionsFunnel_NilAggregator(t *testing.T) {
	svc := NewDunningRecoveryService(newMockRecoveredPaymentRepo(), "")
	funnel, err := svc.GetCollectionsFunnel(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("GetCollectionsFunnel: %v", err)
	}
	if funnel.PastDue.Count != 0 || funnel.Recovered.Count != 0 {
		t.Errorf("expected empty funnel, got %+v", funnel)
	}
}

// The failure breakdown collapses per-currency rows by code and ranks by money
// at risk, descending.
func TestGetFailureBreakdown_RanksByAmount(t *testing.T) {
	svc := NewDunningRecoveryService(newMockRecoveredPaymentRepo(), "")
	svc.SetCollectionsAggregator(&stubCollectionsAgg{
		failures: []domain.CollectionsFailureRow{
			{ErrorCode: "insufficient_funds", Currency: "USD", Count: 2, Amount: 5000},
			{ErrorCode: "card_declined", Currency: "USD", Count: 5, Amount: 20000},
			{ErrorCode: "insufficient_funds", Currency: "USD", Count: 1, Amount: 3000},
		},
	})

	buckets, err := svc.GetFailureBreakdown(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("GetFailureBreakdown: %v", err)
	}
	if len(buckets) != 2 {
		t.Fatalf("expected 2 codes, got %d: %+v", len(buckets), buckets)
	}
	// card_declined (20000) outranks insufficient_funds (5000+3000=8000).
	if buckets[0].ErrorCode != "card_declined" || buckets[0].AmountAtRisk != 20000 {
		t.Errorf("rank 0 = %+v, want card_declined/20000", buckets[0])
	}
	if buckets[1].ErrorCode != "insufficient_funds" || buckets[1].AmountAtRisk != 8000 || buckets[1].Count != 3 {
		t.Errorf("rank 1 = %+v, want insufficient_funds/8000/3", buckets[1])
	}
}
