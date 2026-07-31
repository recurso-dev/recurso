package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// usageAmtSubRepo is a partial SubscriptionRepository returning one subscription
// (GetUsageAmount only calls GetByID).
type usageAmtSubRepo struct {
	port.SubscriptionRepository
	sub *domain.Subscription
}

func (r *usageAmtSubRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Subscription, error) {
	return r.sub, nil
}

// TestGetUsageAmount_ExcludesPayInAdvance proves the usage-amount preview does
// not count pay-in-advance charges. Those are billed per event at ingestion, not
// on the upcoming period-close invoice this preview projects, so including them
// double-represents usage already billed. Before the fix TotalAmount was 20000.
func TestGetUsageAmount_ExcludesPayInAdvance(t *testing.T) {
	tenantID := uuid.New()
	planID := uuid.New()
	subID := uuid.New()
	plan := &domain.Plan{ID: planID, TenantID: tenantID, Prices: []domain.Price{{Currency: "INR"}}}
	sub := &domain.Subscription{ID: subID, TenantID: tenantID, PlanID: planID, CurrentPeriodStart: time.Now().Add(-24 * time.Hour)}
	metric := domain.BillableMetric{ID: uuid.New(), Code: "api", Name: "API", AggregationType: domain.AggregationSum}

	arrears := domain.Charge{ID: uuid.New(), PlanID: planID, ChargeModel: domain.ChargePerUnit,
		Amounts: map[string]domain.ChargeAmounts{"INR": {UnitAmount: "1"}}, Metric: &metric}
	pia := domain.Charge{ID: uuid.New(), PlanID: planID, ChargeModel: domain.ChargePerUnit, PayInAdvance: true,
		Amounts: map[string]domain.ChargeAmounts{"INR": {UnitAmount: "1"}}, Metric: &metric}

	svc := NewMeteringService(
		nil,
		&mockChargeRepoForMeter{charges: []domain.Charge{arrears, pia}},
		&simPlanRepo{plan: plan},
		&usageAmtSubRepo{sub: sub},
		&mockUsageRepoForMeter{qtyByMetricCode: map[string]int64{"api": 100}},
	)

	out, err := svc.GetUsageAmount(context.Background(), tenantID, subID)
	if err != nil {
		t.Fatalf("GetUsageAmount: %v", err)
	}
	// Only the arrears charge (100 × ₹1 = 10000) should appear; the PIA charge is
	// excluded. Before the fix both were summed → 20000.
	if out.TotalAmount != 10000 {
		t.Errorf("TotalAmount = %d, want 10000 (pay-in-advance charge must be excluded from the preview)", out.TotalAmount)
	}
	if len(out.Charges) != 1 {
		t.Errorf("Charges len = %d, want 1 (arrears only)", len(out.Charges))
	}
}
