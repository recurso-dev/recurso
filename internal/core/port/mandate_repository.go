package port

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

type MandateRepository interface {
	Create(ctx context.Context, mandate *domain.Mandate) error
	GetByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.Mandate, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Mandate, error)
	Update(ctx context.Context, mandate *domain.Mandate) error
	GetByRazorpayTokenID(ctx context.Context, tokenID string) (*domain.Mandate, error)
	GetDueForPreNotification(ctx context.Context) ([]*domain.Mandate, error)
	// GetReadyForDebit is a read-only view of mandates currently due for debit.
	// Do NOT use it to drive the debit scheduler — it does not claim rows, so
	// concurrent runners would both charge the same mandate. Use ClaimDueForDebit.
	GetReadyForDebit(ctx context.Context) ([]*domain.Mandate, error)
	// ClaimDueForDebit atomically claims the mandates currently due for debit,
	// advancing their next_debit_at by claimWindow and returning the claimed rows.
	// Concurrent callers get disjoint sets, so each due mandate is charged by
	// exactly one runner even when the distributed scheduler lock is absent
	// (ENG-161). claimWindow also serves as the failure-retry lease.
	ClaimDueForDebit(ctx context.Context, claimWindow time.Duration) ([]*domain.Mandate, error)
}
