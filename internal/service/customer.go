package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/telemetry"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

type CustomerService struct {
	repo      port.CustomerRepository
	subs      port.SubscriptionRepository // nil-safe; gates archiving on active subscriptions
	telemetry *telemetry.Client           // nil-safe; only set when TELEMETRY_OPTIN=true
}

// SetTelemetry injects the opt-in anonymous telemetry client after construction.
func (s *CustomerService) SetTelemetry(t *telemetry.Client) { s.telemetry = t }

// SetSubscriptionRepo enables the archive gate (refuse archiving customers
// with active subscriptions). Without it, archiving skips the check.
func (s *CustomerService) SetSubscriptionRepo(r port.SubscriptionRepository) { s.subs = r }

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

	TaxExempt          bool
	TaxExemptionNumber string
	TaxExemptionCode   string
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
		LedgerAccountID:    ledgerID,
		TaxExempt:          input.TaxExempt,
		TaxExemptionNumber: input.TaxExemptionNumber,
		TaxExemptionCode:   input.TaxExemptionCode,
		Active:             true,
		CreatedAt:          time.Now().UTC(),
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
	// Verify existence + tenant (GetByID returns nil, nil when missing).
	if customer == nil || customer.TenantID != tenantID {
		return nil, nil
	}
	return customer, nil
}

// UpdateCustomerInput carries the editable customer fields. Nil pointers are
// left unchanged, so the same call edits contact/tax details or archives
// (Active=false) / restores (true).
type UpdateCustomerInput struct {
	TenantID   uuid.UUID
	CustomerID uuid.UUID

	Name          *string
	Email         *string
	Phone         *string
	TaxID         *string
	GSTIN         *string
	TaxType       *string
	PlaceOfSupply *string
	Line1         *string
	City          *string
	State         *string
	Zip           *string
	Country       *string
	Active        *bool

	TaxExempt          *bool
	TaxExemptionNumber *string
	TaxExemptionCode   *string
}

// UpdateCustomer applies a partial update. Returns (nil, nil) when the
// customer does not exist for the tenant. Archiving is refused while the
// customer has active subscriptions — cancel or pause them first.
func (s *CustomerService) UpdateCustomer(ctx context.Context, input UpdateCustomerInput) (*domain.Customer, error) {
	customer, err := s.GetCustomer(ctx, input.TenantID, input.CustomerID)
	if err != nil {
		return nil, err
	}
	if customer == nil {
		return nil, nil
	}

	if input.Name != nil {
		customer.Name = input.Name
	}
	if input.Email != nil {
		if *input.Email == "" {
			return nil, fmt.Errorf("email cannot be empty")
		}
		customer.Email = *input.Email
	}
	if input.Phone != nil {
		customer.Phone = *input.Phone
	}
	if input.TaxID != nil {
		customer.TaxID = input.TaxID
	}
	if input.GSTIN != nil {
		customer.GSTIN = input.GSTIN
	}
	if input.TaxType != nil {
		switch *input.TaxType {
		case "business", "consumer":
		default:
			return nil, fmt.Errorf("tax_type must be 'business' or 'consumer'")
		}
		customer.TaxType = *input.TaxType
	}
	if input.PlaceOfSupply != nil {
		customer.PlaceOfSupply = input.PlaceOfSupply
	}
	if input.Line1 != nil {
		customer.BillingAddress.Line1 = *input.Line1
	}
	if input.City != nil {
		customer.BillingAddress.City = *input.City
	}
	if input.State != nil {
		customer.BillingAddress.State = *input.State
	}
	if input.Zip != nil {
		customer.BillingAddress.Zip = *input.Zip
	}
	if input.Country != nil {
		customer.BillingAddress.Country = *input.Country
	}
	if input.TaxExempt != nil {
		customer.TaxExempt = *input.TaxExempt
	}
	if input.TaxExemptionNumber != nil {
		customer.TaxExemptionNumber = *input.TaxExemptionNumber
	}
	if input.TaxExemptionCode != nil {
		customer.TaxExemptionCode = *input.TaxExemptionCode
	}

	if input.Active != nil {
		// Archiving with live subscriptions would orphan active billing.
		if customer.Active && !*input.Active && s.subs != nil {
			counts, err := s.subs.CountActiveByCustomer(ctx, input.TenantID)
			if err != nil {
				return nil, fmt.Errorf("failed to check active subscriptions: %w", err)
			}
			if counts[customer.ID] > 0 {
				return nil, fmt.Errorf("customer has %d active subscription(s) — cancel or pause them before archiving", counts[customer.ID])
			}
		}
		customer.Active = *input.Active
	}

	if err := s.repo.Update(ctx, customer); err != nil {
		return nil, err
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
