package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
	"github.com/swapnull-in/recur-so/internal/adapter/gsp"
)

// --- Mocks ---

type MockInvoiceRepo struct {
	port.InvoiceRepository
	CreatedInvoice *domain.Invoice
}

func (m *MockInvoiceRepo) Create(ctx context.Context, inv *domain.Invoice) error {
	m.CreatedInvoice = inv
	return nil
}

type MockPlanRepo struct {
	port.PlanRepository
	Plan *domain.Plan
}

func (m *MockPlanRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Plan, error) {
	return m.Plan, nil
}

type MockCustomerRepo struct {
	port.CustomerRepository
	Customer *domain.Customer
}

func (m *MockCustomerRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Customer, error) {
	return m.Customer, nil
}

type MockUnbilledChargeRepo struct {
	port.UnbilledChargeRepository
}

func (m *MockUnbilledChargeRepo) ListBySubscriptionID(subID uuid.UUID) ([]*domain.UnbilledCharge, error) {
	return []*domain.UnbilledCharge{}, nil
}

func (m *MockUnbilledChargeRepo) MarkAsInvoiced(ids []uuid.UUID) error {
	return nil
}

type MockSubscriptionRepo struct {
	port.SubscriptionRepository
}

// --- Test ---

func TestGenerateInvoice_EInvoice_B2B(t *testing.T) {
	// Setup Dependencies
	mockInvRepo := &MockInvoiceRepo{}
	mockPlanRepo := &MockPlanRepo{
		Plan: &domain.Plan{
			ID: uuid.New(),
			Prices: []domain.Price{
				{Amount: 100000, Currency: "INR"},
			},
		},
	}
	mockCustRepo := &MockCustomerRepo{
		Customer: &domain.Customer{
			ID: uuid.New(),
			BillingAddress: domain.BillingAddress{
				Country: "India",
				State:   "TN",
			},
			GSTIN:         domain.StringPtr("33ABCDE1234F1Z5"),
			TaxType:       "business",
			PlaceOfSupply: domain.StringPtr("TN"),
		},
	}
	mockUCRepo := &MockUnbilledChargeRepo{}
	mockSubRepo := &MockSubscriptionRepo{}
	mockGSP := gsp.NewMockGSPAdapter()

	svc := NewInvoiceService(mockInvRepo, mockPlanRepo, mockCustRepo, mockUCRepo, mockSubRepo, mockGSP)

	// Setup Input
	subID := uuid.New()
	sub := &domain.Subscription{
		ID:           subID,
		CustomerID:   mockCustRepo.Customer.ID,
		PlanID:       mockPlanRepo.Plan.ID,
		TenantID:     uuid.New(),
		PaymentTerms: "net30",
	}

	// Execution
	ctx := context.Background()
	inv, err := svc.GenerateInvoice(ctx, sub)

	// Assertions
	if err != nil {
		t.Fatalf("GenerateInvoice failed: %v", err)
	}

	if inv.EInvoiceStatus != "GENERATED" {
		t.Errorf("Expected EInvoiceStatus GENERATED, got %s", inv.EInvoiceStatus)
	}
	if inv.IRN == "" {
		t.Error("Expected IRN to be generated")
	}
	if inv.SignedQRCode == "" {
		t.Error("Expected SignedQRCode to be generated")
	}
	if inv.TaxAmount == 0 {
		t.Error("Expected TaxAmount to be calculated")
	}
	// 18% of 100000 = 18000
	if inv.TaxAmount != 18000 {
		t.Errorf("Expected TaxAmount 18000, got %d", inv.TaxAmount)
	}
}

func TestGenerateInvoice_EInvoice_Consumer(t *testing.T) {
	// Setup Dependencies
	mockInvRepo := &MockInvoiceRepo{}
	mockPlanRepo := &MockPlanRepo{
		Plan: &domain.Plan{
			ID: uuid.New(),
			Prices: []domain.Price{
				{Amount: 100000, Currency: "INR"},
			},
		},
	}
	mockCustRepo := &MockCustomerRepo{
		Customer: &domain.Customer{
			ID: uuid.New(),
			BillingAddress: domain.BillingAddress{
				Country: "India",
				State:   "TN",
			},
			GSTIN:         nil, // Consumer
			TaxType:       "consumer",
			PlaceOfSupply: domain.StringPtr("TN"),
		},
	}
	mockUCRepo := &MockUnbilledChargeRepo{}
	mockSubRepo := &MockSubscriptionRepo{}
	mockGSP := gsp.NewMockGSPAdapter()

	svc := NewInvoiceService(mockInvRepo, mockPlanRepo, mockCustRepo, mockUCRepo, mockSubRepo, mockGSP)

	// Setup Input
	subID := uuid.New()
	sub := &domain.Subscription{
		ID:           subID,
		CustomerID:   mockCustRepo.Customer.ID,
		PlanID:       mockPlanRepo.Plan.ID,
		TenantID:     uuid.New(),
		PaymentTerms: "net30",
	}

	// Execution
	ctx := context.Background()
	inv, err := svc.GenerateInvoice(ctx, sub)

	// Assertions
	if err != nil {
		t.Fatalf("GenerateInvoice failed: %v", err)
	}

	if inv.EInvoiceStatus != "NA" {
		t.Errorf("Expected EInvoiceStatus NA, got %s", inv.EInvoiceStatus)
	}
	if inv.IRN != "" {
		t.Error("Expected IRN to be empty for Consumer")
	}
}
