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
	// ErrInvalidSubscriptionState is returned when an operation is illegal
	// for the subscription's current status (pause a trial, resume an active
	// sub, ...). Handlers map it to 409.
	ErrInvalidSubscriptionState = errors.New("invalid subscription state")
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
	paymentAttempts     paymentAttemptLister             // nil-safe; the invoice page's payment-attempt history
	invoiceStatusLog    invoiceStatusHistoryReader       // nil-safe; the invoice page's status timeline
	subHistory          subscriptionHistoryReader        // nil-safe; the subscription page's status+plan timeline
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
	applied, err := s.creditApplier.ApplyAdjustmentCredits(ctx, inv.TenantID, inv.CustomerID, inv.EntityID, inv.Currency, inv.ID, inv.Total)
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
			return nil, ErrSubscriptionNotFound
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
		return nil, ErrPlanNotFound
	}

	// 2. Fetch Customer
	customer, err := s.customerRepo.GetByID(ctx, input.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}
	if customer == nil {
		return nil, ErrCustomerNotFound
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
		// A discount can never exceed the subtotal. Clamping here (not just the
		// header Total) keeps the line's taxable base ≥ 0 — an over-subtotal
		// fixed-amount coupon, or a >100% percent coupon, otherwise persisted a
		// NEGATIVE taxable_amount that corrupts the IRP e-invoice assessable value
		// and the GST liability report.
		if discount > subtotal {
			discount = subtotal
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
		EntityID:       input.EntityID,
		SubscriptionID: &subID,
		CustomerID:     input.CustomerID,
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
	// The initial invoice (built below) applies the coupon, so this is period 1
	// of its duration — record it so renewals continue a `forever`/`repeating`
	// coupon and stop a `once`/expired `repeating` one. NOT for a trial: a trial
	// defers its first invoice to conversion, so the coupon hasn't been applied
	// yet — the counter stays 0 and ConvertTrialToActive advances it.
	if !isTrial && couponID != nil && discount > 0 {
		sub.CouponPeriodsApplied = 1
		// R3: the first period's invoice carries the discount — proration must
		// credit/charge this period at the discounted prices.
		sub.CouponAppliedCurrentPeriod = true
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

// ListInvoicesPaginated returns one page of the tenant's invoices plus the total
// count (for pagination metadata). The API list endpoint uses this instead of
// the unbounded List.
// GetInvoice returns one invoice by id. The repository enforces tenant
// scoping from the context tenant key, so a foreign or missing invoice
// returns (nil, err) and the handler responds with a flat 404. Serves the
// dashboard's addressable /invoices/:id route.
func (s *SubscriptionService) GetInvoice(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	return s.invoiceRepo.GetByID(ctx, id)
}

// GetInvoiceJournalEntries returns the invoice's ledger postings (its journal
// drill), each with DR/CR account names. It verifies the invoice exists
// (tenant-scoped) first and returns (nil, nil) when it doesn't, so a bad id is a
// 404 rather than an empty journal. Without a ledger wired, returns an empty
// journal, never an error.
func (s *SubscriptionService) GetInvoiceJournalEntries(ctx context.Context, tenantID, invoiceID uuid.UUID) ([]domain.GeneralLedgerRow, error) {
	inv, err := s.invoiceRepo.GetByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, nil
	}
	if s.ledger == nil {
		return []domain.GeneralLedgerRow{}, nil
	}
	entries, err := s.ledger.GetJournalEntriesByReference(ctx, tenantID, invoiceID)
	if err != nil {
		return nil, err
	}
	if entries == nil {
		// The invoice exists but has no postings yet (e.g. a draft) — an empty
		// journal, distinct from a missing invoice (which returns nil above → 404).
		entries = []domain.GeneralLedgerRow{}
	}
	return entries, nil
}

// paymentAttemptLister is the narrow read the payment views need: an invoice's
// history, and the tenant-wide payments log.
type paymentAttemptLister interface {
	ListByInvoice(ctx context.Context, tenantID, invoiceID uuid.UUID) ([]*domain.PaymentAttempt, error)
	List(ctx context.Context, tenantID uuid.UUID, status string, limit, offset int) ([]domain.PaymentAttemptListItem, int, error)
}

// SetPaymentAttemptLister enables GetInvoicePaymentAttempts. Without it, that
// endpoint reports an empty history rather than erroring.
func (s *SubscriptionService) SetPaymentAttemptLister(l paymentAttemptLister) { s.paymentAttempts = l }

// GetInvoicePaymentAttempts returns an invoice's payment attempts — its
// settlement/retry history. Verifies the invoice exists (tenant-scoped) first
// and returns (nil, nil) when it doesn't, so a bad id is a 404 rather than an
// empty history.
func (s *SubscriptionService) GetInvoicePaymentAttempts(ctx context.Context, tenantID, invoiceID uuid.UUID) ([]*domain.PaymentAttempt, error) {
	inv, err := s.invoiceRepo.GetByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, nil
	}
	if s.paymentAttempts == nil {
		return []*domain.PaymentAttempt{}, nil
	}
	attempts, err := s.paymentAttempts.ListByInvoice(ctx, tenantID, invoiceID)
	if err != nil {
		return nil, err
	}
	if attempts == nil {
		attempts = []*domain.PaymentAttempt{}
	}
	return attempts, nil
}

// invoiceStatusHistoryReader is the narrow read the invoice status timeline
// needs — an invoice's recorded status transitions (trigger-captured).
type invoiceStatusHistoryReader interface {
	ListByInvoice(ctx context.Context, tenantID, invoiceID uuid.UUID) ([]domain.InvoiceStatusChange, error)
}

// subscriptionHistoryReader is the narrow read the subscription timeline needs —
// a subscription's recorded status and plan transitions (trigger-captured).
type subscriptionHistoryReader interface {
	ListBySubscription(ctx context.Context, tenantID, subscriptionID uuid.UUID) ([]domain.SubscriptionChange, error)
}

// SetSubscriptionHistoryReader enables GetSubscriptionHistory. Without it, the
// endpoint reports an empty timeline rather than erroring.
func (s *SubscriptionService) SetSubscriptionHistoryReader(r subscriptionHistoryReader) {
	s.subHistory = r
}

// GetSubscriptionHistory returns a subscription's recorded status and plan
// transitions (oldest first). Verifies the subscription is tenant-owned first
// and returns (nil, nil) when it's missing or another tenant's, so a bad id is a
// 404 rather than an empty timeline.
func (s *SubscriptionService) GetSubscriptionHistory(ctx context.Context, tenantID, subscriptionID uuid.UUID) ([]domain.SubscriptionChange, error) {
	sub, err := s.GetByID(ctx, tenantID, subscriptionID)
	if err != nil {
		if errors.Is(err, ErrSubscriptionNotFound) {
			return nil, nil // cross-tenant → 404, never confirm existence
		}
		return nil, err
	}
	if sub == nil {
		return nil, nil
	}
	if s.subHistory == nil {
		return []domain.SubscriptionChange{}, nil
	}
	changes, err := s.subHistory.ListBySubscription(ctx, tenantID, subscriptionID)
	if err != nil {
		return nil, err
	}
	if changes == nil {
		changes = []domain.SubscriptionChange{}
	}
	return changes, nil
}

// SetInvoiceStatusHistoryReader enables GetInvoiceStatusHistory. Without it, the
// endpoint reports an empty timeline rather than erroring.
func (s *SubscriptionService) SetInvoiceStatusHistoryReader(r invoiceStatusHistoryReader) {
	s.invoiceStatusLog = r
}

// GetInvoiceStatusHistory returns an invoice's status transitions (oldest
// first). Verifies the invoice exists (tenant-scoped) first and returns
// (nil, nil) when it doesn't, so a bad id is a 404 rather than an empty timeline.
func (s *SubscriptionService) GetInvoiceStatusHistory(ctx context.Context, tenantID, invoiceID uuid.UUID) ([]domain.InvoiceStatusChange, error) {
	inv, err := s.invoiceRepo.GetByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, nil
	}
	if s.invoiceStatusLog == nil {
		return []domain.InvoiceStatusChange{}, nil
	}
	changes, err := s.invoiceStatusLog.ListByInvoice(ctx, tenantID, invoiceID)
	if err != nil {
		return nil, err
	}
	if changes == nil {
		changes = []domain.InvoiceStatusChange{}
	}
	return changes, nil
}

// ListPaymentAttempts returns the tenant-wide payments log (attempts, newest
// first, paginated, optional status filter) with each attempt's invoice number.
// Returns an empty page + zero total when no lister is wired.
func (s *SubscriptionService) ListPaymentAttempts(ctx context.Context, tenantID uuid.UUID, status string, limit, offset int) ([]domain.PaymentAttemptListItem, int, error) {
	if s.paymentAttempts == nil {
		return []domain.PaymentAttemptListItem{}, 0, nil
	}
	items, total, err := s.paymentAttempts.List(ctx, tenantID, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	if items == nil {
		items = []domain.PaymentAttemptListItem{}
	}
	return items, total, nil
}

func (s *SubscriptionService) ListInvoicesPaginated(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Invoice, int, error) {
	invs, err := s.invoiceRepo.ListPaginated(ctx, tenantID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.invoiceRepo.CountByTenant(ctx, tenantID)
	if err != nil {
		return nil, 0, err
	}
	return invs, total, nil
}

// ListInvoicesBySubscriptionPaginated returns one page of a subscription's
// invoices within the tenant plus the scoped total. Serves the dashboard's
// subscription object page (GET /invoices?subscription_id=).
func (s *SubscriptionService) ListInvoicesBySubscriptionPaginated(ctx context.Context, tenantID, subscriptionID uuid.UUID, limit, offset int) ([]*domain.Invoice, int, error) {
	invs, err := s.invoiceRepo.ListBySubscriptionPaginated(ctx, tenantID, subscriptionID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.invoiceRepo.CountBySubscription(ctx, tenantID, subscriptionID)
	if err != nil {
		return nil, 0, err
	}
	return invs, total, nil
}

// ListInvoicesByCustomerPaginated returns one page of a customer's invoices
// within the tenant plus the customer-scoped total. Serves the dashboard's
// customer object page (GET /invoices?customer_id=).
func (s *SubscriptionService) ListInvoicesByCustomerPaginated(ctx context.Context, tenantID, customerID uuid.UUID, limit, offset int) ([]*domain.Invoice, int, error) {
	invs, err := s.invoiceRepo.ListByCustomerPaginated(ctx, tenantID, customerID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.invoiceRepo.CountByCustomer(ctx, tenantID, customerID)
	if err != nil {
		return nil, 0, err
	}
	return invs, total, nil
}

// GetByID retrieves a subscription by ID
func (s *SubscriptionService) GetByID(ctx context.Context, tenantID, subscriptionID uuid.UUID) (*domain.Subscription, error) {
	sub, err := s.subRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}
	if sub != nil && sub.TenantID != tenantID {
		return nil, ErrSubscriptionNotFound
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
	// Exponent-aware: hardcoding /100 misstated non-2-decimal currencies.
	return domain.FormatMoney(amountPaise, currency)
}

// couponAppliesThisPeriod reports whether a subscription's coupon still applies
// for a billing period, given how many periods it has already been applied to:
//   - forever   → every period
//   - once      → only the first (periodsApplied == 0)
//   - repeating → the first DurationMonths periods
//
// A nil/inactive coupon never applies.
func couponAppliesThisPeriod(coupon *domain.Coupon, periodsApplied int) bool {
	if coupon == nil || !coupon.Active {
		return false
	}
	switch coupon.Duration {
	case domain.DurationForever:
		return true
	case domain.DurationOnce:
		return periodsApplied == 0
	case domain.DurationRepeating:
		n := 0
		if coupon.DurationMonths != nil {
			n = *coupon.DurationMonths
		}
		return periodsApplied < n
	default:
		return false
	}
}

// couponDiscountFor returns the discount a coupon applies to a single-line base
// amount, clamped to [0, base] so it can never drive the taxable base negative.
// Returns 0 for a nil/inactive coupon. Shared by the trial-conversion and
// renewal coupon paths so they match the CreateSubscription discount math.
func couponDiscountFor(coupon *domain.Coupon, base int64) int64 {
	if coupon == nil || !coupon.Active || base <= 0 {
		return 0
	}
	var d int64
	if coupon.DiscountType == domain.DiscountTypePercent {
		d = (base * coupon.DiscountValue) / 100
	} else {
		d = coupon.DiscountValue
	}
	if d > base {
		d = base
	}
	if d < 0 {
		d = 0
	}
	return d
}
