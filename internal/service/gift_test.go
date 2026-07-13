package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// --- Mock GiftRepository ---
type mockGiftRepo struct {
	gifts     map[string]*domain.Gift // keyed by code
	listGifts []*domain.Gift
	updated   []*domain.Gift
	createErr error
}

func newMockGiftRepo() *mockGiftRepo {
	return &mockGiftRepo{gifts: make(map[string]*domain.Gift)}
}

func (m *mockGiftRepo) Create(ctx context.Context, g *domain.Gift) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.gifts[g.Code] = g
	return nil
}

func (m *mockGiftRepo) GetByCode(ctx context.Context, tenantID uuid.UUID, code string) (*domain.Gift, error) {
	if g, ok := m.gifts[code]; ok {
		return g, nil
	}
	return nil, nil
}

func (m *mockGiftRepo) List(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Gift, error) {
	return m.listGifts, nil
}

func (m *mockGiftRepo) Update(ctx context.Context, g *domain.Gift) error {
	m.updated = append(m.updated, g)
	m.gifts[g.Code] = g
	return nil
}

func (m *mockGiftRepo) MarkRedeemed(_ context.Context, giftID, tenantID, redeemedBy uuid.UUID, at time.Time) (bool, error) {
	for _, g := range m.gifts {
		if g.ID == giftID && g.TenantID == tenantID {
			if g.Status != domain.GiftStatusPurchased {
				return false, nil // already redeemed — lost the claim
			}
			g.Status = domain.GiftStatusRedeemed
			g.RedeemedByCustomerID = &redeemedBy
			g.RedeemedAt = &at
			g.UpdatedAt = at
			return true, nil
		}
	}
	return false, nil
}

func (m *mockGiftRepo) RevertRedemption(_ context.Context, giftID, tenantID uuid.UUID) error {
	for _, g := range m.gifts {
		if g.ID == giftID && g.TenantID == tenantID {
			g.Status = domain.GiftStatusPurchased
			g.RedeemedByCustomerID = nil
			g.RedeemedAt = nil
		}
	}
	return nil
}

// --- Mock SubscriptionRepository (minimal for gift tests) ---
type mockSubRepoForGift struct {
	created []*domain.Subscription
}

func (m *mockSubRepoForGift) Create(ctx context.Context, s *domain.Subscription) error {
	m.created = append(m.created, s)
	return nil
}

// Implement remaining interface methods as stubs
func (m *mockSubRepoForGift) GetByID(ctx context.Context, id uuid.UUID) (*domain.Subscription, error) {
	return nil, nil
}
func (m *mockSubRepoForGift) Update(ctx context.Context, s *domain.Subscription) error { return nil }
func (m *mockSubRepoForGift) CountActiveByCustomer(ctx context.Context, tenantID uuid.UUID) (map[uuid.UUID]int, error) {
	return nil, nil
}
func (m *mockSubRepoForGift) List(ctx context.Context, tenantID uuid.UUID, filter domain.SubscriptionFilter) ([]*domain.Subscription, error) {
	return nil, nil
}
func (m *mockSubRepoForGift) GetByCustomerID(ctx context.Context, tenantID, customerID uuid.UUID) ([]*domain.Subscription, error) {
	return nil, nil
}
func (m *mockSubRepoForGift) ListDueForRenewal(ctx context.Context, before time.Time) ([]*domain.Subscription, error) {
	return nil, nil
}
func (m *mockSubRepoForGift) GetByStripeSubscriptionID(ctx context.Context, stripeSubID string) (*domain.Subscription, error) {
	return nil, nil
}
func (m *mockSubRepoForGift) GetActiveSubscriptions(ctx context.Context, _ uuid.UUID) ([]*domain.Subscription, error) {
	return nil, nil
}

// --- Mock PlanRepository (minimal for gift tests) ---
type mockPlanRepoForGift struct {
	plan *domain.Plan
}

func (m *mockPlanRepoForGift) GetByID(ctx context.Context, id uuid.UUID) (*domain.Plan, error) {
	return m.plan, nil
}
func (m *mockPlanRepoForGift) Create(ctx context.Context, p *domain.Plan) error { return nil }
func (m *mockPlanRepoForGift) Update(ctx context.Context, p *domain.Plan) error { return nil }
func (m *mockPlanRepoForGift) Delete(ctx context.Context, id uuid.UUID) error   { return nil }
func (m *mockPlanRepoForGift) List(ctx context.Context, tenantID uuid.UUID, filter domain.PlanFilter) ([]*domain.Plan, error) {
	return nil, nil
}
func (m *mockPlanRepoForGift) GetByCode(ctx context.Context, tenantID uuid.UUID, code string) (*domain.Plan, error) {
	return nil, nil
}

// --- Tests ---

func testPlan(planID uuid.UUID) *domain.Plan {
	return &domain.Plan{
		ID:   planID,
		Name: "Test Plan",
		Prices: []domain.Price{
			{ID: uuid.New(), PlanID: planID, Currency: "USD", Amount: 1000, Type: "recurring"},
		},
	}
}

func TestPurchaseGift_Success(t *testing.T) {
	giftRepo := newMockGiftRepo()
	subRepo := &mockSubRepoForGift{}
	planID := uuid.New()
	planRepo := &mockPlanRepoForGift{plan: testPlan(planID)}
	svc := NewGiftService(giftRepo, subRepo, nil, planRepo, nil)

	tenantID := uuid.New()
	buyerID := uuid.New()

	gift, err := svc.PurchaseGift(context.Background(), tenantID, buyerID, planID, "recipient@example.com", 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gift.ID == uuid.Nil {
		t.Error("gift ID should be generated")
	}
	if gift.Code == "" {
		t.Error("gift code should be generated")
	}
	if gift.Status != domain.GiftStatusPurchased {
		t.Errorf("status = %q, want 'purchased'", gift.Status)
	}
	if gift.DurationMonths != 3 {
		t.Errorf("duration = %d, want 3", gift.DurationMonths)
	}
	if gift.BuyerCustomerID != buyerID {
		t.Error("buyer ID mismatch")
	}
	if gift.RecipientEmail != "recipient@example.com" {
		t.Errorf("recipient_email = %q, want recipient@example.com", gift.RecipientEmail)
	}
}

// TestPurchaseGift_RejectsInvalidDuration proves the ENG-165 H3 guard: a
// non-positive or absurdly large duration is refused, so a caller cannot mint a
// negative buyer invoice (crediting themselves) or overflow the invoice amount.
func TestPurchaseGift_RejectsInvalidDuration(t *testing.T) {
	planID := uuid.New()
	svc := NewGiftService(newMockGiftRepo(), &mockSubRepoForGift{}, nil, &mockPlanRepoForGift{plan: testPlan(planID)}, nil)

	for _, d := range []int{0, -1, -12, 121, 100000} {
		if _, err := svc.PurchaseGift(context.Background(), uuid.New(), uuid.New(), planID, "", d); err != ErrInvalidGiftDuration {
			t.Errorf("PurchaseGift(duration=%d): err = %v, want ErrInvalidGiftDuration", d, err)
		}
	}
}

func TestPurchaseGift_CodeFormat(t *testing.T) {
	giftRepo := newMockGiftRepo()
	planID := uuid.New()
	svc := NewGiftService(giftRepo, &mockSubRepoForGift{}, nil, &mockPlanRepoForGift{plan: testPlan(planID)}, nil)

	gift, err := svc.PurchaseGift(context.Background(), uuid.New(), uuid.New(), planID, "", 6)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Code format: GIFT-XXXX (hex)
	if len(gift.Code) < 6 || gift.Code[:5] != "GIFT-" {
		t.Errorf("code = %q, expected GIFT-XXXX format", gift.Code)
	}
}

func TestRedeemGift_Success(t *testing.T) {
	giftRepo := newMockGiftRepo()
	subRepo := &mockSubRepoForGift{}
	planID := uuid.New()
	planRepo := &mockPlanRepoForGift{plan: testPlan(planID)}
	svc := NewGiftService(giftRepo, subRepo, nil, planRepo, nil)

	tenantID := uuid.New()
	buyerID := uuid.New()
	recipientID := uuid.New()

	// Purchase first
	gift, _ := svc.PurchaseGift(context.Background(), tenantID, buyerID, planID, "", 6)

	// Redeem
	sub, err := svc.RedeemGift(context.Background(), tenantID, recipientID, gift.Code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sub.CustomerID != recipientID {
		t.Error("subscription should be for the recipient")
	}
	if sub.PlanID != planID {
		t.Error("subscription should use the gift plan")
	}
	if sub.Status != domain.SubscriptionStatusActive {
		t.Errorf("status = %q, want active", sub.Status)
	}

	// Gift should be marked redeemed
	updatedGift := giftRepo.gifts[gift.Code]
	if updatedGift.Status != domain.GiftStatusRedeemed {
		t.Error("gift should be marked as redeemed")
	}
	if updatedGift.RedeemedByCustomerID == nil || *updatedGift.RedeemedByCustomerID != recipientID {
		t.Error("redeemed_by should be set to recipient")
	}
}

func TestRedeemGift_DoubleRedeem(t *testing.T) {
	giftRepo := newMockGiftRepo()
	subRepo := &mockSubRepoForGift{}
	planID := uuid.New()
	planRepo := &mockPlanRepoForGift{plan: testPlan(planID)}
	svc := NewGiftService(giftRepo, subRepo, nil, planRepo, nil)

	tenantID := uuid.New()
	gift, _ := svc.PurchaseGift(context.Background(), tenantID, uuid.New(), planID, "", 3)

	// First redeem
	_, err := svc.RedeemGift(context.Background(), tenantID, uuid.New(), gift.Code)
	if err != nil {
		t.Fatalf("first redeem should succeed: %v", err)
	}

	// Second redeem should fail
	_, err = svc.RedeemGift(context.Background(), tenantID, uuid.New(), gift.Code)
	if err == nil {
		t.Error("expected error on double redeem")
	}
	if err.Error() != "gift already redeemed" {
		t.Errorf("error = %q, want 'gift already redeemed'", err.Error())
	}
}

func TestRedeemGift_InvalidCode(t *testing.T) {
	giftRepo := newMockGiftRepo()
	svc := NewGiftService(giftRepo, &mockSubRepoForGift{}, nil, &mockPlanRepoForGift{}, nil)

	_, err := svc.RedeemGift(context.Background(), uuid.New(), uuid.New(), "FAKE-CODE")
	if err == nil {
		t.Error("expected error for invalid code")
	}
	if err.Error() != "invalid gift code" {
		t.Errorf("error = %q, want 'invalid gift code'", err.Error())
	}
}

func TestRedeemGift_SubscriptionDuration(t *testing.T) {
	giftRepo := newMockGiftRepo()
	subRepo := &mockSubRepoForGift{}
	planID := uuid.New()
	planRepo := &mockPlanRepoForGift{plan: testPlan(planID)}
	svc := NewGiftService(giftRepo, subRepo, nil, planRepo, nil)

	tenantID := uuid.New()
	gift, _ := svc.PurchaseGift(context.Background(), tenantID, uuid.New(), planID, "", 12)

	sub, err := svc.RedeemGift(context.Background(), tenantID, uuid.New(), gift.Code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Duration should be 12 months
	duration := sub.CurrentPeriodEnd.Sub(sub.CurrentPeriodStart)
	expectedDuration := time.Hour * 24 * 365 // approx 12 months

	if duration < expectedDuration-time.Hour*24*5 || duration > expectedDuration+time.Hour*24*5 {
		t.Errorf("subscription duration = %v, want ~12 months", duration)
	}
}
