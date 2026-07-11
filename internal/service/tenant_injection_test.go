package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// tenantAssertingCustomerRepo fails the test if GetByID is called without the
// expected tenant in ctx — the tenant-context bug class (ENG-134).
type tenantAssertingCustomerRepo struct {
	port.CustomerRepository
	t          *testing.T
	wantTenant uuid.UUID
	customer   *domain.Customer
	called     bool
}

func (f *tenantAssertingCustomerRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Customer, error) {
	f.called = true
	got, ok := ctx.Value(domain.TenantIDKey).(uuid.UUID)
	if !ok || got != f.wantTenant {
		f.t.Errorf("customer GetByID ctx tenant = %v (ok=%v), want %v — background callers must inject", got, ok, f.wantTenant)
	}
	return f.customer, nil
}

// TestEInvoiceGenerate_InjectsInvoiceTenant guards the e-invoice retry worker
// path: GenerateEInvoice is called with a background context by the worker,
// and must inject the invoice's own tenant before the tenant-scoped customer
// read. Regression for ENG-134 (automatic e-invoice retries never worked).
func TestEInvoiceGenerate_InjectsInvoiceTenant(t *testing.T) {
	tenantID := uuid.New()
	custID := uuid.New()
	// Ineligible (non-India) customer: the method returns right after the
	// customer fetch, before touching the concrete config repos.
	repo := &tenantAssertingCustomerRepo{
		t: t, wantTenant: tenantID,
		customer: &domain.Customer{ID: custID, TenantID: tenantID,
			BillingAddress: domain.BillingAddress{Country: "US"}},
	}
	svc := NewEInvoiceService(nil, nil, repo, nil, nil)

	inv := &domain.Invoice{ID: uuid.New(), TenantID: tenantID, CustomerID: custID}
	if _, err := svc.GenerateEInvoice(context.Background(), inv); err != nil {
		t.Fatalf("GenerateEInvoice: %v", err)
	}
	if !repo.called {
		t.Fatal("customer GetByID never called — test is vacuous")
	}
}
