package domain

import (
	"time"

	"github.com/google/uuid"
)

// MagicLink represents a passwordless login link for customers
type MagicLink struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	CustomerID uuid.UUID  `json:"customer_id" db:"customer_id"`
	Token      string     `json:"-" db:"token"`
	ExpiresAt  time.Time  `json:"expires_at" db:"expires_at"`
	UsedAt     *time.Time `json:"used_at" db:"used_at"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}

// IsExpired checks if the magic link has expired
func (m *MagicLink) IsExpired() bool {
	return time.Now().After(m.ExpiresAt)
}

// IsUsed checks if the magic link has been used
func (m *MagicLink) IsUsed() bool {
	return m.UsedAt != nil
}

// PortalSession represents an authenticated customer session
type PortalSession struct {
	ID         uuid.UUID `json:"id" db:"id"`
	CustomerID uuid.UUID `json:"customer_id" db:"customer_id"`
	Token      string    `json:"-" db:"token"`
	ExpiresAt  time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// IsExpired checks if the session has expired
func (s *PortalSession) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}
