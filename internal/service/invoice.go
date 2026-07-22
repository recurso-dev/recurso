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
	EInvoiceService    *EInvoiceService              // P25: E-invoice service (India IRN)
	EUEInvoiceService  *EUEInvoiceService            // Track C: EU e-invoicing (EN 16931/UBL); nil-safe, opt-in
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
	// Usage-based billing v1 (spec_usage_billing.md): plan charges are rated
	// into metered invoice lines at period close. All three are nil-safe —
	// when unset, invoices are byte-identical to pre-metering behaviour.
	ChargeRepo port.ChargeRepository
	UsageRepo  port.UsageRepository
	RatingRepo port.UsageRatingRepository
	// WalletDrainer applies prepaid wallet balance to a committed invoice
	// BEFORE adjustment credit notes and the gateway (Lago-parity B1, D3).
	// nil-safe. Satisfied by *WalletService.
	WalletDrainer walletDrainer
	// Progressive billing (A5). Both nil-safe: when ProgressiveRepo is unset, no
	// subscription is progressive and every path is byte-identical to before.
	// LedgerPoster lets billProgressive post its interim invoice's ledger legs
	// itself (DR AR / CR Revenue) — satisfied by *LedgerService.
	ProgressiveRepo port.ProgressiveBillingRepository
	LedgerPoster    invoiceLedgerPoster
}

// invoiceLedgerPoster is the slice of *LedgerService that billProgressive needs
// to post an interim invoice's legs. Kept narrow so the metering/invoice layer
// does not depend on the concrete ledger service.
type invoiceLedgerPoster interface {
	RecordInvoice(ctx context.Context, invoice *domain.Invoice) error
}

// SetProgressiveBilling wires progressive billing (A5): the watermark repo and
// the ledger poster interim invoices use. nil-safe / optional.
func (s *InvoiceService) SetProgressiveBilling(repo port.ProgressiveBillingRepository, ledger invoiceLedgerPoster) {
	s.ProgressiveRepo = repo
	s.LedgerPoster = ledger
}

// recordInvoiceLeg posts the invoice's base AR→Revenue/Deferred ledger leg (the
// Code-1 the reconciler expects, one per non-draft invoice) via the optional
// ledger poster — BEFORE any reducing legs (wallet drain, credit application)
// relieve AR. Every other invoice-creating flow posts this same leg
// (subscription create/proration/trial, mandate debits, progressive interim);
// the renewal/final-usage/advance generators did NOT, so those invoices carried
// no Code-1 and their deferred revenue was never funded — rev-rec then drained
// Deferred that was never credited, pushing the liability negative.
//
// nil-safe: without a wired ledger it is a no-op. A write failure is logged and
// left for reconciliation rather than failing invoice generation, matching the
// create/proration/mandate paths.
func (s *InvoiceService) recordInvoiceLeg(ctx context.Context, inv *domain.Invoice) {
	if s.LedgerPoster == nil || inv == nil {
		return
	}
	if err := s.LedgerPoster.RecordInvoice(ctx, inv); err != nil {
		slog.Error("ledger write failed on invoice generation — needs reconciliation",
			"invoice_id", inv.ID, "error", err)
	}
}

// walletDrainer consumes prepaid balance against a committed invoice.
type walletDrainer interface {
	DrainForInvoice(ctx context.Context, inv *domain.Invoice) (int64, error)
}

// creditApplier applies open adjustment credit-note balances to an invoice,
// returning the amount applied (ENG-153). Satisfied by *db.CreditNoteRepository.
type creditApplier interface {
	ApplyAdjustmentCredits(ctx context.Context, tenantID, customerID uuid.UUID, entityID *uuid.UUID, currency string, invoiceID uuid.UUID, invoiceTotal int64) (int64, error)
	SumApplicableAdjustments(ctx context.Context, tenantID, customerID uuid.UUID, entityID *uuid.UUID, currency string) (int64, error)
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

// generateEUEInvoiceAfterCommit generates the EN 16931 (UBL) e-invoice for a
// COMMITTED invoice when the tenant has opted into EU e-invoicing. Best-effort
// and fully nil-safe: unset service or a tenant that hasn't opted in is a no-op,
// so behaviour is byte-identical for everyone else. Separate from the India IRN
// path — the two regional regimes never interact.
func (s *InvoiceService) generateEUEInvoiceAfterCommit(ctx context.Context, inv *domain.Invoice, customer *domain.Customer) {
	if s.EUEInvoiceService == nil {
		return
	}
	if _, err := s.EUEInvoiceService.GenerateForInvoice(ctx, inv, customer); err != nil {
		slog.Warn("eu e-invoice generation failed (stored for retry)", "error", err, "invoice_id", inv.ID)
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

	// Usage-based billing v1: rate the subscription's plan charges over the
	// elapsed billing period into metered lines (arrears, D3). The window is
	// the subscription's current period AT GENERATION TIME — the cycle
	// generator runs at period end, before the period advances. Each line
	// flows through newInvoiceLine and the same per-line tax resolution, so
	// the Σ-line == subtotal and GST invariants hold by construction.
	var ratings []*domain.UsageRating
	if s.ChargeRepo != nil && s.UsageRepo != nil {
		metered := s.meteredLines(ctx, sub, customer, plan, price.Currency, invID)
		for _, ml := range metered {
			subtotal += ml.item.Amount
			taxTotal += ml.tax.Total
			igst += ml.tax.IGST
			cgst += ml.tax.CGST
			sgst += ml.tax.SGST
			lines = append(lines, ml.item)
			if ml.rating != nil { // filtered charges carry one claim across several lines
				ratings = append(ratings, ml.rating)
			}
		}
	}

	// Lago-parity B2: minimum-commitment true-up. When the period's
	// subtotal (flat + charges + add-ons + metered usage) falls short of
	// the committed floor, a shortfall line fills exactly the difference —
	// taxed at the plan HSN like the base fee. At-or-above commitment adds
	// nothing; usage past the floor bills normally (overage needs no code).
	if sub.CommitmentAmount > 0 && subtotal < sub.CommitmentAmount {
		shortfall := sub.CommitmentAmount - subtotal
		subtotal += shortfall

		trueUpTax := s.TaxResolver.ResolveInvoiceTax(ctx, sub.TenantID, customer, price.Currency, shortfall, plan.HSNCode)
		taxTotal += trueUpTax.Total
		igst += trueUpTax.IGST
		cgst += trueUpTax.CGST
		sgst += trueUpTax.SGST

		lines = append(lines, newInvoiceLine(invID, "Minimum commitment true-up", trueUpTax.HSN, 1, shortfall, shortfall, trueUpTax, time.Time{}))
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
		EntityID:       sub.EntityID,
		SubscriptionID: &sub.ID,
		CustomerID:     sub.CustomerID,
		InvoiceNumber:  fmt.Sprintf("INV-%d-%s", now.UnixNano(), invID.String()[:8]),
		BillingReason:  domain.BillingReasonSubscriptionCycle,
		Status:         domain.InvoiceStatusOpen,
		Currency:       price.Currency,
		Subtotal:       subtotal,
		TaxAmount:      taxTotal,
		// Invoice-level tax type from the base charge — all lines for a customer
		// share the resolved treatment (D3c). Persisted for the liability report.
		TaxType: baseTax.TaxType,
		Total:   total,

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

	// Post the base AR→Deferred ledger leg on the just-committed invoice, before
	// the wallet/credit reducing legs below relieve AR (see recordInvoiceLeg).
	s.recordInvoiceLeg(ctx, inv)

	// P25 e-invoicing on the now-committed invoice.
	s.generateEInvoiceAfterCommit(ctx, inv, customer)
	s.generateEUEInvoiceAfterCommit(ctx, inv, customer)

	// 6a. Drain the customer's prepaid wallet FIRST (Lago-parity B1, D3
	// ordering: wallet → adjustment credit notes → gateway). The drain is a
	// payment from stored value: AmountPaid rises and the ledger books
	// DR Customer Credit / CR AR inside the drainer. Best-effort.
	if s.WalletDrainer != nil && inv.Total > 0 {
		if drained, err := s.WalletDrainer.DrainForInvoice(ctx, inv); err != nil {
			slog.Error("wallet drain failed on invoice generation", "invoice_id", inv.ID, "error", err)
		} else if drained > 0 {
			inv.AmountPaid += drained
			inv.AmountDue = inv.Total - inv.AmountPaid - inv.CreditApplied
			if inv.AmountPaid+inv.CreditApplied >= inv.Total {
				inv.Status = domain.InvoiceStatusPaid
			}
			if err := s.InvoiceRepo.Update(ctx, inv); err != nil {
				slog.Error("failed to persist wallet application", "invoice_id", inv.ID, "error", err)
			}
			slog.Info("applied wallet balance to invoice", "invoice_id", inv.ID, "wallet_applied", drained)
		}
	}

	// 6b. Apply any account credit (adjustment credit notes) to the new invoice,
	// reducing what the customer owes (ENG-153). Best-effort: a failure leaves the
	// invoice at its full amount (recoverable), never fails invoice generation.
	// The applicable ceiling is what remains AFTER the wallet drain.
	if s.CreditApplier != nil && inv.Total-inv.AmountPaid > 0 {
		if applied, err := s.CreditApplier.ApplyAdjustmentCredits(ctx, inv.TenantID, inv.CustomerID, inv.EntityID, inv.Currency, inv.ID, inv.Total-inv.AmountPaid); err != nil {
			slog.Error("credit application failed on invoice generation", "invoice_id", inv.ID, "error", err)
		} else if applied > 0 {
			inv.CreditApplied = applied
			inv.AmountDue = inv.Total - inv.AmountPaid - applied
			if inv.AmountPaid+applied >= inv.Total {
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

	// Claim the rated usage windows on the now-committed invoice. A claim
	// that conflicts means another generation billed the same window
	// concurrently — loud alarm, since this invoice then double-bills it.
	// (The pre-check in meteredLines makes the plain retry path skip cleanly;
	// this is the last-resort race detector.)
	if s.RatingRepo != nil {
		for _, rating := range ratings {
			claimed, err := s.RatingRepo.Create(ctx, rating)
			if err != nil {
				slog.Error("failed to persist usage rating claim", "invoice_id", inv.ID, "charge_id", rating.ChargeID, "error", err)
			} else if !claimed {
				slog.Error("usage window was rated concurrently — possible double bill",
					"invoice_id", inv.ID, "charge_id", rating.ChargeID, "period_start", rating.PeriodStart)
			}
		}
	}

	return inv, nil
}

// GenerateFinalUsageInvoice bills the metered charges of a just-canceled
// subscription over the partial elapsed window [CurrentPeriodStart, endedAt)
// — the flat fee was already billed in advance at period start; only usage
// is outstanding (spec_usage_billing.md, final invoice). Returns (nil, nil)
// when metering is not wired or the window rates to no lines. The rating
// claim on period_start also blocks any later full-period rating of the
// same window, so a cancel-then-regenerate cannot double-bill.
func (s *InvoiceService) GenerateFinalUsageInvoice(ctx context.Context, sub *domain.Subscription, endedAt time.Time) (*domain.Invoice, error) {
	if s.ChargeRepo == nil || s.UsageRepo == nil {
		return nil, nil
	}

	plan, err := s.PlanRepo.GetByID(ctx, sub.PlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}
	customer, err := s.CustomerRepo.GetByID(ctx, sub.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}
	if len(plan.Prices) == 0 {
		return nil, fmt.Errorf("plan has no prices")
	}
	price := plan.Prices[0]

	invID := uuid.New()

	// Rate over the truncated window by presenting a copy whose period end
	// is the cancellation time.
	windowSub := *sub
	windowSub.CurrentPeriodEnd = endedAt
	metered := s.meteredLines(ctx, &windowSub, customer, plan, price.Currency, invID)
	if len(metered) == 0 {
		return nil, nil
	}

	var subtotal, taxTotal, igst, cgst, sgst int64
	lines := make([]domain.InvoiceItem, 0, len(metered))
	ratings := make([]*domain.UsageRating, 0, len(metered))
	for _, ml := range metered {
		subtotal += ml.item.Amount
		taxTotal += ml.tax.Total
		igst += ml.tax.IGST
		cgst += ml.tax.CGST
		sgst += ml.tax.SGST
		lines = append(lines, ml.item)
		if ml.rating != nil { // filtered charges carry one claim across several lines
			ratings = append(ratings, ml.rating)
		}
	}

	now := time.Now()
	terms := sub.PaymentTerms
	if terms == "" {
		terms = "net0"
	}
	// Invoice-level tax type from the metered lines — all share the resolved
	// treatment (D3c). Persisted for the liability report.
	usageTaxType := ""
	for _, ml := range metered {
		if ml.tax.TaxType != "" {
			usageTaxType = ml.tax.TaxType
			break
		}
	}
	inv := &domain.Invoice{
		ID:             invID,
		TenantID:       sub.TenantID,
		EntityID:       sub.EntityID,
		SubscriptionID: &sub.ID,
		CustomerID:     sub.CustomerID,
		InvoiceNumber:  fmt.Sprintf("INV-%d-%s", now.UnixNano(), invID.String()[:8]),
		BillingReason:  domain.BillingReasonSubscriptionCycle,
		Status:         domain.InvoiceStatusOpen,
		Currency:       price.Currency,
		Subtotal:       subtotal,
		TaxAmount:      taxTotal,
		TaxType:        usageTaxType,
		Total:          subtotal + taxTotal,
		IGSTAmount:     igst,
		CGSTAmount:     cgst,
		SGSTAmount:     sgst,
		LineItems:      lines,
		CreatedAt:      now,
		DueDate:        domain.CalculateDueDate(now, terms),
		PaymentTerms:   terms,
	}

	if err := s.InvoiceRepo.Create(ctx, inv); err != nil {
		return nil, fmt.Errorf("failed to create final usage invoice: %w", err)
	}

	// Base AR→Deferred ledger leg for the final-usage invoice (see recordInvoiceLeg).
	s.recordInvoiceLeg(ctx, inv)

	s.generateEInvoiceAfterCommit(ctx, inv, customer)
	s.generateEUEInvoiceAfterCommit(ctx, inv, customer)

	if s.RatingRepo != nil {
		for _, rating := range ratings {
			claimed, err := s.RatingRepo.Create(ctx, rating)
			if err != nil {
				slog.Error("failed to persist usage rating claim", "invoice_id", inv.ID, "charge_id", rating.ChargeID, "error", err)
			} else if !claimed {
				slog.Error("usage window was rated concurrently — possible double bill",
					"invoice_id", inv.ID, "charge_id", rating.ChargeID, "period_start", rating.PeriodStart)
			}
		}
	}
	return inv, nil
}

// meteredLine is one rated plan charge: the invoice line, the per-line tax
// already summed into the invoice totals by the caller, and the rating
// claim to persist after the invoice commits.
type meteredLine struct {
	item   domain.InvoiceItem
	tax    InvoiceTax
	rating *domain.UsageRating
}

// meteredLines aggregates and prices every plan charge for the elapsed
// window [sub.CurrentPeriodStart, sub.CurrentPeriodEnd). Per charge:
//
//   - already-rated windows are skipped (idempotent retry),
//   - zero usage emits no line (and no claim, so late-arriving events for
//     the window can still bill on a legitimate later generation),
//   - a missing currency entry or rating error skips the charge with a
//     warning, mirroring how currency-mismatched add-ons are handled.
//
// The line carries Quantity 1 with the usage count in the description:
// sub-minor-unit rates (₹0.0035/call) have no representable int64 unit
// price, and quantity 1 × unitAmount == amount keeps the line-math
// invariant every existing renderer assumes.
func (s *InvoiceService) meteredLines(ctx context.Context, sub *domain.Subscription, customer *domain.Customer, plan *domain.Plan, currency string, invID uuid.UUID) []meteredLine {
	charges, err := s.ChargeRepo.ListByPlan(ctx, sub.TenantID, sub.PlanID)
	if err != nil {
		slog.Warn("skipping metered lines: charge list failed", "error", err, "subscription_id", sub.ID)
		return nil
	}

	periodStart, periodEnd := sub.CurrentPeriodStart, sub.CurrentPeriodEnd
	cur := strings.ToUpper(currency)
	now := time.Now()

	// A5: on a progressive subscription, eligible charges bill incrementally via
	// the watermark; at period close their FINAL delta (rate(period_end) minus
	// what interims already billed) settles here as a line on the renewal
	// invoice — the double-billing guard is the watermark CAS, not usage_ratings.
	progressive := s.isProgressive(ctx, sub.ID)

	var out []meteredLine
	for _, ch := range charges {
		if ch.Metric == nil {
			continue
		}
		// Pay-in-advance charges are rated per event at ingestion time and
		// captured as unbilled charges (folded onto this invoice above); never
		// re-bill them at period close (A3).
		if ch.PayInAdvance {
			continue
		}
		// Progressive settle for eligible charges (A5). Non-eligible (volume)
		// charges on a progressive subscription fall through to the classic path.
		if progressive && domain.ProgressiveBillingEligible(ch.ChargeModel) {
			if ml, ok := s.progressiveCloseLine(ctx, sub, customer, plan, currency, cur, invID, ch, periodStart, periodEnd, now); ok {
				out = append(out, ml)
			}
			continue
		}
		if s.RatingRepo != nil {
			rated, err := s.RatingRepo.Exists(ctx, sub.ID, ch.ID, periodStart)
			if err != nil {
				slog.Warn("skipping metered charge: rating check failed", "error", err, "charge_id", ch.ID)
				continue
			}
			if rated {
				continue // window already billed (retried generation)
			}
		}

		// A4: a filtered charge prices distinct values of one event property
		// separately — one line per value plus a default line, under a single
		// per-charge rating claim (the double-billing guard is unchanged).
		if ch.FilterKey != "" {
			out = append(out, s.filteredMeteredLines(ctx, sub, customer, plan, currency, cur, invID, ch, periodStart, periodEnd, now)...)
			continue
		}

		amounts, ok := ch.Amounts[cur]
		if !ok {
			slog.Warn("skipping metered charge without pricing for invoice currency",
				"charge_id", ch.ID, "metric", ch.Metric.Code, "invoice_currency", cur)
			continue
		}

		qtyRat, err := meteredQuantity(ctx, s.UsageRepo, sub.ID, ch, periodStart, periodEnd)
		if err != nil {
			slog.Warn("skipping metered charge: aggregation failed", "error", err, "metric", ch.Metric.Code)
			continue
		}
		if qtyRat.Sign() == 0 {
			continue
		}

		amount, err := RateChargeRat(ch.ChargeModel, amounts, qtyRat)
		if err != nil {
			slog.Warn("skipping metered charge: rating failed", "error", err, "metric", ch.Metric.Code)
			continue
		}

		// Persisted/displayed quantity is the exact aggregate rounded to a whole
		// unit; the amount above is priced from the exact rational, not this.
		qty := roundRatHalfUp(qtyRat)
		hsn := ch.HSNCode
		if hsn == "" {
			hsn = plan.HSNCode
		}
		tax := s.TaxResolver.ResolveInvoiceTax(ctx, sub.TenantID, customer, currency, amount, hsn)
		desc := fmt.Sprintf("%s — %d × usage (%s to %s)",
			ch.Metric.Name, qty, periodStart.Format("2 Jan 2006"), periodEnd.Format("2 Jan 2006"))

		out = append(out, meteredLine{
			item: newInvoiceLine(invID, desc, tax.HSN, 1, amount, amount, tax, time.Time{}),
			tax:  tax,
			rating: &domain.UsageRating{
				ID:             uuid.New(),
				TenantID:       sub.TenantID,
				SubscriptionID: sub.ID,
				ChargeID:       ch.ID,
				PeriodStart:    periodStart,
				PeriodEnd:      periodEnd,
				InvoiceID:      invID,
				Quantity:       qty,
				Amount:         amount,
				CreatedAt:      now,
			},
		})
	}
	return out
}

// filteredMeteredLines rates a dimensional-pricing charge (A4): one line per
// configured filter value (events whose FilterKey property equals the value,
// at the value's amounts) plus a default line for events matching no value (at
// the charge's base amounts). Each line aggregates its own event subset. The
// whole charge carries ONE rating claim (attached to the first line) with the
// summed quantity/amount, so the per-(subscription, charge, period)
// double-billing guard is unchanged.
func (s *InvoiceService) filteredMeteredLines(ctx context.Context, sub *domain.Subscription, customer *domain.Customer, plan *domain.Plan, currency, cur string, invID uuid.UUID, ch domain.Charge, periodStart, periodEnd, now time.Time) []meteredLine {
	hsn := ch.HSNCode
	if hsn == "" {
		hsn = plan.HSNCode
	}

	var lines []meteredLine
	var totalQty, totalAmount int64

	rateSubset := func(amounts domain.ChargeAmounts, values []string, exclude bool, label string) {
		qty, err := s.UsageRepo.AggregateForMetricFiltered(ctx, sub.ID, *ch.Metric, ch.FilterKey, values, exclude, periodStart, periodEnd)
		if err != nil {
			slog.Warn("skipping filtered charge subset: aggregation failed", "error", err, "charge_id", ch.ID, "filter", label)
			return
		}
		if qty == 0 {
			return
		}
		amount, err := RateCharge(ch.ChargeModel, amounts, qty)
		if err != nil {
			slog.Warn("skipping filtered charge subset: rating failed", "error", err, "charge_id", ch.ID, "filter", label)
			return
		}
		tax := s.TaxResolver.ResolveInvoiceTax(ctx, sub.TenantID, customer, currency, amount, hsn)
		desc := fmt.Sprintf("%s [%s] — %d × usage (%s to %s)",
			ch.Metric.Name, label, qty, periodStart.Format("2 Jan 2006"), periodEnd.Format("2 Jan 2006"))
		lines = append(lines, meteredLine{
			item: newInvoiceLine(invID, desc, tax.HSN, 1, amount, amount, tax, time.Time{}),
			tax:  tax,
		})
		totalQty += qty
		totalAmount += amount
	}

	values := make([]string, 0, len(ch.Filters))
	for _, f := range ch.Filters {
		values = append(values, f.Value)
		if amounts, ok := f.Amounts[cur]; ok {
			rateSubset(amounts, []string{f.Value}, false, ch.FilterKey+"="+f.Value)
		}
	}
	// default subset: events whose property matches none of the filter values.
	if baseAmounts, ok := ch.Amounts[cur]; ok {
		rateSubset(baseAmounts, values, true, ch.FilterKey+": other")
	}

	// One claim per charge per period, attached to the first line.
	if len(lines) > 0 {
		lines[0].rating = &domain.UsageRating{
			ID:             uuid.New(),
			TenantID:       sub.TenantID,
			SubscriptionID: sub.ID,
			ChargeID:       ch.ID,
			PeriodStart:    periodStart,
			PeriodEnd:      periodEnd,
			InvoiceID:      invID,
			Quantity:       totalQty,
			Amount:         totalAmount,
			CreatedAt:      now,
		}
	}
	return lines
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
		EntityID:       sub.EntityID,
		SubscriptionID: &sub.ID,
		CustomerID:     sub.CustomerID,
		InvoiceNumber:  fmt.Sprintf("INV-ADV-%d-%s", now.UnixNano(), advInvID.String()[:8]),
		BillingReason:  domain.BillingReasonSubscriptionCycle,
		Status:         domain.InvoiceStatusOpen,
		Currency:       price.Currency,
		Subtotal:       subtotal,
		TaxAmount:      taxRes.Total,
		TaxType:        taxRes.TaxType, // D3c: persist for the liability report
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

	// Base AR→Deferred ledger leg for the advance invoice (see recordInvoiceLeg).
	s.recordInvoiceLeg(ctx, inv)

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
