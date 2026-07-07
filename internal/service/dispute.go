package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

// DisputeService provides the admin-facing operations over invoice disputes.
// The portal-facing (customer) operations live on PortalService, which owns
// the invoice-ownership guard.
type DisputeService struct {
	repo port.DisputeRepository
}

func NewDisputeService(repo port.DisputeRepository) *DisputeService {
	return &DisputeService{repo: repo}
}

// List returns tenant-scoped disputes, optionally filtered by status
// ("open" or "resolved"); an empty status returns all.
func (s *DisputeService) List(ctx context.Context, tenantID uuid.UUID, status string) ([]*domain.InvoiceDispute, error) {
	return s.repo.ListByTenant(ctx, tenantID, status)
}

// Resolve marks an open dispute resolved with an optional note. It is
// tenant-scoped and returns domain.ErrDisputeNotFound when no matching open
// dispute exists.
func (s *DisputeService) Resolve(ctx context.Context, tenantID, id uuid.UUID, note string) error {
	return s.repo.Resolve(ctx, tenantID, id, note)
}
