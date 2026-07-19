package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

type TenantService struct {
	repo *db.TenantRepository
}

func NewTenantService(repo *db.TenantRepository) *TenantService {
	return &TenantService{repo: repo}
}

func (s *TenantService) Register(ctx context.Context, name, email string) (*domain.Tenant, *domain.APIKey, error) {
	tenantID := uuid.New()
	tenant := &domain.Tenant{
		ID:        tenantID,
		Name:      name,
		Email:     email,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
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
