package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/recurso-dev/recurso/internal/adapter/memory"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// newTestRouter builds a router with the idempotency middleware and a
// handler that increments calls and echoes a unique body per invocation.
// Tenant is taken from the X-Test-Tenant header when present.
func newTestRouter(store port.IdempotencyStore, calls *atomic.Int64, status int) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if t := c.GetHeader("X-Test-Tenant"); t != "" {
			c.Set("tenant_id", t)
		}
		c.Next()
	})
	r.Use(IdempotencyMiddleware(store))
	handler := func(c *gin.Context) {
		n := calls.Add(1)
		c.JSON(status, gin.H{"call": n})
	}
	r.POST("/v1/subscriptions", handler)
	r.POST("/v1/credit-notes", handler)
	r.GET("/v1/subscriptions", handler)
	return r
}

func doRequest(r *gin.Engine, method, path, key, tenant string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	if tenant != "" {
		req.Header.Set("X-Test-Tenant", tenant)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestIdempotency_ReplaySameKey(t *testing.T) {
	var calls atomic.Int64
	r := newTestRouter(memory.NewInMemoryIdempotencyStore(time.Minute), &calls, http.StatusCreated)

	first := doRequest(r, http.MethodPost, "/v1/subscriptions", "key-1", "t1")
	second := doRequest(r, http.MethodPost, "/v1/subscriptions", "key-1", "t1")

	if calls.Load() != 1 {
		t.Fatalf("handler called %d times, want 1", calls.Load())
	}
	if first.Code != http.StatusCreated || second.Code != http.StatusCreated {
		t.Fatalf("status codes: first=%d second=%d, want both %d", first.Code, second.Code, http.StatusCreated)
	}
	if first.Body.String() != second.Body.String() {
		t.Fatalf("replayed body %q differs from original %q", second.Body.String(), first.Body.String())
	}
	if second.Header().Get("X-Idempotency-Hit") != "true" {
		t.Fatal("expected X-Idempotency-Hit=true on replayed response")
	}
	if first.Header().Get("X-Idempotency-Hit") == "true" {
		t.Fatal("first response must not be marked as a replay")
	}
}

func TestIdempotency_DifferentKeysProcessIndependently(t *testing.T) {
	var calls atomic.Int64
	r := newTestRouter(memory.NewInMemoryIdempotencyStore(time.Minute), &calls, http.StatusOK)

	first := doRequest(r, http.MethodPost, "/v1/subscriptions", "key-a", "t1")
	second := doRequest(r, http.MethodPost, "/v1/subscriptions", "key-b", "t1")

	if calls.Load() != 2 {
		t.Fatalf("handler called %d times, want 2", calls.Load())
	}
	if first.Body.String() == second.Body.String() {
		t.Fatal("different keys must not share responses")
	}
}

func TestIdempotency_NoHeaderPassesThrough(t *testing.T) {
	var calls atomic.Int64
	r := newTestRouter(memory.NewInMemoryIdempotencyStore(time.Minute), &calls, http.StatusOK)

	doRequest(r, http.MethodPost, "/v1/subscriptions", "", "t1")
	doRequest(r, http.MethodPost, "/v1/subscriptions", "", "t1")

	if calls.Load() != 2 {
		t.Fatalf("handler called %d times, want 2 (no header means no replay)", calls.Load())
	}
}

func TestIdempotency_SameKeyDifferentEndpointsIndependent(t *testing.T) {
	var calls atomic.Int64
	r := newTestRouter(memory.NewInMemoryIdempotencyStore(time.Minute), &calls, http.StatusOK)

	doRequest(r, http.MethodPost, "/v1/subscriptions", "shared-key", "t1")
	second := doRequest(r, http.MethodPost, "/v1/credit-notes", "shared-key", "t1")

	if calls.Load() != 2 {
		t.Fatalf("handler called %d times, want 2 (keys are scoped per path)", calls.Load())
	}
	if second.Header().Get("X-Idempotency-Hit") == "true" {
		t.Fatal("same key on a different endpoint must not replay")
	}
}

func TestIdempotency_SameKeyDifferentTenantsIndependent(t *testing.T) {
	var calls atomic.Int64
	r := newTestRouter(memory.NewInMemoryIdempotencyStore(time.Minute), &calls, http.StatusOK)

	doRequest(r, http.MethodPost, "/v1/subscriptions", "shared-key", "tenant-a")
	doRequest(r, http.MethodPost, "/v1/subscriptions", "shared-key", "tenant-b")

	if calls.Load() != 2 {
		t.Fatalf("handler called %d times, want 2 (keys are scoped per tenant)", calls.Load())
	}
}

func TestIdempotency_GetRequestsNotReplayed(t *testing.T) {
	var calls atomic.Int64
	r := newTestRouter(memory.NewInMemoryIdempotencyStore(time.Minute), &calls, http.StatusOK)

	doRequest(r, http.MethodGet, "/v1/subscriptions", "key-1", "t1")
	doRequest(r, http.MethodGet, "/v1/subscriptions", "key-1", "t1")

	if calls.Load() != 2 {
		t.Fatalf("handler called %d times, want 2 (GET is not idempotency-cached)", calls.Load())
	}
}

func TestIdempotency_ServerErrorsNotStored(t *testing.T) {
	var calls atomic.Int64
	r := newTestRouter(memory.NewInMemoryIdempotencyStore(time.Minute), &calls, http.StatusInternalServerError)

	doRequest(r, http.MethodPost, "/v1/subscriptions", "key-500", "t1")
	second := doRequest(r, http.MethodPost, "/v1/subscriptions", "key-500", "t1")

	if calls.Load() != 2 {
		t.Fatalf("handler called %d times, want 2 (5xx must stay retryable)", calls.Load())
	}
	if second.Header().Get("X-Idempotency-Hit") == "true" {
		t.Fatal("5xx responses must not be replayed")
	}
}

func TestIdempotency_TTLRespected(t *testing.T) {
	var calls atomic.Int64
	r := newTestRouter(memory.NewInMemoryIdempotencyStore(30*time.Millisecond), &calls, http.StatusOK)

	doRequest(r, http.MethodPost, "/v1/subscriptions", "key-ttl", "t1")

	// Within TTL: replayed.
	within := doRequest(r, http.MethodPost, "/v1/subscriptions", "key-ttl", "t1")
	if within.Header().Get("X-Idempotency-Hit") != "true" {
		t.Fatal("expected replay within TTL")
	}

	time.Sleep(50 * time.Millisecond)

	// After TTL: processed again.
	after := doRequest(r, http.MethodPost, "/v1/subscriptions", "key-ttl", "t1")
	if after.Header().Get("X-Idempotency-Hit") == "true" {
		t.Fatal("expired key must not replay")
	}
	if calls.Load() != 2 {
		t.Fatalf("handler called %d times, want 2 (once before, once after expiry)", calls.Load())
	}
}

// failingStore simulates an unavailable backing store (e.g. Redis down
// without the in-memory fallback wired).
type failingStore struct{}

func (failingStore) Get(context.Context, string) (*domain.StoredResponse, error) {
	return nil, errors.New("store unavailable")
}

func (failingStore) Set(context.Context, string, *domain.StoredResponse) error {
	return errors.New("store unavailable")
}

func TestIdempotency_StoreFailureDegradesGracefully(t *testing.T) {
	var calls atomic.Int64
	r := newTestRouter(failingStore{}, &calls, http.StatusOK)

	first := doRequest(r, http.MethodPost, "/v1/subscriptions", "key-1", "t1")
	second := doRequest(r, http.MethodPost, "/v1/subscriptions", "key-1", "t1")

	if first.Code != http.StatusOK || second.Code != http.StatusOK {
		t.Fatalf("requests must succeed when store is down: %d, %d", first.Code, second.Code)
	}
	if calls.Load() != 2 {
		t.Fatalf("handler called %d times, want 2 (idempotency disabled on store failure)", calls.Load())
	}
}

func TestIdempotency_ReplayPreservesCustomHeaders(t *testing.T) {
	store := memory.NewInMemoryIdempotencyStore(time.Minute)
	var calls atomic.Int64
	r := gin.New()
	r.Use(IdempotencyMiddleware(store))
	r.POST("/v1/gifts/purchase", func(c *gin.Context) {
		calls.Add(1)
		c.Header("X-Gift-Code", "ABC123")
		c.JSON(http.StatusCreated, gin.H{"ok": true})
	})

	req := func() *httptest.ResponseRecorder {
		rq := httptest.NewRequest(http.MethodPost, "/v1/gifts/purchase", nil)
		rq.Header.Set("Idempotency-Key", fmt.Sprintf("key-%s", "hdr"))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, rq)
		return w
	}

	req()
	second := req()

	if calls.Load() != 1 {
		t.Fatalf("handler called %d times, want 1", calls.Load())
	}
	if second.Header().Get("X-Gift-Code") != "ABC123" {
		t.Fatal("replayed response must preserve stored headers")
	}
}
