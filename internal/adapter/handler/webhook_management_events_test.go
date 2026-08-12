package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

// objectScopedEventService proves ListEvents routes object_id to the
// object-scoped read.
type objectScopedEventRepo struct {
	tenantID uuid.UUID
	objectID uuid.UUID
	rows     []*domain.Event
}

func (r *objectScopedEventRepo) Create(context.Context, *domain.Event) error { return nil }
func (r *objectScopedEventRepo) GetByID(context.Context, uuid.UUID) (*domain.Event, error) {
	return nil, nil
}
func (r *objectScopedEventRepo) ListByTenantID(context.Context, uuid.UUID, string, int, int) ([]*domain.Event, error) {
	return nil, nil
}
func (r *objectScopedEventRepo) ListByObject(_ context.Context, tenantID, objectID uuid.UUID, _, _ int) ([]*domain.Event, error) {
	if tenantID == r.tenantID && objectID == r.objectID {
		return r.rows, nil
	}
	return nil, nil
}

func TestListEventsObjectFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID, objectID := uuid.New(), uuid.New()

	do := func(query string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/v1/events"+query, nil)
		c.Set("tenant_id", tenantID)
		h := NewWebhookManagementHandler(service.NewWebhookService(nil, &objectScopedEventRepo{
			tenantID: tenantID,
			objectID: objectID,
			rows:     []*domain.Event{{ID: uuid.New(), TenantID: tenantID, Type: "invoice.paid", ObjectID: objectID}},
		}, nil))
		h.ListEvents(c)
		return w
	}

	if got := do("?object_id=" + objectID.String()).Code; got != http.StatusOK {
		t.Fatalf("object-scoped list: got %d want 200", got)
	}
	if got := do("?object_id=nope").Code; got != http.StatusBadRequest {
		t.Fatalf("bad object_id: got %d want 400", got)
	}
}
