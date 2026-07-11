package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/service"
)

// --- fake sso connection repo (handler package) ---

type memSSOConnectionRepo struct {
	byTenant map[uuid.UUID]*domain.SSOConnection
}

func newMemSSOConnectionRepo() *memSSOConnectionRepo {
	return &memSSOConnectionRepo{byTenant: map[uuid.UUID]*domain.SSOConnection{}}
}
func (r *memSSOConnectionRepo) GetByTenant(_ context.Context, tenantID uuid.UUID) (*domain.SSOConnection, error) {
	if c, ok := r.byTenant[tenantID]; ok {
		cp := *c
		return &cp, nil
	}
	return nil, domain.ErrSSOConnectionNotFound
}
func (r *memSSOConnectionRepo) Upsert(_ context.Context, c *domain.SSOConnection) error {
	cp := *c
	cp.UpdatedAt = time.Now()
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = time.Now()
	}
	r.byTenant[c.TenantID] = &cp
	return nil
}
func (r *memSSOConnectionRepo) Delete(_ context.Context, tenantID uuid.UUID) error {
	if _, ok := r.byTenant[tenantID]; !ok {
		return domain.ErrSSOConnectionNotFound
	}
	delete(r.byTenant, tenantID)
	return nil
}

// memSSOReplayStore is an in-memory port.SSOAssertionReplayStore for tests.
type memSSOReplayStore struct{ consumed map[string]bool }

func newMemSSOReplayStore() *memSSOReplayStore {
	return &memSSOReplayStore{consumed: map[string]bool{}}
}
func (r *memSSOReplayStore) MarkConsumed(_ context.Context, _ uuid.UUID, assertionID string, _ time.Time) error {
	if r.consumed[assertionID] {
		return domain.ErrSSOAssertionReplay
	}
	r.consumed[assertionID] = true
	return nil
}

func newSSOHandler(t *testing.T) (*SSOHandler, *service.SSOService, *memUserRepo, *memSSOConnectionRepo) {
	t.Helper()
	ur := newMemUserRepo()
	cr := newMemSSOConnectionRepo()
	key, cert, err := service.LoadOrGenerateSPKeyPair("", "")
	if err != nil {
		t.Fatalf("keypair: %v", err)
	}
	// AuthService for session issuance on ACS.
	auth := service.NewAuthService(ur, newMemSessionRepo(), &memTenants{tenants: map[uuid.UUID]*domain.Tenant{}}, time.Hour)
	sso := service.NewSSOService(cr, ur, newMemSSOReplayStore(), key, cert, "https://api.example.com")
	h := NewSSOHandler(sso, auth, "http://dash.local", false)
	return h, sso, ur, cr
}

// --- admin config: role gating ---

func TestSSOUpsert_MemberForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _, _, _ := newSSOHandler(t)

	c, w := jsonCtx(http.MethodPut, "/v1/sso/connection", `{"idp_entity_id":"e"}`)
	c.Set("tenant_id", uuid.New())
	c.Set("user_role", "member")
	h.UpsertConnection(c)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 for member", w.Code)
	}
}

func TestSSOUpsert_AdminAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _, _, _ := newSSOHandler(t)

	c, w := jsonCtx(http.MethodPut, "/v1/sso/connection", `{"idp_entity_id":"https://idp/e","idp_sso_url":"https://idp/sso","enabled":false}`)
	c.Set("tenant_id", uuid.New())
	c.Set("user_role", "admin")
	h.UpsertConnection(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s, want 200 for admin", w.Code, w.Body.String())
	}
}

// --- admin config: tenant-scoping / not-found ---

func TestSSOGet_TenantScopedNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _, _, cr := newSSOHandler(t)

	tenantA := uuid.New()
	tenantB := uuid.New()
	// Tenant A has a connection.
	cr.byTenant[tenantA] = &domain.SSOConnection{ID: uuid.New(), TenantID: tenantA, IDPEntityID: "e", Enabled: false}

	// Tenant B asks for its own (nonexistent) connection → 404, and can never see A's.
	c, w := jsonCtx(http.MethodGet, "/v1/sso/connection", "")
	c.Set("tenant_id", tenantB)
	c.Set("user_role", "owner")
	h.GetConnection(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 for tenant with no connection", w.Code)
	}

	// Tenant A sees its own.
	c2, w2 := jsonCtx(http.MethodGet, "/v1/sso/connection", "")
	c2.Set("tenant_id", tenantA)
	c2.Set("user_role", "owner")
	h.GetConnection(c2)
	if w2.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 for tenant A", w2.Code)
	}
}

// --- public SP endpoints ---

func TestSSOMetadata_Renders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _, _, cr := newSSOHandler(t)
	tenantID := uuid.New()
	cr.byTenant[tenantID] = &domain.SSOConnection{ID: uuid.New(), TenantID: tenantID, IDPEntityID: "e", Enabled: false}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/auth/saml/"+tenantID.String()+"/metadata", nil)
	c.Params = gin.Params{{Key: "tenantID", Value: tenantID.String()}}
	h.Metadata(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "EntityDescriptor") {
		t.Fatalf("metadata not rendered: %s", w.Body.String())
	}
}

func TestSSOMetadata_NoConnection404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _, _, _ := newSSOHandler(t)
	tenantID := uuid.New()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/auth/saml/"+tenantID.String()+"/metadata", nil)
	c.Params = gin.Params{{Key: "tenantID", Value: tenantID.String()}}
	h.Metadata(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 with no connection", w.Code)
	}
}

func TestSSOLogin_DisabledTenant404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _, _, cr := newSSOHandler(t)
	tenantID := uuid.New()
	// Configured but disabled.
	cr.byTenant[tenantID] = &domain.SSOConnection{
		ID: uuid.New(), TenantID: tenantID,
		IDPEntityID: "https://idp/e", IDPSSOURL: "https://idp/sso", IDPCertificate: "x", Enabled: false,
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/auth/saml/"+tenantID.String()+"/login", nil)
	c.Params = gin.Params{{Key: "tenantID", Value: tenantID.String()}}
	h.Login(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 for disabled tenant", w.Code)
	}
}

func TestSSOACS_UnknownEmailForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, sso, ur, cr := newSSOHandler(t)
	tenantID := uuid.New()

	// Enabled connection so ACS is reachable (validation will still fail on the
	// fake body, so we assert the mapping path via MapEmailToUser separately).
	cr.byTenant[tenantID] = &domain.SSOConnection{
		ID: uuid.New(), TenantID: tenantID,
		IDPEntityID: "https://idp/e", IDPSSOURL: "https://idp/sso", IDPCertificate: "x", Enabled: true,
	}

	// A validated-but-unknown email maps to 403 (no JIT provisioning).
	if _, err := sso.MapEmailToUser(context.Background(), tenantID, "stranger@corp.com"); err != domain.ErrSSOUserNotFound {
		t.Fatalf("unknown email err = %v, want ErrSSOUserNotFound", err)
	}
	// And a known one succeeds.
	_ = ur.Create(context.Background(), &domain.User{
		ID: uuid.New(), TenantID: tenantID, Email: "known@corp.com", Name: "K", Role: domain.RoleMember,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	if _, err := sso.MapEmailToUser(context.Background(), tenantID, "known@corp.com"); err != nil {
		t.Fatalf("known email: %v", err)
	}
}
