package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/gsp"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

type mockCouponRepoForInvoice struct {
	port.CouponRepository
	coupon *domain.Coupon
}

func (m *mockCouponRepoForInvoice) GetByID(_ context.Context, tenantID, id uuid.UUID) (*domain.Coupon, error) {
	if m.coupon != nil && m.coupon.ID == id && m.coupon.TenantID == tenantID {
		return m.coupon, nil
	}
	return nil, nil
}

// TestGenerateInvoice_AppliesForeverCoupon proves a `forever` coupon on the
// subscription discounts the flat plan fee on every renewal. Before the fix
// GenerateInvoice ignored sub.CouponID entirely, so the customer was billed the
// full price + full tax every period after the first.
func TestGenerateInvoice_AppliesForeverCoupon(t *testing.T) {
	planID := uuid.New()
	custID := uuid.New()
	tenantID := uuid.New()
	couponID := uuid.New()

	svc := NewInvoiceService(
		&MockInvoiceRepo{},
		&MockPlanRepo{Plan: &domain.Plan{ID: planID, Prices: []domain.Price{{Amount: 100000, Currency: "INR"}}}},
		&MockCustomerRepo{Customer: &domain.Customer{
			ID: custID, PlaceOfSupply: domain.StringPtr("TN"),
			BillingAddress: domain.BillingAddress{Country: "India", State: "TN"},
		}},
		&MockUnbilledChargeRepo{}, &MockSubscriptionRepo{}, gsp.NewMockGSPAdapter(), nil,
	)
	svc.CouponRepo = &mockCouponRepoForInvoice{coupon: &domain.Coupon{
		ID: couponID, TenantID: tenantID,
		DiscountType: domain.DiscountTypePercent, DiscountValue: 20,
		Duration: domain.DurationForever, Active: true,
	}}

	sub := &domain.Subscription{ID: uuid.New(), CustomerID: custID, PlanID: planID, TenantID: tenantID, CouponID: &couponID}

	inv, err := svc.GenerateInvoice(context.Background(), sub)
	if err != nil {
		t.Fatalf("GenerateInvoice: %v", err)
	}
	// 20% off ₹1000 → taxable ₹800; intra-state 18% on 80000 = 14400; total 94400.
	if inv.Subtotal != 100000 {
		t.Errorf("subtotal = %d, want 100000 (gross)", inv.Subtotal)
	}
	if inv.TaxAmount != 14400 {
		t.Errorf("tax = %d, want 14400 (18%% of the discounted 80000, not full 100000)", inv.TaxAmount)
	}
	if inv.Total != 94400 {
		t.Errorf("total = %d, want 94400 (80000 net + 14400 tax)", inv.Total)
	}
	if len(inv.LineItems) == 0 || inv.LineItems[0].TaxableAmount != 80000 {
		t.Errorf("base line taxable = %+v, want 80000", inv.LineItems)
	}
}

// TestGenerateInvoice_OnceCouponNotReappliedOnRenewal pins that a `once` coupon
// is NOT applied on renewals (it belongs to the first invoice only) — so this
// change never over-discounts a one-time coupon.
func TestGenerateInvoice_OnceCouponNotReappliedOnRenewal(t *testing.T) {
	planID := uuid.New()
	custID := uuid.New()
	tenantID := uuid.New()
	couponID := uuid.New()

	svc := NewInvoiceService(
		&MockInvoiceRepo{},
		&MockPlanRepo{Plan: &domain.Plan{ID: planID, Prices: []domain.Price{{Amount: 100000, Currency: "INR"}}}},
		&MockCustomerRepo{Customer: &domain.Customer{
			ID: custID, PlaceOfSupply: domain.StringPtr("TN"),
			BillingAddress: domain.BillingAddress{Country: "India", State: "TN"},
		}},
		&MockUnbilledChargeRepo{}, &MockSubscriptionRepo{}, gsp.NewMockGSPAdapter(), nil,
	)
	svc.CouponRepo = &mockCouponRepoForInvoice{coupon: &domain.Coupon{
		ID: couponID, TenantID: tenantID,
		DiscountType: domain.DiscountTypePercent, DiscountValue: 20,
		Duration: domain.DurationOnce, Active: true,
	}}
	// A renewal: the `once` coupon was already consumed at create (period 1).
	sub := &domain.Subscription{ID: uuid.New(), CustomerID: custID, PlanID: planID, TenantID: tenantID, CouponID: &couponID, CouponPeriodsApplied: 1}

	inv, err := svc.GenerateInvoice(context.Background(), sub)
	if err != nil {
		t.Fatalf("GenerateInvoice: %v", err)
	}
	if inv.Total != 118000 {
		t.Errorf("total = %d, want 118000 — a `once` coupon must NOT be re-applied on renewals", inv.Total)
	}
}

// TestGenerateInvoice_RepeatingCouponStopsAfterN proves a `repeating` coupon
// (N months) applies for the first N renewal periods and then stops, driven by
// the subscription's CouponPeriodsApplied counter.
func TestGenerateInvoice_RepeatingCouponStopsAfterN(t *testing.T) {
	planID := uuid.New()
	custID := uuid.New()
	tenantID := uuid.New()
	couponID := uuid.New()
	months := 3

	svc := NewInvoiceService(
		&MockInvoiceRepo{},
		&MockPlanRepo{Plan: &domain.Plan{ID: planID, Prices: []domain.Price{{Amount: 100000, Currency: "INR"}}}},
		&MockCustomerRepo{Customer: &domain.Customer{
			ID: custID, PlaceOfSupply: domain.StringPtr("TN"),
			BillingAddress: domain.BillingAddress{Country: "India", State: "TN"},
		}},
		&MockUnbilledChargeRepo{}, &MockSubscriptionRepo{}, gsp.NewMockGSPAdapter(), nil,
	)
	svc.CouponRepo = &mockCouponRepoForInvoice{coupon: &domain.Coupon{
		ID: couponID, TenantID: tenantID,
		DiscountType: domain.DiscountTypePercent, DiscountValue: 20,
		Duration: domain.DurationRepeating, DurationMonths: &months, Active: true,
	}}

	// Period 1 was create; renewals are periods 2..N. Walk the counter forward
	// as the renewal worker would (each GenerateInvoice advances it).
	sub := &domain.Subscription{ID: uuid.New(), CustomerID: custID, PlanID: planID, TenantID: tenantID, CouponID: &couponID, CouponPeriodsApplied: 1}
	for period := 2; period <= 5; period++ {
		inv, err := svc.GenerateInvoice(context.Background(), sub)
		if err != nil {
			t.Fatalf("period %d: %v", period, err)
		}
		if period <= months {
			if inv.Total != 94400 {
				t.Errorf("period %d total = %d, want 94400 (coupon applies for the first %d periods)", period, inv.Total, months)
			}
		} else if inv.Total != 118000 {
			t.Errorf("period %d total = %d, want 118000 (coupon exhausted after %d periods)", period, inv.Total, months)
		}
	}
	if sub.CouponPeriodsApplied != months {
		t.Errorf("CouponPeriodsApplied = %d, want %d (capped at DurationMonths)", sub.CouponPeriodsApplied, months)
	}
}
