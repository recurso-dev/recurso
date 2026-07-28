package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/recurso-dev/recurso/internal/service"
)

// oneSubRepo is a minimal SubscriptionRepository holding a single subscription.
// Only GetByID is exercised; the rest satisfy the interface. GetByID is
// cross-tenant at the repo layer (it returns the row regardless of tenant) —
// the SERVICE is responsible for the tenant check, which is exactly what this
// test guards.
type oneSubRepo struct {
	port.SubscriptionRepository
	sub *domain.Subscription
}

func (r *oneSubRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Subscription, error) {
	if r.sub != nil && r.sub.ID == id {
		return r.sub, nil
	}
	return nil, nil
}

func newSubGetHandler(sub *domain.Subscription) *SubscriptionHandler {
	svc := service.NewSubscriptionService(&oneSubRepo{sub: sub}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	return NewSubscriptionHandler(svc)
}

func doGetSubscription(h *SubscriptionHandler, tenantID uuid.UUID, id string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/subscriptions/"+id, nil)
	c.Params = gin.Params{{Key: "id", Value: id}}
	c.Set("tenant_id", tenantID)
	h.GetSubscription(c)
	return w
}

func TestGetSubscriptionReturnsOwnedRow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID := uuid.New()
	sub := &domain.Subscription{ID: uuid.New(), TenantID: tenantID, Status: "active", CreatedAt: time.Now()}
	h := newSubGetHandler(sub)

	if got := doGetSubscription(h, tenantID, sub.ID.String()).Code; got != http.StatusOK {
		t.Fatalf("owned subscription: got %d want 200", got)
	}
}

func TestGetSubscriptionCrossTenantIs404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	owner := uuid.New()
	sub := &domain.Subscription{ID: uuid.New(), TenantID: owner, Status: "active"}
	h := newSubGetHandler(sub)

	// A different tenant asking for someone else's subscription must get a
	// flat 404 — never 200 (IDOR) and never a 403 that confirms existence.
	other := uuid.New()
	if got := doGetSubscription(h, other, sub.ID.String()).Code; got != http.StatusNotFound {
		t.Fatalf("cross-tenant read: got %d want 404", got)
	}
}

func TestGetSubscriptionUnknownIs404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID := uuid.New()
	h := newSubGetHandler(&domain.Subscription{ID: uuid.New(), TenantID: tenantID})
	if got := doGetSubscription(h, tenantID, uuid.New().String()).Code; got != http.StatusNotFound {
		t.Fatalf("unknown id: got %d want 404", got)
	}
}

func TestGetSubscriptionBadIdIs400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newSubGetHandler(nil)
	if got := doGetSubscription(h, uuid.New(), "not-a-uuid").Code; got != http.StatusBadRequest {
		t.Fatalf("bad id: got %d want 400", got)
	}
}
