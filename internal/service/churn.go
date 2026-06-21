package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
	"github.com/recur-so/recurso/internal/core/port"
)

type ChurnService struct {
	customerRepo port.CustomerRepository
	invoiceRepo  port.InvoiceRepository
	// usageRepo    port.UsageRepository // Future: Analyze usage drop
}

func NewChurnService(
	customerRepo port.CustomerRepository,
	invoiceRepo port.InvoiceRepository,
) *ChurnService {
	return &ChurnService{
		customerRepo: customerRepo,
		invoiceRepo:  invoiceRepo,
	}
}

// AnalyzeCustomer calculates the risk score for a customer based on "Intelligence Heuristics"
func (s *ChurnService) AnalyzeCustomer(ctx context.Context, customerID uuid.UUID) error {
	score := 0
	factors := make(map[string]interface{})
	riskFactorsList := []string{}

	// 1. Analyze Recent Invoices (Last 90 Days)
	invoices, err := s.invoiceRepo.GetByCustomerID(ctx, customerID)
	if err != nil {
		return err
	}

	failedInvoices := 0
	for _, inv := range invoices {
		// Logic: If invoice is void or past_due or uncollectible, it counts.
		// Checking last 90 days.
		if time.Since(inv.CreatedAt) < 90*24*time.Hour {
			if inv.Status == domain.InvoiceStatusVoid || inv.Status == domain.InvoiceStatusPastDue || inv.Status == domain.InvoiceStatusUncollectible {
				failedInvoices++
			}
		}
	}

	if failedInvoices > 0 {
		score += failedInvoices * 20
		riskFactorsList = append(riskFactorsList, "payment_failures")
		factors["failed_invoices_count"] = failedInvoices
	}

	// Cap score at 100
	if score > 100 {
		score = 100
	}

	factors["factors"] = riskFactorsList

	// Update Risk in DB
	return s.customerRepo.UpdateRisk(ctx, customerID, score, factors)
}

// AnalyzeAllCustomers iterates and analyzes all (Placeholder for Worker)
func (s *ChurnService) AnalyzeAllCustomers(ctx context.Context, tenantID uuid.UUID) error {
	// 1. List all customers
	// 2. Loop and Analyze
	return nil
}
