package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// TestRateLimitMiddleware_BlocksPastLimit proves the fixed-window limiter (used
// for the "expensive" bucket on import/PDF/GL-export routes) admits exactly
// `limit` requests in the window and 429s the next one — so one caller can't
// hammer the CPU/IO-heavy endpoints. Redis is nil here, exercising the
// in-memory fallback path.
func TestRateLimitMiddleware_BlocksPastLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimitMiddleware(nil, "test-expensive", 3, time.Minute))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	do := func() int {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.RemoteAddr = "203.0.113.7:1234"
		r.ServeHTTP(w, req)
		return w.Code
	}

	for i := 1; i <= 3; i++ {
		if got := do(); got != http.StatusOK {
			t.Fatalf("request %d: got %d, want 200 (within limit)", i, got)
		}
	}
	if got := do(); got != http.StatusTooManyRequests {
		t.Fatalf("4th request: got %d, want 429 (past limit)", got)
	}
}

// TestRateLimitMiddleware_SeparateScopesSeparateBuckets guards the bug the
// scope parameter exists to prevent: two limiters with different scopes must
// not share a counter (a shared key once made the strictest limiter judge the
// combined total and 429 normal traffic).
func TestRateLimitMiddleware_SeparateScopesSeparateBuckets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/a", RateLimitMiddleware(nil, "scope-a", 1, time.Minute), func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/b", RateLimitMiddleware(nil, "scope-b", 1, time.Minute), func(c *gin.Context) { c.Status(http.StatusOK) })

	call := func(path string) int {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.RemoteAddr = "203.0.113.8:1234"
		r.ServeHTTP(w, req)
		return w.Code
	}

	if got := call("/a"); got != http.StatusOK {
		t.Fatalf("first /a: got %d, want 200", got)
	}
	// A different scope must have its own fresh bucket, not inherit /a's count.
	if got := call("/b"); got != http.StatusOK {
		t.Fatalf("first /b (separate scope): got %d, want 200 — scopes share a bucket", got)
	}
}

// TestRateLimitMiddleware_RedisErrorFallsBackToLocal guards the fail-closed
// behaviour: when Redis is configured but unreachable, the limiter must still
// enforce the window from its in-memory fallback rather than admit every
// request — this middleware is the brute-force ceiling on the public auth
// endpoints, and a Redis blip must not lift it.
func TestRateLimitMiddleware_RedisErrorFallsBackToLocal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// A client pointed at a closed port errors on every command.
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: 200 * time.Millisecond, MaxRetries: -1})
	r := gin.New()
	r.Use(RateLimitMiddleware(rdb, "test-redis-down", 2, time.Minute))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	do := func() int {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.RemoteAddr = "203.0.113.9:1234"
		r.ServeHTTP(w, req)
		return w.Code
	}
	if got := do(); got != http.StatusOK {
		t.Fatalf("1st: got %d, want 200", got)
	}
	if got := do(); got != http.StatusOK {
		t.Fatalf("2nd: got %d, want 200", got)
	}
	if got := do(); got != http.StatusTooManyRequests {
		t.Fatalf("3rd with redis down: got %d, want 429 (fallback must still limit)", got)
	}
}
