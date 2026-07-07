package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/adapter/telemetry"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

type CustomerService struct {
	repo      port.CustomerRepository
	telemetry *telemetry.Client // nil-safe; only set when TELEMETRY_OPTIN=true
}

// SetTelemetry injects the opt-in anonymous telemetry client after construction.
func (s *CustomerService) SetTelemetry(t *telemetry.Client) { s.telemetry = t }

func NewCustomerService(repo port.CustomerRepository) *CustomerService {
	return &CustomerService{repo: repo}
}

type CreateCustomerInput struct {
	TenantID      uuid.UUID
	Email         string
	Name          string
	Phone         string
	TaxID         string
	GSTIN         string // P24
	TaxType       string // P25
	PlaceOfSupply string // P24
	Line1         string
	City          string
	State         string
	Zip           string
	Country       string
}

func (s *CustomerService) CreateCustomer(ctx context.Context, input CreateCustomerInput) (*domain.Customer, error) {
	// 1. Generate ID
	id := uuid.New()

	// 2. Generate Ledger Account ID (In a real app, this would call TigerBeetle)
	// For P0 without a running TB sidecar, we assume the ID is generated
	ledgerID := uuid.New()

	// Auto-set TaxType if GSTIN is present
	taxType := input.TaxType
	if taxType == "" {
		if input.GSTIN != "" {
			taxType = "business"
		} else {
			taxType = "consumer"
		}
	}

	customer := &domain.Customer{
		ID:            id,
		TenantID:      input.TenantID,
		Email:         input.Email,
		Name:          &input.Name,
		Phone:         input.Phone,
		TaxID:         &input.TaxID,
		GSTIN:         &input.GSTIN,         // P24
		TaxType:       taxType,              // P25
		PlaceOfSupply: &input.PlaceOfSupply, // P24
		BillingAddress: domain.BillingAddress{
			Line1:   input.Line1,
			City:    input.City,
			State:   input.State,
			Zip:     input.Zip,
			Country: input.Country,
		},
		LedgerAccountID: ledgerID,
		CreatedAt:       time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, customer); err != nil {
		return nil, err
	}

	s.telemetry.MilestoneFirstCustomer() // opt-in anonymous milestone; no-op when disabled

	return customer, nil
}

func (s *CustomerService) ListCustomers(ctx context.Context, tenantID uuid.UUID, filter domain.CustomerFilter) ([]*domain.Customer, error) {
	return s.repo.List(ctx, tenantID, filter)
}

func (s *CustomerService) GetCustomer(ctx context.Context, tenantID uuid.UUID, customerID uuid.UUID) (*domain.Customer, error) {
	customer, err := s.repo.GetByID(ctx, customerID)
	if err != nil {
		return nil, err
	}
	// Verify tenant
	if customer.TenantID != tenantID {
		return nil, nil // Or return a not found error
	}
	return customer, nil
}

type UpdatePaymentMethodInput struct {
	CustomerID uuid.UUID
	CardBrand  string
	CardLast4  string
	ExpMonth   int
	ExpYear    int
}

func (s *CustomerService) UpdatePaymentMethod(ctx context.Context, input UpdatePaymentMethodInput) error {
	return s.repo.UpdatePaymentMethod(ctx, input.CustomerID, input.CardBrand, input.CardLast4, input.ExpMonth, input.ExpYear)
}
