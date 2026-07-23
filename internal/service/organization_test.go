package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

type mockOrgRepo struct {
	orgs map[uuid.UUID]*domain.Organization
}

func newMockOrgRepo() *mockOrgRepo {
	return &mockOrgRepo{orgs: make(map[uuid.UUID]*domain.Organization)}
}

func (m *mockOrgRepo) Create(_ context.Context, org *domain.Organization) error {
	m.orgs[org.ID] = org
	return nil
}

func (m *mockOrgRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Organization, error) {
	if org, ok := m.orgs[id]; ok {
		return org, nil
	}
	return nil, domain.ErrOrganizationNotFound
}

func (m *mockOrgRepo) ListByOwner(_ context.Context, ownerTenantID uuid.UUID) ([]*domain.Organization, error) {
	var list []*domain.Organization
	for _, o := range m.orgs {
		if o.OwnerTenantID == ownerTenantID {
			list = append(list, o)
		}
	}
	return list, nil
}

func (m *mockOrgRepo) Update(_ context.Context, _ *domain.Organization) error { return nil }
func (m *mockOrgRepo) Delete(_ context.Context, _ uuid.UUID) error            { return nil }
func (m *mockOrgRepo) AddTenant(_ context.Context, _, _ uuid.UUID) error      { return nil }
func (m *mockOrgRepo) RemoveTenant(_ context.Context, _, _ uuid.UUID) error   { return nil }
func (m *mockOrgRepo) ListTenants(_ context.Context, _ uuid.UUID) ([]*domain.Tenant, error) {
	return nil, nil
}

func TestOrganizationService_TenantIsolation(t *testing.T) {
	repo := newMockOrgRepo()
	svc := service.NewOrganizationService(repo, nil, nil)

	ownerTenantID := uuid.New()
	otherTenantID := uuid.New()
	orgID := uuid.New()

	repo.orgs[orgID] = &domain.Organization{
		ID:            orgID,
		OwnerTenantID: ownerTenantID,
		Name:          "Test Corp",
	}

	// Fetch as non-owner tenant should return ErrOrganizationNotFound for security
	_, err := svc.GetByID(context.Background(), otherTenantID, orgID)
	if err == nil {
		t.Error("expected error accessing organization owned by another tenant")
	}

	// Fetch as owner tenant should succeed
	org, err := svc.GetByID(context.Background(), ownerTenantID, orgID)
	if err != nil {
		t.Fatalf("unexpected error accessing owned organization: %v", err)
	}

	if org.Name != "Test Corp" {
		t.Errorf("expected org name 'Test Corp', got %s", org.Name)
	}
}

func TestOrganizationService_AddTenantSelfRestriction(t *testing.T) {
	repo := newMockOrgRepo()
	svc := service.NewOrganizationService(repo, nil, nil)

	ownerTenantID := uuid.New()
	foreignTenantID := uuid.New()
	orgID := uuid.New()

	repo.orgs[orgID] = &domain.Organization{
		ID:            orgID,
		OwnerTenantID: ownerTenantID,
		Name:          "Parent Org",
	}

	// Attaching foreign tenant without consent must return ErrCrossTenantAttach
	err := svc.AddTenant(context.Background(), ownerTenantID, orgID, foreignTenantID)
	if err != domain.ErrCrossTenantAttach {
		t.Errorf("expected ErrCrossTenantAttach, got %v", err)
	}

	// Attaching self must succeed
	err = svc.AddTenant(context.Background(), ownerTenantID, orgID, ownerTenantID)
	if err != nil {
		t.Errorf("unexpected error attaching self to owned org: %v", err)
	}
}
