package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// inviteTTL is how long a team-invite link stays valid. Longer than a password
// reset (a new teammate may not act immediately), but still bounded.
const inviteTTL = 7 * 24 * time.Hour

// InviteUser adds a teammate WITHOUT the admin ever choosing their password.
// The account is created with a long random password the admin never learns, so
// it cannot be logged into; the invitee receives a single-use link and sets
// their own password (which runs through the existing ResetPassword flow).
// Reuses the password-reset token machinery and all of CreateUser's guards
// (role validity, owner-escalation, duplicate email).
func (s *AuthService) InviteUser(ctx context.Context, tenantID uuid.UUID, actorRole domain.Role, email, name string, role domain.Role) (*domain.User, error) {
	if s.resetTokens == nil {
		return nil, fmt.Errorf("invitations are not configured")
	}

	// A random password the admin never sees: the account stays unusable until
	// the invitee accepts and sets their own.
	randomPassword, _, err := newSessionToken()
	if err != nil {
		return nil, err
	}
	user, err := s.CreateUser(ctx, tenantID, actorRole, email, name, role, randomPassword)
	if err != nil {
		return nil, err
	}

	raw, tokenHash, err := newSessionToken()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if err := s.resetTokens.Create(ctx, &domain.PasswordResetToken{
		ID:        uuid.New(),
		TokenHash: tokenHash,
		UserID:    user.ID,
		ExpiresAt: now.Add(inviteTTL),
		CreatedAt: now,
	}); err != nil {
		return nil, err
	}

	// The accept page calls POST /auth/reset-password with this token.
	link := fmt.Sprintf("%s/accept-invite?token=%s", s.appBaseURL, raw)
	if s.mailer != nil {
		if err := s.mailer.SendInvite(ctx, user.Email, user.Name, link); err != nil {
			// Best-effort: the account + token already exist, so surface to logs
			// but don't fail the invite.
			s.logger.Error("failed to send team invite email", "error", err)
		}
	}
	return user, nil
}
