package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// ErrEntityValidation marks caller-correctable failures (bad request). Handlers
// map errors wrapping it to HTTP 400.
var ErrEntityValidation = errors.New("entity validation failed")

// entityStore is the persistence surface the service needs.
type entityStore interface {
	List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Entity, error)
	GetByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.Entity, error)
	GetPrimary(ctx context.Context, tenantID uuid.UUID) (*domain.Entity, error)
	Create(ctx context.Context, e *domain.Entity) error
	Update(ctx context.Context, e *domain.Entity) error
	Delete(ctx context.Context, id, tenantID uuid.UUID) error
}

// EntityService manages a tenant's legal entities (Multi-Entity Books, Inc 1).
type EntityService struct {
	repo entityStore
}

func NewEntityService(repo entityStore) *EntityService {
	return &EntityService{repo: repo}
}

func (s *EntityService) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Entity, error) {
	return s.repo.List(ctx, tenantID)
}

func (s *EntityService) Get(ctx context.Context, id, tenantID uuid.UUID) (*domain.Entity, error) {
	e, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, fmt.Errorf("%w: entity not found", ErrEntityValidation)
	}
	return e, nil
}

// CreateEntityInput is the create/update payload.
type CreateEntityInput struct {
	Name          string
	LegalName     string
	InvoicePrefix string
	CountryCode   string
}

// Create adds a new (non-primary) entity. The ledger id and invoice sequence are
// allocated by the repository.
func (s *EntityService) Create(ctx context.Context, tenantID uuid.UUID, in CreateEntityInput) (*domain.Entity, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrEntityValidation)
	}
	prefix := sanitizePrefix(in.InvoicePrefix, name)
	e := &domain.Entity{
		TenantID:      tenantID,
		Name:          name,
		LegalName:     strings.TrimSpace(in.LegalName),
		InvoicePrefix: prefix,
		CountryCode:   entityCountry(in.CountryCode),
	}
	if e.CountryCode != "" && len(e.CountryCode) != 2 {
		return nil, fmt.Errorf("%w: country_code must be a 2-letter ISO code", ErrEntityValidation)
	}
	if err := s.repo.Create(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

// Update edits an entity's name, legal name, invoice prefix, and country.
func (s *EntityService) Update(ctx context.Context, tenantID, id uuid.UUID, in CreateEntityInput) (*domain.Entity, error) {
	e, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, fmt.Errorf("%w: entity not found", ErrEntityValidation)
	}
	if strings.TrimSpace(in.Name) != "" {
		e.Name = strings.TrimSpace(in.Name)
	}
	e.LegalName = strings.TrimSpace(in.LegalName)
	if p := strings.TrimSpace(in.InvoicePrefix); p != "" {
		e.InvoicePrefix = sanitizePrefix(p, e.Name)
	}
	if c := entityCountry(in.CountryCode); c != "" {
		if len(c) != 2 {
			return nil, fmt.Errorf("%w: country_code must be a 2-letter ISO code", ErrEntityValidation)
		}
		e.CountryCode = c
	}
	if err := s.repo.Update(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

// Delete removes a non-primary entity. The primary can never be deleted (it is
// the backfill target every tenant needs).
func (s *EntityService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	e, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return err
	}
	if e == nil {
		return fmt.Errorf("%w: entity not found", ErrEntityValidation)
	}
	if e.IsPrimary {
		return fmt.Errorf("%w: the primary entity cannot be deleted", ErrEntityValidation)
	}
	return s.repo.Delete(ctx, id, tenantID)
}

func entityCountry(c string) string {
	return strings.ToUpper(strings.TrimSpace(c))
}

// sanitizePrefix keeps invoice prefixes to a safe, uppercase, hyphen-friendly
// slug; falls back to a slug of the name, then "INV".
func sanitizePrefix(prefix, name string) string {
	src := strings.TrimSpace(prefix)
	if src == "" {
		src = name
	}
	var b strings.Builder
	for _, r := range strings.ToUpper(src) {
		switch {
		case r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == ' ':
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "INV"
	}
	if len(out) > 12 {
		out = out[:12]
	}
	return out
}
