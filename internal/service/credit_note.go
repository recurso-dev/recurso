package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/adapter/db"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

type CreditNoteService struct {
	repo         *db.CreditNoteRepository
	customerRepo *db.CustomerRepository // To validate customer and fetch name if needed
}

func NewCreditNoteService(repo *db.CreditNoteRepository, customerRepo *db.CustomerRepository) *CreditNoteService {
	return &CreditNoteService{repo: repo, customerRepo: customerRepo}
}

func (s *CreditNoteService) Create(ctx context.Context, tenantID uuid.UUID, req domain.CreateCreditNoteRequest) (*domain.CreditNote, error) {
	// 1. Validate Customer belongs to Tenant
	customer, err := s.customerRepo.GetByID(ctx, req.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("invalid customer: %w", err)
	}
	if customer.TenantID != tenantID {
		return nil, fmt.Errorf("customer does not belong to tenant")
	}

	// 2. Create Credit Note
	// Generate Reference if not provided
	ref := fmt.Sprintf("CN-%d", time.Now().Unix())

	cn := &domain.CreditNote{
		TenantID:   tenantID,
		CustomerID: req.CustomerID,
		InvoiceID:  req.InvoiceID,
		Reference:  &ref, // Simple reference generation
		Amount:     req.Amount,
		Balance:    req.Amount, // Initially balance = amount
		Currency:   req.Currency,
		Status:     domain.CreditNoteStatusIssued,
		Reason:     req.Reason,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.repo.Create(ctx, cn); err != nil {
		return nil, err
	}

	cn.Customer = customer // Populate customer for response

	return cn, nil
}

func (s *CreditNoteService) List(ctx context.Context, tenantID uuid.UUID, filter domain.CreditNoteFilter) ([]*domain.CreditNote, error) {
	cns, err := s.repo.List(ctx, tenantID, filter)
	if err != nil {
		return nil, err
	}

	// Hydrate Customers
	// Optimization: Fetch all needed customers in one go if this becomes slow.
	// For now, simple loop is fine for MVP.
	for _, cn := range cns {
		customer, _ := s.customerRepo.GetByID(ctx, cn.CustomerID)
		if customer != nil {
			cn.Customer = customer
		}
	}

	return cns, nil
}
