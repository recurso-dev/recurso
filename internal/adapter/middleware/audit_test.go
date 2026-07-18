package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

type fakeAuditRepo struct {
	mu      sync.Mutex
	entries []*domain.AuditLog
}

func (f *fakeAuditRepo) Insert(ctx context.Context, a *domain.AuditLog) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entries = append(f.entries, a)
	return nil
}

func (f *fakeAuditRepo) List(ctx context.Context, tenantID uuid.UUID, filter domain.AuditLogFilter) ([]domain.AuditLog, error) {
	return nil, nil
}

func (f *fakeAuditRepo) wait(t *testing.T, want int) []*domain.AuditLog {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		f.mu.Lock()
		n := len(f.entries)
		entries := append([]*domain.AuditLog(nil), f.entries...)
		f.mu.Unlock()
		if n >= want {
			return entries
		}
		time.Sleep(5 * time.Millisecond)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*domain.AuditLog(nil), f.entries...)
}

func auditTestRouter(repo *fakeAuditRepo, tenantID uuid.UUID, userID *uuid.UUID, handlerStatus int) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenantID)
		if userID != nil {
			c.Set("user_id", *userID)
		}
	})
	r.Use(Audit(repo))
	h := func(c *gin.Context) { c.JSON(handlerStatus, gin.H{"ok": true}) }
	r.PUT("/v1/plans/:id/charges", h)
	r.POST("/v1/wallets", h)
	r.POST("/v1/usage/events", h)            // high-volume: never audited
	r.GET("/v1/plans/:id/charges", h)        // read: never audited
	r.POST("/v1/subscriptions/:id/pause", h) // not allowlisted
	return r
}

func TestAuditRecordsConfigMutations(t *testing.T) {
	repo := &fakeAuditRepo{}
	tenantID := uuid.New()
	userID := uuid.New()
	r := auditTestRouter(repo, tenantID, &userID, http.StatusOK)

	req := httptest.NewRequest("PUT", "/v1/plans/plan_1/charges", strings.NewReader(`[{"metric_id":"m"}]`))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	entries := repo.wait(t, 1)
	if len(entries) != 1 {
		t.Fatalf("audit entries = %d, want 1", len(entries))
	}
	e := entries[0]
	if e.TenantID != tenantID || e.Actor != userID.String() {
		t.Fatalf("entry actor/tenant = %q/%s, want the session user", e.Actor, e.TenantID)
	}
	if e.Action != "PUT /v1/plans/:id/charges" || e.EntityType != "plans" || e.EntityID != "plan_1" {
		t.Fatalf("entry = %+v, want route template + entity", e)
	}
	if !strings.Contains(e.RequestBody, "metric_id") {
		t.Fatalf("request body not captured: %q", e.RequestBody)
	}
}

func TestAuditActorFallsBackToAPIKey(t *testing.T) {
	repo := &fakeAuditRepo{}
	r := auditTestRouter(repo, uuid.New(), nil, http.StatusCreated)

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/v1/wallets", strings.NewReader(`{}`)))
	entries := repo.wait(t, 1)
	if len(entries) != 1 || entries[0].Actor != "api_key" {
		t.Fatalf("entries = %+v, want one api_key entry", entries)
	}
}

func TestAuditSkipsReadsHighVolumeAndFailures(t *testing.T) {
	repo := &fakeAuditRepo{}
	r := auditTestRouter(repo, uuid.New(), nil, http.StatusOK)

	// Reads, non-allowlisted resources, and high-volume ingest never audit.
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/v1/plans/p/charges", nil))
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/v1/usage/events", strings.NewReader(`{}`)))
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/v1/subscriptions/s/pause", nil))

	// Failed mutations never audit either.
	failing := auditTestRouter(repo, uuid.New(), nil, http.StatusBadRequest)
	failing.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/v1/wallets", strings.NewReader(`{}`)))

	time.Sleep(50 * time.Millisecond)
	if entries := repo.wait(t, 0); len(entries) != 0 {
		t.Fatalf("audit entries = %+v, want none", entries)
	}
}

func TestAuditBodyTruncated(t *testing.T) {
	repo := &fakeAuditRepo{}
	r := auditTestRouter(repo, uuid.New(), nil, http.StatusOK)

	big := strings.Repeat("x", maxAuditBodyLen*2)
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/v1/wallets", strings.NewReader(big)))
	entries := repo.wait(t, 1)
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	if len(entries[0].RequestBody) > maxAuditBodyLen+3 || !strings.HasSuffix(entries[0].RequestBody, "...") {
		t.Fatalf("body len = %d, want truncated with ellipsis", len(entries[0].RequestBody))
	}
}
