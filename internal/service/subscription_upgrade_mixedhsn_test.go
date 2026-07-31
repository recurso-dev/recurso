package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestComputePlanChangeProration_MixedHSNRates proves that a plan change taxes
// each side on its OWN plan's HSN rate, not one rate applied to the net.
//
// Seller in TN; buyer in KA (inter-state → IGST). Half the cycle remains.
//
//	Old plan: ₹1000 (100000 minor), HSN 998314 → 18%
//	New plan: ₹400  (40000 minor),  HSN 9963   → 5%
//
//	credit = trunc(100000 × 0.5) = 50000   (unused old plan, reversed @ 18%)
//	charge = trunc(40000  × 0.5) = 20000   (remaining new plan, collected @ 5%)
//	net    = 20000 − 50000       = −30000  (a downgrade credit)
//
// Correct net tax = chargeTax − creditTax = trunc(20000×0.05) − trunc(50000×0.18)
// = 1000 − 9000 = −8000. The old single-rate code taxed −net (30000) at the OLD
// plan's 18% = −5400, under-reversing the customer's GST by ₹26, so this test
// FAILS on the pre-fix code.
func TestComputePlanChangeProration_MixedHSNRates(t *testing.T) {
	tenantID := uuid.New()
	// Indian seller registered in Tamil Nadu (state code 33).
	resolver := NewTaxResolver(&mockGSTConfigProvider{cfg: &domain.TenantGSTConfig{
		GSTIN:     "33ABCDE1234F1Z5",
		StateCode: "33",
		SACCode:   "998314",
	}}, "IN", "TN")
	svc := &SubscriptionService{taxResolver: resolver}

	now := time.Now().UTC()
	sub := &domain.Subscription{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		CustomerID:         uuid.New(),
		CurrentPeriodStart: now.AddDate(0, 0, -15),
		CurrentPeriodEnd:   now.AddDate(0, 0, 15),
	}
	// Buyer in Karnataka (≠ seller's TN) → inter-state IGST, single number to assert.
	customer := inCustomer("KA")

	oldPlan := &domain.Plan{ID: uuid.New(), HSNCode: "998314", // 18%
		Prices: []domain.Price{{Amount: 100000, Currency: "INR"}}}
	newPlan := &domain.Plan{ID: uuid.New(), HSNCode: "9963", // 5%
		Prices: []domain.Price{{Amount: 40000, Currency: "INR"}}}

	pcp := svc.computePlanChangeProration(context.Background(), tenantID, sub, oldPlan, newPlan, customer, now)

	if pcp.Proration.NetAmount != -30000 {
		t.Fatalf("net = %d, want -30000 (downgrade)", pcp.Proration.NetAmount)
	}
	// The core assertion: net tax must reflect BOTH rates, not one applied to net.
	if pcp.Tax.Total != -8000 {
		t.Errorf("net tax Total = %d, want -8000 (charge@5%% − credit@18%% = 1000 − 9000). "+
			"A result of -5400 means the old single-rate-on-net bug is back.", pcp.Tax.Total)
	}
	if pcp.Tax.IGST != -8000 {
		t.Errorf("net tax IGST = %d, want -8000 (inter-state)", pcp.Tax.IGST)
	}
	// The credit note the caller builds must net principal + tax correctly:
	// -(net + tax) = -(-30000 + -8000) = 38000 (₹380 credit).
	if got := -(pcp.Proration.NetAmount + pcp.Tax.Total); got != 38000 {
		t.Errorf("resulting credit amount = %d, want 38000 (₹590 unused old − ₹210 remaining new)", got)
	}
}

// TestComputePlanChangeProration_SameHSN_Unchanged pins the common case: when
// both plans share a rate, the per-side tax equals the old net-based tax, so
// existing single-rate behaviour is preserved.
func TestComputePlanChangeProration_SameHSN_Unchanged(t *testing.T) {
	tenantID := uuid.New()
	resolver := NewTaxResolver(&mockGSTConfigProvider{cfg: &domain.TenantGSTConfig{
		GSTIN: "33ABCDE1234F1Z5", StateCode: "33", SACCode: "998314",
	}}, "IN", "TN")
	svc := &SubscriptionService{taxResolver: resolver}

	now := time.Now().UTC()
	sub := &domain.Subscription{
		ID: uuid.New(), TenantID: tenantID, CustomerID: uuid.New(),
		CurrentPeriodStart: now.AddDate(0, 0, -15),
		CurrentPeriodEnd:   now.AddDate(0, 0, 15),
	}
	customer := inCustomer("KA")

	// Both plans HSN 998314 (18%). Upgrade ₹1000 → ₹2000, half remaining.
	oldPlan := &domain.Plan{ID: uuid.New(), HSNCode: "998314",
		Prices: []domain.Price{{Amount: 100000, Currency: "INR"}}}
	newPlan := &domain.Plan{ID: uuid.New(), HSNCode: "998314",
		Prices: []domain.Price{{Amount: 200000, Currency: "INR"}}}

	pcp := svc.computePlanChangeProration(context.Background(), tenantID, sub, oldPlan, newPlan, customer, now)

	// charge=trunc(200000×0.5)=100000, credit=trunc(100000×0.5)=50000, net=50000.
	// tax = trunc(100000×0.18) − trunc(50000×0.18) = 18000 − 9000 = 9000,
	// which equals the old trunc(50000×0.18)=9000.
	if pcp.Proration.NetAmount != 50000 {
		t.Fatalf("net = %d, want 50000", pcp.Proration.NetAmount)
	}
	if pcp.Tax.Total != 9000 {
		t.Errorf("net tax = %d, want 9000 (18%% on the ₹500 net)", pcp.Tax.Total)
	}
}
