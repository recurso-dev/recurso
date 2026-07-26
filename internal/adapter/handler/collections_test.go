package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// stubQueueLister records the filter it was handed so we can assert the handler
// only forwards whitelisted status/managed_by values.
type stubQueueLister struct {
	gotFilter domain.CollectionsQueueFilter
	items     []domain.CollectionsQueueItem
	count     int
}

func (s *stubQueueLister) ListCollectionsQueue(_ context.Context, _ uuid.UUID, f domain.CollectionsQueueFilter) ([]domain.CollectionsQueueItem, error) {
	s.gotFilter = f
	return s.items, nil
}
func (s *stubQueueLister) CountCollectionsQueue(_ context.Context, _ uuid.UUID, _ domain.CollectionsQueueFilter) (int, error) {
	return s.count, nil
}

// Valid filters pass through; the response carries data + pagination meta.
func TestCollectionsQueue_ValidFiltersAndShape(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &stubQueueLister{
		items: []domain.CollectionsQueueItem{{ID: uuid.New(), Status: "past_due"}},
		count: 1,
	}
	h := NewCollectionsHandler(stub, nil)

	c, w := jsonCtx(http.MethodGet, "/v1/collections/queue?status=past_due&managed_by=worker&per_page=25", "")
	c.Set("tenant_id", uuid.New())
	h.GetQueue(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if stub.gotFilter.Status != "past_due" || stub.gotFilter.ManagedBy != "worker" {
		t.Errorf("filter not forwarded: %+v", stub.gotFilter)
	}
	if stub.gotFilter.Limit != 25 {
		t.Errorf("limit = %d, want 25", stub.gotFilter.Limit)
	}
	var body struct {
		Data []domain.CollectionsQueueItem `json:"data"`
		Meta struct {
			Total   int `json:"total"`
			PerPage int `json:"per_page"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(body.Data) != 1 || body.Meta.Total != 1 || body.Meta.PerPage != 25 {
		t.Errorf("response shape wrong: %+v", body)
	}
}

// Garbage status/managed_by values are dropped, not forwarded to the query.
func TestCollectionsQueue_InvalidFiltersDropped(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &stubQueueLister{}
	h := NewCollectionsHandler(stub, nil)

	c, w := jsonCtx(http.MethodGet, "/v1/collections/queue?status=DROP+TABLE&managed_by=hacker", "")
	c.Set("tenant_id", uuid.New())
	h.GetQueue(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if stub.gotFilter.Status != "" || stub.gotFilter.ManagedBy != "" {
		t.Errorf("invalid filters must be dropped, got %+v", stub.gotFilter)
	}
}

// A caller whose tenant_id isn't a uuid (never happens behind real auth, but
// guards the type assertion) → 401.
func TestCollectionsQueue_RejectsNonUUIDTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &stubQueueLister{}
	h := NewCollectionsHandler(stub, nil)

	c, w := jsonCtx(http.MethodGet, "/v1/collections/queue", "")
	c.Set("tenant_id", "not-a-uuid")
	h.GetQueue(c)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}
