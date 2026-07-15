package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/telemetry"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
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
	// AddonRepo enables multi-product add-on lines on recurring invoices
	// (Multi-product catalog v1). nil-safe: when unset, GenerateInvoice
	// produces byte-identical single-plan invoices.
	AddonRepo port.SubscriptionAddonRepository
	Telemetry *telemetry.Client // nil-safe; only set when TELEMETRY_OPTIN=true
	// CreditApplier applies a customer's adjustment credit-note balances to a
	// freshly-created invoice (ENG-153). nil-safe: when unset, invoices bill the
	// full amount and no credit is consumed.
	CreditApplier creditApplier
}

// creditApplier applies open adjustment credit-note balances to an invoice,
// returning the amount applied (ENG-153). Satisfied by *db.CreditNoteRepository.
type creditApplier interface {
	ApplyAdjustmentCredits(ctx context.Context, tenantID, customerID uuid.UUID, currency string, invoiceID uuid.UUID, invoiceTotal int64) (int64, error)
	SumApplicableAdjustments(ctx context.Context, tenantID, customerID uuid.UUID, currency string) (int64, error)
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

// generateEInvoiceAfterCommit registers the government e-invoice (IRN) for a
// COMMITTED invoice and persists the result. It must run only after the invoice
// is durably committed — an IRN is irreversible, so requesting it before commit
// orphans it if the insert rolls back (PHASE2 #3). Best-effort.
func (s *InvoiceService) generateEInvoiceAfterCommit(ctx context.Context, inv *domain.Invoice, customer *domain.Customer) {
	switch {
	case s.EInvoiceService != nil:
		if _, err := s.EInvoiceService.GenerateEInvoice(ctx, inv); err != nil {
			slog.Error("e-invoice generation failed (will retry)", "error", err, "invoice_id", inv.ID)
		}
	case s.GSPAdapter != nil && customer.BillingAddress.Country == "India" && domain.PtrToString(customer.GSTIN) != "" && customer.TaxType == "business":
		resp, err := s.GSPAdapter.GenerateIRN(ctx, inv)
		if err == nil {
			inv.IRN = resp.IRN
			inv.SignedQRCode = resp.SignedQRCode
			inv.EInvoiceStatus = "GENERATED"
			inv.AckNo = resp.AckNo
		} else {
			slog.Error("error generating IRN", "error", err, "invoice_id", inv.ID)
			inv.EInvoiceStatus = "FAILED"
		}
	default:
		inv.EInvoiceStatus = "NA"
	}
	// Persist the IRN/status (and any retry scheduling) onto the committed row.
	if err := s.InvoiceRepo.Update(ctx, inv); err != nil {
		slog.Error("failed to persist e-invoice result", "invoice_id", inv.ID, "error", err)
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

	// Invoice id is created up front so line items can reference it as they are
	// accumulated below.
	invID := uuid.New()

	// 3. Calculate Amounts. The base plan is its own line; each billable
	// unbilled charge is its own line too (Phase 3), taxed at its own HSN.
	subtotal := price.Amount

	// Base line: plan price only, taxed on the plan price at the plan's HSN.
	baseTax := s.TaxResolver.ResolveInvoiceTax(ctx, sub.TenantID, customer, price.Currency, price.Amount, plan.HSNCode)

	// Running invoice totals seeded from the base line. Charges and add-ons are
	// accumulated onto these as their own lines below.
	taxTotal, igst, cgst, sgst := baseTax.Total, baseTax.IGST, baseTax.CGST, baseTax.SGST

	baseDesc := plan.Name
	if baseDesc == "" {
		baseDesc = "Subscription"
	}
	lines := []domain.InvoiceItem{
		newInvoiceLine(invID, baseDesc, baseTax.HSN, 1, price.Amount, price.Amount, baseTax, time.Time{}),
	}

	// P15: Add Unbilled Charges as their own line items (Phase 3). Charges in a
	// different currency than the plan price cannot be billed on this invoice;
	// they stay unbilled. Each billable charge is taxed independently at its own
	// HSN (falling back to the tenant SAC when unset), mirroring add-ons.
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

			chargeTax := s.TaxResolver.ResolveInvoiceTax(ctx, sub.TenantID, customer, price.Currency, c.Amount, c.HSNCode)
			taxTotal += chargeTax.Total
			igst += chargeTax.IGST
			cgst += chargeTax.CGST
			sgst += chargeTax.SGST

			chargeDesc := c.Description
			if chargeDesc == "" {
				chargeDesc = "One-time charge"
			}
			lines = append(lines, newInvoiceLine(invID, chargeDesc, chargeTax.HSN, 1, c.Amount, c.Amount, chargeTax, time.Time{}))
		}
	}

	// Multi-product catalog v1: each add-on attached to the subscription is
	// billed as its own line — the add-on plan's price × quantity — taxed
	// independently through the same resolver, then summed onto the base
	// invoice. Currency-mismatched or unresolvable add-ons are skipped (never
	// summed into a different-currency invoice), mirroring unbilled charges.
	if s.AddonRepo != nil {
		addons, addonErr := s.AddonRepo.ListBySubscriptionID(ctx, sub.TenantID, sub.ID)
		if addonErr != nil {
			slog.Warn("skipping subscription add-ons: list failed",
				"error", addonErr, "subscription_id", sub.ID)
		}
		for _, a := range addons {
			addonPlan, planErr := s.PlanRepo.GetByID(ctx, a.PlanID)
			if planErr != nil || addonPlan == nil || len(addonPlan.Prices) == 0 {
				slog.Warn("skipping add-on: plan unavailable",
					"error", planErr, "add_on_id", a.ID, "add_on_plan_id", a.PlanID)
				continue
			}
			addonPrice := addonPlan.Prices[0]
			if addonPrice.Currency != "" && !strings.EqualFold(addonPrice.Currency, price.Currency) {
				slog.Warn("skipping add-on with mismatched currency",
					"add_on_id", a.ID, "add_on_currency", addonPrice.Currency, "invoice_currency", price.Currency)
				continue
			}
			lineAmount := addonPrice.Amount * int64(a.Quantity)
			subtotal += lineAmount

			// Tax each add-on line independently to avoid rounding drift from
			// taxing the aggregate, and to keep the base line untouched.
			lineTax := s.TaxResolver.ResolveInvoiceTax(ctx, sub.TenantID, customer, price.Currency, lineAmount, addonPlan.HSNCode)
			taxTotal += lineTax.Total
			igst += lineTax.IGST
			cgst += lineTax.CGST
			sgst += lineTax.SGST

			// Itemization (Phase 1): record this add-on as its own line, with the
			// same per-line tax that was just summed into the totals.
			addonDesc := addonPlan.Name
			if addonDesc == "" {
				addonDesc = "Add-on"
			}
			lines = append(lines, newInvoiceLine(invID, addonDesc, lineTax.HSN, a.Quantity, addonPrice.Amount, lineAmount, lineTax, time.Time{}))
		}
	}

	total := subtotal + taxTotal

	// 4. Determine Payment Terms & Due Date (P15)
	terms := sub.PaymentTerms
	if terms == "" {
		terms = "net0"
	}

	now := time.Now()
	dueDate := domain.CalculateDueDate(now, terms)

	// 5. Create Invoice
	inv := &domain.Invoice{
		ID:             invID,
		TenantID:       sub.TenantID,
		SubscriptionID: &sub.ID,
		CustomerID:     sub.CustomerID,
		InvoiceNumber:  fmt.Sprintf("INV-%d-%s", now.UnixNano(), invID.String()[:8]),
		BillingReason:  domain.BillingReasonSubscriptionCycle,
		Status:         domain.InvoiceStatusOpen,
		Currency:       price.Currency,
		Subtotal:       subtotal,
		TaxAmount:      taxTotal,
		Total:          total,

		IGSTAmount: igst,
		CGSTAmount: cgst,
		SGSTAmount: sgst,
		// HSNCode?

		LineItems: lines,

		CreatedAt:    now,
		DueDate:      dueDate,
		PaymentTerms: terms,
		RetryCount:   0,
	}

	// 6. Persist FIRST — P25 e-invoicing runs after commit (below) so a failed
	// insert can't orphan an irreversible government IRN (PHASE2 #3).
	if err := s.InvoiceRepo.Create(ctx, inv); err != nil {
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	// P25 e-invoicing on the now-committed invoice.
	s.generateEInvoiceAfterCommit(ctx, inv, customer)

	// 6b. Apply any account credit (adjustment credit notes) to the new invoice,
	// reducing what the customer owes (ENG-153). Best-effort: a failure leaves the
	// invoice at its full amount (recoverable), never fails invoice generation.
	if s.CreditApplier != nil && inv.Total > 0 {
		if applied, err := s.CreditApplier.ApplyAdjustmentCredits(ctx, inv.TenantID, inv.CustomerID, inv.Currency, inv.ID, inv.Total); err != nil {
			slog.Error("credit application failed on invoice generation", "invoice_id", inv.ID, "error", err)
		} else if applied > 0 {
			inv.CreditApplied = applied
			inv.AmountDue = inv.Total - inv.AmountPaid - applied
			if applied >= inv.Total {
				inv.Status = domain.InvoiceStatusPaid
			}
			slog.Info("applied account credit to invoice", "invoice_id", inv.ID, "credit_applied", applied)
		}
	}

	s.Telemetry.MilestoneFirstInvoice() // opt-in anonymous milestone; no-op when disabled

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
// maxAdvancePeriods caps how many billing periods can be pre-charged in one
// advance invoice. It bounds both the invoice amount (price * periods stays
// clear of int64 overflow) and the O(periods) period-extension loop below, so a
// typo (periods: 1000) can neither over-charge a customer nor hang the request.
const maxAdvancePeriods = 60

func (s *InvoiceService) GenerateAdvanceInvoice(ctx context.Context, subID uuid.UUID, periods int) (*domain.Invoice, error) {
	if periods < 1 || periods > maxAdvancePeriods {
		return nil, fmt.Errorf("periods must be between 1 and %d", maxAdvancePeriods)
	}

	sub, err := s.SubscriptionRepo.GetByID(ctx, subID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}
	if sub == nil {
		return nil, fmt.Errorf("subscription not found")
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

	taxRes := s.TaxResolver.ResolveInvoiceTax(ctx, sub.TenantID, customer, price.Currency, subtotal, plan.HSNCode)
	total := subtotal + taxRes.Total

	now := time.Now()
	terms := sub.PaymentTerms
	if terms == "" {
		terms = "net0"
	}
	dueDate := domain.CalculateDueDate(now, terms)

	advInvID := uuid.New()
	advDesc := plan.Name
	if advDesc == "" {
		advDesc = "Subscription"
	}
	inv := &domain.Invoice{
		ID:             advInvID,
		TenantID:       sub.TenantID,
		SubscriptionID: &sub.ID,
		CustomerID:     sub.CustomerID,
		InvoiceNumber:  fmt.Sprintf("INV-ADV-%d-%s", now.UnixNano(), advInvID.String()[:8]),
		BillingReason:  domain.BillingReasonSubscriptionCycle,
		Status:         domain.InvoiceStatusOpen,
		Currency:       price.Currency,
		Subtotal:       subtotal,
		TaxAmount:      taxRes.Total,
		Total:          total,
		IGSTAmount:     taxRes.IGST,
		CGSTAmount:     taxRes.CGST,
		SGSTAmount:     taxRes.SGST,
		LineItems: []domain.InvoiceItem{
			newInvoiceLine(advInvID, advDesc, taxRes.HSN, periods, price.Amount, subtotal, taxRes, time.Time{}),
		},
		CreatedAt:    now,
		DueDate:      dueDate,
		PaymentTerms: terms,
	}

	if err := s.InvoiceRepo.Create(ctx, inv); err != nil {
		return nil, err
	}

	s.Telemetry.MilestoneFirstInvoice() // opt-in anonymous milestone; no-op when disabled

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
