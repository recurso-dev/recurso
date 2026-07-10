package worker

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

// churnTenantAssertingRepo fails the test when List runs without the tenant in
// ctx — the tenant-context bug class (ENG-134): the churn worker's background
// context carried no tenant, so churn analysis never ran.
type churnTenantAssertingRepo struct {
	port.CustomerRepository
	t          *testing.T
	wantTenant uuid.UUID
	called     bool
}

func (f *churnTenantAssertingRepo) List(ctx context.Context, tenantID uuid.UUID, filter domain.CustomerFilter) ([]*domain.Customer, error) {
	f.called = true
	got, ok := ctx.Value(domain.TenantIDKey).(uuid.UUID)
	if !ok || got != f.wantTenant {
		f.t.Errorf("customer List ctx tenant = %v (ok=%v), want %v — worker must inject before service calls", got, ok, f.wantTenant)
	}
	return nil, nil // empty page ends the loop before AnalyzeCustomer
}

func TestChurnWorker_InjectsTenantForAnalysis(t *testing.T) {
	tenantID := uuid.New()
	repo := &churnTenantAssertingRepo{t: t, wantTenant: tenantID}
	w := NewChurnWorker(nil, repo, nil, time.Hour)

	w.AnalyzeTenantCustomers(context.Background(), tenantID)

	if !repo.called {
		t.Fatal("customer List never called — test is vacuous")
	}
}
