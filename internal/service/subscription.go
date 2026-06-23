package service

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/adapter/db"
	"github.com/recur-so/recurso/internal/core/domain"
	"github.com/recur-so/recurso/internal/core/port"
	"github.com/recur-so/recurso/internal/core/service/tax"
)

type SubscriptionService struct {
	subRepo         port.SubscriptionRepository
	invoiceRepo     port.InvoiceRepository
	planRepo        port.PlanRepository
	customerRepo    port.CustomerRepository
	couponRepo      port.CouponRepository
	notifier        port.Notifier
	ledger          *LedgerService
	gateway         port.PaymentGateway
	gspAdapter      port.GSPAdapter
	notificationService *NotificationService
	einvoiceService     *EInvoiceService
	txManager           *db.TxManager
	subRepoImpl     *db.SubscriptionRepository // Concrete type for TX methods
	invRepoImpl     *db.InvoiceRepository      // Concrete type for TX methods
	revrecService   *RevRecService
	logger          *slog.Logger
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

type CreateSubscriptionInput struct {
	TenantID          uuid.UUID
	CustomerID        uuid.UUID
	PlanID            uuid.UUID
	StartDate         time.Time
	CouponCode        string
	BillingAnchorType string // "acquisition" (default) or "first_of_month"
	PaymentTerms      string // "net0", "net15", "net30", "net60", "due_on_receipt"
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

	// Calculate Tax (GST)
	taxEngine := tax.NewGSTEngine("TN")
	pos := customer.PlaceOfSupply
	if domain.PtrToString(pos) == "" {
		pos = nil
	}
	taxRes := taxEngine.CalculateTaxLegacy(total, domain.PtrToString(pos))
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

	// P25: E-Invoicing via EInvoiceService
	if s.einvoiceService != nil {
		_, einvErr := s.einvoiceService.GenerateEInvoice(ctx, invoice)
		if einvErr != nil {
			s.logger.Error("e-invoice generation failed (will retry)", "error", einvErr, "invoice_id", invID)
		}
	} else if customer.BillingAddress.Country == "India" && domain.PtrToString(customer.GSTIN) != "" && customer.TaxType == "business" {
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
		Status:             domain.SubscriptionStatusActive,
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

	// Atomic: Create subscription + invoice in a single transaction
	if s.txManager != nil && s.subRepoImpl != nil && s.invRepoImpl != nil {
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

	if err := s.invoiceRepo.Update(ctx, inv); err != nil {
		return err
	}

	// Phase 5: Create Revenue Recognition Schedule
	if s.revrecService != nil {
		if err := s.revrecService.CreateScheduleForInvoice(ctx, inv); err != nil {
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

	// 3. Calculate Proration
	now := time.Now().UTC()

	// Assuming single price for MVP simplification
	currentPrice := currentPlan.Prices[0].Amount
	newPrice := newPlan.Prices[0].Amount

	proration := s.CalculateProration(
		currentPrice,
		newPrice,
		sub.CurrentPeriodStart,
		sub.CurrentPeriodEnd,
		now,
	)

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
			Currency:       newPlan.Prices[0].Currency,
			Subtotal:       proration.NetAmount,
			TaxAmount:      0,
			Total:          proration.NetAmount,
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

	// Sync with Gateway (Razorpay/Stripe)
	if s.gateway != nil && sub.RazorpaySubscriptionID != "" {
		// Mock implementation call to gateway update
		// s.gateway.UpdateSubscription(ctx, sub.RazorpaySubscriptionID, newPlan.Code)
	}

	return sub, nil
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
