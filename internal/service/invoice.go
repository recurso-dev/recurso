package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
	"github.com/recur-so/recurso/internal/core/port"
	"github.com/recur-so/recurso/internal/core/service/tax"
)

type InvoiceService struct {
	InvoiceRepo        port.InvoiceRepository
	PlanRepo           port.PlanRepository
	CustomerRepo       port.CustomerRepository
	UnbilledChargeRepo port.UnbilledChargeRepository // P15
	SubscriptionRepo   port.SubscriptionRepository   // P15
	GSPAdapter         port.GSPAdapter               // P25
	EInvoiceService    *EInvoiceService              // P25: E-invoice service
}

func NewInvoiceService(
	invRepo port.InvoiceRepository,
	planRepo port.PlanRepository,
	custRepo port.CustomerRepository,
	ucRepo port.UnbilledChargeRepository,
	subRepo port.SubscriptionRepository,
	gspAdapter port.GSPAdapter,
) *InvoiceService {
	return &InvoiceService{
		InvoiceRepo:        invRepo,
		PlanRepo:           planRepo,
		CustomerRepo:       custRepo,
		UnbilledChargeRepo: ucRepo,
		SubscriptionRepo:   subRepo,
		GSPAdapter:         gspAdapter,
	}
}

func (s *InvoiceService) GenerateInvoice(ctx context.Context, sub *domain.Subscription) (*domain.Invoice, error) {
	// 1. Fetch Plan
	plan, err := s.PlanRepo.GetByID(ctx, sub.PlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}

	// 2. Fetch Customer
	customer, err := s.CustomerRepo.GetByID(ctx, sub.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}

	if len(plan.Prices) == 0 {
		return nil, fmt.Errorf("plan has no prices")
	}
	// Use first price for now.
	price := plan.Prices[0]

	// 3. Calculate Amounts
	subtotal := price.Amount

	// P15: Add Unbilled Charges
	charges, err := s.UnbilledChargeRepo.ListBySubscriptionID(sub.ID)
	if err == nil {
		for _, c := range charges {
			subtotal += c.Amount
		}
	}

	// Calculate Tax (P24)
	// Initialize Tax Engine (TODO: Move to struct/dependency injection)
	// Assume Org State is "TN" (Tamil Nadu)
	taxEngine := tax.NewGSTEngine("TN")

	// Determine Place of Supply
	// If customer has PlaceOfSupply set, use it. Else fall back to some logic or treat as intra-state/inter-state default.
	pos := customer.PlaceOfSupply
	if domain.PtrToString(pos) == "" {
		// Try to infer from Address? For now assume it matches Org State if not set (Consumer) or handle as ERROR?
		// Let's assume consumer in same state for simplicity if missing, OR better, default to Inter-state (IGST) to be safe/conservative?
		// Actually for B2C SaaS in India, if location unknown, it's complicated.
		// Let's default to "TN" if missing to keep it simple for local dev, or empty string -> IGST.
		pos = domain.StringPtr("TN") // Defaulting to Intra-state for dev simplicity? Or Inter-state.
		// Let's use empty string which GSTEngine handles as IGST.
		pos = nil
	}

	taxRes := taxEngine.CalculateTax(subtotal, domain.PtrToString(pos))

	total := subtotal + taxRes.Total

	// 4. Determine Payment Terms & Due Date (P15)
	terms := sub.PaymentTerms
	if terms == "" {
		terms = "net0"
	}

	now := time.Now()
	dueDate := domain.CalculateDueDate(now, terms)

	// 5. Create Invoice
	invID := uuid.New()
	inv := &domain.Invoice{
		ID:             invID,
		TenantID:       sub.TenantID,
		SubscriptionID: &sub.ID,
		CustomerID:     sub.CustomerID,
		InvoiceNumber:  fmt.Sprintf("INV-%d-%s", now.UnixNano(), invID.String()[:8]),
		Status:         domain.InvoiceStatusOpen,
		Currency:       price.Currency,
		Subtotal:       subtotal,
		TaxAmount:      taxRes.Total,
		Total:          total,

		IGSTAmount: taxRes.IGST,
		CGSTAmount: taxRes.CGST,
		SGSTAmount: taxRes.SGST,
		// HSNCode?

		CreatedAt:    now,
		DueDate:      dueDate,
		PaymentTerms: terms,
		RetryCount:   0,
	}

	// P25: E-Invoicing via EInvoiceService
	if s.EInvoiceService != nil {
		_, einvErr := s.EInvoiceService.GenerateEInvoice(ctx, inv)
		if einvErr != nil {
			// Soft fail: invoice still gets created, e-invoice retried later
			slog.Error("e-invoice generation failed (will retry)", "error", einvErr)
		}
	} else if customer.BillingAddress.Country == "India" && domain.PtrToString(customer.GSTIN) != "" && customer.TaxType == "business" {
		// Fallback: direct GSP call (backward compat when EInvoiceService is nil)
		resp, err := s.GSPAdapter.GenerateIRN(ctx, inv)
		if err == nil {
			inv.IRN = resp.IRN
			inv.SignedQRCode = resp.SignedQRCode
			inv.EInvoiceStatus = "GENERATED"
			inv.AckNo = resp.AckNo
		} else {
			slog.Error("error generating IRN", "error", err)
			inv.EInvoiceStatus = "FAILED"
		}
	} else {
		inv.EInvoiceStatus = "NA"
	}

	// 6. Persist
	if err := s.InvoiceRepo.Create(ctx, inv); err != nil {
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	// P15: Mark Charges as Invoiced
	if len(charges) > 0 {
		var ids []uuid.UUID
		for _, c := range charges {
			ids = append(ids, c.ID)
		}
		_ = s.UnbilledChargeRepo.MarkAsInvoiced(ids)
	}

	return inv, nil
}

// GenerateAdvanceInvoice generates an invoice for N future periods immediately.
// It extends the subscription's CurrentPeriodEnd.
func (s *InvoiceService) GenerateAdvanceInvoice(ctx context.Context, subID uuid.UUID, periods int) (*domain.Invoice, error) {
	sub, err := s.SubscriptionRepo.GetByID(ctx, subID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	plan, err := s.PlanRepo.GetByID(ctx, sub.PlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}

	if len(plan.Prices) == 0 {
		return nil, fmt.Errorf("plan has no prices")
	}
	price := plan.Prices[0]

	// Calculate Advance Amount
	advanceAmount := price.Amount * int64(periods)

	subtotal := advanceAmount

	// Tax calculation (matching GenerateInvoice)
	customer, err := s.CustomerRepo.GetByID(ctx, sub.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}

	taxEngine := tax.NewGSTEngine("TN")
	pos := customer.PlaceOfSupply
	if domain.PtrToString(pos) == "" {
		pos = nil
	}
	taxRes := taxEngine.CalculateTax(subtotal, domain.PtrToString(pos))
	total := subtotal + taxRes.Total

	now := time.Now()
	terms := sub.PaymentTerms
	if terms == "" {
		terms = "net0"
	}
	dueDate := domain.CalculateDueDate(now, terms)

	advInvID := uuid.New()
	inv := &domain.Invoice{
		ID:             advInvID,
		TenantID:       sub.TenantID,
		SubscriptionID: &sub.ID,
		CustomerID:     sub.CustomerID,
		InvoiceNumber:  fmt.Sprintf("INV-ADV-%d-%s", now.UnixNano(), advInvID.String()[:8]),
		Status:         domain.InvoiceStatusOpen,
		Currency:       price.Currency,
		Subtotal:       subtotal,
		TaxAmount:      taxRes.Total,
		Total:          total,
		IGSTAmount:     taxRes.IGST,
		CGSTAmount:     taxRes.CGST,
		SGSTAmount:     taxRes.SGST,
		CreatedAt:      now,
		DueDate:        dueDate,
		PaymentTerms:   terms,
	}

	if err := s.InvoiceRepo.Create(ctx, inv); err != nil {
		return nil, err
	}

	// Update Subscription Period using plan's interval unit and count
	newEndDate := sub.CurrentPeriodEnd
	if newEndDate.Before(now) {
		newEndDate = now
	}
	for i := 0; i < periods; i++ {
		newEndDate = domain.AddInterval(newEndDate, string(plan.IntervalUnit), plan.IntervalCount)
	}

	sub.CurrentPeriodEnd = newEndDate
	if err := s.SubscriptionRepo.Update(ctx, sub); err != nil {
		slog.Warn("failed to update subscription period after advance invoice", "error", err, "subscription_id", sub.ID)
	}

	return inv, nil
}
