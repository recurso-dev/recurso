package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// fakeConnRepo is an in-memory AccountingConnectionRepository holding at most
// one connection per (tenant, provider) — all ConnectTokenBased needs.
type fakeConnRepo struct {
	conns map[string]*domain.AccountingConnection
}

func newFakeConnRepo() *fakeConnRepo {
	return &fakeConnRepo{conns: map[string]*domain.AccountingConnection{}}
}

func (f *fakeConnRepo) key(tenantID uuid.UUID, provider string) string {
	return tenantID.String() + "/" + provider
}

func (f *fakeConnRepo) Create(_ context.Context, conn *domain.AccountingConnection) error {
	f.conns[f.key(conn.TenantID, conn.Provider)] = conn
	return nil
}

func (f *fakeConnRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.AccountingConnection, error) {
	for _, c := range f.conns {
		if c.ID == id {
			return c, nil
		}
	}
	return nil, errors.New("not found")
}

func (f *fakeConnRepo) GetByTenantAndProvider(_ context.Context, tenantID uuid.UUID, provider string) (*domain.AccountingConnection, error) {
	if c, ok := f.conns[f.key(tenantID, provider)]; ok {
		return c, nil
	}
	return nil, errors.New("not found")
}

func (f *fakeConnRepo) ListByTenant(_ context.Context, tenantID uuid.UUID) ([]*domain.AccountingConnection, error) {
	var out []*domain.AccountingConnection
	for _, c := range f.conns {
		if c.TenantID == tenantID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (f *fakeConnRepo) Update(_ context.Context, conn *domain.AccountingConnection) error {
	f.conns[f.key(conn.TenantID, conn.Provider)] = conn
	return nil
}

func (f *fakeConnRepo) Delete(_ context.Context, id uuid.UUID) error {
	for k, c := range f.conns {
		if c.ID == id {
			delete(f.conns, k)
		}
	}
	return nil
}

func (f *fakeConnRepo) GetActiveConnections(context.Context) ([]*domain.AccountingConnection, error) {
	return nil, nil
}
func (f *fakeConnRepo) CreateSyncLog(context.Context, *domain.AccountingSyncLog) error { return nil }
func (f *fakeConnRepo) ListSyncLogs(context.Context, uuid.UUID, int) ([]*domain.AccountingSyncLog, error) {
	return nil, nil
}

func newConnectTokenRouter(repo *fakeConnRepo, tenantID uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewAccountingHandler(repo, nil, []byte("test-secret"), "https://app.example.com")
	r := gin.New()
	r.POST("/v1/accounting/connect-token/:provider", func(c *gin.Context) {
		c.Set("tenant_id", tenantID)
		h.ConnectTokenBased(c)
	})
	return r
}

// TestConnectTokenBased covers the token-based connection flow for the two
// non-OAuth providers: NetSuite (account id + SuiteTalk token required),
// Tally (no credentials), validation failures, reconnect-updates-in-place,
// and rejection of OAuth-only providers.
func TestConnectTokenBased(t *testing.T) {
	tenantID := uuid.New()

	post := func(r *gin.Engine, path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		var reader *strings.Reader
		if body == "" {
			reader = strings.NewReader("")
		} else {
			reader = strings.NewReader(body)
		}
		req := httptest.NewRequest(http.MethodPost, path, reader)
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(rec, req)
		return rec
	}

	t.Run("netsuite requires credentials", func(t *testing.T) {
		r := newConnectTokenRouter(newFakeConnRepo(), tenantID)
		rec := post(r, "/v1/accounting/connect-token/netsuite", `{"account_id":"ACME123"}`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400 (body: %s)", rec.Code, rec.Body.String())
		}
	})

	t.Run("netsuite creates a connection", func(t *testing.T) {
		repo := newFakeConnRepo()
		r := newConnectTokenRouter(repo, tenantID)
		rec := post(r, "/v1/accounting/connect-token/netsuite",
			`{"account_id":"ACME123","access_token":"tok_ns"}`)
		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201 (body: %s)", rec.Code, rec.Body.String())
		}
		conn, err := repo.GetByTenantAndProvider(context.Background(), tenantID, "netsuite")
		if err != nil {
			t.Fatalf("connection not stored: %v", err)
		}
		if conn.RealmID != "ACME123" || conn.AccessToken != "tok_ns" || !conn.IsActive {
			t.Fatalf("stored connection wrong: %+v", conn)
		}
	})

	t.Run("netsuite reconnect updates in place", func(t *testing.T) {
		repo := newFakeConnRepo()
		existing := &domain.AccountingConnection{
			ID: uuid.New(), TenantID: tenantID, Provider: "netsuite",
			RealmID: "ACME123", AccessToken: "tok_old", IsActive: false, LastError: "invalid_grant",
		}
		_ = repo.Create(context.Background(), existing)
		r := newConnectTokenRouter(repo, tenantID)
		rec := post(r, "/v1/accounting/connect-token/netsuite",
			`{"account_id":"ACME123","access_token":"tok_new"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
		}
		conn, _ := repo.GetByTenantAndProvider(context.Background(), tenantID, "netsuite")
		if conn.ID != existing.ID {
			t.Fatalf("reconnect created a duplicate connection")
		}
		if conn.AccessToken != "tok_new" || !conn.IsActive || conn.LastError != "" {
			t.Fatalf("reconnect did not refresh the connection: %+v", conn)
		}
	})

	t.Run("tally connects with no credentials", func(t *testing.T) {
		repo := newFakeConnRepo()
		r := newConnectTokenRouter(repo, tenantID)
		rec := post(r, "/v1/accounting/connect-token/tally", "")
		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201 (body: %s)", rec.Code, rec.Body.String())
		}
		conn, err := repo.GetByTenantAndProvider(context.Background(), tenantID, "tally")
		if err != nil || !conn.IsActive {
			t.Fatalf("tally connection not stored/active: %v %+v", err, conn)
		}
	})

	t.Run("oauth-only provider rejected", func(t *testing.T) {
		r := newConnectTokenRouter(newFakeConnRepo(), tenantID)
		rec := post(r, "/v1/accounting/connect-token/quickbooks", "{}")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400 (body: %s)", rec.Code, rec.Body.String())
		}
	})
}
