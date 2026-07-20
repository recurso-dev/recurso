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
)

// fakeGatewayConnRepo is an in-memory GatewayConnectionRepository.
type fakeGatewayConnRepo struct {
	byID     map[uuid.UUID]*domain.GatewayConnection
	upserted int
}

func newFakeGatewayConnRepo() *fakeGatewayConnRepo {
	return &fakeGatewayConnRepo{byID: map[uuid.UUID]*domain.GatewayConnection{}}
}

func (f *fakeGatewayConnRepo) Upsert(_ context.Context, conn *domain.GatewayConnection) error {
	f.upserted++
	// mimic the partial-unique index: deactivate prior active same-provider rows
	for _, c := range f.byID {
		if c.TenantID == conn.TenantID && c.Provider == conn.Provider && c.Active {
			c.Active = false
		}
	}
	cp := *conn
	f.byID[conn.ID] = &cp
	return nil
}

func (f *fakeGatewayConnRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.GatewayConnection, error) {
	if c, ok := f.byID[id]; ok {
		return c, nil
	}
	return nil, domain.ErrGatewayConnectionNotFound
}

func (f *fakeGatewayConnRepo) GetActive(_ context.Context, tenantID uuid.UUID, provider domain.GatewayProvider) (*domain.GatewayConnection, error) {
	for _, c := range f.byID {
		if c.TenantID == tenantID && c.Provider == provider && c.Active {
			return c, nil
		}
	}
	return nil, domain.ErrGatewayConnectionNotFound
}

func (f *fakeGatewayConnRepo) ListByTenant(_ context.Context, tenantID uuid.UUID) ([]*domain.GatewayConnection, error) {
	var out []*domain.GatewayConnection
	for _, c := range f.byID {
		if c.TenantID == tenantID && c.Active {
			out = append(out, c)
		}
	}
	return out, nil
}

func (f *fakeGatewayConnRepo) Deactivate(_ context.Context, tenantID uuid.UUID, provider domain.GatewayProvider) error {
	for _, c := range f.byID {
		if c.TenantID == tenantID && c.Provider == provider && c.Active {
			c.Active = false
			return nil
		}
	}
	return domain.ErrGatewayConnectionNotFound
}

func testVault(t *testing.T) *secretbox.Box {
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

func TestConnectSealsSecretAndRoundTrips(t *testing.T) {
	repo := newFakeGatewayConnRepo()
	vault := testVault(t)
	svc := NewGatewayConnectionService(repo, vault)
	tenant := uuid.New()

	conn, err := svc.Connect(context.Background(), tenant, ConnectInput{
		Provider:      "stripe",
		Mode:          "live",
		PublicKey:     "pk_live_abc",
		SecretKey:     "sk_live_topsecret",
		WebhookSecret: "whsec_123",
	})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// Secret must be stored sealed, not plaintext.
	if conn.SecretKeyEnc == "" || conn.SecretKeyEnc == "sk_live_topsecret" {
		t.Fatalf("secret not sealed: %q", conn.SecretKeyEnc)
	}
	// ... but decryptable via the service.
	got, err := svc.OpenSecret(conn)
	if err != nil || got != "sk_live_topsecret" {
		t.Fatalf("OpenSecret = %q, %v", got, err)
	}
	wh, err := svc.OpenWebhookSecret(conn)
	if err != nil || wh != "whsec_123" {
		t.Fatalf("OpenWebhookSecret = %q, %v", wh, err)
	}
	if conn.Mode != domain.GatewayModeLive || conn.PublicKey != "pk_live_abc" {
		t.Fatalf("unexpected mode/publicKey: %+v", conn)
	}
}

func TestConnectReplacesActiveConnection(t *testing.T) {
	repo := newFakeGatewayConnRepo()
	svc := NewGatewayConnectionService(repo, testVault(t))
	tenant := uuid.New()
	ctx := context.Background()

	_, _ = svc.Connect(ctx, tenant, ConnectInput{Provider: "razorpay", PublicKey: "rzp_key", SecretKey: "s1"})
	_, _ = svc.Connect(ctx, tenant, ConnectInput{Provider: "razorpay", PublicKey: "rzp_key2", SecretKey: "s2"})

	active, err := svc.GetActive(ctx, tenant, domain.GatewayRazorpay)
	if err != nil {
		t.Fatalf("GetActive: %v", err)
	}
	if active.PublicKey != "rzp_key2" {
		t.Fatalf("expected latest connection active, got %q", active.PublicKey)
	}
	list, _ := svc.List(ctx, tenant)
	if len(list) != 1 {
		t.Fatalf("expected exactly one active connection, got %d", len(list))
	}
}

func TestConnectValidation(t *testing.T) {
	svc := NewGatewayConnectionService(newFakeGatewayConnRepo(), testVault(t))
	tenant := uuid.New()
	ctx := context.Background()

	cases := []struct {
		name string
		in   ConnectInput
	}{
		{"bad provider", ConnectInput{Provider: "paypal", SecretKey: "x"}},
		{"missing secret", ConnectInput{Provider: "stripe", SecretKey: ""}},
		{"razorpay needs key_id", ConnectInput{Provider: "razorpay", SecretKey: "x"}},
		{"bad mode", ConnectInput{Provider: "stripe", SecretKey: "x", Mode: "sandbox"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.Connect(ctx, tenant, tc.in); err == nil || !IsGatewayConnectionValidationError(err) {
				t.Fatalf("want validation error, got %v", err)
			}
		})
	}
}

func TestConnectWithoutVaultFails(t *testing.T) {
	svc := NewGatewayConnectionService(newFakeGatewayConnRepo(), nil)
	if svc.VaultReady() {
		t.Fatal("VaultReady should be false without a vault")
	}
	_, err := svc.Connect(context.Background(), uuid.New(), ConnectInput{Provider: "stripe", SecretKey: "x"})
	if !errors.Is(err, domain.ErrGatewayVaultUnavailable) {
		t.Fatalf("want ErrGatewayVaultUnavailable, got %v", err)
	}
}

func TestDisconnect(t *testing.T) {
	repo := newFakeGatewayConnRepo()
	svc := NewGatewayConnectionService(repo, testVault(t))
	tenant := uuid.New()
	ctx := context.Background()

	_, _ = svc.Connect(ctx, tenant, ConnectInput{Provider: "stripe", PublicKey: "pk", SecretKey: "s"})
	if err := svc.Disconnect(ctx, tenant, "stripe"); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}
	if _, err := svc.GetActive(ctx, tenant, domain.GatewayStripe); !errors.Is(err, domain.ErrGatewayConnectionNotFound) {
		t.Fatalf("expected no active connection, got %v", err)
	}
	// Disconnecting again is a not-found.
	if err := svc.Disconnect(ctx, tenant, "stripe"); !errors.Is(err, domain.ErrGatewayConnectionNotFound) {
		t.Fatalf("want ErrGatewayConnectionNotFound, got %v", err)
	}
}
