package db

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestSubscriptionRepository_CouponRoundTrip_Postgres proves the subscription
// repo now PERSISTS and LOADS coupon_id and coupon_periods_applied. Before this,
// coupon_id was a column the repo never wrote or scanned, so sub.CouponID was
// always nil after a load — silently making recurring-coupon application inert.
func TestSubscriptionRepository_CouponRoundTrip_Postgres(t *testing.T) {
	conn := openProgressiveTestDB(t)
	ctx := context.Background()
	run := uuid.NewString()[:8]

	tenantID := uuid.New()
	must(t, conn, `INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "SC-"+run, "sc-"+run+"@t.com")
	custID := uuid.New()
	must(t, conn, `INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1,$2,$3,$4,NOW())`,
		custID, tenantID, "c-"+run+"@t.com", uuid.New())
	planID := uuid.New()
	must(t, conn, `INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active) VALUES ($1,$2,'P','p-`+run+`','month',1,TRUE)`,
		planID, tenantID)
	couponID := uuid.New()
	must(t, conn, `INSERT INTO coupons (id, tenant_id, code, discount_type, discount_value, duration, active, created_at, updated_at) VALUES ($1,$2,$3,'percent',20,'repeating',TRUE,NOW(),NOW())`,
		couponID, tenantID, "SAVE20-"+run)

	repo := NewSubscriptionRepository(conn)
	tctx := context.WithValue(ctx, domain.TenantIDKey, tenantID)

	sub := &domain.Subscription{
		ID: uuid.New(), TenantID: tenantID, CustomerID: custID, PlanID: planID,
		Status: domain.SubscriptionStatusActive, CurrentPeriodStart: time.Now(), CurrentPeriodEnd: time.Now().Add(720 * time.Hour),
		BillingAnchor: time.Now(), CreatedAt: time.Now(), UpdatedAt: time.Now(),
		CouponID: &couponID, CouponPeriodsApplied: 1,
	}
	if err := repo.Create(ctx, sub); err != nil {
		t.Fatalf("create sub: %v", err)
	}

	got, err := repo.GetByID(tctx, sub.ID)
	if err != nil {
		t.Fatalf("get sub: %v", err)
	}
	if got.CouponID == nil || *got.CouponID != couponID {
		t.Errorf("loaded coupon_id = %v, want %s (coupon must round-trip, else recurring coupons are inert)", got.CouponID, couponID)
	}
	if got.CouponPeriodsApplied != 1 {
		t.Errorf("loaded coupon_periods_applied = %d, want 1", got.CouponPeriodsApplied)
	}

	// The renewal worker advances the counter via Update; it must persist.
	got.CouponPeriodsApplied = 2
	got.UpdatedAt = time.Now()
	if err := repo.Update(tctx, got); err != nil {
		t.Fatalf("update sub: %v", err)
	}
	again, err := repo.GetByID(tctx, sub.ID)
	if err != nil {
		t.Fatalf("get sub after update: %v", err)
	}
	if again.CouponPeriodsApplied != 2 {
		t.Errorf("after Update, coupon_periods_applied = %d, want 2 (counter must persist across renewals)", again.CouponPeriodsApplied)
	}
	if again.CouponID == nil || *again.CouponID != couponID {
		t.Errorf("coupon_id lost across Update: %v", again.CouponID)
	}
}
