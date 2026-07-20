package service

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/secretbox"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// IntegrationConnectionValidationError maps to HTTP 400.
type IntegrationConnectionValidationError string

func (e IntegrationConnectionValidationError) Error() string { return string(e) }

// IsIntegrationConnectionValidationError reports whether err is caller-fixable.
func IsIntegrationConnectionValidationError(err error) bool {
	var v IntegrationConnectionValidationError
	return errors.As(err, &v)
}

// requiredFields lists the config keys each provider must supply; used for
// validation and to drive the connect form.
var integrationRequiredFields = map[string][]string{
	"taxjar":  {"api_key"},
	"avalara": {"account_id", "license_key", "company_code"},
	"hubspot": {"access_token"},
	"s3":      {"bucket", "region", "access_key_id", "secret_access_key"},
}

// nonSecretFields are safe to surface back to the dashboard (endpoints, region,
// bucket) — everything else is treated as a secret and never returned.
var integrationNonSecretFields = map[string]bool{
	"api_url": true, "base_url": true, "endpoint": true,
	"region": true, "bucket": true, "prefix": true, "company_code": true,
}

// IntegrationConnectionService owns per-tenant credentials for tax/CRM/storage
// integrations. Config is sealed as one JSON blob; a nil vault (no
// GATEWAY_ENCRYPTION_KEY) fails writes with ErrGatewayVaultUnavailable so
// plaintext is never persisted.
type IntegrationConnectionService struct {
	repo  port.IntegrationConnectionRepository
	vault *secretbox.Box
	now   func() time.Time
	// allowPrivateEgress permits tenant-supplied endpoints that resolve to
	// private/reserved IPs. False (the default) on multi-tenant deployments so a
	// tenant can't SSRF the host into internal services / cloud metadata; set
	// true on single-tenant self-hosted where a private MinIO endpoint is legit.
	allowPrivateEgress bool
}

func NewIntegrationConnectionService(repo port.IntegrationConnectionRepository, vault *secretbox.Box) *IntegrationConnectionService {
	return &IntegrationConnectionService{repo: repo, vault: vault, now: time.Now}
}

// SetAllowPrivateEgress permits private/reserved endpoint hosts (self-hosted).
func (s *IntegrationConnectionService) SetAllowPrivateEgress(v bool) { s.allowPrivateEgress = v }

// urlConfigFields are tenant-supplied config values the server later fetches —
// the SSRF surface. Validated at connect time.
var urlConfigFields = map[string]bool{"endpoint": true, "api_url": true, "base_url": true}

// validateEndpointURL rejects a tenant-supplied URL that would let the server
// reach an internal/reserved address (SSRF). Requires http(s); on multi-tenant
// deployments every resolved IP must be a global unicast address, blocking
// loopback, link-local (169.254.169.254 cloud metadata), private, and
// unique-local ranges. NOTE: this is a connect-time check; a short-TTL DNS
// record could still rebind at fetch time (TOCTOU) — acceptable as a first
// mitigation for an authenticated owner/admin actor.
func (s *IntegrationConnectionService) validateEndpointURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return IntegrationConnectionValidationError("endpoint must be an http(s) URL")
	}
	if s.allowPrivateEgress {
		return nil
	}
	if u.Scheme != "https" {
		return IntegrationConnectionValidationError("endpoint must use https")
	}
	host := u.Hostname()
	var ips []net.IP
	if ip := net.ParseIP(host); ip != nil {
		ips = []net.IP{ip}
	} else {
		resolved, err := net.LookupIP(host)
		if err != nil || len(resolved) == 0 {
			return IntegrationConnectionValidationError("endpoint host could not be resolved")
		}
		ips = resolved
	}
	for _, ip := range ips {
		if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
			return IntegrationConnectionValidationError("endpoint resolves to a private or reserved address")
		}
	}
	return nil
}

func (s *IntegrationConnectionService) VaultReady() bool { return s.vault != nil }

// Connect validates and stores (sealed) a tenant's integration config,
// replacing any existing active connection for that (category, provider).
func (s *IntegrationConnectionService) Connect(ctx context.Context, tenantID uuid.UUID, category, provider string, config map[string]string) (*domain.IntegrationConnection, error) {
	if s.vault == nil {
		return nil, domain.ErrGatewayVaultUnavailable
	}
	cat := domain.IntegrationCategory(strings.ToLower(strings.TrimSpace(category)))
	prov := strings.ToLower(strings.TrimSpace(provider))
	if !domain.ValidIntegration(cat, prov) {
		return nil, IntegrationConnectionValidationError("unsupported integration category/provider")
	}
	// Trim values and require the provider's mandatory fields.
	clean := map[string]string{}
	for k, v := range config {
		clean[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	for _, f := range integrationRequiredFields[prov] {
		if clean[f] == "" {
			return nil, IntegrationConnectionValidationError(prov + ": " + f + " is required")
		}
	}
	// SSRF guard: any tenant-supplied URL the server will later fetch must not
	// point at an internal/reserved address.
	for field, val := range clean {
		if urlConfigFields[field] && val != "" {
			if err := s.validateEndpointURL(val); err != nil {
				return nil, err
			}
		}
	}

	blob, err := json.Marshal(clean)
	if err != nil {
		return nil, err
	}
	sealed, err := s.vault.Seal(string(blob))
	if err != nil {
		return nil, err
	}

	now := s.now().UTC()
	conn := &domain.IntegrationConnection{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Category:  cat,
		Provider:  prov,
		ConfigEnc: sealed,
		Active:    true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.Upsert(ctx, conn); err != nil {
		return nil, err
	}
	return conn, nil
}

// List returns the tenant's active connections (secret-free).
func (s *IntegrationConnectionService) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.IntegrationConnection, error) {
	return s.repo.ListByTenant(ctx, tenantID)
}

// Disconnect soft-deletes the tenant's active connection for a (category, provider).
func (s *IntegrationConnectionService) Disconnect(ctx context.Context, tenantID uuid.UUID, category, provider string) error {
	cat := domain.IntegrationCategory(strings.ToLower(strings.TrimSpace(category)))
	prov := strings.ToLower(strings.TrimSpace(provider))
	if !domain.ValidIntegration(cat, prov) {
		return IntegrationConnectionValidationError("unsupported integration category/provider")
	}
	return s.repo.Deactivate(ctx, tenantID, cat, prov)
}

// Resolve returns the decrypted config for a tenant's active connection to a
// (category, provider), or (nil, false) when none exists / the vault is
// unavailable. Callers fall back to env config on false.
func (s *IntegrationConnectionService) Resolve(ctx context.Context, tenantID uuid.UUID, category domain.IntegrationCategory, provider string) (map[string]string, bool) {
	if s == nil || s.vault == nil {
		return nil, false
	}
	conn, err := s.repo.GetActive(ctx, tenantID, category, provider)
	if err != nil || conn == nil {
		return nil, false
	}
	plain, err := s.vault.Open(conn.ConfigEnc)
	if err != nil || plain == "" {
		return nil, false
	}
	var cfg map[string]string
	if err := json.Unmarshal([]byte(plain), &cfg); err != nil {
		return nil, false
	}
	return cfg, true
}

// SafeConfig returns the non-secret config fields for display (region, bucket,
// endpoints), plus which secret fields are set — never the secret values.
func (s *IntegrationConnectionService) SafeConfig(ctx context.Context, conn *domain.IntegrationConnection) (present map[string]string, hasSecret bool) {
	present = map[string]string{}
	cfg, ok := s.Resolve(ctx, conn.TenantID, conn.Category, conn.Provider)
	if !ok {
		return present, false
	}
	for k, v := range cfg {
		if integrationNonSecretFields[k] {
			present[k] = v
		} else if v != "" {
			hasSecret = true
		}
	}
	return present, hasSecret
}

// RequiredFields exposes the mandatory config keys for a provider (drives the UI).
func RequiredFields(provider string) []string {
	return integrationRequiredFields[strings.ToLower(provider)]
}
