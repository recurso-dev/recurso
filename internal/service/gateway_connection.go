package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/secretbox"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// GatewayConnectionValidationError maps to HTTP 400.
type GatewayConnectionValidationError string

func (e GatewayConnectionValidationError) Error() string { return string(e) }

// GatewayConnectionService owns per-tenant BYO gateway credentials: it seals
// secrets before they hit the repository and opens them for the resolver /
// webhook router. A nil vault means no GATEWAY_ENCRYPTION_KEY is configured —
// every write fails with domain.ErrGatewayVaultUnavailable so plaintext secrets
// can never be persisted.
type GatewayConnectionService struct {
	repo  port.GatewayConnectionRepository
	vault *secretbox.Box
	now   func() time.Time
}

func NewGatewayConnectionService(repo port.GatewayConnectionRepository, vault *secretbox.Box) *GatewayConnectionService {
	return &GatewayConnectionService{repo: repo, vault: vault, now: time.Now}
}

// VaultReady reports whether credential storage is available (an encryption key
// is configured). The dashboard uses this to show a "not configured" state
// instead of failing connect attempts.
func (s *GatewayConnectionService) VaultReady() bool { return s.vault != nil }

// ConnectInput is the caller-facing shape for connecting a gateway. Secrets are
// plaintext here (submitted once by the tenant) and never stored as-is.
type ConnectInput struct {
	Provider      string
	Mode          string
	PublicKey     string // Razorpay key_id / Stripe publishable key
	SecretKey     string // Razorpay key_secret / Stripe secret key
	WebhookSecret string // optional; the connection's own signing secret
}

// Connect validates and stores (sealed) a tenant's gateway credentials,
// replacing any existing active connection for that provider.
func (s *GatewayConnectionService) Connect(ctx context.Context, tenantID uuid.UUID, in ConnectInput) (*domain.GatewayConnection, error) {
	if s.vault == nil {
		return nil, domain.ErrGatewayVaultUnavailable
	}

	provider := domain.GatewayProvider(strings.ToLower(strings.TrimSpace(in.Provider)))
	if !domain.ValidGatewayProvider(provider) {
		return nil, GatewayConnectionValidationError("provider must be one of: stripe, razorpay")
	}

	mode := domain.GatewayMode(strings.ToLower(strings.TrimSpace(in.Mode)))
	if mode == "" {
		mode = domain.GatewayModeTest
	}
	if mode != domain.GatewayModeTest && mode != domain.GatewayModeLive {
		return nil, GatewayConnectionValidationError("mode must be 'test' or 'live'")
	}

	secret := strings.TrimSpace(in.SecretKey)
	if secret == "" {
		return nil, GatewayConnectionValidationError("secret key is required")
	}
	publicKey := strings.TrimSpace(in.PublicKey)
	if provider == domain.GatewayRazorpay && publicKey == "" {
		return nil, GatewayConnectionValidationError("key_id is required for Razorpay")
	}

	secretEnc, err := s.vault.Seal(secret)
	if err != nil {
		return nil, err
	}
	webhookEnc, err := s.vault.Seal(strings.TrimSpace(in.WebhookSecret))
	if err != nil {
		return nil, err
	}

	now := s.now().UTC()
	conn := &domain.GatewayConnection{
		ID:               uuid.New(),
		TenantID:         tenantID,
		Provider:         provider,
		Mode:             mode,
		PublicKey:        publicKey,
		SecretKeyEnc:     secretEnc,
		WebhookSecretEnc: webhookEnc,
		Active:           true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.repo.Upsert(ctx, conn); err != nil {
		return nil, err
	}
	return conn, nil
}

// List returns the tenant's active connections. Secrets stay sealed on the
// struct and are omitted from JSON by the domain tags.
func (s *GatewayConnectionService) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.GatewayConnection, error) {
	return s.repo.ListByTenant(ctx, tenantID)
}

// Disconnect soft-deletes the tenant's active connection for a provider.
func (s *GatewayConnectionService) Disconnect(ctx context.Context, tenantID uuid.UUID, provider string) error {
	p := domain.GatewayProvider(strings.ToLower(strings.TrimSpace(provider)))
	if !domain.ValidGatewayProvider(p) {
		return GatewayConnectionValidationError("provider must be one of: stripe, razorpay")
	}
	return s.repo.Deactivate(ctx, tenantID, p)
}

// SetWebhookSecret seals and stores the webhook signing secret on the tenant's
// active connection in place (two-step connect: the tenant creates the webhook
// in their gateway console using the per-connection URL, then pastes back the
// secret). An empty secret clears it.
func (s *GatewayConnectionService) SetWebhookSecret(ctx context.Context, tenantID uuid.UUID, provider, secret string) error {
	if s.vault == nil {
		return domain.ErrGatewayVaultUnavailable
	}
	p := domain.GatewayProvider(strings.ToLower(strings.TrimSpace(provider)))
	if !domain.ValidGatewayProvider(p) {
		return GatewayConnectionValidationError("provider must be one of: stripe, razorpay")
	}
	sealed, err := s.vault.Seal(strings.TrimSpace(secret))
	if err != nil {
		return err
	}
	return s.repo.SetWebhookSecret(ctx, tenantID, p, sealed)
}

// OpenSecret decrypts a connection's gateway secret. Used by the resolver
// (increment 2) to build a live gateway client for a tenant.
func (s *GatewayConnectionService) OpenSecret(conn *domain.GatewayConnection) (string, error) {
	if s.vault == nil {
		return "", domain.ErrGatewayVaultUnavailable
	}
	return s.vault.Open(conn.SecretKeyEnc)
}

// OpenWebhookSecret decrypts a connection's webhook signing secret. Used by the
// per-connection webhook router (increment 3). Returns "" when none is set.
func (s *GatewayConnectionService) OpenWebhookSecret(conn *domain.GatewayConnection) (string, error) {
	if s.vault == nil {
		return "", domain.ErrGatewayVaultUnavailable
	}
	return s.vault.Open(conn.WebhookSecretEnc)
}

// GetByID resolves a connection for webhook routing.
func (s *GatewayConnectionService) GetByID(ctx context.Context, id uuid.UUID) (*domain.GatewayConnection, error) {
	return s.repo.GetByID(ctx, id)
}

// GetActive resolves a tenant's active connection for a provider, or
// domain.ErrGatewayConnectionNotFound.
func (s *GatewayConnectionService) GetActive(ctx context.Context, tenantID uuid.UUID, provider domain.GatewayProvider) (*domain.GatewayConnection, error) {
	return s.repo.GetActive(ctx, tenantID, provider)
}

// IsValidationError reports whether err is a caller-fixable validation error
// (for HTTP status mapping).
func IsGatewayConnectionValidationError(err error) bool {
	var v GatewayConnectionValidationError
	return errors.As(err, &v)
}
