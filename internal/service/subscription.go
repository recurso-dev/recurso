package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
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
	TenantID          uuid.UUID
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

// MarkInvoicePaid settles an invoice and returns whether THIS caller performed
// the paid transition. Multiple settlers (inline checkout, gateway webhook,
// retry worker, offline payment) can all call it for the same invoice, but only
// one gets transitioned=true — callers that fire once-per-settlement side
// effects (e.g. recording a dunning outcome) must gate on it so a redelivered
// webhook or a second settler doesn't double-count.
func (s *SubscriptionService) MarkInvoicePaid(ctx context.Context, invoiceID uuid.UUID) (transitioned bool, err error) {
	inv, err := s.invoiceRepo.GetByID(ctx, invoiceID)
	if err != nil {
		return false, err
	}
	if inv == nil {
		return false, fmt.Errorf("invoice not found")
	}

	if inv.Status == domain.InvoiceStatusPaid {
		return false, nil // Already paid
	}

	// Amount already settled by a non-cash channel before this payment — today
	// only a prepaid wallet drain at invoice generation writes amount_paid ahead
	// of settlement (partial offline payments leave the invoice open without
	// touching it). That drain already relieved AR and was booked as cash at
	// top-up, so the ledger cash leg below must EXCLUDE it; otherwise the wallet
	// portion double-books as cash and drives AR negative. Captured from the
	// freshly-loaded invoice, before MarkPaid / the AmountPaid overwrite below.
	walletSettled := inv.AmountPaid

	now := time.Now().UTC()
	// Atomically claim the paid transition. Only the settler whose conditional
	// UPDATE actually flips the row runs the side-effects below; concurrent
	// settlers get transitioned=false and return without double-posting the
	// ledger or double-counting recovered revenue.
	transitioned, err = s.invoiceRepo.MarkPaid(ctx, inv.TenantID, invoiceID, now)
	if err != nil {
		return false, err
	}
	if !transitioned {
		return false, nil // another settler already marked it paid
	}
	inv.Status = domain.InvoiceStatusPaid
	inv.PaidAt = &now
	inv.AmountPaid = inv.Total

	s.telemetry.MilestoneFirstPayment() // opt-in anonymous milestone; no-op when disabled

	// Dunning recovery attribution: if this invoice needed retries or dunning
	// to get paid, record it as recovered revenue (idempotent, non-fatal).
	if s.recoveryRecorder != nil {
		s.recoveryRecorder.RecordIfRecovered(ctx, inv)
	}

	// Record payment in ledger — cash leg net of the wallet portion already
	// settled at generation (see walletSettled above).
	if s.ledger != nil {
		if err := s.ledger.RecordPaymentWithSettled(ctx, inv, walletSettled); err != nil {
			s.logger.Error("ledger payment write failed", "error", err, "invoice_id", inv.ID)
		}
	}

	// Phase 5: Create Revenue Recognition Schedule
	if s.revrecService != nil {
		var sub *domain.Subscription
		if inv.SubscriptionID != nil {
			sub, _ = s.subRepo.GetByID(ctx, *inv.SubscriptionID)
		}
		if err := s.revrecService.CreateScheduleForInvoice(ctx, inv, sub); err != nil {
			s.logger.Error("failed to create revrec schedule", "invoice_id", inv.ID, "error", err)
			// Don't fail the whole payment mark-paid for now, just log.
		}
	}

	// Send payment received notification
	if s.notificationService != nil {
		customer, custErr := s.customerRepo.GetByID(ctx, inv.CustomerID)
		if custErr != nil {
			s.logger.Error("failed to fetch customer for payment notification", "error", custErr, "customer_id", inv.CustomerID)
		} else if customer != nil {
			err := s.notificationService.SendPaymentReceived(ctx, PaymentData{
				CustomerName:  domain.PtrToString(customer.Name),
				CustomerEmail: customer.Email,
				InvoiceNumber: inv.InvoiceNumber,
				Amount:        formatAmount(inv.Total, inv.Currency),
				PaymentDate:   now.Format("Jan 02, 2006"),
			})
			if err != nil {
				s.logger.Error("failed to send payment received notification", "error", err, "invoice_id", inv.ID)
			}
		}
	}

	return true, nil
}

func (s *SubscriptionService) ListSubscriptions(ctx context.Context, tenantID uuid.UUID, filter domain.SubscriptionFilter) ([]*domain.Subscription, error) {
	return s.subRepo.List(ctx, tenantID, filter)
}

func (s *SubscriptionService) ListInvoices(ctx context.Context, tenantID uuid.UUID) ([]*domain.Invoice, error) {
	return s.invoiceRepo.List(ctx, tenantID)
}

// CancelResult contains the result of a subscription cancellation
type CancelResult struct {
	ID               uuid.UUID
	Status           string
	CurrentPeriodEnd time.Time
	CustomerEmail    string
	CustomerName     string
	PlanName         string
}

// Cancel cancels a subscription
func (s *SubscriptionService) Cancel(ctx context.Context, tenantID, subscriptionID uuid.UUID, immediately bool, reason, feedback string) (*CancelResult, error) {
	sub, err := s.subRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}
	if sub == nil {
		return nil, fmt.Errorf("subscription not found")
	}
	if sub.TenantID != tenantID {
		return nil, fmt.Errorf("subscription not found for tenant")
	}

	// Idempotent: an already-canceled subscription is a no-op. Re-running would
	// overwrite CanceledAt with a later time, re-call the gateway cancel, and
	// re-invoke the rev-rec unwind — so guard the terminal state.
	if sub.Status == domain.SubscriptionStatusCanceled {
		return &CancelResult{ID: sub.ID, Status: string(sub.Status), CurrentPeriodEnd: sub.CurrentPeriodEnd}, nil
	}

	// Get customer and plan info for notification (best-effort — the cancel
	// still succeeds if these fail; the notification fields just stay blank).
	customer, err := s.customerRepo.GetByID(ctx, sub.CustomerID)
	if err != nil {
		s.logger.Warn("cancel: customer lookup failed; notification fields may be blank",
			"subscription_id", sub.ID, "error", err)
	}
	plan, err := s.planRepo.GetByID(ctx, sub.PlanID)
	if err != nil {
		s.logger.Warn("cancel: plan lookup failed; notification fields may be blank",
			"subscription_id", sub.ID, "error", err)
	}

	now := time.Now().UTC()

	if immediately {
		sub.Status = domain.SubscriptionStatusCanceled
		sub.CanceledAt = &now
	} else {
		sub.CancelAtPeriodEnd = true
	}

	sub.CancellationReason = reason
	sub.CancellationFeedback = feedback
	sub.UpdatedAt = now

	if err := s.subRepo.Update(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to update subscription: %w", err)
	}

	// Cancel on payment gateway (best-effort)
	if s.gateway != nil {
		if sub.RazorpaySubscriptionID != "" {
			if err := s.gateway.CancelSubscription(ctx, sub.RazorpaySubscriptionID); err != nil {
				s.logger.Error("failed to cancel subscription on payment gateway", "error", err, "gateway", "razorpay", "subscription_id", sub.RazorpaySubscriptionID)
			}
		}
		if sub.StripeSubscriptionID != "" {
			if err := s.gateway.CancelSubscription(ctx, sub.StripeSubscriptionID); err != nil {
				s.logger.Error("failed to cancel subscription on payment gateway", "error", err, "gateway", "stripe", "subscription_id", sub.StripeSubscriptionID)
			}
		}
	}

	// Usage-based billing v1: bill the metered usage of the partial elapsed
	// window on immediate cancel — the flat fee was paid in advance, but the
	// usage since period start would otherwise never be invoiced. Best-effort:
	// a failure logs loudly and leaves the window unclaimed for a manual rerun.
	// Cancel-at-period-end needs nothing here: the normal cycle generator
	// rates the full period when it closes.
	if immediately && s.finalUsageInvoicer != nil {
		if finalInv, err := s.finalUsageInvoicer.GenerateFinalUsageInvoice(ctx, sub, now); err != nil {
			s.logger.Error("final usage invoice on cancel failed", "error", err, "subscription_id", sub.ID)
		} else if finalInv != nil {
			s.logger.Info("final usage invoice generated on cancel",
				"subscription_id", sub.ID, "invoice_id", finalInv.ID, "total", finalInv.Total)
		}
	}

	// Rev-rec unwind on immediate cancel: forfeit (recognize) the remaining
	// deferred revenue and void future recognition events, so a mid-period
	// cancel doesn't leave deferred sitting forever or keep firing recognition
	// (ENG-147). Only for immediate cancels — cancel-at-period-end keeps service
	// (and the natural recognition schedule) running to period end. Best-effort.
	if immediately && s.revrecService != nil {
		if forfeited, err := s.revrecService.UnwindOnCancel(ctx, tenantID, subscriptionID); err != nil {
			s.logger.Error("rev-rec unwind on cancel failed", "error", err, "subscription_id", subscriptionID)
		} else if forfeited > 0 {
			s.logger.Info("rev-rec deferred forfeited on cancel", "subscription_id", subscriptionID, "amount", forfeited)
		}
	}

	result := &CancelResult{
		ID:               sub.ID,
		Status:           string(sub.Status),
		CurrentPeriodEnd: sub.CurrentPeriodEnd,
	}

	if customer != nil {
		result.CustomerEmail = customer.Email
		result.CustomerName = domain.PtrToString(customer.Name)
	}
	if plan != nil {
		result.PlanName = plan.Name
	}

	return result, nil
}

// Reactivate reactivates a cancelled subscription
func (s *SubscriptionService) Reactivate(ctx context.Context, tenantID, subscriptionID uuid.UUID) (*domain.Subscription, error) {
	sub, err := s.subRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}
	if sub == nil {
		return nil, fmt.Errorf("subscription not found")
	}
	if sub.TenantID != tenantID {
		return nil, fmt.Errorf("subscription not found for tenant")
	}

	// Can only reactivate if cancel_at_period_end is true or within grace period
	if !sub.CancelAtPeriodEnd && sub.Status != domain.SubscriptionStatusCanceled {
		return nil, fmt.Errorf("subscription cannot be reactivated")
	}

	// Check if still within period
	if time.Now().After(sub.CurrentPeriodEnd) {
		return nil, fmt.Errorf("subscription period has ended, please create a new subscription")
	}

	sub.CancelAtPeriodEnd = false
	sub.CancellationReason = ""
	sub.CancellationFeedback = ""
	sub.CanceledAt = nil // clear the cancel timestamp — otherwise churn/MRR/rev-rec queries that filter canceled_at IS NOT NULL misclassify the reactivated (live) subscription as churned
	sub.Status = domain.SubscriptionStatusActive
	sub.UpdatedAt = time.Now().UTC()

	if err := s.subRepo.Update(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to reactivate subscription: %w", err)
	}

	return sub, nil
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

// ProrationResult contains the calculation for an upgrade/downgrade
type ProrationResult struct {
	CreditAmount       int64     // Amount credited for unused time on old plan
	ChargeAmount       int64     // Amount charged for remaining time on new plan
	NetAmount          int64     // Net amount to invoice (Charge - Credit)
	ProrationDate      time.Time // Date the proration occurred
	UnusedSeconds      float64   // Number of seconds unused in the period
	RemainingSeconds   float64   // Number of seconds remaining in the period
	PeriodTotalSeconds float64   // Total seconds in the period
}

// CalculateProration calculates the credit and charge amounts for a plan change
func (s *SubscriptionService) CalculateProration(
	currentPlanPrice int64,
	newPlanPrice int64,
	periodStart time.Time,
	periodEnd time.Time,
	prorationDate time.Time,
) *ProrationResult {
	totalDuration := periodEnd.Sub(periodStart).Seconds()
	if totalDuration <= 0 {
		return &ProrationResult{}
	}

	remainingDuration := periodEnd.Sub(prorationDate).Seconds()
	if remainingDuration < 0 {
		remainingDuration = 0
	}

	unusedDuration := remainingDuration // In simple terms, unused time on old plan matches remaining time on new plan

	// Calculate Credit for Old Plan (Unused Time)
	creditAmount := int64(float64(currentPlanPrice) * (unusedDuration / totalDuration))

	// Calculate Charge for New Plan (Remaining Time)
	chargeAmount := int64(float64(newPlanPrice) * (remainingDuration / totalDuration))

	return &ProrationResult{
		CreditAmount:       creditAmount,
		ChargeAmount:       chargeAmount,
		NetAmount:          chargeAmount - creditAmount,
		ProrationDate:      prorationDate,
		UnusedSeconds:      unusedDuration,
		RemainingSeconds:   remainingDuration,
		PeriodTotalSeconds: totalDuration,
	}
}

// PlanChangeProration bundles a proration result with the tax computed on its
// net amount. Both UpdateSubscription (apply) and PreviewPlanChange (preview)
// obtain it from computePlanChangeProration, guaranteeing the previewed numbers
// equal what apply will actually charge.
type PlanChangeProration struct {
	Proration     *ProrationResult
	Tax           InvoiceTax
	Currency      string
	EffectiveDate time.Time
}

// computePlanChangeProration is the single source of truth for plan-change
// math. A positive net (a charge) is taxed on the new plan's HSN; a negative net
// (a downgrade credit) carries the reversed tax computed on the old plan's HSN
// (ENG-150), so the credit refunds the GST originally collected.
func (s *SubscriptionService) computePlanChangeProration(
	ctx context.Context,
	tenantID uuid.UUID,
	sub *domain.Subscription,
	currentPlan, newPlan *domain.Plan,
	customer *domain.Customer,
	now time.Time,
) PlanChangeProration {
	if currentPlan == nil || newPlan == nil || len(currentPlan.Prices) == 0 || len(newPlan.Prices) == 0 {
		return PlanChangeProration{Proration: &ProrationResult{ProrationDate: now}, EffectiveDate: now}
	}

	currency := newPlan.Prices[0].Currency
	proration := s.CalculateProration(
		currentPlan.Prices[0].Amount,
		newPlan.Prices[0].Amount,
		sub.CurrentPeriodStart,
		sub.CurrentPeriodEnd,
		now,
	)

	var taxRes InvoiceTax
	if customer != nil && proration.NetAmount != 0 {
		if proration.NetAmount > 0 {
			taxRes = s.taxResolver.ResolveInvoiceTax(ctx, tenantID, customer, currency, proration.NetAmount, newPlan.HSNCode)
		} else {
			// Downgrade credit: reverse the GST collected on the old plan's unused
			// portion. Resolve on the positive base with the OLD plan's HSN, then
			// negate so the credit carries the tax reversal (ENG-150) — previously
			// a credit reversed no tax and under-refunded the customer.
			t := s.taxResolver.ResolveInvoiceTax(ctx, tenantID, customer, currency, -proration.NetAmount, currentPlan.HSNCode)
			taxRes = InvoiceTax{
				Total: -t.Total, IGST: -t.IGST, CGST: -t.CGST, SGST: -t.SGST,
				TaxType: t.TaxType, Note: t.Note, Rate: t.Rate, HSN: t.HSN,
			}
		}
	}

	return PlanChangeProration{Proration: proration, Tax: taxRes, Currency: currency, EffectiveDate: now}
}

// PlanChangePreview is the read-only breakdown returned by PreviewPlanChange.
// All monetary fields are in the currency's smallest unit (e.g. paise/cents).
type PlanChangePreview struct {
	SubscriptionID    uuid.UUID `json:"subscription_id"`
	CurrentPlanID     uuid.UUID `json:"current_plan_id"`
	NewPlanID         uuid.UUID `json:"new_plan_id"`
	Currency          string    `json:"currency"`
	CreditAmount      int64     `json:"credit_amount"`       // credit for unused time on the current plan
	ChargeAmount      int64     `json:"charge_amount"`       // prorated charge for the remaining period on the new plan
	NetAmount         int64     `json:"net_amount"`          // charge - credit, before tax
	TaxAmount         int64     `json:"tax_amount"`          // tax on the net: positive on a charge, negative (reversed GST) on a downgrade credit (ENG-150)
	TotalAmount       int64     `json:"total_amount"`        // net + tax: the immediate proration charge (positive) or credit (negative)
	EffectiveDate     time.Time `json:"effective_date"`      // when the change would take effect (now)
	NextInvoiceAmount int64     `json:"next_invoice_amount"` // full new-plan charge incl. tax at the next renewal
	IsUpgrade         bool      `json:"is_upgrade"`          // true when the new plan costs more than the current one
}

// PreviewPlanChange computes the proration for switching a subscription to
// newPlanID WITHOUT applying it. It reuses computePlanChangeProration — the
// exact function UpdateSubscription uses — so the preview matches the charge.
func (s *SubscriptionService) PreviewPlanChange(ctx context.Context, tenantID, subscriptionID, newPlanID uuid.UUID) (*PlanChangePreview, error) {
	sub, err := s.subRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, ErrSubscriptionNotFound
	}
	if sub.TenantID != tenantID {
		return nil, ErrSubscriptionNotFound
	}

	currentPlan, err := s.planRepo.GetByID(ctx, sub.PlanID)
	if err != nil {
		return nil, err
	}
	if currentPlan == nil || len(currentPlan.Prices) == 0 {
		return nil, fmt.Errorf("current plan unavailable for preview")
	}

	newPlan, err := s.planRepo.GetByID(ctx, newPlanID)
	if err != nil {
		return nil, err
	}
	if newPlan == nil {
		return nil, ErrPlanNotFound
	}
	if len(newPlan.Prices) == 0 {
		return nil, fmt.Errorf("new plan has no prices")
	}

	customer, err := s.customerRepo.GetByID(ctx, sub.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}

	now := time.Now().UTC()
	pcp := s.computePlanChangeProration(ctx, tenantID, sub, currentPlan, newPlan, customer, now)

	// Resulting next-invoice amount: the full new-plan price plus tax, i.e. what
	// the customer pays at the next renewal once fully on the new plan.
	newPrice := newPlan.Prices[0].Amount
	var nextTax InvoiceTax
	if newPrice > 0 && customer != nil {
		nextTax = s.taxResolver.ResolveInvoiceTax(ctx, tenantID, customer, pcp.Currency, newPrice, newPlan.HSNCode)
	}

	return &PlanChangePreview{
		SubscriptionID:    sub.ID,
		CurrentPlanID:     sub.PlanID,
		NewPlanID:         newPlanID,
		Currency:          pcp.Currency,
		CreditAmount:      pcp.Proration.CreditAmount,
		ChargeAmount:      pcp.Proration.ChargeAmount,
		NetAmount:         pcp.Proration.NetAmount,
		TaxAmount:         pcp.Tax.Total,
		TotalAmount:       pcp.Proration.NetAmount + pcp.Tax.Total,
		EffectiveDate:     pcp.EffectiveDate,
		NextInvoiceAmount: newPrice + nextTax.Total,
		IsUpgrade:         newPrice > currentPlan.Prices[0].Amount,
	}, nil
}

// UpdateSubscription updates a subscription's plan and handles proration
// persistPlanChange writes the proration invoice and/or downgrade credit note
// and flips the subscription's plan. See the atomicity note at the transaction
// branch below (PHASE2 #1).
func (s *SubscriptionService) persistPlanChange(ctx context.Context, chargeInvoice *domain.Invoice, creditNote *domain.CreditNote, sub *domain.Subscription) error {
	// Atomic path: the proration invoice OR downgrade credit note commits together
	// with the plan flip in one transaction. A failed flip can never leave an
	// orphaned charge — or (the exploit) a spendable credit without the actual
	// downgrade, which a caller could loop for unbounded credit (PHASE2 #1).
	canTx := s.txManager != nil && s.subRepoImpl != nil && s.invRepoImpl != nil &&
		(creditNote == nil || s.creditNoteRepo != nil)
	if canTx {
		return s.txManager.WithTx(ctx, func(tx *sql.Tx) error {
			if chargeInvoice != nil {
				if err := s.invRepoImpl.CreateWithTx(ctx, tx, chargeInvoice); err != nil {
					return fmt.Errorf("failed to create proration invoice: %w", err)
				}
			}
			if creditNote != nil {
				if err := s.creditNoteRepo.CreateWithTx(ctx, tx, creditNote); err != nil {
					return fmt.Errorf("failed to create downgrade credit note: %w", err)
				}
			}
			return s.subRepoImpl.UpdateWithTx(ctx, tx, sub)
		})
	}

	// Fallback for mock/partial wiring (tests without concrete repos): sequential
	// best-effort. Not atomic, but only reached when the tx path is unavailable.
	// Tightened: Update the subscription FIRST. If this fails, no credit is
	// issued. If it succeeds but the credit fails, it's bad UX but financially
	// safe (prevents the downgrade exploit).
	if err := s.subRepo.Update(ctx, sub); err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	if chargeInvoice != nil {
		if err := s.invoiceRepo.Create(ctx, chargeInvoice); err != nil {
			s.logger.Error("failed to create proration invoice after plan flip", "subscription_id", sub.ID, "error", err)
			return fmt.Errorf("plan updated but failed to create proration invoice: %w", err)
		}
	}
	if creditNote != nil {
		if s.creditNoteRepo != nil {
			if err := s.creditNoteRepo.Create(ctx, creditNote); err != nil {
				s.logger.Error("failed to create downgrade credit note after plan flip", "subscription_id", sub.ID, "error", err)
				return fmt.Errorf("plan updated but failed to create downgrade credit note: %w", err)
			}
		} else {
			s.logger.Warn("downgrade proration credit not persisted (no credit-note repo configured)",
				"subscription_id", sub.ID, "amount", creditNote.Amount)
		}
	}
	return nil
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

func (s *SubscriptionService) UpdateSubscription(ctx context.Context, tenantID, subscriptionID, newPlanID uuid.UUID) (*domain.Subscription, error) {
	// 1. Fetch Subscription & Current Plan
	sub, err := s.subRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, fmt.Errorf("subscription not found")
	}
	if sub.TenantID != tenantID {
		return nil, fmt.Errorf("subscription not found for tenant")
	}

	// 1.5 Fetch Customer
	customer, err := s.customerRepo.GetByID(ctx, sub.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}
	if customer == nil {
		return nil, fmt.Errorf("customer not found")
	}

	if sub.PlanID == newPlanID {
		return sub, nil // No change
	}

	currentPlan, err := s.planRepo.GetByID(ctx, sub.PlanID)
	if err != nil {
		return nil, err
	}

	// 2. Fetch New Plan
	newPlan, err := s.planRepo.GetByID(ctx, newPlanID)
	if err != nil {
		return nil, err
	}
	if newPlan == nil {
		return nil, fmt.Errorf("new plan not found")
	}

	// 3. Calculate Proration via the shared helper so apply and preview
	// (PreviewPlanChange) always agree on credit/charge/tax.
	now := time.Now().UTC()
	pcp := s.computePlanChangeProration(ctx, tenantID, sub, currentPlan, newPlan, customer, now)
	proration := pcp.Proration
	taxRes := pcp.Tax

	// 4. Build the proration record and apply the plan change atomically.
	//   - NetAmount > 0 (upgrade): issue a CHARGE invoice (tax-inclusive) and
	//     flip the plan in the same DB transaction, so a plan change can never
	//     land without its charge (or vice versa).
	//   - NetAmount < 0 (downgrade): persist the credit as a spendable
	//     adjustment CREDIT NOTE including the reversed tax. Previously the
	//     credit was force-zeroed onto a $0 "paid" invoice and silently vanished
	//     (ENG-150). The credit note is created first, then the plan flips.
	sub.PlanID = newPlanID
	sub.UpdatedAt = now

	var chargeInvoice *domain.Invoice
	var creditNote *domain.CreditNote

	switch {
	case proration.NetAmount > 0:
		prInvID := uuid.New()
		prDesc := "Plan change proration"
		if newPlan.Name != "" {
			prDesc = fmt.Sprintf("Proration: %s", newPlan.Name)
		}
		chargeInvoice = &domain.Invoice{
			ID:             prInvID,
			TenantID:       tenantID,
			SubscriptionID: &sub.ID,
			CustomerID:     sub.CustomerID,
			InvoiceNumber:  fmt.Sprintf("INV-PR-%d-%s", time.Now().UnixNano(), prInvID.String()[:8]),
			BillingReason:  domain.BillingReasonSubscriptionUpdate,
			Status:         domain.InvoiceStatusOpen,
			Currency:       pcp.Currency,
			Subtotal:       proration.NetAmount,
			TaxAmount:      taxRes.Total,
			TaxType:        taxRes.TaxType, // D3c: persist for the liability report
			IGSTAmount:     taxRes.IGST,
			CGSTAmount:     taxRes.CGST,
			SGSTAmount:     taxRes.SGST,
			Total:          proration.NetAmount + taxRes.Total,
			LineItems: []domain.InvoiceItem{
				newInvoiceLine(prInvID, prDesc, taxRes.HSN, 1, proration.NetAmount, proration.NetAmount, taxRes, time.Time{}),
			},
			CreatedAt: now,
			DueDate:   now,
		}

		// P25 e-invoicing is deferred to AFTER the invoice is committed (below the
		// persist call) — see generateEInvoiceAfterCommit (PHASE2 #3).

	case proration.NetAmount < 0:
		// Both proration.NetAmount and taxRes.Total are negative here; negating
		// their sum yields a positive, spendable credit balance.
		creditAmount := -(proration.NetAmount + taxRes.Total)
		creditNote = &domain.CreditNote{
			ID:           uuid.New(),
			TenantID:     tenantID,
			CustomerID:   sub.CustomerID,
			Amount:       creditAmount,
			Balance:      creditAmount,
			Currency:     pcp.Currency,
			Status:       domain.CreditNoteStatusIssued,
			Reason:       "Plan downgrade proration credit",
			Type:         domain.CreditNoteTypeAdjustment,
			RefundStatus: domain.RefundStatusNone,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
	}

	// 5. Persist. Charge path: invoice + plan flip in one transaction. Credit
	// path: credit note first (an orphaned credit is recoverable), then plan
	// flip. If the txManager/concrete repos are unavailable (e.g. tests with a
	// mock repo), fall back to sequential writes.
	if err := s.persistPlanChange(ctx, chargeInvoice, creditNote, sub); err != nil {
		return nil, err
	}

	// P25 e-invoicing runs AFTER the invoice is durably committed: registering a
	// government IRN before commit would orphan an irreversible IRN at NIC if the
	// transaction rolled back (PHASE2 #3).
	if chargeInvoice != nil {
		s.generateEInvoiceAfterCommit(ctx, chargeInvoice, customer)
	}

	// Post the upgrade charge's invoice leg (DR AR / CR Deferred/Revenue) to the
	// ledger, symmetric to the downgrade-credit posting below and to the initial
	// invoice in CreateSubscription. Without it the charge invoice would only ever
	// get its cash leg on payment, leaving AR/Deferred permanently imbalanced —
	// the reconciler flags it as missing_invoice_transaction (F1). Best-effort,
	// after commit: a post failure is logged for reconciliation, never fails the
	// plan change.
	if chargeInvoice != nil && s.ledger != nil {
		if err := s.ledger.RecordInvoice(ctx, chargeInvoice); err != nil {
			s.logger.Error("upgrade proration ledger invoice post failed — reconciliation needed",
				"invoice_id", chargeInvoice.ID, "amount", chargeInvoice.Total, "error", err)
		}
	}

	// Apply account credit to the upgrade charge invoice (ENG-154).
	if chargeInvoice != nil {
		s.applyCreditToInvoice(ctx, chargeInvoice)
	}

	// Rev-rec + ledger for a downgrade credit (ENG-154): the over-deferred
	// portion of the current period stops being revenue we'll earn and becomes
	// account credit we owe. Shrink the recognition schedule by the credit amount
	// (so it recognizes only the new plan's remaining service) and post
	// DR Deferred / CR Customer-Credit for the same amount, keeping Deferred and
	// the schedule in step. Best-effort: failures are logged for reconciliation.
	if creditNote != nil && creditNote.Amount > 0 {
		// The credit note is GROSS (net + reversed GST). Split it: the NET reduces
		// Deferred and the recognition schedule (both of which hold net-of-tax
		// after ENG-191), while the tax portion reverses out of Tax Payable — the
		// two together credit the customer the gross they paid. Passing the gross
		// to the net-holding Deferred/schedule would drive Deferred negative by
		// the tax (ENG-191c).
		netCredit := -proration.NetAmount
		if netCredit < 0 {
			netCredit = 0
		}
		taxCredit := -taxRes.Total
		if taxCredit < 0 {
			taxCredit = 0
		}
		if s.revrecService != nil && netCredit > 0 {
			if reduced, err := s.revrecService.ReduceScheduleForDowngrade(ctx, tenantID, subscriptionID, netCredit); err != nil {
				s.logger.Error("downgrade schedule reduction failed", "subscription_id", subscriptionID, "error", err)
			} else if reduced != netCredit {
				s.logger.Warn("downgrade schedule reduced less than the net credit (deferred/credit may need reconciliation)",
					"subscription_id", subscriptionID, "net_credit", netCredit, "reduced", reduced)
			}
		}
		if s.ledger != nil {
			if netCredit > 0 {
				if _, err := s.ledger.RecordDowngradeCredit(ctx, tenantID, creditNote.ID, netCredit, "Plan downgrade credit (net)"); err != nil {
					s.logger.Error("downgrade credit ledger post failed — reconciliation needed",
						"credit_note_id", creditNote.ID, "amount", netCredit, "error", err)
				}
			}
			if taxCredit > 0 {
				if _, err := s.ledger.RecordDowngradeTaxReversal(ctx, tenantID, creditNote.ID, taxCredit, "Plan downgrade GST reversal"); err != nil {
					s.logger.Error("downgrade tax reversal ledger post failed — reconciliation needed",
						"credit_note_id", creditNote.ID, "amount", taxCredit, "error", err)
				}
			}
		}
	}

	// Sync with Gateway (Razorpay/Stripe) — not implemented yet.
	// When s.gateway != nil && sub.RazorpaySubscriptionID != "":
	// s.gateway.UpdateSubscription(ctx, sub.RazorpaySubscriptionID, newPlan.Code)

	return sub, nil
}

// ConvertTrialToActive converts a trialing subscription to active and generates
// its first real invoice. The invoice is created "open" with a due date, so it
// enters the normal payment/dunning path (GetOverdueInvoices picks up open
// invoices once past due). Returns an error if the subscription is not trialing.
func (s *SubscriptionService) ConvertTrialToActive(ctx context.Context, sub *domain.Subscription) (*domain.Invoice, error) {
	if sub == nil {
		return nil, fmt.Errorf("subscription is nil")
	}
	if sub.Status != domain.SubscriptionStatusTrialing {
		return nil, fmt.Errorf("subscription %s is not trialing (status=%s)", sub.ID, sub.Status)
	}

	// The trial scheduler calls with a background context, but every repo
	// below is tenant-scoped — inject the subscription's own tenant (same
	// pattern as the payment webhooks). Without this no trial ever converts.
	ctx = context.WithValue(ctx, domain.TenantIDKey, sub.TenantID)

	plan, err := s.planRepo.GetByID(ctx, sub.PlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}
	if plan == nil || len(plan.Prices) == 0 {
		return nil, fmt.Errorf("plan has no prices")
	}

	customer, err := s.customerRepo.GetByID(ctx, sub.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}
	if customer == nil {
		return nil, fmt.Errorf("customer not found")
	}

	price := plan.Prices[0]
	now := time.Now().UTC()

	// The first paid period starts where the trial ended.
	start := now
	if sub.TrialEnd != nil {
		start = *sub.TrialEnd
	}
	end := domain.AddInterval(start, string(plan.IntervalUnit), plan.IntervalCount)

	subtotal := price.Amount
	taxRes := s.taxResolver.ResolveInvoiceTax(ctx, sub.TenantID, customer, price.Currency, subtotal, plan.HSNCode)
	total := subtotal + taxRes.Total

	paymentTerms := sub.PaymentTerms
	if paymentTerms == "" {
		paymentTerms = "due_on_receipt"
	}
	dueDate := domain.CalculateDueDate(now, paymentTerms)

	invID := uuid.New()
	convDesc := plan.Name
	if convDesc == "" {
		convDesc = "Subscription"
	}
	invoice := &domain.Invoice{
		ID:             invID,
		TenantID:       sub.TenantID,
		SubscriptionID: &sub.ID,
		CustomerID:     sub.CustomerID,
		InvoiceNumber:  fmt.Sprintf("INV-%d-%s", now.UnixNano(), invID.String()[:8]),
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
		// Itemization (Phase 1): single plan line reconciling to the totals.
		LineItems: []domain.InvoiceItem{
			newInvoiceLine(invID, convDesc, taxRes.HSN, 1, subtotal, subtotal, taxRes, time.Time{}),
		},
		PaymentTerms: paymentTerms,
		CreatedAt:    now,
		DueDate:      dueDate,
	}

	// P25 e-invoicing is deferred to AFTER the invoice + activation commit (below),
	// so a rolled-back conversion — including the atomic trial-race loss — can't
	// orphan an irreversible government IRN (PHASE2 #3).

	sub.Status = domain.SubscriptionStatusActive
	sub.CurrentPeriodStart = start
	sub.CurrentPeriodEnd = end
	sub.UpdatedAt = now

	// Persist the first invoice and activate the subscription atomically, so a
	// mid-write failure can't leave a trial billed but still trialing (or
	// activated with no invoice) (ENG-150). The activation is a CONDITIONAL
	// transition (WHERE status='trialing'): with the scheduler lock a no-op under
	// multi-instance, two runners can both read this trial as expired, but only
	// the one that flips the row creates an invoice — the loser matches zero rows
	// and bails out, so a trial is billed exactly once (ENG-161). Falls back to
	// sequential writes when the txManager/concrete repos are unavailable (e.g.
	// mock-repo tests), where there is no cross-instance concurrency.
	if s.txManager != nil && s.invRepoImpl != nil && s.subRepoImpl != nil {
		if err := s.txManager.WithTx(ctx, func(tx *sql.Tx) error {
			won, err := s.subRepoImpl.ActivateTrialWithTx(ctx, tx, sub)
			if err != nil {
				return err
			}
			if !won {
				return errTrialRaceLost // roll back; another runner already converted
			}
			if err := s.invRepoImpl.CreateWithTx(ctx, tx, invoice); err != nil {
				return fmt.Errorf("failed to create trial conversion invoice: %w", err)
			}
			return nil
		}); err != nil {
			if errors.Is(err, errTrialRaceLost) {
				return nil, ErrTrialAlreadyConverted
			}
			return nil, err
		}
	} else {
		if err := s.invoiceRepo.Create(ctx, invoice); err != nil {
			return nil, fmt.Errorf("failed to create trial conversion invoice: %w", err)
		}
		if err := s.subRepo.Update(ctx, sub); err != nil {
			return nil, fmt.Errorf("failed to activate subscription after trial: %w", err)
		}
	}

	// P25 e-invoicing AFTER the invoice is durably committed (PHASE2 #3).
	s.generateEInvoiceAfterCommit(ctx, invoice, customer)

	// Dual-write to ledger (best-effort; reconciliation covers gaps).
	if s.ledger != nil {
		if err := s.ledger.RecordInvoice(ctx, invoice); err != nil {
			s.logger.Error("ledger write failed on trial conversion — will need reconciliation",
				"error", err, "invoice_id", invID, "amount", total)
		}
	}

	// Apply any account credit to the first invoice (ENG-154).
	s.applyCreditToInvoice(ctx, invoice)

	s.logger.Info("trial converted to active",
		"subscription_id", sub.ID, "invoice_id", invID, "amount", total)

	// Notify the customer that their first invoice is due (best-effort).
	if s.notificationService != nil {
		if err := s.notificationService.SendInvoiceCreated(ctx, InvoiceData{
			CustomerName:  domain.PtrToString(customer.Name),
			CustomerEmail: customer.Email,
			InvoiceNumber: invoice.InvoiceNumber,
			InvoiceID:     invoice.ID.String(),
			Amount:        formatAmount(total, price.Currency),
			DueDate:       dueDate.Format("Jan 02, 2006"),
		}); err != nil {
			s.logger.Error("failed to send trial conversion invoice notification", "error", err, "invoice_id", invID)
		}
	}

	return invoice, nil
}

// ExtendCurrentPeriod extends a subscription's current period end by the given number of days
func (s *SubscriptionService) ExtendCurrentPeriod(ctx context.Context, tenantID, subscriptionID uuid.UUID, days int) (*domain.Subscription, error) {
	sub, err := s.subRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}
	if sub == nil {
		return nil, fmt.Errorf("subscription not found")
	}
	if sub.TenantID != tenantID {
		return nil, fmt.Errorf("subscription not found for tenant")
	}

	sub.CurrentPeriodEnd = sub.CurrentPeriodEnd.AddDate(0, 0, days)
	sub.UpdatedAt = time.Now().UTC()

	if err := s.subRepo.Update(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to extend subscription period: %w", err)
	}

	return sub, nil
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

// AddAddon attaches an add-on plan to a subscription with a quantity. It guards
// tenant ownership of the subscription, validates the quantity, confirms the
// add-on plan exists, and requires the add-on plan's currency to match the
// subscription's base-plan currency. The add-on takes effect from the next
// recurring invoice.
func (s *SubscriptionService) AddAddon(ctx context.Context, tenantID, subscriptionID, planID uuid.UUID, quantity int) (*domain.SubscriptionAddon, error) {
	if s.addonRepo == nil {
		return nil, fmt.Errorf("add-ons are not enabled")
	}
	if quantity <= 0 {
		return nil, ErrInvalidQuantity
	}

	sub, err := s.requireOwnedSubscription(ctx, tenantID, subscriptionID)
	if err != nil {
		return nil, err
	}

	subCurrency, err := s.subscriptionCurrency(ctx, sub)
	if err != nil {
		return nil, err
	}

	addonPlan, err := s.planRepo.GetByID(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("failed to get add-on plan: %w", err)
	}
	if addonPlan == nil || len(addonPlan.Prices) == 0 {
		return nil, ErrPlanNotFound
	}
	if !strings.EqualFold(addonPlan.Prices[0].Currency, subCurrency) {
		return nil, ErrAddonCurrencyMismatch
	}

	addon := &domain.SubscriptionAddon{
		ID:             uuid.New(),
		TenantID:       tenantID,
		SubscriptionID: subscriptionID,
		PlanID:         planID,
		Quantity:       quantity,
		CreatedAt:      time.Now().UTC(),
	}
	if err := s.addonRepo.Create(ctx, addon); err != nil {
		return nil, err
	}

	s.logger.Info("subscription add-on attached",
		"subscription_id", subscriptionID, "add_on_plan_id", planID, "quantity", quantity)

	return addon, nil
}

// ListAddons returns the add-ons attached to a subscription (tenant-scoped).
func (s *SubscriptionService) ListAddons(ctx context.Context, tenantID, subscriptionID uuid.UUID) ([]*domain.SubscriptionAddon, error) {
	if s.addonRepo == nil {
		return nil, fmt.Errorf("add-ons are not enabled")
	}
	if _, err := s.requireOwnedSubscription(ctx, tenantID, subscriptionID); err != nil {
		return nil, err
	}
	return s.addonRepo.ListBySubscriptionID(ctx, tenantID, subscriptionID)
}

// RemoveAddon detaches an add-on from a subscription. It guards subscription
// ownership and confirms the add-on belongs to that subscription before
// deleting, so a valid add-on ID from another subscription cannot be removed.
func (s *SubscriptionService) RemoveAddon(ctx context.Context, tenantID, subscriptionID, addonID uuid.UUID) error {
	if s.addonRepo == nil {
		return fmt.Errorf("add-ons are not enabled")
	}
	if _, err := s.requireOwnedSubscription(ctx, tenantID, subscriptionID); err != nil {
		return err
	}

	addon, err := s.addonRepo.GetByID(ctx, tenantID, addonID)
	if err != nil {
		return err
	}
	if addon == nil || addon.SubscriptionID != subscriptionID {
		return ErrAddonNotFound
	}

	return s.addonRepo.Delete(ctx, tenantID, addonID)
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
