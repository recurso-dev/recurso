package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// GatewayProvider identifies a payment gateway a tenant can connect their own
// merchant account to. Kept deliberately small for v1 — the two production
// gateways; the experimental adapters (gocardless/adyen) stay env-only until
// certified.
type GatewayProvider string

const (
	GatewayStripe   GatewayProvider = "stripe"
	GatewayRazorpay GatewayProvider = "razorpay"
)

// ValidGatewayProvider reports whether p is a connectable provider.
func ValidGatewayProvider(p GatewayProvider) bool {
	switch p {
	case GatewayStripe, GatewayRazorpay:
		return true
	}
	return false
}

// GatewayMode distinguishes a connection's test vs live credentials so the
// dashboard can badge it and callers never mix modes.
type GatewayMode string

const (
	GatewayModeTest GatewayMode = "test"
	GatewayModeLive GatewayMode = "live"
)

// GatewayConnection is a tenant's own payment-gateway credentials (BYO
// gateway). At most one active connection per (tenant, provider). The secret
// fields are AES-256-GCM sealed at rest via the secretbox — the *Enc columns
// hold ciphertext, never plaintext, and are never serialized to API clients.
//
// The public-ish identifiers (KeyID for Razorpay, PublishableKey for Stripe)
// are stored plaintext: they are shipped to the browser at checkout anyway.
type GatewayConnection struct {
	ID       uuid.UUID       `db:"id" json:"id"`
	TenantID uuid.UUID       `db:"tenant_id" json:"-"`
	Provider GatewayProvider `db:"provider" json:"provider"`
	Mode     GatewayMode     `db:"mode" json:"mode"`

	// PublicKey is the non-secret identifier surfaced to clients: Razorpay
	// key_id or Stripe publishable key. Safe to return in API responses.
	PublicKey string `db:"public_key" json:"public_key"`

	// SecretKeyEnc is the sealed gateway secret (Stripe secret key / Razorpay
	// key secret). Never serialized.
	SecretKeyEnc string `db:"secret_key_enc" json:"-"`
	// WebhookSecretEnc is the sealed signing secret for THIS connection's
	// webhook endpoint. Never serialized. May be empty until the tenant sets
	// up the webhook in their gateway dashboard.
	WebhookSecretEnc string `db:"webhook_secret_enc" json:"-"`

	// Active gates whether this connection is used for routing. Disconnecting
	// flips it false rather than deleting, preserving the audit trail.
	Active    bool      `db:"active" json:"active"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// HasSecret reports whether a sealed secret is present (a connection with no
// secret can't charge).
func (c *GatewayConnection) HasSecret() bool {
	return c != nil && c.SecretKeyEnc != ""
}

// Gateway connection domain errors.
var (
	ErrGatewayConnectionNotFound = errors.New("gateway connection not found")
	// ErrGatewayVaultUnavailable is returned when a BYO operation is attempted
	// but no GATEWAY_ENCRYPTION_KEY is configured — the credentials could not be
	// sealed, so the platform (env) gateway remains the only path.
	ErrGatewayVaultUnavailable = errors.New("gateway credential vault unavailable (no encryption key configured)")
)
