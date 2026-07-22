package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

type mockEntityStore struct {
	byID    map[uuid.UUID]*domain.Entity
	created []*domain.Entity
	updated []*domain.Entity
	deleted []uuid.UUID
}

func newMockEntityStore() *mockEntityStore {
	return &mockEntityStore{byID: map[uuid.UUID]*domain.Entity{}}
}

func (m *mockEntityStore) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Entity, error) {
	var out []*domain.Entity
	for _, e := range m.byID {
		if e.TenantID == tenantID {
			out = append(out, e)
		}
	}
	return out, nil
}
func (m *mockEntityStore) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.Entity, error) {
	e := m.byID[id]
	if e == nil || e.TenantID != tenantID {
		return nil, nil
	}
	return e, nil
}
func (m *mockEntityStore) GetPrimary(ctx context.Context, tenantID uuid.UUID) (*domain.Entity, error) {
	for _, e := range m.byID {
		if e.TenantID == tenantID && e.IsPrimary {
			return e, nil
		}
	}
	return nil, nil
}
func (m *mockEntityStore) Create(ctx context.Context, e *domain.Entity) error {
	e.ID = uuid.New()
	e.TBLedgerID = len(m.byID) + 1
	m.byID[e.ID] = e
	m.created = append(m.created, e)
	return nil
}
func (m *mockEntityStore) Update(ctx context.Context, e *domain.Entity) error {
	m.byID[e.ID] = e
	m.updated = append(m.updated, e)
	return nil
}
func (m *mockEntityStore) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	delete(m.byID, id)
	m.deleted = append(m.deleted, id)
	return nil
}

func TestEntity_Create_RequiresName(t *testing.T) {
	svc := NewEntityService(newMockEntityStore())
	if _, err := svc.Create(context.Background(), uuid.New(), CreateEntityInput{Name: "  "}); err == nil {
		t.Error("expected an error for a blank name")
	}
}

func TestEntity_Create_SanitizesPrefixAndDefaults(t *testing.T) {
	store := newMockEntityStore()
	svc := NewEntityService(store)
	e, err := svc.Create(context.Background(), uuid.New(), CreateEntityInput{Name: "ACME India!"})
	if err != nil {
		t.Fatal(err)
	}
	if e.InvoicePrefix != "ACME-INDIA" {
		t.Errorf("invoice_prefix = %q, want ACME-INDIA (slug of the name)", e.InvoicePrefix)
	}
	if e.IsPrimary {
		t.Error("a created entity must never be primary")
	}
	if len(store.created) != 1 {
		t.Errorf("expected one create, got %d", len(store.created))
	}
}

func TestEntity_Create_RejectsBadCountry(t *testing.T) {
	svc := NewEntityService(newMockEntityStore())
	if _, err := svc.Create(context.Background(), uuid.New(), CreateEntityInput{Name: "X", CountryCode: "USA"}); err == nil {
		t.Error("expected an error for a non-2-letter country code")
	}
}

func TestEntity_Delete_PrimaryRejected(t *testing.T) {
	store := newMockEntityStore()
	tenant := uuid.New()
	primary := &domain.Entity{ID: uuid.New(), TenantID: tenant, IsPrimary: true, Name: "Primary"}
	store.byID[primary.ID] = primary

	svc := NewEntityService(store)
	if err := svc.Delete(context.Background(), tenant, primary.ID); err == nil {
		t.Error("the primary entity must not be deletable")
	}
	if len(store.deleted) != 0 {
		t.Error("no delete should have been issued for the primary entity")
	}
}

func TestEntity_Delete_NonPrimary(t *testing.T) {
	store := newMockEntityStore()
	tenant := uuid.New()
	e := &domain.Entity{ID: uuid.New(), TenantID: tenant, IsPrimary: false, Name: "ACME UK"}
	store.byID[e.ID] = e

	svc := NewEntityService(store)
	if err := svc.Delete(context.Background(), tenant, e.ID); err != nil {
		t.Fatal(err)
	}
	if len(store.deleted) != 1 || store.deleted[0] != e.ID {
		t.Error("expected the non-primary entity to be deleted")
	}
}

func TestEntity_Delete_NotFound(t *testing.T) {
	svc := NewEntityService(newMockEntityStore())
	err := svc.Delete(context.Background(), uuid.New(), uuid.New())
	if err == nil || !errors.Is(err, ErrEntityValidation) {
		t.Errorf("expected a validation error for a missing entity, got %v", err)
	}
}
