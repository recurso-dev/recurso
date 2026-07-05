package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

type InvoiceService struct {
	InvoiceRepo        port.InvoiceRepository
	PlanRepo           port.PlanRepository
	CustomerRepo       port.CustomerRepository
	UnbilledChargeRepo port.UnbilledChargeRepository // P15
	SubscriptionRepo   port.SubscriptionRepository   // P15
	GSPAdapter         port.GSPAdapter               // P25
	EInvoiceService    *EInvoiceService              // P25: E-invoice service
	TaxResolver        *TaxResolver                  // Jurisdiction-aware tax
}

func NewInvoiceService(
	invRepo port.InvoiceRepository,
	planRepo port.PlanRepository,
	custRepo port.CustomerRepository,
	ucRepo port.UnbilledChargeRepository,
	subRepo port.SubscriptionRepository,
	gspAdapter port.GSPAdapter,
	taxResolver *TaxResolver,
) *InvoiceService {
	if taxResolver == nil {
		// Env-default resolver (IN/TN) preserves historical behavior when no
		// resolver is wired.
		taxResolver = NewTaxResolver(nil, "", "")
	}
	return &InvoiceService{
		InvoiceRepo:        invRepo,
		PlanRepo:           planRepo,
		CustomerRepo:       custRepo,
		UnbilledChargeRepo: ucRepo,
		SubscriptionRepo:   subRepo,
		GSPAdapter:         gspAdapter,
		TaxResolver:        taxResolver,
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

	// P15: Add Unbilled Charges. Charges in a different currency than the
	// plan price cannot be summed into this invoice; they stay unbilled.
	charges, err := s.UnbilledChargeRepo.ListBySubscriptionID(sub.ID)
	var billableCharges []*domain.UnbilledCharge
	if err == nil {
		for _, c := range charges {
			if c.Currency != "" && !strings.EqualFold(c.Currency, price.Currency) {
				slog.Warn("skipping unbilled charge with mismatched currency",
					"charge_id", c.ID, "charge_currency", c.Currency, "invoice_currency", price.Currency)
				continue
			}
			billableCharges = append(billableCharges, c)
			subtotal += c.Amount
		}
	}

	// Jurisdiction-aware tax: tenant GST config (India) or env company
	// defaults decide the engine; buyer location decides the treatment.
	taxRes := s.TaxResolver.ResolveInvoiceTax(ctx, sub.TenantID, customer, price.Currency, subtotal)

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

	// P15: Mark Charges as Invoiced (only the ones actually billed)
	if len(billableCharges) > 0 {
		var ids []uuid.UUID
		for _, c := range billableCharges {
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

	taxRes := s.TaxResolver.ResolveInvoiceTax(ctx, sub.TenantID, customer, price.Currency, subtotal)
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
