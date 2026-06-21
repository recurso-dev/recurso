package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
)

// --- Mock CustomerRepository ---
type mockCustomerRepo struct {
	customers    map[uuid.UUID]*domain.Customer
	createErr    error
	getByIDErr   error
	listResult   []*domain.Customer
	byReferral   *domain.Customer
	updateCalled bool
}

func newMockCustomerRepo() *mockCustomerRepo {
	return &mockCustomerRepo{
		customers: make(map[uuid.UUID]*domain.Customer),
	}
}

func (m *mockCustomerRepo) Create(ctx context.Context, c *domain.Customer) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.customers[c.ID] = c
	return nil
}

func (m *mockCustomerRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Customer, error) {
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	if c, ok := m.customers[id]; ok {
		return c, nil
	}
	return nil, errors.New("customer not found")
}

func (m *mockCustomerRepo) List(ctx context.Context, tenantID uuid.UUID, filter domain.CustomerFilter) ([]*domain.Customer, error) {
	return m.listResult, nil
}

func (m *mockCustomerRepo) Update(ctx context.Context, c *domain.Customer) error {
	m.updateCalled = true
	m.customers[c.ID] = c
	return nil
}

func (m *mockCustomerRepo) GetByIDPublic(ctx context.Context, id uuid.UUID) (*domain.Customer, error) {
	return m.GetByID(ctx, id)
}

func (m *mockCustomerRepo) GetByReferralCode(ctx context.Context, tenantID uuid.UUID, code string) (*domain.Customer, error) {
	return m.byReferral, nil
}

func (m *mockCustomerRepo) UpdateRisk(ctx context.Context, customerID uuid.UUID, score int, factors map[string]interface{}) error {
	return nil
}

func (m *mockCustomerRepo) UpdatePaymentMethod(ctx context.Context, customerID uuid.UUID, brand, last4 string, expMonth, expYear int) error {
	return nil
}

// --- Tests ---

func TestCreateCustomer_Success(t *testing.T) {
	repo := newMockCustomerRepo()
	svc := NewCustomerService(repo)

	tenantID := uuid.New()
	input := CreateCustomerInput{
		TenantID: tenantID,
		Email:    "alice@example.com",
		Name:     "Alice Smith",
		Phone:    "+1-555-0100",
		Country:  "US",
	}

	customer, err := svc.CreateCustomer(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if customer.ID == uuid.Nil {
		t.Error("customer ID should be generated")
	}
	if customer.Email != "alice@example.com" {
		t.Errorf("email = %q, want alice@example.com", customer.Email)
	}
	if customer.TenantID != tenantID {
		t.Error("tenant ID mismatch")
	}
	if customer.LedgerAccountID == uuid.Nil {
		t.Error("ledger account ID should be generated")
	}
}

func TestCreateCustomer_AutoTaxType_Business(t *testing.T) {
	repo := newMockCustomerRepo()
	svc := NewCustomerService(repo)

	customer, err := svc.CreateCustomer(context.Background(), CreateCustomerInput{
		TenantID: uuid.New(),
		Email:    "biz@corp.in",
		Name:     "CorpInd",
		GSTIN:    "29ABCDE1234F1ZK",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if customer.TaxType != "business" {
		t.Errorf("tax type = %q, want 'business' (auto-set from GSTIN)", customer.TaxType)
	}
}

func TestCreateCustomer_AutoTaxType_Consumer(t *testing.T) {
	repo := newMockCustomerRepo()
	svc := NewCustomerService(repo)

	customer, err := svc.CreateCustomer(context.Background(), CreateCustomerInput{
		TenantID: uuid.New(),
		Email:    "consumer@gmail.com",
		Name:     "Joe",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if customer.TaxType != "consumer" {
		t.Errorf("tax type = %q, want 'consumer' (default when no GSTIN)", customer.TaxType)
	}
}

func TestCreateCustomer_ExplicitTaxType(t *testing.T) {
	repo := newMockCustomerRepo()
	svc := NewCustomerService(repo)

	customer, err := svc.CreateCustomer(context.Background(), CreateCustomerInput{
		TenantID: uuid.New(),
		Email:    "biz@corp.in",
		Name:     "CorpInd",
		GSTIN:    "29ABCDE1234F1ZK",
		TaxType:  "exempt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if customer.TaxType != "exempt" {
		t.Errorf("tax type = %q, want 'exempt' (explicitly provided)", customer.TaxType)
	}
}

func TestCreateCustomer_RepoError(t *testing.T) {
	repo := newMockCustomerRepo()
	repo.createErr = errors.New("db connection refused")
	svc := NewCustomerService(repo)

	_, err := svc.CreateCustomer(context.Background(), CreateCustomerInput{
		TenantID: uuid.New(),
		Email:    "test@test.com",
		Name:     "Test",
	})
	if err == nil {
		t.Error("expected error from repo, got nil")
	}
}

func TestGetCustomer_TenantIsolation(t *testing.T) {
	repo := newMockCustomerRepo()
	svc := NewCustomerService(repo)

	tenantA := uuid.New()
	tenantB := uuid.New()

	// Create customer belonging to tenant A
	customer := &domain.Customer{
		ID:       uuid.New(),
		TenantID: tenantA,
		Email:    "alice@a.com",
		Name:     domain.StringPtr("Alice"),
	}
	repo.customers[customer.ID] = customer

	// Tenant A can access it
	result, err := svc.GetCustomer(context.Background(), tenantA, customer.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected customer, got nil")
	}

	// Tenant B cannot access it
	result, err = svc.GetCustomer(context.Background(), tenantB, customer.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("tenant B should not be able to access tenant A's customer")
	}
}

func TestCreateCustomer_BillingAddress(t *testing.T) {
	repo := newMockCustomerRepo()
	svc := NewCustomerService(repo)

	customer, err := svc.CreateCustomer(context.Background(), CreateCustomerInput{
		TenantID: uuid.New(),
		Email:    "test@test.com",
		Name:     "Test",
		Line1:    "123 Main St",
		City:     "San Francisco",
		State:    "CA",
		Zip:      "94102",
		Country:  "US",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if customer.BillingAddress.Line1 != "123 Main St" {
		t.Errorf("line1 = %q, want '123 Main St'", customer.BillingAddress.Line1)
	}
	if customer.BillingAddress.City != "San Francisco" {
		t.Errorf("city = %q, want 'San Francisco'", customer.BillingAddress.City)
	}
}
