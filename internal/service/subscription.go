package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/adapter/telemetry"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// Sentinel errors let handlers map service failures to the right HTTP status
// (e.g. 404) without brittle string matching.
var (
	ErrSubscriptionNotFound = errors.New("subscription not found")
	ErrPlanNotFound         = errors.New("plan not found")
	// ErrAddonNotFound is returned when an add-on does not exist for the
	// tenant/subscription, keeping tenant isolation opaque to callers.
	ErrAddonNotFound = errors.New("add-on not found")
	// ErrAddonCurrencyMismatch is returned when an add-on plan's price
	// currency differs from the subscription's base-plan currency; add-ons
	// must invoice in the same currency as the base line.
	ErrAddonCurrencyMismatch = errors.New("add-on plan currency does not match subscription currency")
	// ErrInvalidQuantity is returned when an add-on quantity is not positive.
	ErrInvalidQuantity = errors.New("quantity must be greater than 0")
	// ErrTrialAlreadyConverted means another runner won the atomic trial->active
	// transition first, so this caller did nothing. The trial scheduler treats it
	// as a benign skip, not a failure (ENG-161).
	ErrTrialAlreadyConverted = errors.New("trial already converted to active")
	// errTrialRaceLost is the internal signal returned from the conversion tx when
	// the conditional activate matched zero rows; it rolls the tx back and is
	// mapped to ErrTrialAlreadyConverted for callers.
	errTrialRaceLost = errors.New("trial conversion race lost")
)

type SubscriptionService struct {
	subRepo             port.SubscriptionRepository
	invoiceRepo         port.InvoiceRepository
	planRepo            port.PlanRepository
	customerRepo        port.CustomerRepository
	couponRepo          port.CouponRepository
	notifier            port.Notifier
	ledger              *LedgerService
	gateway             port.PaymentGateway
	gspAdapter          port.GSPAdapter
	notificationService *NotificationService
	einvoiceService     *EInvoiceService
	txManager           *db.TxManager
	subRepoImpl         *db.SubscriptionRepository // Concrete type for TX methods
	invRepoImpl         *db.InvoiceRepository      // Concrete type for TX methods
	creditNoteRepo      *db.CreditNoteRepository   // Downgrade proration credits (ENG-150); nil-safe
	creditApplier       creditApplier              // Apply account credit to charge invoices (ENG-154); nil-safe
	revrecService       *RevRecService
	taxResolver         *TaxResolver
	recoveryRecorder    PaymentRecoveryRecorder
	addonRepo           port.SubscriptionAddonRepository // Multi-product catalog v1; nil-safe (add-ons disabled)
	telemetry           *telemetry.Client                // nil-safe; only set when TELEMETRY_OPTIN=true
	finalUsageInvoicer  finalUsageInvoicer               // Usage-based billing v1: bill the partial window on immediate cancel; nil-safe
	logger              *slog.Logger
}

// finalUsageInvoicer bills a canceled subscription's metered usage for the
// partial elapsed window. Satisfied by *InvoiceService.
type finalUsageInvoicer interface {
	GenerateFinalUsageInvoice(ctx context.Context, sub *domain.Subscription, endedAt time.Time) (*domain.Invoice, error)
}

func NewSubscriptionService(
	subRepo port.SubscriptionRepository,
	invoiceRepo port.InvoiceRepository,
	planRepo port.PlanRepository,
	customerRepo port.CustomerRepository,
	couponRepo port.CouponRepository,
	notifier port.Notifier,
	ledger *LedgerService,
	gateway port.PaymentGateway,
	gspAdapter port.GSPAdapter,
	txManager *db.TxManager,
	revrecService *RevRecService,
	taxResolver *TaxResolver,
) *SubscriptionService {
	// Try to extract concrete types for TX methods
	var subImpl *db.SubscriptionRepository
	var invImpl *db.InvoiceRepository
	if sr, ok := subRepo.(*db.SubscriptionRepository); ok {
		subImpl = sr
	}
	if ir, ok := invoiceRepo.(*db.InvoiceRepository); ok {
		invImpl = ir
	}
	if taxResolver == nil {
		// Env-default resolver (IN/TN) preserves historical behavior when no
		// resolver is wired.
		taxResolver = NewTaxResolver(nil, "", "")
	}

	return &SubscriptionService{
		subRepo:         subRepo,
		invoiceRepo:     invoiceRepo,
		planRepo:        planRepo,
		customerRepo:    customerRepo,
		couponRepo:      couponRepo,
		notifier:        notifier,
		ledger:          ledger,
		gateway:         gateway,
		gspAdapter:      gspAdapter,
		einvoiceService: nil, // Set via SetEInvoiceService after construction
		txManager:       txManager,
		subRepoImpl:     subImpl,
		invRepoImpl:     invImpl,
		revrecService:   revrecService,
		taxResolver:     taxResolver,
		logger:          slog.Default().With("service", "subscription"),
	}
}

// SetCreditNoteRepo injects the credit-note repository used to persist downgrade
// proration credits as spendable adjustment credit notes (ENG-150). Nil-safe: if
// unset, a downgrade credit is logged rather than dropped.
func (s *SubscriptionService) SetCreditNoteRepo(r *db.CreditNoteRepository) {
	s.creditNoteRepo = r
}

// SetCreditApplier wires account-credit application into the proration-upgrade
// and trial-conversion charge invoices (ENG-154). Nil-safe.
func (s *SubscriptionService) SetCreditApplier(a creditApplier) { s.creditApplier = a }

// applyCreditToInvoice draws the customer's account credit against a just-created
// charge invoice, updating the in-memory struct to reflect it. Best-effort: a
// failure leaves the invoice at full amount. Shared by the proration-upgrade and
// trial-conversion paths (ENG-154).
func (s *SubscriptionService) applyCreditToInvoice(ctx context.Context, inv *domain.Invoice) {
	if s.creditApplier == nil || inv == nil || inv.Total <= 0 {
		return
	}
	applied, err := s.creditApplier.ApplyAdjustmentCredits(ctx, inv.TenantID, inv.CustomerID, inv.Currency, inv.ID, inv.Total)
	if err != nil {
		s.logger.Error("credit application failed", "invoice_id", inv.ID, "error", err)
		return
	}
	if applied > 0 {
		inv.CreditApplied = applied
		inv.AmountDue = inv.Total - inv.AmountPaid - applied
		if applied >= inv.Total {
			inv.Status = domain.InvoiceStatusPaid
		}
		s.logger.Info("applied account credit to charge invoice", "invoice_id", inv.ID, "credit_applied", applied)
	}
}

// SetEInvoiceService injects the EInvoiceService after construction (avoids circular deps).
func (s *SubscriptionService) SetEInvoiceService(einvoiceSvc *EInvoiceService) {
	s.einvoiceService = einvoiceSvc
}

// SetNotificationService injects the NotificationService after construction.
func (s *SubscriptionService) SetNotificationService(ns *NotificationService) {
	s.notificationService = ns
}

// SetRecoveryRecorder injects the dunning recovery recorder after construction.
func (s *SubscriptionService) SetRecoveryRecorder(rr PaymentRecoveryRecorder) {
	s.recoveryRecorder = rr
}

// SetTelemetry injects the opt-in anonymous telemetry client after construction.
func (s *SubscriptionService) SetTelemetry(t *telemetry.Client) { s.telemetry = t }

// SetFinalUsageInvoicer wires the metered final invoice on immediate cancel
// (usage-based billing v1); nil-safe when unset.
func (s *SubscriptionService) SetFinalUsageInvoicer(f finalUsageInvoicer) { s.finalUsageInvoicer = f }

// SetCommitment sets the subscription's per-period minimum (Lago-parity
// B2): when a period's subtotal falls short, a true-up line fills the gap
// on the renewal invoice. amount 0 clears the commitment.
func (s *SubscriptionService) SetCommitment(ctx context.Context, tenantID, subscriptionID uuid.UUID, amount int64) (*domain.Subscription, error) {
	if amount < 0 {
		return nil, fmt.Errorf("commitment amount must not be negative")
	}
	if s.subRepoImpl == nil {
		return nil, fmt.Errorf("commitment persistence not configured")
	}
	if err := s.subRepoImpl.SetCommitment(ctx, tenantID, subscriptionID, amount); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("subscription not found")
		}
		return nil, err
	}
	return s.GetByID(ctx, tenantID, subscriptionID)
}

// SetAddonRepository injects the subscription add-on repository after
// construction (Multi-product catalog v1). Left nil, the add-on service
// methods return ErrAddonNotFound / errors and the money path is unchanged.
func (s *SubscriptionService) SetAddonRepository(r port.SubscriptionAddonRepository) {
	s.addonRepo = r
}

type CreateSubscriptionInput struct {
	TenantID uuid.UUID
	// EntityID is the issuing legal entity (Multi-Entity Books). Nil ⇒ the
	// tenant's primary entity (backward-compatible default).
	EntityID          *uuid.UUID
	CustomerID        uuid.UUID
	PlanID            uuid.UUID
	StartDate         time.Time
	CouponCode        string
	BillingAnchorType string // "acquisition" (default) or "first_of_month"
	PaymentTerms      string // "net0", "net15", "net30", "net60", "due_on_receipt"
	TrialDays         int    // >0 starts the subscription in "trialing"; first invoice is generated at trial conversion
}

func (s *SubscriptionService) CreateSubscription(ctx context.Context, input CreateSubscriptionInput) (*domain.Subscription, error) {
	// 1. Fetch Plan
	plan, err := s.planRepo.GetByID(ctx, input.PlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}
	if plan == nil {
		return nil, fmt.Errorf("plan not found")
	}

	// 2. Fetch Customer
	customer, err := s.customerRepo.GetByID(ctx, input.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}
	if customer == nil {
		return nil, fmt.Errorf("customer not found")
	}

	// 3. Calculate Dates
	start := input.StartDate
	if start.IsZero() {
		start = time.Now().UTC()
	}

	// Determine End Date
	var end time.Time
	// Handle Calendar Billing
	anchorType := input.BillingAnchorType
	if anchorType == "" {
		anchorType = "acquisition"
	}

	// The natural end of a full billing interval from `start`. AddInterval clamps
	// month/year math to the target month's last day (no Jan 31 -> Mar 3 drift).
	fullEnd := domain.AddInterval(start, string(plan.IntervalUnit), plan.IntervalCount)

	// firstPeriodFactor prorates the first charge. With first_of_month the first
	// period is a short stub (start → 1st of next month); billing the full plan
	// price for it over-charged the customer (ENG-144). Prorate to the stub's
	// share of a full interval.
	firstPeriodFactor := 1.0
	if anchorType == "first_of_month" && start.Day() != 1 {
		year, month, _ := start.Date()
		end = time.Date(year, month, 1, 0, 0, 0, 0, start.Location()).AddDate(0, 1, 0)
		if fullEnd.After(start) {
			firstPeriodFactor = end.Sub(start).Seconds() / fullEnd.Sub(start).Seconds()
		}
	} else {
		end = fullEnd
	}

	// Trial handling: a trialing subscription defers its first invoice until the
	// trial-expiry scheduler converts it to active. During the trial the current
	// period is the trial window itself.
	isTrial := input.TrialDays > 0
	var trialEndPtr *time.Time
	subStatus := domain.SubscriptionStatusActive
	if isTrial {
		trialEnd := start.AddDate(0, 0, input.TrialDays)
		trialEndPtr = &trialEnd
		subStatus = domain.SubscriptionStatusTrialing
		end = trialEnd
	}

	// 4. Calculate Price & Apply Coupon
	if len(plan.Prices) == 0 {
		return nil, fmt.Errorf("plan has no prices")
	}
	price := plan.Prices[0]

	// firstPeriodFactor is 1.0 except for a prorated first_of_month stub period.
	subtotal := int64(float64(price.Amount) * firstPeriodFactor)
	discount := int64(0)
	var couponID *uuid.UUID

	if input.CouponCode != "" {
		coupon, err := s.couponRepo.GetByCode(ctx, input.TenantID, input.CouponCode)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch coupon: %w", err)
		}
		if coupon == nil {
			return nil, fmt.Errorf("invalid coupon code")
		}
		if !coupon.Active {
			return nil, fmt.Errorf("coupon is no longer active")
		}

		couponID = &coupon.ID

		if coupon.DiscountType == domain.DiscountTypePercent {
			discount = (subtotal * coupon.DiscountValue) / 100
		} else {
			discount = coupon.DiscountValue
		}
	}

	total := subtotal - discount
	if total < 0 {
		total = 0
	}

	// Jurisdiction-aware tax on the post-discount amount: tenant GST config
	// (India) or env company defaults decide the engine; buyer location
	// decides the treatment.
	taxRes := s.taxResolver.ResolveInvoiceTax(ctx, input.TenantID, customer, price.Currency, total, plan.HSNCode)
	total = total + taxRes.Total

	subID := uuid.New()
	invID := uuid.New()

	// Calculate Due Date based on payment terms
	paymentTerms := input.PaymentTerms
	if paymentTerms == "" {
		paymentTerms = "due_on_receipt"
	}
	dueDate := domain.CalculateDueDate(time.Now().UTC(), paymentTerms)

	// Itemization: the initial invoice is a single plan line. Its Amount is the
	// gross Subtotal and its taxable_amount is the post-discount base the tax was
	// computed on, so the line stays consistent (amount − discount == taxable).
	planLineDesc := plan.Name
	if planLineDesc == "" {
		planLineDesc = "Subscription"
	}

	// Header GST comes from the resolver (tax computed on the post-discount
	// amount). Line-level taxable_amount is set below.
	taxTotal, taxIGST, taxCGST, taxSGST := taxRes.Total, taxRes.IGST, taxRes.CGST, taxRes.SGST

	lines := []domain.InvoiceItem{
		newInvoiceLine(invID, planLineDesc, taxRes.HSN, 1, subtotal, subtotal, taxRes, time.Time{}),
	}

	// Per-line discount distribution (Phase 3). The initial invoice is always a
	// single line today, so we record its post-discount taxable base directly and
	// keep the engine-computed header tax verbatim (no total shifts). Should the
	// invoice ever grow multiple lines, distributeDiscount spreads the discount
	// pro-rata (largest-remainder) and re-aggregates the header from the lines.
	if discount > 0 {
		if len(lines) == 1 {
			lines[0].TaxableAmount = subtotal - discount
		} else {
			taxIGST, taxCGST, taxSGST, taxTotal = distributeDiscount(lines, discount)
			total = (subtotal - discount) + taxTotal
		}
	}

	// Create Invoice with Discount applied
	invoice := &domain.Invoice{
		ID:             invID,
		TenantID:       input.TenantID,
		SubscriptionID: &subID,
		CustomerID:     input.CustomerID,
		InvoiceNumber:  fmt.Sprintf("INV-%d-%s", time.Now().UnixNano(), invID.String()[:8]),
		BillingReason:  domain.BillingReasonSubscriptionCreate,
		Status:         domain.InvoiceStatusOpen,
		Currency:       price.Currency,
		Subtotal:       subtotal,
		TaxAmount:      taxTotal,
		TaxType:        taxRes.TaxType, // D3c: persist for the liability report
		Total:          total,
		IGSTAmount:     taxIGST,
		CGSTAmount:     taxCGST,
		SGSTAmount:     taxSGST,
		LineItems:      lines,
		PaymentTerms:   paymentTerms,
		CreatedAt:      time.Now().UTC(),
		DueDate:        dueDate,
		PaidAt:         nil,
	}

	// P25 e-invoicing is deferred to AFTER the subscription + invoice commit
	// (below), so a rolled-back create can't orphan an irreversible government IRN
	// (PHASE2 #3). Skipped for trials — the IRN is generated on trial conversion.

	anchorDay := 0
	if anchorType == "first_of_month" {
		anchorDay = 1
	}

	sub := &domain.Subscription{
		ID:                 subID,
		TenantID:           input.TenantID,
		EntityID:           input.EntityID,
		CustomerID:         input.CustomerID,
		PlanID:             input.PlanID,
		Status:             subStatus,
		TrialEnd:           trialEndPtr,
		CurrentPeriodStart: start,
		CurrentPeriodEnd:   end,
		BillingAnchor:      start,
		BillingAnchorType:  anchorType,
		BillingAnchorDay:   anchorDay,
		PaymentTerms:       paymentTerms,
		CouponID:           couponID,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	// Create gateway subscription (Razorpay/Stripe)
	if s.gateway != nil {
		totalCount := 120 // 10 years for ongoing
		rpPlanID := plan.Code

		gwSubID, err := s.gateway.CreateSubscription(ctx, rpPlanID, totalCount, customer.Email, nil, price.Currency)
		if err != nil {
			// Best-effort mirror only: it uses the Recurso plan code as the
			// gateway plan/price id, which rarely exists on real gateways.
			// Billing is unaffected — Recurso's own invoicing + checkout/retry
			// collect the money — so this is a Warn, not an Error.
			s.logger.Warn("optional gateway-side subscription not created; billing proceeds via Recurso invoicing",
				"error", err,
				"plan_code", plan.Code,
			)
		} else {
			if price.Currency == "INR" {
				sub.RazorpaySubscriptionID = gwSubID
			} else {
				sub.StripeSubscriptionID = gwSubID
			}
		}
	}

	if isTrial {
		// Trial: persist the subscription only. The first invoice is generated
		// when the trial-expiry scheduler converts it to active.
		if err := s.subRepo.Create(ctx, sub); err != nil {
			return nil, fmt.Errorf("failed to create trial subscription: %w", err)
		}
	} else if s.txManager != nil && s.subRepoImpl != nil && s.invRepoImpl != nil {
		// Atomic: Create subscription + invoice in a single transaction
		err := s.txManager.WithTx(ctx, func(tx *sql.Tx) error {
			if err := s.subRepoImpl.CreateWithTx(ctx, tx, sub); err != nil {
				return fmt.Errorf("failed to create subscription: %w", err)
			}
			if err := s.invRepoImpl.CreateWithTx(ctx, tx, invoice); err != nil {
				return fmt.Errorf("failed to create invoice: %w", err)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		// Fallback: non-transactional (for tests or when TxManager not available)
		if err := s.subRepo.Create(ctx, sub); err != nil {
			return nil, fmt.Errorf("failed to create subscription: %w", err)
		}
		if err := s.invoiceRepo.Create(ctx, invoice); err != nil {
			return nil, fmt.Errorf("failed to create invoice: %w", err)
		}
	}

	if !isTrial {
		s.telemetry.MilestoneFirstInvoice() // opt-in anonymous milestone; no-op when disabled

		// P25 e-invoicing AFTER the invoice is durably committed (PHASE2 #3).
		s.generateEInvoiceAfterCommit(ctx, invoice, customer)

		// Dual-write to ledger (outside TX — TigerBeetle is a separate system)
		if s.ledger != nil {
			if err := s.ledger.RecordInvoice(ctx, invoice); err != nil {
				s.logger.Error("ledger write failed — will need reconciliation",
					"error", err,
					"invoice_id", invID,
					"amount", total,
				)
			}
		}
	}

	s.logger.Info("subscription created",
		"subscription_id", subID,
		"customer_id", input.CustomerID,
		"plan_id", input.PlanID,
		"billing_anchor_type", anchorType,
		"payment_terms", paymentTerms,
	)

	// Send subscription created notification
	if s.notificationService != nil {
		err := s.notificationService.SendSubscriptionCreated(ctx, SubscriptionData{
			CustomerName:    domain.PtrToString(customer.Name),
			CustomerEmail:   customer.Email,
			PlanName:        plan.Name,
			Price:           formatAmount(price.Amount, price.Currency),
			Interval:        fmt.Sprintf("%d %s", plan.IntervalCount, plan.IntervalUnit),
			StartDate:       start.Format("Jan 02, 2006"),
			NextBillingDate: end.Format("Jan 02, 2006"),
		})
		if err != nil {
			s.logger.Error("failed to send subscription created notification", "error", err, "subscription_id", subID)
		}
	}

	return sub, nil
}

func (s *SubscriptionService) ListSubscriptions(ctx context.Context, tenantID uuid.UUID, filter domain.SubscriptionFilter) ([]*domain.Subscription, error) {
	return s.subRepo.List(ctx, tenantID, filter)
}

func (s *SubscriptionService) ListInvoices(ctx context.Context, tenantID uuid.UUID) ([]*domain.Invoice, error) {
	return s.invoiceRepo.List(ctx, tenantID)
}

// GetByID retrieves a subscription by ID
func (s *SubscriptionService) GetByID(ctx context.Context, tenantID, subscriptionID uuid.UUID) (*domain.Subscription, error) {
	sub, err := s.subRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}
	if sub != nil && sub.TenantID != tenantID {
		return nil, fmt.Errorf("subscription not found for tenant")
	}
	return sub, nil
}

// generateEInvoiceAfterCommit registers the government e-invoice (IRN) for a
// committed invoice and persists the result. Shared by every plan path — first
// invoice, plan-change proration, and trial conversion. It MUST be called only
// after the invoice is durably committed: an IRN is an irreversible external
// side-effect, so requesting it before commit would leave an orphaned government
// registration if the transaction rolled back (PHASE2 #3 / I1).
// Best-effort — a failure is logged and the e-invoice status/retry is persisted
// so the retry worker can pick it up.
func (s *SubscriptionService) generateEInvoiceAfterCommit(ctx context.Context, invoice *domain.Invoice, customer *domain.Customer) {
	switch {
	case s.einvoiceService != nil:
		if _, err := s.einvoiceService.GenerateEInvoice(ctx, invoice); err != nil {
			s.logger.Error("e-invoice generation failed for proration (will retry)", "error", err, "invoice_id", invoice.ID)
		}
	case s.gspAdapter != nil && customer.BillingAddress.Country == "India" && domain.PtrToString(customer.GSTIN) != "" && customer.TaxType == "business":
		resp, err := s.gspAdapter.GenerateIRN(ctx, invoice)
		if err == nil {
			invoice.IRN = resp.IRN
			invoice.SignedQRCode = resp.SignedQRCode
			invoice.EInvoiceStatus = "GENERATED"
			invoice.AckNo = resp.AckNo
		} else {
			s.logger.Error("error generating IRN for proration invoice", "error", err, "invoice_id", invoice.ID)
			invoice.EInvoiceStatus = "FAILED"
		}
	default:
		return // not e-invoice eligible; nothing generated or to persist
	}

	// Persist the IRN/status (and any retry scheduling) onto the committed row.
	if err := s.invoiceRepo.Update(ctx, invoice); err != nil {
		s.logger.Error("failed to persist proration e-invoice result", "invoice_id", invoice.ID, "error", err)
	}
}

// --- Multi-product catalog v1: subscription add-ons ---------------------
//
// An add-on is an existing plan attached to a subscription with a quantity.
// The subscription's base plan_id is unchanged; add-ons become extra invoice
// lines (price × quantity, taxed independently) starting from the NEXT
// recurring invoice. Mid-cycle proration is a deliberate follow-up.

// requireOwnedSubscription loads a subscription and enforces tenant ownership,
// returning ErrSubscriptionNotFound for both a missing row and a cross-tenant
// row so isolation stays opaque.
func (s *SubscriptionService) requireOwnedSubscription(ctx context.Context, tenantID, subscriptionID uuid.UUID) (*domain.Subscription, error) {
	sub, err := s.subRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}
	if sub == nil || sub.TenantID != tenantID {
		return nil, ErrSubscriptionNotFound
	}
	return sub, nil
}

// subscriptionCurrency derives the subscription's billing currency from its
// base plan's first price. Add-ons must match it.
func (s *SubscriptionService) subscriptionCurrency(ctx context.Context, sub *domain.Subscription) (string, error) {
	plan, err := s.planRepo.GetByID(ctx, sub.PlanID)
	if err != nil {
		return "", fmt.Errorf("failed to get base plan: %w", err)
	}
	if plan == nil || len(plan.Prices) == 0 {
		return "", fmt.Errorf("base plan has no prices")
	}
	return plan.Prices[0].Currency, nil
}

func formatAmount(amountPaise int64, currency string) string {
	amount := float64(amountPaise) / 100
	switch currency {
	case "INR":
		return fmt.Sprintf("₹%.2f", amount)
	case "USD":
		return fmt.Sprintf("$%.2f", amount)
	default:
		return fmt.Sprintf("%s %.2f", currency, amount)
	}
}
