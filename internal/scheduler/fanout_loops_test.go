package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/adapter/memory"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

// fanoutTenantRepo returns a fixed tenant list for the fan-out schedulers.
type fanoutTenantRepo struct{ tenants []*domain.Tenant }

func (r *fanoutTenantRepo) ListTenants(_ context.Context) ([]*domain.Tenant, error) {
	return r.tenants, nil
}

// --- MRR snapshot fan-out ---

type fanoutCapturer struct {
	called  []uuid.UUID
	failFor uuid.UUID
}

func (c *fanoutCapturer) CaptureMRRSnapshot(_ context.Context, tenantID uuid.UUID, _ time.Time) (int, error) {
	c.called = append(c.called, tenantID)
	if tenantID == c.failFor {
		return 0, errors.New("boom")
	}
	return 1, nil
}

// TestMRRSnapshotScheduler_RunFansOutOverAllTenants guards the loop's control
// flow: it captures every tenant, and one tenant's failure does not abort the
// rest (the per-day capture is separately DB-tested).
func TestMRRSnapshotScheduler_RunFansOutOverAllTenants(t *testing.T) {
	t1, t2, t3 := uuid.New(), uuid.New(), uuid.New()
	tr := &fanoutTenantRepo{tenants: []*domain.Tenant{{ID: t1}, {ID: t2}, {ID: t3}}}
	cap := &fanoutCapturer{failFor: t2} // middle tenant errors

	s := NewMRRSnapshotScheduler(tr, cap, memory.NewNoOpLocker())
	s.run()

	if len(cap.called) != 3 {
		t.Fatalf("captured %d tenants, want 3 (a per-tenant error must not abort the fan-out)", len(cap.called))
	}
	for _, want := range []uuid.UUID{t1, t2, t3} {
		if !containsID(cap.called, want) {
			t.Errorf("tenant %s was not captured", want)
		}
	}
}

// --- Reconciliation fan-out ---

type fanoutReconRunner struct {
	called  []uuid.UUID
	failFor uuid.UUID
}

func (r *fanoutReconRunner) Run(_ context.Context, tenantID uuid.UUID) (*service.ReconciliationReport, error) {
	r.called = append(r.called, tenantID)
	if tenantID == r.failFor {
		return nil, errors.New("boom")
	}
	return &service.ReconciliationReport{TenantID: tenantID}, nil
}

// TestReconciliationScheduler_RunFansOutOverAllTenants guards the loop's control
// flow: it reconciles every tenant, and one tenant's failure does not abort the
// rest (the per-tenant reconciliation is separately DB-tested).
func TestReconciliationScheduler_RunFansOutOverAllTenants(t *testing.T) {
	t1, t2, t3 := uuid.New(), uuid.New(), uuid.New()
	tr := &fanoutTenantRepo{tenants: []*domain.Tenant{{ID: t1}, {ID: t2}, {ID: t3}}}
	runner := &fanoutReconRunner{failFor: t1} // first tenant errors

	s := NewReconciliationScheduler(tr, runner, memory.NewNoOpLocker())
	s.runReconciliation()

	if len(runner.called) != 3 {
		t.Fatalf("reconciled %d tenants, want 3 (a per-tenant error must not abort the fan-out)", len(runner.called))
	}
	for _, want := range []uuid.UUID{t1, t2, t3} {
		if !containsID(runner.called, want) {
			t.Errorf("tenant %s was not reconciled", want)
		}
	}
}

func containsID(ids []uuid.UUID, want uuid.UUID) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}
