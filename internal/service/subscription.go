package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/adapter/db"
	"github.com/swapnull-in/recur-so/internal/adapter/telemetry"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

// Sentinel errors let handlers map service failures to the right HTTP status
// (e.g. 404) without brittle string matching.
var (
	ErrSubscriptionNotFound = errors.New("subscription not found")
	ErrPlanNotFound         = errors.New("plan not found")
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
	revrecService       *RevRecService
	taxResolver         *TaxResolver
	recoveryRecorder    PaymentRecoveryRecorder
	telemetry           *telemetry.Client // nil-safe; only set when TELEMETRY_OPTIN=true
	logger              *slog.Logger
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

	if anchorType == "first_of_month" && start.Day() != 1 {
		// Prorate to first of next month
		year, month, _ := start.Date()
		end = time.Date(year, month, 1, 0, 0, 0, 0, start.Location()).AddDate(0, 1, 0)
	} else {
		switch plan.IntervalUnit {
		case domain.IntervalMonth:
			end = start.AddDate(0, plan.IntervalCount, 0)
		case domain.IntervalYear:
			end = start.AddDate(plan.IntervalCount, 0, 0)
		case domain.IntervalWeek:
			end = start.AddDate(0, 0, 7*plan.IntervalCount)
		case domain.IntervalDay:
			end = start.AddDate(0, 0, plan.IntervalCount)
		}
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

	subtotal := price.Amount
	discount := int64(0)
	var couponID *uuid.UUID

	if input.CouponCode != "" {
		coupon, err := s.couponRepo.GetByCode(ctx, input.CouponCode)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch coupon: %w", err)
		}
		if coupon == nil {
			return nil, fmt.Errorf("invalid coupon code")
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
	taxRes := s.taxResolver.ResolveInvoiceTax(ctx, input.TenantID, customer, price.Currency, total)
	total = total + taxRes.Total

	subID := uuid.New()
	invID := uuid.New()

	// Calculate Due Date based on payment terms
	paymentTerms := input.PaymentTerms
	if paymentTerms == "" {
		paymentTerms = "due_on_receipt"
	}
	dueDate := domain.CalculateDueDate(time.Now().UTC(), paymentTerms)

	// Create Invoice with Discount applied
	invoice := &domain.Invoice{
		ID:             invID,
		TenantID:       input.TenantID,
		SubscriptionID: &subID,
		CustomerID:     input.CustomerID,
		InvoiceNumber:  fmt.Sprintf("INV-%d-%s", time.Now().UnixNano(), invID.String()[:8]),
		Status:         domain.InvoiceStatusOpen,
		Currency:       price.Currency,
		Subtotal:       subtotal,
		TaxAmount:      taxRes.Total,
		Total:          total,
		IGSTAmount:     taxRes.IGST,
		CGSTAmount:     taxRes.CGST,
		SGSTAmount:     taxRes.SGST,
		PaymentTerms:   paymentTerms,
		CreatedAt:      time.Now().UTC(),
		DueDate:        dueDate,
		PaidAt:         nil,
	}

	// P25: E-Invoicing via EInvoiceService. Skipped for trials — the first
	// invoice (and its IRN) is generated when the trial converts to active.
	if !isTrial && s.einvoiceService != nil {
		_, einvErr := s.einvoiceService.GenerateEInvoice(ctx, invoice)
		if einvErr != nil {
			s.logger.Error("e-invoice generation failed (will retry)", "error", einvErr, "invoice_id", invID)
		}
	} else if !isTrial && customer.BillingAddress.Country == "India" && domain.PtrToString(customer.GSTIN) != "" && customer.TaxType == "business" {
		// Fallback: direct GSP call (backward compat)
		resp, err := s.gspAdapter.GenerateIRN(ctx, invoice)
		if err == nil {
			invoice.IRN = resp.IRN
			invoice.SignedQRCode = resp.SignedQRCode
			invoice.EInvoiceStatus = "GENERATED"
			invoice.AckNo = resp.AckNo
		} else {
			s.logger.Error("e-invoicing IRN generation failed", "error", err, "invoice_id", invID)
			invoice.EInvoiceStatus = "FAILED"
		}
	} else {
		invoice.EInvoiceStatus = "PENDING"
	}

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
			s.logger.Error("payment gateway subscription creation failed",
				"error", err,
				"plan_code", plan.Code,
				"customer_email", customer.Email,
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

func (s *SubscriptionService) MarkInvoicePaid(ctx context.Context, invoiceID uuid.UUID) error {
	inv, err := s.invoiceRepo.GetByID(ctx, invoiceID)
	if err != nil {
		return err
	}
	if inv == nil {
		return fmt.Errorf("invoice not found")
	}

	if inv.Status == domain.InvoiceStatusPaid {
		return nil // Already paid
	}

	now := time.Now().UTC()
	inv.Status = domain.InvoiceStatusPaid
	inv.PaidAt = &now
	inv.AmountPaid = inv.Total

	if err := s.invoiceRepo.Update(ctx, inv); err != nil {
		return err
	}

	s.telemetry.MilestoneFirstPayment() // opt-in anonymous milestone; no-op when disabled

	// Dunning recovery attribution: if this invoice needed retries or dunning
	// to get paid, record it as recovered revenue (idempotent, non-fatal).
	if s.recoveryRecorder != nil {
		s.recoveryRecorder.RecordIfRecovered(ctx, inv)
	}

	// Record payment in ledger
	if s.ledger != nil {
		if err := s.ledger.RecordPayment(ctx, inv); err != nil {
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

	return nil
}

func (s *SubscriptionService) ListSubscriptions(ctx context.Context, tenantID uuid.UUID, filter domain.SubscriptionFilter) ([]*domain.Subscription, error) {
	return s.subRepo.List(ctx, tenantID, filter)
}

// PauseSubscription pauses an active subscription (Phase 49)
func (s *SubscriptionService) PauseSubscription(ctx context.Context, tenantID, subscriptionID uuid.UUID) (*domain.Subscription, error) {
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

	if sub.Status != domain.SubscriptionStatusActive {
		return nil, fmt.Errorf("only active subscriptions can be paused")
	}

	sub.Status = domain.SubscriptionStatusPaused

	if err := s.subRepo.Update(ctx, sub); err != nil {
		return nil, err
	}

	return sub, nil
}

// ResumeSubscription resumes a paused subscription (Phase 49)
func (s *SubscriptionService) ResumeSubscription(ctx context.Context, tenantID, subscriptionID uuid.UUID) (*domain.Subscription, error) {
	sub, err := s.subRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}
	if sub.TenantID != tenantID {
		return nil, fmt.Errorf("subscription not found for tenant")
	}

	if sub.Status != domain.SubscriptionStatusPaused {
		return nil, fmt.Errorf("only paused subscriptions can be resumed")
	}

	sub.Status = domain.SubscriptionStatusActive

	if err := s.subRepo.Update(ctx, sub); err != nil {
		return nil, err
	}

	return sub, nil
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

	// Get customer and plan info for notification
	customer, _ := s.customerRepo.GetByID(ctx, sub.CustomerID)
	plan, _ := s.planRepo.GetByID(ctx, sub.PlanID)

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
// math. Tax is only applied to a positive net (a charge); credits carry none,
// mirroring the invoice built in UpdateSubscription.
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
	if proration.NetAmount > 0 && customer != nil {
		taxRes = s.taxResolver.ResolveInvoiceTax(ctx, tenantID, customer, currency, proration.NetAmount)
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
	TaxAmount         int64     `json:"tax_amount"`          // tax on a positive net (0 for credits)
	TotalAmount       int64     `json:"total_amount"`        // net + tax: the immediate proration invoice total
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
		nextTax = s.taxResolver.ResolveInvoiceTax(ctx, tenantID, customer, pcp.Currency, newPrice)
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

	// 4. Create Invoice for Proration (if diff > 0)
	// If NetAmount is positive, charge user immediately or add to next bill
	// If NetAmount is negative, add credit
	if proration.NetAmount != 0 {
		invoice := &domain.Invoice{
			ID:             uuid.New(),
			TenantID:       tenantID,
			SubscriptionID: &sub.ID,
			CustomerID:     sub.CustomerID,
			InvoiceNumber:  fmt.Sprintf("INV-PR-%d-%s", time.Now().UnixNano(), uuid.New().String()[:8]),
			Status:         domain.InvoiceStatusOpen, // Or Draft if adding to next bill
			Currency:       pcp.Currency,
			Subtotal:       proration.NetAmount,
			TaxAmount:      taxRes.Total,
			IGSTAmount:     taxRes.IGST,
			CGSTAmount:     taxRes.CGST,
			SGSTAmount:     taxRes.SGST,
			Total:          proration.NetAmount + taxRes.Total,
			CreatedAt:      now,
			DueDate:        now,
		}

		if proration.NetAmount < 0 {
			invoice.Status = domain.InvoiceStatusPaid // Credit note essentially
			// In real world, we'd create a Credit Note entity, but reusing Invoice for P1 simplification
			invoice.Total = 0 // Credits don't require payment
			// Improve: Record 'CreditBalance' on Customer entity
		}

		// P25: E-Invoicing via EInvoiceService
		if proration.NetAmount > 0 && s.einvoiceService != nil {
			_, einvErr := s.einvoiceService.GenerateEInvoice(ctx, invoice)
			if einvErr != nil {
				s.logger.Error("e-invoice generation failed for proration (will retry)", "error", einvErr)
			}
		} else if proration.NetAmount > 0 && customer.BillingAddress.Country == "India" && domain.PtrToString(customer.GSTIN) != "" && customer.TaxType == "business" {
			// Fallback: direct GSP call
			resp, err := s.gspAdapter.GenerateIRN(ctx, invoice)
			if err == nil {
				invoice.IRN = resp.IRN
				invoice.SignedQRCode = resp.SignedQRCode
				invoice.EInvoiceStatus = "GENERATED"
				invoice.AckNo = resp.AckNo
			} else {
				s.logger.Error("error generating IRN for proration invoice", "error", err)
				invoice.EInvoiceStatus = "FAILED"
			}
		}

		if err := s.invoiceRepo.Create(ctx, invoice); err != nil {
			return nil, fmt.Errorf("failed to create proration invoice: %w", err)
		}
	}

	// 5. Update Subscription
	sub.PlanID = newPlanID
	sub.UpdatedAt = now

	// Reset period? Usually we keep the same period but change the plan
	// Stripe keeps the anchor. So period end remains same.

	if err := s.subRepo.Update(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to update subscription: %w", err)
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
	taxRes := s.taxResolver.ResolveInvoiceTax(ctx, sub.TenantID, customer, price.Currency, subtotal)
	total := subtotal + taxRes.Total

	paymentTerms := sub.PaymentTerms
	if paymentTerms == "" {
		paymentTerms = "due_on_receipt"
	}
	dueDate := domain.CalculateDueDate(now, paymentTerms)

	invID := uuid.New()
	invoice := &domain.Invoice{
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
		IGSTAmount:     taxRes.IGST,
		CGSTAmount:     taxRes.CGST,
		SGSTAmount:     taxRes.SGST,
		PaymentTerms:   paymentTerms,
		CreatedAt:      now,
		DueDate:        dueDate,
	}

	// E-invoicing follows the same rules as the first invoice in CreateSubscription.
	if s.einvoiceService != nil {
		if _, einvErr := s.einvoiceService.GenerateEInvoice(ctx, invoice); einvErr != nil {
			s.logger.Error("e-invoice generation failed on trial conversion (will retry)", "error", einvErr, "invoice_id", invID)
		}
	} else {
		invoice.EInvoiceStatus = "PENDING"
	}

	// Persist the first invoice, then activate the subscription. Mirrors the
	// non-atomic invoice-then-subscription ordering used by UpdateSubscription.
	if err := s.invoiceRepo.Create(ctx, invoice); err != nil {
		return nil, fmt.Errorf("failed to create trial conversion invoice: %w", err)
	}

	sub.Status = domain.SubscriptionStatusActive
	sub.CurrentPeriodStart = start
	sub.CurrentPeriodEnd = end
	sub.UpdatedAt = now
	if err := s.subRepo.Update(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to activate subscription after trial: %w", err)
	}

	// Dual-write to ledger (best-effort; reconciliation covers gaps).
	if s.ledger != nil {
		if err := s.ledger.RecordInvoice(ctx, invoice); err != nil {
			s.logger.Error("ledger write failed on trial conversion — will need reconciliation",
				"error", err, "invoice_id", invID, "amount", total)
		}
	}

	s.logger.Info("trial converted to active",
		"subscription_id", sub.ID, "invoice_id", invID, "amount", total)

	// Notify the customer that their first invoice is due (best-effort).
	if s.notificationService != nil {
		if err := s.notificationService.SendInvoiceCreated(ctx, InvoiceData{
			CustomerName:  domain.PtrToString(customer.Name),
			CustomerEmail: customer.Email,
			InvoiceNumber: invoice.InvoiceNumber,
			Amount:        formatAmount(total, price.Currency),
			DueDate:       dueDate.Format("Jan 02, 2006"),
			PaymentURL:    "",
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
