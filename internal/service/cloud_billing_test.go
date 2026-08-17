package service

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// --- fakes (no database) ---

type fakeCustomerCreator struct {
	calls   int
	lastIn  CreateCustomerInput
	created []*domain.Customer
}

func (f *fakeCustomerCreator) CreateCustomer(_ context.Context, in CreateCustomerInput) (*domain.Customer, error) {
	f.calls++
	f.lastIn = in
	c := &domain.Customer{ID: uuid.New(), TenantID: in.TenantID, Email: in.Email, Name: &in.Name}
	f.created = append(f.created, c)
	return c, nil
}

type fakeMappingRepo struct {
	rows map[uuid.UUID]*domain.CloudTenantCustomer // keyed by signup tenant id
}

func newFakeMappingRepo() *fakeMappingRepo {
	return &fakeMappingRepo{rows: map[uuid.UUID]*domain.CloudTenantCustomer{}}
}

func (f *fakeMappingRepo) GetByTenant(_ context.Context, _ uuid.UUID, tenantID uuid.UUID) (*domain.CloudTenantCustomer, error) {
	return f.rows[tenantID], nil
}

func (f *fakeMappingRepo) Create(_ context.Context, m *domain.CloudTenantCustomer) error {
	f.rows[m.TenantID] = m
	return nil
}

type fakeTenantLister struct{ tenants []*domain.Tenant }

func (f *fakeTenantLister) ListTenants(_ context.Context) ([]*domain.Tenant, error) {
	return f.tenants, nil
}

func newTenant(name string) *domain.Tenant {
	return &domain.Tenant{ID: uuid.New(), Name: name, Email: name + "@example.com"}
}

// --- tests ---

func TestProvisionTenant_CreatesCustomerAndMapping(t *testing.T) {
	platform := uuid.New()
	creator := &fakeCustomerCreator{}
	repo := newFakeMappingRepo()
	svc := NewCloudBillingService(platform, creator, repo, &fakeTenantLister{}, nil)

	tenant := newTenant("spotify")
	if err := svc.ProvisionTenant(context.Background(), tenant); err != nil {
		t.Fatalf("ProvisionTenant: %v", err)
	}
	if creator.calls != 1 {
		t.Fatalf("expected 1 CreateCustomer call, got %d", creator.calls)
	}
	if creator.lastIn.TenantID != platform {
		t.Fatalf("customer should be created inside the platform tenant, got %s", creator.lastIn.TenantID)
	}
	if creator.lastIn.Name != "spotify" || creator.lastIn.Email != "spotify@example.com" {
		t.Fatalf("customer name/email not carried from tenant: %+v", creator.lastIn)
	}
	m := repo.rows[tenant.ID]
	if m == nil || m.PlatformTenantID != platform || m.CustomerID != creator.created[0].ID {
		t.Fatalf("mapping not recorded correctly: %+v", m)
	}
	if m.SubscriptionID != nil {
		t.Fatalf("increment 1 must not set a subscription, got %v", m.SubscriptionID)
	}
}

func TestProvisionTenant_Idempotent(t *testing.T) {
	platform := uuid.New()
	creator := &fakeCustomerCreator{}
	repo := newFakeMappingRepo()
	svc := NewCloudBillingService(platform, creator, repo, &fakeTenantLister{}, nil)

	tenant := newTenant("acme")
	for i := 0; i < 3; i++ {
		if err := svc.ProvisionTenant(context.Background(), tenant); err != nil {
			t.Fatalf("ProvisionTenant #%d: %v", i, err)
		}
	}
	if creator.calls != 1 {
		t.Fatalf("expected exactly 1 CreateCustomer across repeated provisions, got %d", creator.calls)
	}
}

func TestProvisionTenant_SkipsPlatformTenantItself(t *testing.T) {
	platform := uuid.New()
	creator := &fakeCustomerCreator{}
	repo := newFakeMappingRepo()
	svc := NewCloudBillingService(platform, creator, repo, &fakeTenantLister{}, nil)

	self := &domain.Tenant{ID: platform, Name: "founder", Email: "founder@example.com"}
	if err := svc.ProvisionTenant(context.Background(), self); err != nil {
		t.Fatalf("ProvisionTenant: %v", err)
	}
	if creator.calls != 0 {
		t.Fatalf("the founder tenant must not be mirrored as its own customer, got %d calls", creator.calls)
	}
}

func TestBackfill_ProvisionsMissingSkipsExistingAndSelf(t *testing.T) {
	platform := uuid.New()
	creator := &fakeCustomerCreator{}
	repo := newFakeMappingRepo()

	a := newTenant("a")
	b := newTenant("b")
	self := &domain.Tenant{ID: platform, Name: "founder", Email: "founder@example.com"}
	lister := &fakeTenantLister{tenants: []*domain.Tenant{a, b, self}}
	svc := NewCloudBillingService(platform, creator, repo, lister, nil)

	// Pre-provision "a" so backfill must skip it.
	if err := svc.ProvisionTenant(context.Background(), a); err != nil {
		t.Fatalf("pre-provision: %v", err)
	}
	if creator.calls != 1 {
		t.Fatalf("setup: expected 1 call, got %d", creator.calls)
	}

	n, err := svc.Backfill(context.Background())
	if err != nil {
		t.Fatalf("Backfill: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 newly provisioned (b), got %d", n)
	}
	if creator.calls != 2 {
		t.Fatalf("expected 2 total CreateCustomer calls (a setup + b backfill), got %d", creator.calls)
	}
	if repo.rows[b.ID] == nil {
		t.Fatalf("tenant b should be provisioned after backfill")
	}
	if repo.rows[platform] != nil {
		t.Fatalf("the platform tenant must never be provisioned")
	}
}
