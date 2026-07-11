package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// OAuthIdentityRepository persists links between dashboard users and external
// identity-provider accounts (Google/GitHub).
type OAuthIdentityRepository interface {
	// Create inserts a new identity. It returns domain.ErrDuplicateEmail-style
	// uniqueness only via GetByProviderUserID; callers check existence first.
	Create(ctx context.Context, identity *domain.OAuthIdentity) error
	// GetByProviderUserID resolves the identity for a (provider, providerUserID)
	// pair, or domain.ErrUserNotFound if none exists.
	GetByProviderUserID(ctx context.Context, provider, providerUserID string) (*domain.OAuthIdentity, error)
	// ListByUser returns every identity linked to a user.
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.OAuthIdentity, error)
}
