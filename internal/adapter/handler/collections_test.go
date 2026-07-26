package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
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
	h := NewCollectionsHandler(stub, nil, nil)

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
	h := NewCollectionsHandler(stub, nil, nil)

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
	h := NewCollectionsHandler(stub, nil, nil)

	c, w := jsonCtx(http.MethodGet, "/v1/collections/queue", "")
	c.Set("tenant_id", "not-a-uuid")
	h.GetQueue(c)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

// --- Inc 3: manual action endpoints ---

// stubActions returns a scripted error (nil = success) for every action.
type stubActions struct {
	err        error
	lastPaused *bool
}

func (s *stubActions) RetryNow(_ context.Context, _, _ uuid.UUID) error { return s.err }
func (s *stubActions) SetPaused(_ context.Context, _, _ uuid.UUID, paused bool) error {
	s.lastPaused = &paused
	return s.err
}
func (s *stubActions) MarkUncollectible(_ context.Context, _, _ uuid.UUID) error { return s.err }

func TestCollectionsRetryNow_StatusMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)
	id := uuid.New().String()
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"success", nil, http.StatusOK},
		{"mandate", service.ErrRetryMandate, http.StatusConflict},
		{"in flight", service.ErrRetryInFlight, http.StatusConflict},
		{"not past due", service.ErrRetryNotPastDue, http.StatusConflict},
		{"not found", service.ErrCollectionInvoiceNotFound, http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := NewCollectionsHandler(&stubQueueLister{}, nil, &stubActions{err: tc.err})
			c, w := jsonCtx(http.MethodPost, "/v1/collections/invoices/"+id+"/retry-now", "")
			c.Set("tenant_id", uuid.New())
			c.Params = gin.Params{{Key: "id", Value: id}}
			h.RetryNow(c)
			if w.Code != tc.want {
				t.Fatalf("status = %d, want %d", w.Code, tc.want)
			}
		})
	}
}

func TestCollectionsPause_ForwardsFlag(t *testing.T) {
	gin.SetMode(gin.TestMode)
	id := uuid.New().String()
	actions := &stubActions{}
	h := NewCollectionsHandler(&stubQueueLister{}, nil, actions)
	c, w := jsonCtx(http.MethodPost, "/v1/collections/invoices/"+id+"/pause", `{"paused":true}`)
	c.Set("tenant_id", uuid.New())
	c.Params = gin.Params{{Key: "id", Value: id}}
	h.PauseDunning(c)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if actions.lastPaused == nil || *actions.lastPaused != true {
		t.Error("paused=true must reach the service")
	}
}

func TestCollectionsActions_InvalidInvoiceID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewCollectionsHandler(&stubQueueLister{}, nil, &stubActions{})
	c, w := jsonCtx(http.MethodPost, "/v1/collections/invoices/not-a-uuid/retry-now", "")
	c.Set("tenant_id", uuid.New())
	c.Params = gin.Params{{Key: "id", Value: "not-a-uuid"}}
	h.RetryNow(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestCollectionsActions_NotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	id := uuid.New().String()
	h := NewCollectionsHandler(&stubQueueLister{}, nil, nil) // no actions wired
	c, w := jsonCtx(http.MethodPost, "/v1/collections/invoices/"+id+"/mark-uncollectible", "")
	c.Set("tenant_id", uuid.New())
	c.Params = gin.Params{{Key: "id", Value: id}}
	h.MarkUncollectible(c)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
}
