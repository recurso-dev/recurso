package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
)

// --- Mock ReferralRepository ---
type mockReferralRepo struct {
	referrals      []*domain.Referral
	byReferredID   *domain.Referral
	created        []*domain.Referral
	createErr      error
}

func newMockReferralRepo() *mockReferralRepo {
	return &mockReferralRepo{}
}

func (m *mockReferralRepo) Create(ctx context.Context, r *domain.Referral) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.created = append(m.created, r)
	return nil
}

func (m *mockReferralRepo) GetByCode(ctx context.Context, tenantID uuid.UUID, code string) (*domain.Referral, error) {
	return nil, nil
}

func (m *mockReferralRepo) GetByReferrerID(ctx context.Context, tenantID uuid.UUID, referrerID uuid.UUID) ([]*domain.Referral, error) {
	return m.referrals, nil
}

func (m *mockReferralRepo) GetByReferredID(ctx context.Context, tenantID uuid.UUID, referredID uuid.UUID) (*domain.Referral, error) {
	return m.byReferredID, nil
}

func (m *mockReferralRepo) List(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Referral, error) {
	return m.referrals, nil
}

// --- Tests ---

func TestCreateReferral_SelfReferral(t *testing.T) {
	refRepo := newMockReferralRepo()
	custRepo := newMockCustomerRepo()
	svc := NewReferralService(refRepo, custRepo)

	customerID := uuid.New()
	tenantID := uuid.New()

	// Set up the customer in mock
	custRepo.customers[customerID] = &domain.Customer{
		ID:       customerID,
		TenantID: tenantID,
		Name:     domain.StringPtr("Alice"),
	}

	_, err := svc.CreateReferral(context.Background(), tenantID, customerID, customerID, 500, "USD")
	if err != ErrSelfReferral {
		t.Errorf("expected ErrSelfReferral, got %v", err)
	}
}

func TestCreateReferral_Success(t *testing.T) {
	refRepo := newMockReferralRepo()
	custRepo := newMockCustomerRepo()
	svc := NewReferralService(refRepo, custRepo)

	tenantID := uuid.New()
	referrerID := uuid.New()
	referredID := uuid.New()

	// Set up the referrer customer
	custRepo.customers[referrerID] = &domain.Customer{
		ID:       referrerID,
		TenantID: tenantID,
		Name:     domain.StringPtr("Alice"),
	}

	referral, err := svc.CreateReferral(context.Background(), tenantID, referrerID, referredID, 1000, "INR")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if referral.ReferrerID != referrerID {
		t.Errorf("referrer ID = %v, want %v", referral.ReferrerID, referrerID)
	}
	if referral.ReferredID != referredID {
		t.Errorf("referred ID = %v, want %v", referral.ReferredID, referredID)
	}
	if referral.RewardAmount != 1000 {
		t.Errorf("reward = %d, want 1000", referral.RewardAmount)
	}
	if referral.Currency != "INR" {
		t.Errorf("currency = %q, want INR", referral.Currency)
	}
	if referral.Status != domain.ReferralStatusPending {
		t.Errorf("status = %q, want pending", referral.Status)
	}
}

func TestGenerateCode_NewCustomer(t *testing.T) {
	refRepo := newMockReferralRepo()
	custRepo := newMockCustomerRepo()
	svc := NewReferralService(refRepo, custRepo)

	tenantID := uuid.New()
	customerID := uuid.New()

	custRepo.customers[customerID] = &domain.Customer{
		ID:       customerID,
		TenantID: tenantID,
		Name:     domain.StringPtr("Bob Smith"),
	}

	code, err := svc.GenerateCode(context.Background(), tenantID, customerID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Code should start with "BOB-"
	if len(code) < 5 || code[:4] != "BOB-" {
		t.Errorf("code = %q, want prefix BOB-", code)
	}

	// Should have been saved back to customer
	if !custRepo.updateCalled {
		t.Error("expected customer update to be called")
	}
	if domain.PtrToString(custRepo.customers[customerID].ReferralCode) != code {
		t.Error("referral code should be saved on customer record")
	}
}

func TestGenerateCode_ExistingCode(t *testing.T) {
	refRepo := newMockReferralRepo()
	custRepo := newMockCustomerRepo()
	svc := NewReferralService(refRepo, custRepo)

	tenantID := uuid.New()
	customerID := uuid.New()

	// Customer already has a code
	custRepo.customers[customerID] = &domain.Customer{
		ID:           customerID,
		TenantID:     tenantID,
		Name:         domain.StringPtr("Eve"),
		ReferralCode: domain.StringPtr("EVE-ABCD"),
	}

	code, err := svc.GenerateCode(context.Background(), tenantID, customerID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return existing code, not generate new
	if code != "EVE-ABCD" {
		t.Errorf("code = %q, want EVE-ABCD (existing)", code)
	}
	if custRepo.updateCalled {
		t.Error("should not call Update when code already exists")
	}
}

func TestTrackReferral_AlreadyReferred(t *testing.T) {
	refRepo := newMockReferralRepo()
	custRepo := newMockCustomerRepo()
	svc := NewReferralService(refRepo, custRepo)

	tenantID := uuid.New()
	referrerID := uuid.New()
	referredID := uuid.New()

	// Set up referrer via referral code lookup
	custRepo.byReferral = &domain.Customer{
		ID:       referrerID,
		TenantID: tenantID,
		Name:     domain.StringPtr("Alice"),
	}

	// Mark the referred customer as already referred
	refRepo.byReferredID = &domain.Referral{
		ID:         uuid.New(),
		ReferrerID: referrerID,
		ReferredID: referredID,
	}

	err := svc.TrackReferral(context.Background(), tenantID, referredID, "ALICE-1234")
	if err != ErrAlreadyReferred {
		t.Errorf("expected ErrAlreadyReferred, got %v", err)
	}
}

func TestTrackReferral_CodeNotFound(t *testing.T) {
	refRepo := newMockReferralRepo()
	custRepo := newMockCustomerRepo()
	svc := NewReferralService(refRepo, custRepo)

	// byReferral is nil — code doesn't match any customer
	custRepo.byReferral = nil

	err := svc.TrackReferral(context.Background(), uuid.New(), uuid.New(), "INVALID-CODE")
	if err != ErrReferralCodeNotFound {
		t.Errorf("expected ErrReferralCodeNotFound, got %v", err)
	}
}

func TestListReferrals_Pagination(t *testing.T) {
	refRepo := newMockReferralRepo()
	custRepo := newMockCustomerRepo()
	svc := NewReferralService(refRepo, custRepo)

	refRepo.referrals = []*domain.Referral{
		{ID: uuid.New(), Status: domain.ReferralStatusPending},
		{ID: uuid.New(), Status: domain.ReferralStatusPending},
	}

	results, err := svc.ListReferrals(context.Background(), uuid.New(), 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("got %d results, want 2", len(results))
	}
}
