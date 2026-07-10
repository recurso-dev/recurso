package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// MagicLinkRepository handles magic link persistence
type MagicLinkRepository interface {
	Create(ctx context.Context, link *domain.MagicLink) error
	GetByToken(ctx context.Context, token string) (*domain.MagicLink, error)
	// MarkUsed atomically consumes the link, returning true only if this call
	// claimed a still-unused link (closes the single-use race).
	MarkUsed(ctx context.Context, id uuid.UUID) (bool, error)
	DeleteExpired(ctx context.Context) error
}

// PortalSessionRepository handles portal session persistence
type PortalSessionRepository interface {
	Create(ctx context.Context, session *domain.PortalSession) error
	GetByToken(ctx context.Context, token string) (*domain.PortalSession, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteExpired(ctx context.Context) error
}
