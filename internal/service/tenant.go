package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/adapter/db"
	"github.com/recur-so/recurso/internal/core/domain"
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

	// Generate a simple secret key
	// In production, use a secure random generator and larger entropy
	keyID := uuid.New()
	randomPart := uuid.New().String()
	keyValue := fmt.Sprintf("sk_live_%s", randomPart)

	apiKey := &domain.APIKey{
		ID:        keyID,
		TenantID:  tenantID,
		KeyValue:  keyValue,
		Type:      "secret",
		IsActive:  true,
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

func (s *TenantService) GenerateKey(ctx context.Context, tenantID uuid.UUID, name string) (*domain.APIKey, error) {
	// Name is unused in MVP schema, but good for future
	keyID := uuid.New()
	randomPart := uuid.New().String()
	keyValue := fmt.Sprintf("sk_live_%s", randomPart)

	apiKey := &domain.APIKey{
		ID:        keyID,
		TenantID:  tenantID,
		KeyValue:  keyValue,
		Type:      "secret",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.repo.CreateAPIKey(ctx, apiKey); err != nil {
		return nil, err
	}

	return apiKey, nil
}
