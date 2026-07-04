package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// ConsentRepository interface for consent data operations
type ConsentRepository interface {
	Create(ctx context.Context, tenantID uuid.UUID, record domain.ConsentRecord) (*domain.Consent, error)
	GetByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) ([]domain.Consent, error)
	GetBySubscription(ctx context.Context, tenantID, subscriptionID uuid.UUID) (*domain.Consent, error)
	Revoke(ctx context.Context, tenantID, consentID uuid.UUID) error
	RevokeBySubscription(ctx context.Context, tenantID, subscriptionID uuid.UUID) error
	GetActiveByType(ctx context.Context, tenantID, customerID uuid.UUID, consentType domain.ConsentType) (*domain.Consent, error)
}

// ConsentService handles consent business logic
type ConsentService struct {
	repo ConsentRepository
}

// NewConsentService creates a new ConsentService
func NewConsentService(repo ConsentRepository) *ConsentService {
	return &ConsentService{repo: repo}
}

// RecordRecurringBillingConsent records consent for recurring billing (RBI compliance)
func (s *ConsentService) RecordRecurringBillingConsent(ctx context.Context, tenantID uuid.UUID, customerID uuid.UUID, subscriptionID *uuid.UUID, ipAddress, userAgent string) (*domain.Consent, error) {
	record := domain.ConsentRecord{
		CustomerID:     customerID,
		SubscriptionID: subscriptionID,
		ConsentType:    domain.ConsentTypeRecurringBilling,
		Granted:        true,
		IPAddress:      ipAddress,
		UserAgent:      userAgent,
		ConsentText:    domain.RecurringBillingConsentText,
		Version:        domain.CurrentConsentVersion,
	}

	return s.repo.Create(ctx, tenantID, record)
}

// RecordConsent records a general consent
func (s *ConsentService) RecordConsent(ctx context.Context, tenantID uuid.UUID, record domain.ConsentRecord) (*domain.Consent, error) {
	if record.Version == "" {
		record.Version = domain.CurrentConsentVersion
	}
	return s.repo.Create(ctx, tenantID, record)
}

// GetCustomerConsents retrieves all consents for a customer
func (s *ConsentService) GetCustomerConsents(ctx context.Context, tenantID, customerID uuid.UUID) ([]domain.Consent, error) {
	return s.repo.GetByCustomer(ctx, tenantID, customerID)
}

// GetSubscriptionConsent retrieves the recurring billing consent for a subscription
func (s *ConsentService) GetSubscriptionConsent(ctx context.Context, tenantID, subscriptionID uuid.UUID) (*domain.Consent, error) {
	return s.repo.GetBySubscription(ctx, tenantID, subscriptionID)
}

// RevokeConsent revokes a specific consent
func (s *ConsentService) RevokeConsent(ctx context.Context, tenantID, consentID uuid.UUID) error {
	return s.repo.Revoke(ctx, tenantID, consentID)
}

// RevokeSubscriptionConsent revokes consent for a subscription (on cancellation)
func (s *ConsentService) RevokeSubscriptionConsent(ctx context.Context, tenantID, subscriptionID uuid.UUID) error {
	return s.repo.RevokeBySubscription(ctx, tenantID, subscriptionID)
}

// HasActiveRecurringBillingConsent checks if customer has active recurring billing consent
func (s *ConsentService) HasActiveRecurringBillingConsent(ctx context.Context, tenantID, customerID uuid.UUID) (bool, error) {
	consent, err := s.repo.GetActiveByType(ctx, tenantID, customerID, domain.ConsentTypeRecurringBilling)
	if err != nil {
		return false, err
	}
	return consent != nil && consent.Granted, nil
}
