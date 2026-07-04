package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

var (
	ErrReferralCodeNotFound = errors.New("referral code not found")
	ErrSelfReferral         = errors.New("cannot refer yourself")
	ErrAlreadyReferred      = errors.New("customer already referred")
)

type ReferralService struct {
	referralRepo port.ReferralRepository
	customerRepo port.CustomerRepository
}

func NewReferralService(referralRepo port.ReferralRepository, customerRepo port.CustomerRepository) *ReferralService {
	return &ReferralService{
		referralRepo: referralRepo,
		customerRepo: customerRepo,
	}
}

// GenerateCode creates a unique referral code for a customer and stores it on their profile
func (s *ReferralService) GenerateCode(ctx context.Context, tenantID uuid.UUID, customerID uuid.UUID) (string, error) {
	customer, err := s.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return "", err
	}

	// If customer already has a code, return it
	if customer.ReferralCode != nil && *customer.ReferralCode != "" {
		return *customer.ReferralCode, nil
	}

	// Format: First 3 letters of name (upper) + 4 random hex chars
	prefix := "REF"
	if customer.Name != nil && len(*customer.Name) >= 3 {
		prefix = strings.ToUpper((*customer.Name)[:3])
	}

	bytes := make([]byte, 2)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	suffix := strings.ToUpper(hex.EncodeToString(bytes))

	code := fmt.Sprintf("%s-%s", prefix, suffix)

	// Save the code on the customer record
	customer.ReferralCode = &code
	if err := s.customerRepo.Update(ctx, customer); err != nil {
		return "", err
	}

	return code, nil
}

// TrackReferral links a new customer to a referrer via the referrer's referral code
func (s *ReferralService) TrackReferral(ctx context.Context, tenantID uuid.UUID, newCustomerID uuid.UUID, code string) error {
	// 1. Find the referrer by their referral_code on the customer table
	referrer, err := s.customerRepo.GetByReferralCode(ctx, tenantID, code)
	if err != nil {
		return err
	}
	if referrer == nil {
		return ErrReferralCodeNotFound
	}

	// 2. Prevent self-referral
	if referrer.ID == newCustomerID {
		return ErrSelfReferral
	}

	// 3. Check if this customer was already referred
	existing, err := s.referralRepo.GetByReferredID(ctx, tenantID, newCustomerID)
	if err != nil {
		return err
	}
	if existing != nil {
		return ErrAlreadyReferred
	}

	// 4. Create referral record
	now := time.Now()
	referral := &domain.Referral{
		ID:           uuid.New(),
		TenantID:     tenantID,
		ReferrerID:   referrer.ID,
		ReferredID:   newCustomerID,
		Code:         code,
		Status:       domain.ReferralStatusPending,
		RewardAmount: 500, // Default $5.00 reward
		Currency:     "USD",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	return s.referralRepo.Create(ctx, referral)
}

// CreateReferral explicitly creates a referral record (admin action)
func (s *ReferralService) CreateReferral(ctx context.Context, tenantID uuid.UUID, referrerID uuid.UUID, referredID uuid.UUID, rewardAmount int64, currency string) (*domain.Referral, error) {
	if referrerID == referredID {
		return nil, ErrSelfReferral
	}

	// Generate a unique code for this referral
	code, err := s.GenerateCode(ctx, tenantID, referrerID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	referral := &domain.Referral{
		ID:           uuid.New(),
		TenantID:     tenantID,
		ReferrerID:   referrerID,
		ReferredID:   referredID,
		Code:         code,
		Status:       domain.ReferralStatusPending,
		RewardAmount: rewardAmount,
		Currency:     currency,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.referralRepo.Create(ctx, referral); err != nil {
		return nil, err
	}

	return referral, nil
}

// ListReferrals returns referrals for a tenant with pagination
func (s *ReferralService) ListReferrals(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Referral, error) {
	return s.referralRepo.List(ctx, tenantID, limit, offset)
}

// QualifyReferral marks a referral as qualified (e.g., when the referred customer makes their first payment)
func (s *ReferralService) QualifyReferral(ctx context.Context, tenantID uuid.UUID, referralID uuid.UUID) (*domain.Referral, error) {
	referral, err := s.referralRepo.GetByID(ctx, tenantID, referralID)
	if err != nil {
		return nil, err
	}
	if referral == nil {
		return nil, errors.New("referral not found")
	}

	if referral.Status != domain.ReferralStatusPending {
		return nil, fmt.Errorf("referral is already %s", referral.Status)
	}

	now := time.Now()
	referral.Status = domain.ReferralStatusQualified
	referral.QualifiedAt = &now
	referral.UpdatedAt = now

	if err := s.referralRepo.Update(ctx, referral); err != nil {
		return nil, err
	}

	return referral, nil
}

// ApplyReward marks a qualified referral as rewarded
func (s *ReferralService) ApplyReward(ctx context.Context, tenantID uuid.UUID, referralID uuid.UUID) (*domain.Referral, error) {
	referral, err := s.referralRepo.GetByID(ctx, tenantID, referralID)
	if err != nil {
		return nil, err
	}
	if referral == nil {
		return nil, errors.New("referral not found")
	}

	if referral.Status != domain.ReferralStatusQualified {
		return nil, fmt.Errorf("referral must be qualified before reward can be applied, current status: %s", referral.Status)
	}

	now := time.Now()
	referral.Status = domain.ReferralStatusRewarded
	referral.UpdatedAt = now

	if err := s.referralRepo.Update(ctx, referral); err != nil {
		return nil, err
	}

	return referral, nil
}
