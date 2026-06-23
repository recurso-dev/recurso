package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/adapter/gsp"
	"github.com/recur-so/recurso/internal/core/domain"
	"github.com/recur-so/recurso/internal/core/port"
)

// --- Mocks for subscription tests ---

type subMockSubRepo struct {
	port.SubscriptionRepository
	created *domain.Subscription
	sub     *domain.Subscription
	updated *domain.Subscription
}

func (m *subMockSubRepo) Create(ctx context.Context, sub *domain.Subscription) error {
	m.created = sub
	return nil
}

func (m *subMockSubRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Subscription, error) {
	return m.sub, nil
}

func (m *subMockSubRepo) Update(ctx context.Context, sub *domain.Subscription) error {
	m.updated = sub
	return nil
}

type subMockInvoiceRepo struct {
	port.InvoiceRepository
	created *domain.Invoice
}

func (m *subMockInvoiceRepo) Create(ctx context.Context, inv *domain.Invoice) error {
	m.created = inv
	return nil
}

type subMockPlanRepo struct {
	port.PlanRepository
	plan *domain.Plan
}

func (m *subMockPlanRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Plan, error) {
	return m.plan, nil
}

type subMockCustomerRepo struct {
	port.CustomerRepository
	customer *domain.Customer
}

func (m *subMockCustomerRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Customer, error) {
	return m.customer, nil
}

type subMockCouponRepo struct {
	port.CouponRepository
	coupon *domain.Coupon
}

func (m *subMockCouponRepo) GetByCode(ctx context.Context, code string) (*domain.Coupon, error) {
	if m.coupon != nil && m.coupon.Code == code {
		return m.coupon, nil
	}
	return nil, nil
}

type subMockNotifier struct{}

func (m *subMockNotifier) SendEmail(ctx context.Context, to, subject, body string) error {
	return nil
}

type subMockGateway struct {
	port.PaymentGateway
	cancelCalls []string
	cancelErr   error
}

func (m *subMockGateway) CreateSubscription(ctx context.Context, planID string, totalCount int, customerEmail string, startAt *int64, currency string) (string, error) {
	return "sub_mock_" + planID, nil
}

func (m *subMockGateway) CancelSubscription(ctx context.Context, subscriptionID string) error {
	m.cancelCalls = append(m.cancelCalls, subscriptionID)
	return m.cancelErr
}

// newTestSubscriptionService creates a SubscriptionService with mocks for testing.
func newTestSubscriptionService(
	subRepo port.SubscriptionRepository,
	invRepo port.InvoiceRepository,
	planRepo port.PlanRepository,
	custRepo port.CustomerRepository,
	couponRepo port.CouponRepository,
	gw port.PaymentGateway,
) *SubscriptionService {
	return NewSubscriptionService(
		subRepo,
		invRepo,
		planRepo,
		custRepo,
		couponRepo,
		&subMockNotifier{},
		NewLedgerService(nil, nil),
		gw,
		gsp.NewMockGSPAdapter(),
		nil, // txManager
		nil, // revrecService
	)
}

// --- CreateSubscription Tax Tests ---

func TestCreateSubscription_IntraStateTax(t *testing.T) {
	planID := uuid.New()
	customerID := uuid.New()

	planRepo := &subMockPlanRepo{plan: &domain.Plan{
		ID:            planID,
		IntervalUnit:  domain.IntervalMonth,
		IntervalCount: 1,
		Prices:        []domain.Price{{Amount: 100000, Currency: "INR"}},
	}}
	custRepo := &subMockCustomerRepo{customer: &domain.Customer{
		ID:            customerID,
		PlaceOfSupply: domain.StringPtr("TN"),
	}}
	invRepo := &subMockInvoiceRepo{}

	svc := newTestSubscriptionService(
		&subMockSubRepo{}, invRepo, planRepo, custRepo,
		&subMockCouponRepo{}, &subMockGateway{},
	)

	sub, err := svc.CreateSubscription(context.Background(), CreateSubscriptionInput{
		TenantID:   uuid.New(),
		CustomerID: customerID,
		PlanID:     planID,
		StartDate:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sub == nil {
		t.Fatal("expected subscription, got nil")
	}

	inv := invRepo.created
	if inv == nil {
		t.Fatal("expected invoice to be created")
	}

	// Intra-state TN→TN: 9% CGST + 9% SGST = 18%
	// 18% of 100000 = 18000
	if inv.TaxAmount != 18000 {
		t.Errorf("TaxAmount = %d, want 18000", inv.TaxAmount)
	}
	if inv.CGSTAmount != 9000 {
		t.Errorf("CGSTAmount = %d, want 9000", inv.CGSTAmount)
	}
	if inv.SGSTAmount != 9000 {
		t.Errorf("SGSTAmount = %d, want 9000", inv.SGSTAmount)
	}
	if inv.IGSTAmount != 0 {
		t.Errorf("IGSTAmount = %d, want 0 for intra-state", inv.IGSTAmount)
	}
	// Total = subtotal + tax = 100000 + 18000
	if inv.Total != 118000 {
		t.Errorf("Total = %d, want 118000", inv.Total)
	}
	if inv.Subtotal != 100000 {
		t.Errorf("Subtotal = %d, want 100000", inv.Subtotal)
	}
}

func TestCreateSubscription_InterStateTax(t *testing.T) {
	planID := uuid.New()
	customerID := uuid.New()

	planRepo := &subMockPlanRepo{plan: &domain.Plan{
		ID:            planID,
		IntervalUnit:  domain.IntervalMonth,
		IntervalCount: 1,
		Prices:        []domain.Price{{Amount: 100000, Currency: "INR"}},
	}}
	custRepo := &subMockCustomerRepo{customer: &domain.Customer{
		ID:            customerID,
		PlaceOfSupply: domain.StringPtr("KA"), // Karnataka != TN
	}}
	invRepo := &subMockInvoiceRepo{}

	svc := newTestSubscriptionService(
		&subMockSubRepo{}, invRepo, planRepo, custRepo,
		&subMockCouponRepo{}, &subMockGateway{},
	)

	_, err := svc.CreateSubscription(context.Background(), CreateSubscriptionInput{
		TenantID:   uuid.New(),
		CustomerID: customerID,
		PlanID:     planID,
		StartDate:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inv := invRepo.created
	if inv == nil {
		t.Fatal("expected invoice to be created")
	}

	// Inter-state: 18% IGST
	if inv.IGSTAmount != 18000 {
		t.Errorf("IGSTAmount = %d, want 18000", inv.IGSTAmount)
	}
	if inv.CGSTAmount != 0 {
		t.Errorf("CGSTAmount = %d, want 0 for inter-state", inv.CGSTAmount)
	}
	if inv.SGSTAmount != 0 {
		t.Errorf("SGSTAmount = %d, want 0 for inter-state", inv.SGSTAmount)
	}
	if inv.Total != 118000 {
		t.Errorf("Total = %d, want 118000", inv.Total)
	}
}

func TestCreateSubscription_NilPlaceOfSupply(t *testing.T) {
	planID := uuid.New()
	customerID := uuid.New()

	planRepo := &subMockPlanRepo{plan: &domain.Plan{
		ID:            planID,
		IntervalUnit:  domain.IntervalMonth,
		IntervalCount: 1,
		Prices:        []domain.Price{{Amount: 50000, Currency: "INR"}},
	}}
	custRepo := &subMockCustomerRepo{customer: &domain.Customer{
		ID:            customerID,
		PlaceOfSupply: nil, // No PlaceOfSupply
	}}
	invRepo := &subMockInvoiceRepo{}

	svc := newTestSubscriptionService(
		&subMockSubRepo{}, invRepo, planRepo, custRepo,
		&subMockCouponRepo{}, &subMockGateway{},
	)

	_, err := svc.CreateSubscription(context.Background(), CreateSubscriptionInput{
		TenantID:   uuid.New(),
		CustomerID: customerID,
		PlanID:     planID,
		StartDate:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inv := invRepo.created
	if inv == nil {
		t.Fatal("expected invoice to be created")
	}

	// Nil PlaceOfSupply → treated as inter-state → IGST
	if inv.IGSTAmount != 9000 {
		t.Errorf("IGSTAmount = %d, want 9000 (18%% of 50000)", inv.IGSTAmount)
	}
	if inv.TaxAmount != 9000 {
		t.Errorf("TaxAmount = %d, want 9000", inv.TaxAmount)
	}
}

func TestCreateSubscription_TaxOnZeroTotal(t *testing.T) {
	planID := uuid.New()
	customerID := uuid.New()
	couponID := uuid.New()

	planRepo := &subMockPlanRepo{plan: &domain.Plan{
		ID:            planID,
		IntervalUnit:  domain.IntervalMonth,
		IntervalCount: 1,
		Prices:        []domain.Price{{Amount: 10000, Currency: "INR"}},
	}}
	custRepo := &subMockCustomerRepo{customer: &domain.Customer{
		ID:            customerID,
		PlaceOfSupply: domain.StringPtr("TN"),
	}}
	invRepo := &subMockInvoiceRepo{}
	couponRepo := &subMockCouponRepo{coupon: &domain.Coupon{
		ID:            couponID,
		Code:          "FREE100",
		DiscountType:  domain.DiscountTypePercent,
		DiscountValue: 100, // 100% off
	}}

	svc := newTestSubscriptionService(
		&subMockSubRepo{}, invRepo, planRepo, custRepo,
		couponRepo, &subMockGateway{},
	)

	_, err := svc.CreateSubscription(context.Background(), CreateSubscriptionInput{
		TenantID:   uuid.New(),
		CustomerID: customerID,
		PlanID:     planID,
		StartDate:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		CouponCode: "FREE100",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inv := invRepo.created
	if inv == nil {
		t.Fatal("expected invoice to be created")
	}

	// 100% discount → total before tax = 0 → tax = 0
	if inv.TaxAmount != 0 {
		t.Errorf("TaxAmount = %d, want 0 for fully discounted plan", inv.TaxAmount)
	}
	if inv.Total != 0 {
		t.Errorf("Total = %d, want 0 for fully discounted plan", inv.Total)
	}
}

func TestCreateSubscription_PartialDiscount(t *testing.T) {
	planID := uuid.New()
	customerID := uuid.New()
	couponID := uuid.New()

	planRepo := &subMockPlanRepo{plan: &domain.Plan{
		ID:            planID,
		IntervalUnit:  domain.IntervalMonth,
		IntervalCount: 1,
		Prices:        []domain.Price{{Amount: 10000, Currency: "INR"}},
	}}
	custRepo := &subMockCustomerRepo{customer: &domain.Customer{
		ID:            customerID,
		PlaceOfSupply: domain.StringPtr("TN"),
	}}
	invRepo := &subMockInvoiceRepo{}
	couponRepo := &subMockCouponRepo{coupon: &domain.Coupon{
		ID:            couponID,
		Code:          "HALF",
		DiscountType:  domain.DiscountTypePercent,
		DiscountValue: 50, // 50% off
	}}

	svc := newTestSubscriptionService(
		&subMockSubRepo{}, invRepo, planRepo, custRepo,
		couponRepo, &subMockGateway{},
	)

	_, err := svc.CreateSubscription(context.Background(), CreateSubscriptionInput{
		TenantID:   uuid.New(),
		CustomerID: customerID,
		PlanID:     planID,
		StartDate:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		CouponCode: "HALF",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inv := invRepo.created
	if inv == nil {
		t.Fatal("expected invoice to be created")
	}

	// 50% of 10000 = 5000 after discount
	// 18% tax on 5000 = 900 (intra-state: CGST 450 + SGST 450)
	if inv.TaxAmount != 900 {
		t.Errorf("TaxAmount = %d, want 900", inv.TaxAmount)
	}
	if inv.Total != 5900 {
		t.Errorf("Total = %d, want 5900 (5000 + 900 tax)", inv.Total)
	}
	if inv.Subtotal != 10000 {
		t.Errorf("Subtotal = %d, want 10000 (pre-discount)", inv.Subtotal)
	}
}

// --- Cancel Gateway Tests ---

func TestCancel_CallsGateway_Razorpay(t *testing.T) {
	subID := uuid.New()
	gw := &subMockGateway{}
	subRepo := &subMockSubRepo{sub: &domain.Subscription{
		ID:                     subID,
		TenantID:               uuid.New(),
		CustomerID:             uuid.New(),
		PlanID:                 uuid.New(),
		Status:                 domain.SubscriptionStatusActive,
		RazorpaySubscriptionID: "sub_rp_123",
	}}

	svc := newTestSubscriptionService(
		subRepo, &subMockInvoiceRepo{}, &subMockPlanRepo{plan: &domain.Plan{}},
		&subMockCustomerRepo{customer: &domain.Customer{}},
		&subMockCouponRepo{}, gw,
	)

	_, err := svc.Cancel(context.Background(), subRepo.sub.TenantID, subID, true, "too expensive", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(gw.cancelCalls) != 1 {
		t.Fatalf("expected 1 gateway cancel call, got %d", len(gw.cancelCalls))
	}
	if gw.cancelCalls[0] != "sub_rp_123" {
		t.Errorf("gateway called with %q, want 'sub_rp_123'", gw.cancelCalls[0])
	}
	if subRepo.updated.Status != domain.SubscriptionStatusCanceled {
		t.Errorf("status = %q, want 'canceled'", subRepo.updated.Status)
	}
}

func TestCancel_CallsGateway_Stripe(t *testing.T) {
	subID := uuid.New()
	gw := &subMockGateway{}
	subRepo := &subMockSubRepo{sub: &domain.Subscription{
		ID:                   subID,
		TenantID:             uuid.New(),
		CustomerID:           uuid.New(),
		PlanID:               uuid.New(),
		Status:               domain.SubscriptionStatusActive,
		StripeSubscriptionID: "sub_stripe_abc",
	}}

	svc := newTestSubscriptionService(
		subRepo, &subMockInvoiceRepo{}, &subMockPlanRepo{plan: &domain.Plan{}},
		&subMockCustomerRepo{customer: &domain.Customer{}},
		&subMockCouponRepo{}, gw,
	)

	_, err := svc.Cancel(context.Background(), subRepo.sub.TenantID, subID, true, "switching", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(gw.cancelCalls) != 1 {
		t.Fatalf("expected 1 gateway cancel call, got %d", len(gw.cancelCalls))
	}
	if gw.cancelCalls[0] != "sub_stripe_abc" {
		t.Errorf("gateway called with %q, want 'sub_stripe_abc'", gw.cancelCalls[0])
	}
}

func TestCancel_CallsBothGateways(t *testing.T) {
	subID := uuid.New()
	gw := &subMockGateway{}
	subRepo := &subMockSubRepo{sub: &domain.Subscription{
		ID:                     subID,
		TenantID:               uuid.New(),
		CustomerID:             uuid.New(),
		PlanID:                 uuid.New(),
		Status:                 domain.SubscriptionStatusActive,
		RazorpaySubscriptionID: "sub_rp_999",
		StripeSubscriptionID:   "sub_stripe_999",
	}}

	svc := newTestSubscriptionService(
		subRepo, &subMockInvoiceRepo{}, &subMockPlanRepo{plan: &domain.Plan{}},
		&subMockCustomerRepo{customer: &domain.Customer{}},
		&subMockCouponRepo{}, gw,
	)

	_, err := svc.Cancel(context.Background(), subRepo.sub.TenantID, subID, true, "test", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(gw.cancelCalls) != 2 {
		t.Fatalf("expected 2 gateway cancel calls, got %d", len(gw.cancelCalls))
	}
}

func TestCancel_GatewayError_DoesNotFail(t *testing.T) {
	subID := uuid.New()
	gw := &subMockGateway{cancelErr: errors.New("gateway timeout")}
	subRepo := &subMockSubRepo{sub: &domain.Subscription{
		ID:                     subID,
		TenantID:               uuid.New(),
		CustomerID:             uuid.New(),
		PlanID:                 uuid.New(),
		Status:                 domain.SubscriptionStatusActive,
		RazorpaySubscriptionID: "sub_rp_fail",
	}}

	svc := newTestSubscriptionService(
		subRepo, &subMockInvoiceRepo{}, &subMockPlanRepo{plan: &domain.Plan{}},
		&subMockCustomerRepo{customer: &domain.Customer{}},
		&subMockCouponRepo{}, gw,
	)

	result, err := svc.Cancel(context.Background(), subRepo.sub.TenantID, subID, true, "test", "")
	if err != nil {
		t.Fatalf("cancel should succeed even if gateway fails, got: %v", err)
	}
	if result.Status != "canceled" {
		t.Errorf("status = %q, want 'canceled'", result.Status)
	}
	// Gateway was still called
	if len(gw.cancelCalls) != 1 {
		t.Errorf("expected 1 gateway call despite error, got %d", len(gw.cancelCalls))
	}
}

func TestCancel_NoGatewayIDs_NoGatewayCalls(t *testing.T) {
	subID := uuid.New()
	gw := &subMockGateway{}
	subRepo := &subMockSubRepo{sub: &domain.Subscription{
		ID:         subID,
		TenantID:   uuid.New(),
		CustomerID: uuid.New(),
		PlanID:     uuid.New(),
		Status:     domain.SubscriptionStatusActive,
		// No gateway IDs set
	}}

	svc := newTestSubscriptionService(
		subRepo, &subMockInvoiceRepo{}, &subMockPlanRepo{plan: &domain.Plan{}},
		&subMockCustomerRepo{customer: &domain.Customer{}},
		&subMockCouponRepo{}, gw,
	)

	_, err := svc.Cancel(context.Background(), subRepo.sub.TenantID, subID, true, "test", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(gw.cancelCalls) != 0 {
		t.Errorf("expected 0 gateway calls, got %d", len(gw.cancelCalls))
	}
}

func TestCancel_AtPeriodEnd(t *testing.T) {
	subID := uuid.New()
	periodEnd := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	subRepo := &subMockSubRepo{sub: &domain.Subscription{
		ID:               subID,
		TenantID:         uuid.New(),
		CustomerID:       uuid.New(),
		PlanID:           uuid.New(),
		Status:           domain.SubscriptionStatusActive,
		CurrentPeriodEnd: periodEnd,
	}}

	svc := newTestSubscriptionService(
		subRepo, &subMockInvoiceRepo{}, &subMockPlanRepo{plan: &domain.Plan{}},
		&subMockCustomerRepo{customer: &domain.Customer{}},
		&subMockCouponRepo{}, &subMockGateway{},
	)

	result, err := svc.Cancel(context.Background(), subRepo.sub.TenantID, subID, false, "downgrading", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Not immediately cancelled — set to cancel at period end
	if result.Status != string(domain.SubscriptionStatusActive) {
		t.Errorf("status = %q, want 'active' (cancel at period end)", result.Status)
	}
	if !subRepo.updated.CancelAtPeriodEnd {
		t.Error("expected CancelAtPeriodEnd to be true")
	}
	if subRepo.updated.CanceledAt != nil {
		t.Error("expected CanceledAt to be nil for end-of-period cancel")
	}
}

// --- Tenant Isolation Tests ---

func TestCancel_WrongTenant_Rejected(t *testing.T) {
	subID := uuid.New()
	ownerTenant := uuid.New()
	attackerTenant := uuid.New()

	subRepo := &subMockSubRepo{sub: &domain.Subscription{
		ID:         subID,
		TenantID:   ownerTenant,
		CustomerID: uuid.New(),
		PlanID:     uuid.New(),
		Status:     domain.SubscriptionStatusActive,
	}}

	svc := newTestSubscriptionService(
		subRepo, &subMockInvoiceRepo{}, &subMockPlanRepo{plan: &domain.Plan{}},
		&subMockCustomerRepo{customer: &domain.Customer{}},
		&subMockCouponRepo{}, &subMockGateway{},
	)

	_, err := svc.Cancel(context.Background(), attackerTenant, subID, true, "test", "")
	if err == nil {
		t.Fatal("expected error when cancelling subscription from wrong tenant")
	}
	if err.Error() != "subscription not found for tenant" {
		t.Errorf("error = %q, want 'subscription not found for tenant'", err.Error())
	}
	if subRepo.updated != nil {
		t.Error("subscription should not be updated when tenant doesn't match")
	}
}

func TestReactivate_WrongTenant_Rejected(t *testing.T) {
	subID := uuid.New()
	ownerTenant := uuid.New()
	attackerTenant := uuid.New()

	subRepo := &subMockSubRepo{sub: &domain.Subscription{
		ID:                subID,
		TenantID:          ownerTenant,
		CustomerID:        uuid.New(),
		PlanID:            uuid.New(),
		Status:            domain.SubscriptionStatusActive,
		CancelAtPeriodEnd: true,
		CurrentPeriodEnd:  time.Now().Add(24 * time.Hour),
	}}

	svc := newTestSubscriptionService(
		subRepo, &subMockInvoiceRepo{}, &subMockPlanRepo{plan: &domain.Plan{}},
		&subMockCustomerRepo{customer: &domain.Customer{}},
		&subMockCouponRepo{}, &subMockGateway{},
	)

	_, err := svc.Reactivate(context.Background(), attackerTenant, subID)
	if err == nil {
		t.Fatal("expected error when reactivating subscription from wrong tenant")
	}
	if err.Error() != "subscription not found for tenant" {
		t.Errorf("error = %q, want 'subscription not found for tenant'", err.Error())
	}
	if subRepo.updated != nil {
		t.Error("subscription should not be updated when tenant doesn't match")
	}
}

func TestCancel_CorrectTenant_Succeeds(t *testing.T) {
	subID := uuid.New()
	tenantID := uuid.New()

	subRepo := &subMockSubRepo{sub: &domain.Subscription{
		ID:         subID,
		TenantID:   tenantID,
		CustomerID: uuid.New(),
		PlanID:     uuid.New(),
		Status:     domain.SubscriptionStatusActive,
	}}

	svc := newTestSubscriptionService(
		subRepo, &subMockInvoiceRepo{}, &subMockPlanRepo{plan: &domain.Plan{}},
		&subMockCustomerRepo{customer: &domain.Customer{}},
		&subMockCouponRepo{}, &subMockGateway{},
	)

	result, err := svc.Cancel(context.Background(), tenantID, subID, true, "test", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "canceled" {
		t.Errorf("status = %q, want 'canceled'", result.Status)
	}
}

func TestReactivate_CorrectTenant_Succeeds(t *testing.T) {
	subID := uuid.New()
	tenantID := uuid.New()

	subRepo := &subMockSubRepo{sub: &domain.Subscription{
		ID:                subID,
		TenantID:          tenantID,
		CustomerID:        uuid.New(),
		PlanID:            uuid.New(),
		Status:            domain.SubscriptionStatusActive,
		CancelAtPeriodEnd: true,
		CurrentPeriodEnd:  time.Now().Add(24 * time.Hour),
	}}

	svc := newTestSubscriptionService(
		subRepo, &subMockInvoiceRepo{}, &subMockPlanRepo{plan: &domain.Plan{}},
		&subMockCustomerRepo{customer: &domain.Customer{}},
		&subMockCouponRepo{}, &subMockGateway{},
	)

	result, err := svc.Reactivate(context.Background(), tenantID, subID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != domain.SubscriptionStatusActive {
		t.Errorf("status = %q, want 'active'", result.Status)
	}
	if result.CancelAtPeriodEnd {
		t.Error("expected CancelAtPeriodEnd to be false after reactivation")
	}
}
