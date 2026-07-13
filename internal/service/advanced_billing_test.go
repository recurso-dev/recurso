package service

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// fakeUCSubRepo mimics the tenant-scoped SubscriptionRepository.GetByID: the
// subscription is only visible when the context tenant matches its owner.
type fakeUCSubRepo struct {
	port.SubscriptionRepository
	subs map[uuid.UUID]uuid.UUID // subID -> owning tenant
}

func (r *fakeUCSubRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Subscription, error) {
	tenantID, ok := ctx.Value(domain.TenantIDKey).(uuid.UUID)
	if !ok {
		return nil, sql.ErrNoRows
	}
	owner, ok := r.subs[id]
	if !ok || owner != tenantID {
		return nil, sql.ErrNoRows
	}
	return &domain.Subscription{ID: id, TenantID: owner}, nil
}

type fakeUCRepo struct {
	port.UnbilledChargeRepository
	bySub map[uuid.UUID][]*domain.UnbilledCharge
}

func (r *fakeUCRepo) ListBySubscriptionID(subscriptionID uuid.UUID) ([]*domain.UnbilledCharge, error) {
	return r.bySub[subscriptionID], nil
}

// TestAddUnbilledCharge_RejectsNonPositiveAmount proves the ENG-165 H3 guard: a
// zero or negative unbilled charge is refused before any repo write, so it
// cannot be used to credit or zero out an invoice.
func TestAddUnbilledCharge_RejectsNonPositiveAmount(t *testing.T) {
	subID := uuid.New()
	owner := uuid.New()
	subRepo := &fakeUCSubRepo{subs: map[uuid.UUID]uuid.UUID{subID: owner}}
	svc := NewAdvancedBillingService(&fakeUCRepo{}, subRepo)
	ctx := context.WithValue(context.Background(), domain.TenantIDKey, owner)

	for _, amt := range []int64{0, -1, -50000} {
		if _, err := svc.AddUnbilledCharge(ctx, subID, amt, "INR", "x", ""); err != ErrInvalidChargeAmount {
			t.Errorf("AddUnbilledCharge(%d): err = %v, want ErrInvalidChargeAmount", amt, err)
		}
	}
}

// TestListUnbilledCharges_TenantIsolation proves the ENG-165 H1 fix: listing a
// subscription's unbilled charges first verifies the subscription belongs to
// the caller's tenant, so a foreign tenant cannot read the amounts.
func TestListUnbilledCharges_TenantIsolation(t *testing.T) {
	subID := uuid.New()
	owner := uuid.New()
	attacker := uuid.New()

	subRepo := &fakeUCSubRepo{subs: map[uuid.UUID]uuid.UUID{subID: owner}}
	ucRepo := &fakeUCRepo{bySub: map[uuid.UUID][]*domain.UnbilledCharge{
		subID: {{ID: uuid.New(), SubscriptionID: subID, Amount: 50000, Currency: "INR"}},
	}}
	svc := NewAdvancedBillingService(ucRepo, subRepo)

	// Attacker cannot list the owner's charges — subscription is invisible.
	attackerCtx := context.WithValue(context.Background(), domain.TenantIDKey, attacker)
	if _, err := svc.ListUnbilledCharges(attackerCtx, subID); err == nil {
		t.Fatal("cross-tenant ListUnbilledCharges: expected error, got nil")
	}

	// Owner sees its charge.
	ownerCtx := context.WithValue(context.Background(), domain.TenantIDKey, owner)
	charges, err := svc.ListUnbilledCharges(ownerCtx, subID)
	if err != nil {
		t.Fatalf("owner ListUnbilledCharges: %v", err)
	}
	if len(charges) != 1 || charges[0].Amount != 50000 {
		t.Fatalf("owner charges = %+v, want one charge of 50000", charges)
	}
}
