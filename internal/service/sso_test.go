package service

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/crewjam/saml"
	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// --- fake sso connection repo ---

type fakeSSOConnectionRepo struct {
	byTenant map[uuid.UUID]*domain.SSOConnection
}

func newFakeSSOConnectionRepo() *fakeSSOConnectionRepo {
	return &fakeSSOConnectionRepo{byTenant: map[uuid.UUID]*domain.SSOConnection{}}
}

func (r *fakeSSOConnectionRepo) GetByTenant(_ context.Context, tenantID uuid.UUID) (*domain.SSOConnection, error) {
	if c, ok := r.byTenant[tenantID]; ok {
		cp := *c
		return &cp, nil
	}
	return nil, domain.ErrSSOConnectionNotFound
}

func (r *fakeSSOConnectionRepo) Upsert(_ context.Context, c *domain.SSOConnection) error {
	cp := *c
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = time.Now()
	}
	cp.UpdatedAt = time.Now()
	r.byTenant[c.TenantID] = &cp
	return nil
}

func (r *fakeSSOConnectionRepo) Delete(_ context.Context, tenantID uuid.UUID) error {
	if _, ok := r.byTenant[tenantID]; !ok {
		return domain.ErrSSOConnectionNotFound
	}
	delete(r.byTenant, tenantID)
	return nil
}

// fakeSSOReplayStore is an in-memory port.SSOAssertionReplayStore for tests.
type fakeSSOReplayStore struct {
	consumed map[string]bool
}

func newFakeSSOReplayStore() *fakeSSOReplayStore {
	return &fakeSSOReplayStore{consumed: map[string]bool{}}
}

func (r *fakeSSOReplayStore) MarkConsumed(_ context.Context, _ uuid.UUID, assertionID string, _ time.Time) error {
	if r.consumed[assertionID] {
		return domain.ErrSSOAssertionReplay
	}
	r.consumed[assertionID] = true
	return nil
}

func newSSOTestService(t *testing.T) (*SSOService, *fakeUserRepo, *fakeSSOConnectionRepo) {
	t.Helper()
	ur := newFakeUserRepo()
	cr := newFakeSSOConnectionRepo()
	key, cert, err := LoadOrGenerateSPKeyPair("", "")
	if err != nil {
		t.Fatalf("keypair: %v", err)
	}
	svc := NewSSOService(cr, ur, newFakeSSOReplayStore(), key, cert, "https://api.example.com")
	return svc, ur, cr
}

// idpCertB64 returns a base64 DER cert usable as an IdP signing cert in tests.
func idpCertB64(t *testing.T) string {
	t.Helper()
	_, cert, err := LoadOrGenerateSPKeyPair("", "")
	if err != nil {
		t.Fatalf("cert: %v", err)
	}
	return base64.StdEncoding.EncodeToString(cert.Raw)
}

func TestSSO_UpsertRequiresConfigToEnable(t *testing.T) {
	svc, _, _ := newSSOTestService(t)
	tenantID := uuid.New()

	// Enabling with nothing configured is rejected.
	if _, err := svc.UpsertConnection(context.Background(), tenantID, UpsertConnectionInput{Enabled: true}); err == nil {
		t.Fatal("expected error enabling an unconfigured connection")
	}

	// Disabled upsert with partial config is fine.
	conn, err := svc.UpsertConnection(context.Background(), tenantID, UpsertConnectionInput{
		IDPEntityID: "https://idp.example.com/entity",
		Enabled:     false,
	})
	if err != nil {
		t.Fatalf("upsert disabled: %v", err)
	}
	if conn.Enabled {
		t.Fatal("connection should be disabled")
	}

	// Fully configured + enabled succeeds.
	conn, err = svc.UpsertConnection(context.Background(), tenantID, UpsertConnectionInput{
		IDPEntityID:    "https://idp.example.com/entity",
		IDPSSOURL:      "https://idp.example.com/sso",
		IDPCertificate: idpCertB64(t),
		Enabled:        true,
	})
	if err != nil {
		t.Fatalf("upsert enabled: %v", err)
	}
	if !conn.Enabled || !conn.Configured() {
		t.Fatal("expected enabled + configured")
	}
}

func TestSSO_MapEmailToUser_KnownAndUnknown(t *testing.T) {
	svc, ur, _ := newSSOTestService(t)
	tenantID := uuid.New()
	otherTenant := uuid.New()

	_ = ur.Create(context.Background(), &domain.User{
		ID: uuid.New(), TenantID: tenantID, Email: "sso-user@corp.com",
		Name: "SSO User", Role: domain.RoleMember, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	// Known email in the tenant → mapped.
	u, err := svc.MapEmailToUser(context.Background(), tenantID, "SSO-User@Corp.com")
	if err != nil {
		t.Fatalf("known email: %v", err)
	}
	if u.Email != "sso-user@corp.com" {
		t.Fatalf("wrong user: %+v", u)
	}

	// Unknown email → 403 sentinel.
	if _, err := svc.MapEmailToUser(context.Background(), tenantID, "nobody@corp.com"); err != domain.ErrSSOUserNotFound {
		t.Fatalf("unknown email err = %v, want ErrSSOUserNotFound", err)
	}

	// Known email but WRONG tenant → not found (tenant isolation).
	if _, err := svc.MapEmailToUser(context.Background(), otherTenant, "sso-user@corp.com"); err != domain.ErrSSOUserNotFound {
		t.Fatalf("cross-tenant err = %v, want ErrSSOUserNotFound", err)
	}
}

func TestSSO_MetadataRenders(t *testing.T) {
	svc, _, _ := newSSOTestService(t)
	tenantID := uuid.New()

	// No connection → metadata 404 (ErrSSOConnectionNotFound).
	if _, err := svc.Metadata(context.Background(), tenantID); err == nil {
		t.Fatal("expected error rendering metadata with no connection")
	}

	if _, err := svc.UpsertConnection(context.Background(), tenantID, UpsertConnectionInput{
		IDPEntityID: "https://idp.example.com/entity", Enabled: false,
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	xmlBytes, err := svc.Metadata(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("metadata: %v", err)
	}
	xml := string(xmlBytes)
	if !strings.Contains(xml, "EntityDescriptor") || !strings.Contains(xml, "SPSSODescriptor") {
		t.Fatalf("metadata missing SP descriptor: %s", xml)
	}
	if !strings.Contains(xml, tenantID.String()) {
		t.Fatalf("metadata should embed the tenant ACS/metadata URLs: %s", xml)
	}
}

func TestSSO_LoginAndACSGatedWhenDisabled(t *testing.T) {
	svc, _, _ := newSSOTestService(t)
	tenantID := uuid.New()

	// Unconfigured tenant → not enabled.
	if _, err := svc.LoginRedirectURL(context.Background(), tenantID); err != domain.ErrSSONotEnabled {
		t.Fatalf("login (no conn) err = %v, want ErrSSONotEnabled", err)
	}

	// Configured but disabled → still not enabled.
	if _, err := svc.UpsertConnection(context.Background(), tenantID, UpsertConnectionInput{
		IDPEntityID:    "https://idp.example.com/entity",
		IDPSSOURL:      "https://idp.example.com/sso",
		IDPCertificate: idpCertB64(t),
		Enabled:        false,
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if _, err := svc.LoginRedirectURL(context.Background(), tenantID); err != domain.ErrSSONotEnabled {
		t.Fatalf("login (disabled) err = %v, want ErrSSONotEnabled", err)
	}
}

func TestSSO_LoginRedirectWhenEnabled(t *testing.T) {
	svc, _, _ := newSSOTestService(t)
	tenantID := uuid.New()
	if _, err := svc.UpsertConnection(context.Background(), tenantID, UpsertConnectionInput{
		IDPEntityID:    "https://idp.example.com/entity",
		IDPSSOURL:      "https://idp.example.com/sso",
		IDPCertificate: idpCertB64(t),
		Enabled:        true,
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	redirectURL, err := svc.LoginRedirectURL(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("login redirect: %v", err)
	}
	if !strings.HasPrefix(redirectURL, "https://idp.example.com/sso") {
		t.Fatalf("redirect should target the IdP SSO URL, got %s", redirectURL)
	}
	if !strings.Contains(redirectURL, "SAMLRequest=") {
		t.Fatalf("redirect missing SAMLRequest: %s", redirectURL)
	}
}

func TestSSO_DeleteConnection(t *testing.T) {
	svc, _, _ := newSSOTestService(t)
	tenantID := uuid.New()
	if err := svc.DeleteConnection(context.Background(), tenantID); err != domain.ErrSSOConnectionNotFound {
		t.Fatalf("delete missing err = %v, want ErrSSOConnectionNotFound", err)
	}
	_, _ = svc.UpsertConnection(context.Background(), tenantID, UpsertConnectionInput{IDPEntityID: "e"})
	if err := svc.DeleteConnection(context.Background(), tenantID); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

// TestSSO_AssertionReplayRejected covers the replay guard directly (past the
// signature round-trip that ProcessACS does): the first consume of an assertion
// ID succeeds, a second consume of the same ID is rejected as an invalid
// assertion, and a fresh ID is accepted again.
func TestSSO_AssertionReplayRejected(t *testing.T) {
	svc, _, _ := newSSOTestService(t)
	ctx := context.Background()
	tenantID := uuid.New()

	a1 := &saml.Assertion{ID: "_assertion-abc", Conditions: &saml.Conditions{NotOnOrAfter: time.Now().Add(5 * time.Minute)}}
	if err := svc.consumeAssertion(ctx, tenantID, a1); err != nil {
		t.Fatalf("first consume: %v", err)
	}
	if err := svc.consumeAssertion(ctx, tenantID, a1); !errors.Is(err, domain.ErrSSOInvalidAssertion) {
		t.Fatalf("replay consume err = %v, want wraps ErrSSOInvalidAssertion", err)
	}

	// A different assertion ID is still accepted.
	a2 := &saml.Assertion{ID: "_assertion-def"}
	if err := svc.consumeAssertion(ctx, tenantID, a2); err != nil {
		t.Fatalf("second (distinct) consume: %v", err)
	}

	// An assertion with no ID has nothing to key replay protection on → rejected.
	if err := svc.consumeAssertion(ctx, tenantID, &saml.Assertion{}); !errors.Is(err, domain.ErrSSOInvalidAssertion) {
		t.Fatalf("empty-ID consume err = %v, want wraps ErrSSOInvalidAssertion", err)
	}
}
