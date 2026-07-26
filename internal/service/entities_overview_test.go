package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// stubOverviewInvoiceRepo supplies the open-AR-by-entity rows the overview needs;
// everything else on the port is unused here.
type stubOverviewInvoiceRepo struct {
	port.InvoiceRepository
	rows []domain.EntityOutstandingRow
}

func (s *stubOverviewInvoiceRepo) GetOutstandingByEntity(_ context.Context, _ uuid.UUID) ([]domain.EntityOutstandingRow, error) {
	return s.rows, nil
}

// The overview merges per-entity MRR with open AR: each entity carries its own
// MRR + receivables, a NULL-entity AR row rolls up to the primary, totals sum,
// and rows stay sorted by MRR descending (inherited from GetMRRByEntity).
func TestGetEntitiesOverview(t *testing.T) {
	primaryID := uuid.New()
	entityB := uuid.New()

	planA := mrrPlan("USD", 1000) // primary
	planB := mrrPlan("USD", 3000) // entity B
	planRepo := &mockPlanRepoForMRR{plans: map[uuid.UUID]*domain.Plan{planA.ID: planA, planB.ID: planB}}
	subRepo := &mockSubRepoForMRR{active: []*domain.Subscription{
		{ID: uuid.New(), PlanID: planA.ID, EntityID: nil},
		{ID: uuid.New(), PlanID: planB.ID, EntityID: &entityB},
	}}
	invRepo := &stubOverviewInvoiceRepo{rows: []domain.EntityOutstandingRow{
		{EntityID: &primaryID, Currency: "USD", Amount: 5000},
		{EntityID: &entityB, Currency: "USD", Amount: 12000},
		{EntityID: nil, Currency: "USD", Amount: 2000}, // NULL → primary
	}}

	svc := NewAnalyticsService(subRepo, invRepo, planRepo, nil)
	svc.SetFX(&mockFXForMRR{source: "live", asOf: time.Now()}, nil, "USD")
	svc.SetEntityReader(&fakeAnalyticsEntityReader{
		primary: &domain.Entity{ID: primaryID, IsPrimary: true},
		all: []*domain.Entity{
			{ID: primaryID, Name: "HQ", IsPrimary: true},
			{ID: entityB, Name: "Entity B"},
		},
	})

	ov, err := svc.GetEntitiesOverview(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("GetEntitiesOverview: %v", err)
	}
	if ov.TotalMRR != 4000 {
		t.Errorf("total MRR = %d, want 4000", ov.TotalMRR)
	}
	if ov.TotalAROutstanding != 19000 { // 5000 + 2000 (primary) + 12000 (B)
		t.Errorf("total AR = %d, want 19000", ov.TotalAROutstanding)
	}
	if len(ov.Entities) != 2 {
		t.Fatalf("expected 2 entities, got %d", len(ov.Entities))
	}
	// Sorted by MRR desc → Entity B first.
	if ov.Entities[0].EntityID != entityB || ov.Entities[0].MRR != 3000 || ov.Entities[0].AROutstanding != 12000 {
		t.Errorf("rank 0 = %+v, want Entity B mrr 3000 ar 12000", ov.Entities[0])
	}
	// Primary: MRR 1000, AR 5000 + 2000 (NULL rolled up) = 7000.
	if ov.Entities[1].EntityID != primaryID || ov.Entities[1].MRR != 1000 || ov.Entities[1].AROutstanding != 7000 {
		t.Errorf("rank 1 = %+v, want HQ mrr 1000 ar 7000", ov.Entities[1])
	}
	if !ov.Entities[1].IsPrimary {
		t.Error("HQ should be flagged primary")
	}
}

// No entity reader → empty overview (single-entity tenants use the consolidated
// dashboards), never an error.
func TestGetEntitiesOverview_NoReader(t *testing.T) {
	subRepo, planRepo := mrrFixture(mrrPlan("USD", 1000), mrrPlan("USD", 2000))
	invRepo := &stubOverviewInvoiceRepo{}
	svc := NewAnalyticsService(subRepo, invRepo, planRepo, nil)
	svc.SetFX(&mockFXForMRR{source: "live", asOf: time.Now()}, nil, "USD")

	ov, err := svc.GetEntitiesOverview(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("GetEntitiesOverview: %v", err)
	}
	if len(ov.Entities) != 0 {
		t.Errorf("expected empty overview without an entity reader, got %d", len(ov.Entities))
	}
}
