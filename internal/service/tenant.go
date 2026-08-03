package service

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// defaultTrialDays is the managed-cloud trial length; override with TRIAL_DAYS.
const defaultTrialDays = 14

func trialDays() int {
	if v := os.Getenv("TRIAL_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return defaultTrialDays
}

type TenantService struct {
	repo *db.TenantRepository
}

func NewTenantService(repo *db.TenantRepository) *TenantService {
	return &TenantService{repo: repo}
}

func (s *TenantService) Register(ctx context.Context, name, email string) (*domain.Tenant, *domain.APIKey, error) {
	tenantID := uuid.New()
	now := time.Now()
	trialEnd := now.Add(time.Duration(trialDays()) * 24 * time.Hour)
	tenant := &domain.Tenant{
		ID:        tenantID,
		Name:      name,
		Email:     email,
		CreatedAt: now,
		UpdatedAt: now,
		// New signups start a managed-cloud trial.
		TrialEndsAt:   &trialEnd,
		BillingStatus: domain.BillingStatusTrialing,
		PlanTier:      domain.PlanTierTrial,
	}

	// New tenants get a test-mode key by default — safe to develop against,
	// and rejected by a live-money server until the tenant mints a live key.
	keyID := uuid.New()
	randomPart := uuid.New().String()
	keyValue := domain.NewAPIKeyValue(false, randomPart)

	apiKey := &domain.APIKey{
		ID:        keyID,
		TenantID:  tenantID,
		KeyValue:  keyValue,
		Type:      "secret",
		IsActive:  true,
		Livemode:  false,
		CreatedAt: time.Now(),
	}

	// Transaction would be better here, but simple sequential writes for MVP
	if err := s.repo.CreateTenant(ctx, tenant); err != nil {
		return nil, nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	if err := s.repo.CreateAPIKey(ctx, apiKey); err != nil {
		return nil, nil, fmt.Errorf("failed to create api key: %w", err)
	}

	return tenant, apiKey, nil
}

func (s *TenantService) ListKeys(ctx context.Context, tenantID uuid.UUID) ([]*domain.APIKey, error) {
	return s.repo.ListAPIKeys(ctx, tenantID)
}

// RevokeKey deactivates an API key; it stops authenticating immediately.
func (s *TenantService) RevokeKey(ctx context.Context, tenantID, keyID uuid.UUID) error {
	return s.repo.RevokeAPIKey(ctx, tenantID, keyID)
}

// TenantName returns the tenant's display name, or "" — used on printed
// documents where a lookup failure should degrade, not fail the render.
func (s *TenantService) TenantName(ctx context.Context, tenantID uuid.UUID) string {
	t, err := s.GetAccount(ctx, tenantID)
	if err != nil || t == nil {
		return ""
	}
	return t.Name
}

func (s *TenantService) GetAccount(ctx context.Context, tenantID uuid.UUID) (*domain.Tenant, error) {
	return s.repo.GetByID(ctx, tenantID)
}

func (s *TenantService) UpdateAccount(ctx context.Context, tenantID uuid.UUID, name, email string) (*domain.Tenant, error) {
	tenant, err := s.repo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	tenant.Name = name
	tenant.Email = email
	tenant.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, tenant); err != nil {
		return nil, err
	}
	return tenant, nil
}

func (s *TenantService) GenerateKey(ctx context.Context, tenantID uuid.UUID, name string, livemode bool) (*domain.APIKey, error) {
	// Name is unused in MVP schema, but good for future
	keyID := uuid.New()
	randomPart := uuid.New().String()
	keyValue := domain.NewAPIKeyValue(livemode, randomPart)

	apiKey := &domain.APIKey{
		ID:        keyID,
		TenantID:  tenantID,
		KeyValue:  keyValue,
		Type:      "secret",
		IsActive:  true,
		Livemode:  livemode,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.repo.CreateAPIKey(ctx, apiKey); err != nil {
		return nil, err
	}

	return apiKey, nil
}
