package port

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// SSOAssertionReplayStore records consumed SAML assertion IDs so a replayed
// SAMLResponse (same assertion presented twice) is rejected. It is the
// server-side half of SAML replay protection; the store must be shared across
// all instances that terminate the ACS endpoint.
type SSOAssertionReplayStore interface {
	// MarkConsumed atomically records assertionID as consumed for tenantID,
	// retained until expiresAt (the assertion's NotOnOrAfter). It returns
	// domain.ErrSSOAssertionReplay if assertionID was already recorded, and nil
	// on the first (winning) consume.
	MarkConsumed(ctx context.Context, tenantID uuid.UUID, assertionID string, expiresAt time.Time) error
}
