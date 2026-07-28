package accounting

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// captureSleeps stubs rateLimitSleep to record waits without sleeping.
func captureSleeps(t *testing.T) *[]time.Duration {
	t.Helper()
	var waits []time.Duration
	orig := rateLimitSleep
	rateLimitSleep = func(_ context.Context, d time.Duration) error {
		waits = append(waits, d)
		return nil
	}
	t.Cleanup(func() { rateLimitSleep = orig })
	return &waits
}

func TestRateLimitRetryHonorsRetryAfterAndReplaysBody(t *testing.T) {
	waits := captureSleeps(t)

	var bodies []string
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(b))
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "7")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	req, _ := http.NewRequestWithContext(context.Background(), "POST", srv.URL, bytes.NewReader([]byte(`{"n":1}`)))
	resp, err := doWithRateLimitRetry(context.Background(), req)
	if err != nil {
		t.Fatalf("doWithRateLimitRetry: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 after retry", resp.StatusCode)
	}
	if len(*waits) != 1 || (*waits)[0] != 7*time.Second {
		t.Fatalf("waits = %v, want exactly the 7s Retry-After", *waits)
	}
	// The retried POST must carry the SAME body — a consumed body silently
	// replayed as empty would corrupt the record being pushed.
	if len(bodies) != 2 || bodies[0] != `{"n":1}` || bodies[1] != `{"n":1}` {
		t.Fatalf("bodies = %q, want the payload on both attempts", bodies)
	}
}

func TestRateLimitRetryGivesUpAfterCap(t *testing.T) {
	waits := captureSleeps(t)

	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	req, _ := http.NewRequestWithContext(context.Background(), "GET", srv.URL, nil)
	resp, err := doWithRateLimitRetry(context.Background(), req)
	if err != nil {
		t.Fatalf("doWithRateLimitRetry: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// The final 429 is RETURNED (callers turn it into a logged sync error);
	// the helper must not loop forever.
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want the final 429 surfaced", resp.StatusCode)
	}
	if attempts != rateLimitMaxRetries+1 {
		t.Fatalf("attempts = %d, want %d", attempts, rateLimitMaxRetries+1)
	}
	// No usable Retry-After -> default wait each time.
	for _, w := range *waits {
		if w != rateLimitDefaultWait {
			t.Fatalf("wait = %v, want default %v", w, rateLimitDefaultWait)
		}
	}
}

func TestRateLimitRetryCapsAbsurdRetryAfter(t *testing.T) {
	waits := captureSleeps(t)

	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "3600") // must not stall the sweep an hour
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	req, _ := http.NewRequestWithContext(context.Background(), "GET", srv.URL, nil)
	resp, err := doWithRateLimitRetry(context.Background(), req)
	if err != nil {
		t.Fatalf("doWithRateLimitRetry: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if len(*waits) != 1 || (*waits)[0] != rateLimitMaxWait {
		t.Fatalf("waits = %v, want capped at %v", *waits, rateLimitMaxWait)
	}
}

func TestRateLimitRetryAbortsOnContextCancel(t *testing.T) {
	orig := rateLimitSleep
	rateLimitSleep = func(ctx context.Context, _ time.Duration) error { return ctx.Err() }
	t.Cleanup(func() { rateLimitSleep = orig })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", srv.URL, nil)
	if _, err := doWithRateLimitRetry(ctx, req); err == nil {
		t.Fatal("want context error when the wait is interrupted")
	}
}
