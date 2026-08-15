package service

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

type mockRunStore struct {
	created         []*domain.ReconciliationRun
	createdDiscreps [][]domain.ReconciliationRunDiscrepancy
	list            []domain.ReconciliationRun
	detail          *domain.ReconciliationRunDetail
}

func (m *mockRunStore) Create(_ context.Context, r *domain.ReconciliationRun, d []domain.ReconciliationRunDiscrepancy) error {
	m.created = append(m.created, r)
	m.createdDiscreps = append(m.createdDiscreps, d)
	return nil
}
func (m *mockRunStore) ListByTenant(_ context.Context, _ uuid.UUID, _ int) ([]domain.ReconciliationRun, error) {
	return m.list, nil
}
func (m *mockRunStore) GetByID(_ context.Context, _, _ uuid.UUID) (*domain.ReconciliationRunDetail, error) {
	return m.detail, nil
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

func TestRecordRunPersistsDiscrepancyDetail(t *testing.T) {
	store := &mockRunStore{}
	svc := NewReconciliationService(nil, nil)
	svc.SetRunStore(store)

	invID := uuid.New()
	report := &ReconciliationReport{
		TotalDiscrepancies: 2,
		Discrepancies: []ReconciliationDiscrepancy{
			{Type: "invoice_amount_mismatch", InvoiceID: &invID, ExpectedAmount: 10000, FoundAmount: 9000},
			{Type: "ledger_unbalanced", ExpectedAmount: 500, FoundAmount: 0},
		},
	}
	if err := svc.RecordRun(context.Background(), uuid.New(), uuid.New(), report); err != nil {
		t.Fatalf("RecordRun: %v", err)
	}
	if len(store.createdDiscreps) != 1 || len(store.createdDiscreps[0]) != 2 {
		t.Fatalf("expected 2 discrepancy rows persisted, got %#v", store.createdDiscreps)
	}
	got := store.createdDiscreps[0][0]
	if got.Type != "invoice_amount_mismatch" || got.InvoiceID == nil || *got.InvoiceID != invID ||
		got.ExpectedAmount != 10000 || got.FoundAmount != 9000 {
		t.Fatalf("discrepancy not mapped onto the persisted row: %+v", got)
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
