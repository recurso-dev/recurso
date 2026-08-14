package service

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

type mockRunStore struct {
	created []*domain.ReconciliationRun
	list    []domain.ReconciliationRun
}

func (m *mockRunStore) Create(_ context.Context, r *domain.ReconciliationRun) error {
	m.created = append(m.created, r)
	return nil
}
func (m *mockRunStore) ListByTenant(_ context.Context, _ uuid.UUID, _ int) ([]domain.ReconciliationRun, error) {
	return m.list, nil
}

func TestRecordRunMapsReportAndActor(t *testing.T) {
	store := &mockRunStore{}
	svc := NewReconciliationService(nil, nil)
	svc.SetRunStore(store)

	tenantID, actorID := uuid.New(), uuid.New()
	report := &ReconciliationReport{
		InvoicesChecked:    12,
		TotalDiscrepancies: 3,
		TBCompared:         true,
		TBAccountsChecked:  8,
	}
	if err := svc.RecordRun(context.Background(), tenantID, actorID, report); err != nil {
		t.Fatalf("RecordRun: %v", err)
	}
	if len(store.created) != 1 {
		t.Fatalf("expected 1 recorded run, got %d", len(store.created))
	}
	got := store.created[0]
	if got.TenantID != tenantID || got.TotalDiscrepancies != 3 || got.InvoicesChecked != 12 || !got.TBCompared {
		t.Fatalf("report not mapped onto the run: %+v", got)
	}
	if got.RunBy == nil || *got.RunBy != actorID {
		t.Fatalf("actor not recorded: %+v", got.RunBy)
	}
}

func TestRecordRunNilActorLeavesRunByNull(t *testing.T) {
	store := &mockRunStore{}
	svc := NewReconciliationService(nil, nil)
	svc.SetRunStore(store)
	if err := svc.RecordRun(context.Background(), uuid.New(), uuid.Nil, &ReconciliationReport{}); err != nil {
		t.Fatalf("RecordRun: %v", err)
	}
	if store.created[0].RunBy != nil {
		t.Fatalf("expected null run_by for a system run")
	}
}

func TestRunStoreIsNilSafe(t *testing.T) {
	svc := NewReconciliationService(nil, nil) // no run store wired
	if err := svc.RecordRun(context.Background(), uuid.New(), uuid.New(), &ReconciliationReport{}); err != nil {
		t.Fatalf("RecordRun without a store should be a no-op, got %v", err)
	}
	runs, err := svc.ListRuns(context.Background(), uuid.New(), 10)
	if err != nil {
		t.Fatalf("ListRuns without a store: %v", err)
	}
	if runs == nil || len(runs) != 0 {
		t.Fatalf("expected an empty (non-nil) run list, got %#v", runs)
	}
}

func TestListRunsDelegatesToStore(t *testing.T) {
	store := &mockRunStore{list: []domain.ReconciliationRun{{TotalDiscrepancies: 0}, {TotalDiscrepancies: 5}}}
	svc := NewReconciliationService(nil, nil)
	svc.SetRunStore(store)
	runs, err := svc.ListRuns(context.Background(), uuid.New(), 10)
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if len(runs) != 2 || !runs[0].Balanced() || runs[1].Balanced() {
		t.Fatalf("unexpected runs: %+v", runs)
	}
}
