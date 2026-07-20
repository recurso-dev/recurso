package service

import (
	"context"
	"crypto/rand"
	"errors"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/secretbox"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/service/tax"
)

type fakeIntegrationRepo struct {
	rows map[uuid.UUID]*domain.IntegrationConnection
}

func newFakeIntegrationRepo() *fakeIntegrationRepo {
	return &fakeIntegrationRepo{rows: map[uuid.UUID]*domain.IntegrationConnection{}}
}

func (f *fakeIntegrationRepo) Upsert(_ context.Context, c *domain.IntegrationConnection) error {
	for _, e := range f.rows {
		if e.TenantID == c.TenantID && e.Category == c.Category && e.Provider == c.Provider && e.Active {
			e.Active = false
		}
	}
	cp := *c
	f.rows[c.ID] = &cp
	return nil
}
func (f *fakeIntegrationRepo) GetActive(_ context.Context, t uuid.UUID, cat domain.IntegrationCategory, prov string) (*domain.IntegrationConnection, error) {
	for _, e := range f.rows {
		if e.TenantID == t && e.Category == cat && e.Provider == prov && e.Active {
			return e, nil
		}
	}
	return nil, domain.ErrIntegrationConnectionNotFound
}
func (f *fakeIntegrationRepo) ListByTenant(_ context.Context, t uuid.UUID) ([]*domain.IntegrationConnection, error) {
	var out []*domain.IntegrationConnection
	for _, e := range f.rows {
		if e.TenantID == t && e.Active {
			out = append(out, e)
		}
	}
	return out, nil
}
func (f *fakeIntegrationRepo) Deactivate(_ context.Context, t uuid.UUID, cat domain.IntegrationCategory, prov string) error {
	for _, e := range f.rows {
		if e.TenantID == t && e.Category == cat && e.Provider == prov && e.Active {
			e.Active = false
			return nil
		}
	}
	return domain.ErrIntegrationConnectionNotFound
}

func integrationVault(t *testing.T) *secretbox.Box {
	t.Helper()
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		t.Fatal(err)
	}
	b, err := secretbox.New(key)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestIntegrationConnectSealsAndResolves(t *testing.T) {
	svc := NewIntegrationConnectionService(newFakeIntegrationRepo(), integrationVault(t))
	tenant := uuid.New()
	ctx := context.Background()

	conn, err := svc.Connect(ctx, tenant, "tax", "taxjar", map[string]string{"api_key": "tj_secret", "api_url": "https://api.taxjar.com"})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if conn.ConfigEnc == "" || conn.ConfigEnc == "tj_secret" {
		t.Fatalf("config not sealed: %q", conn.ConfigEnc)
	}
	cfg, ok := svc.Resolve(ctx, tenant, domain.IntegrationTax, "taxjar")
	if !ok || cfg["api_key"] != "tj_secret" || cfg["api_url"] != "https://api.taxjar.com" {
		t.Fatalf("Resolve = %v, %v", cfg, ok)
	}
	// Non-secret fields surface; secrets don't.
	present, hasSecret := svc.SafeConfig(ctx, conn)
	if present["api_url"] != "https://api.taxjar.com" || !hasSecret {
		t.Fatalf("SafeConfig = %v hasSecret=%v", present, hasSecret)
	}
	if _, leaked := present["api_key"]; leaked {
		t.Fatal("SafeConfig leaked a secret field")
	}
}

func TestIntegrationConnectValidation(t *testing.T) {
	svc := NewIntegrationConnectionService(newFakeIntegrationRepo(), integrationVault(t))
	tenant := uuid.New()
	ctx := context.Background()
	cases := []struct {
		name, cat, prov string
		cfg             map[string]string
	}{
		{"bad provider", "tax", "vertex", map[string]string{"api_key": "x"}},
		{"bad category", "email", "mailgun", map[string]string{"api_key": "x"}},
		{"taxjar missing key", "tax", "taxjar", map[string]string{}},
		{"avalara missing fields", "tax", "avalara", map[string]string{"account_id": "1"}},
		{"s3 missing fields", "storage", "s3", map[string]string{"bucket": "b"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.Connect(ctx, tenant, tc.cat, tc.prov, tc.cfg); err == nil || !IsIntegrationConnectionValidationError(err) {
				t.Fatalf("want validation error, got %v", err)
			}
		})
	}
}

func TestIntegrationConnectBlocksSSRFEndpoint(t *testing.T) {
	svc := NewIntegrationConnectionService(newFakeIntegrationRepo(), integrationVault(t))
	tenant := uuid.New()
	ctx := context.Background()
	base := map[string]string{"bucket": "b", "region": "us", "access_key_id": "k", "secret_access_key": "s"}
	with := func(endpoint string) map[string]string {
		m := map[string]string{}
		for k, v := range base {
			m[k] = v
		}
		m["endpoint"] = endpoint
		return m
	}

	// Multi-tenant (default): internal/reserved endpoints are rejected.
	for _, bad := range []string{
		"http://169.254.169.254/latest/meta-data/", // cloud metadata
		"http://localhost:9000",
		"http://127.0.0.1/",
		"http://10.0.0.5:9000",
		"http://minio:9000",       // resolves to nothing / not public
		"https://192.168.1.10",    // private
		"ftp://example.com",       // bad scheme
		"http://s3.amazonaws.com", // http (must be https) on multi-tenant
	} {
		if _, err := svc.Connect(ctx, tenant, "storage", "s3", with(bad)); err == nil || !IsIntegrationConnectionValidationError(err) {
			t.Fatalf("endpoint %q should be rejected, got %v", bad, err)
		}
	}

	// A public https endpoint is allowed (IP literal keeps the test off DNS).
	if _, err := svc.Connect(ctx, tenant, "storage", "s3", with("https://8.8.8.8")); err != nil {
		t.Fatalf("public https endpoint should be allowed, got %v", err)
	}
}

func TestIntegrationSelfHostedAllowsPrivateEndpoint(t *testing.T) {
	svc := NewIntegrationConnectionService(newFakeIntegrationRepo(), integrationVault(t))
	svc.SetAllowPrivateEgress(true) // self-hosted
	tenant := uuid.New()
	cfg := map[string]string{"bucket": "b", "region": "us", "access_key_id": "k", "secret_access_key": "s", "endpoint": "http://minio:9000"}
	if _, err := svc.Connect(context.Background(), tenant, "storage", "s3", cfg); err != nil {
		t.Fatalf("self-hosted should allow a private MinIO endpoint, got %v", err)
	}
}

func TestIntegrationConnectWithoutVault(t *testing.T) {
	svc := NewIntegrationConnectionService(newFakeIntegrationRepo(), nil)
	if svc.VaultReady() {
		t.Fatal("VaultReady should be false")
	}
	_, err := svc.Connect(context.Background(), uuid.New(), "crm", "hubspot", map[string]string{"access_token": "x"})
	if !errors.Is(err, domain.ErrGatewayVaultUnavailable) {
		t.Fatalf("want ErrGatewayVaultUnavailable, got %v", err)
	}
}

func TestIntegrationDisconnectAndReplace(t *testing.T) {
	svc := NewIntegrationConnectionService(newFakeIntegrationRepo(), integrationVault(t))
	tenant := uuid.New()
	ctx := context.Background()
	_, _ = svc.Connect(ctx, tenant, "crm", "hubspot", map[string]string{"access_token": "v1"})
	_, _ = svc.Connect(ctx, tenant, "crm", "hubspot", map[string]string{"access_token": "v2"})
	cfg, _ := svc.Resolve(ctx, tenant, domain.IntegrationCRM, "hubspot")
	if cfg["access_token"] != "v2" {
		t.Fatalf("replace failed: %v", cfg)
	}
	if err := svc.Disconnect(ctx, tenant, "crm", "hubspot"); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}
	if _, ok := svc.Resolve(ctx, tenant, domain.IntegrationCRM, "hubspot"); ok {
		t.Fatal("expected no active connection after disconnect")
	}
}

// stubSalesTax is a minimal tax.SalesTaxProvider recording its identity.
type stubSalesTax struct{ name string }

func (s *stubSalesTax) Name() string { return s.name }
func (s *stubSalesTax) LookupSalesTax(_ context.Context, _ *tax.SalesTaxQuery) (*tax.SalesTaxResult, error) {
	return &tax.SalesTaxResult{}, nil
}

func TestSalesTaxResolverPerTenantAndFallback(t *testing.T) {
	svc := NewIntegrationConnectionService(newFakeIntegrationRepo(), integrationVault(t))
	tenant := uuid.New()
	other := uuid.New()
	ctx := context.Background()
	_, _ = svc.Connect(ctx, tenant, "tax", "taxjar", map[string]string{"api_key": "tj_tenant"})

	built := ""
	resolver := NewSalesTaxProviderResolver(svc, func(provider string, cfg map[string]string) tax.SalesTaxProvider {
		built = provider + ":" + cfg["api_key"]
		return &stubSalesTax{name: built}
	})

	// Tenant with a connection -> their provider is built.
	if p := resolver.For(ctx, tenant); p == nil || built != "taxjar:tj_tenant" {
		t.Fatalf("expected tenant provider, built=%q p=%v", built, p)
	}
	// Cached: a second call doesn't rebuild.
	built = ""
	if p := resolver.For(ctx, tenant); p == nil || built != "" {
		t.Fatalf("expected cached provider (no rebuild), built=%q", built)
	}
	// Tenant with no connection -> nil (caller falls back to env).
	if p := resolver.For(ctx, other); p != nil {
		t.Fatalf("expected nil for un-connected tenant, got %v", p)
	}
}
