package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// OAuthIdentity links a dashboard user to an external identity-provider account
// (Google or GitHub). It stores no access/refresh tokens — only the stable
// provider user id and the email seen at link time. Lookups by
// (Provider, ProviderUserID) are unique.
type OAuthIdentity struct {
	ID             uuid.UUID `db:"id"`
	UserID         uuid.UUID `db:"user_id"`
	Provider       string    `db:"provider"`         // "google" | "github"
	ProviderUserID string    `db:"provider_user_id"` // stable, opaque id from the provider
	Email          string    `db:"email"`            // email seen at link time (lower-cased)
	CreatedAt      time.Time `db:"created_at"`
}

// OAuth domain errors. Kept coarse at the HTTP boundary (the callback only ever
// redirects to DASHBOARD_URL/login?error=oauth), but distinguishable internally
// for logging and tests.
var (
	// ErrOAuthEmailUnverified is returned when a provider reports an email that
	// is not verified and no existing identity matches. We refuse to create or
	// link an account on an unverified email (prevents account takeover).
	ErrOAuthEmailUnverified = errors.New("oauth email is not verified")
	// ErrOAuthProviderDisabled is returned for an unknown or unconfigured
	// provider (its client id/secret env vars are unset). Surfaces as 404.
	ErrOAuthProviderDisabled = errors.New("oauth provider is not enabled")
	// ErrOAuthStateMismatch is returned when the CSRF state in the callback does
	// not match the value bound in the short-lived cookie. Surfaces as 403.
	ErrOAuthStateMismatch = errors.New("oauth state mismatch")
)
