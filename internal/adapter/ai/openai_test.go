package ai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// TestOpenAI_RetriesTransientThenSucceeds proves the GenerateCompletion retry
// loop: two 429s are retried and the third (200) succeeds.
func TestOpenAI_RetriesTransientThenSucceeds(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if atomic.AddInt32(&calls, 1) < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"message":"rate limited"}}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer srv.Close()

	p := NewOpenAIProvider("test-key")
	p.baseURL = srv.URL

	// Keep backoff from slowing the test; it's exercised, just briefly.
	out, err := p.GenerateCompletion(context.Background(), "sys", "user")
	if err != nil {
		t.Fatalf("GenerateCompletion: %v", err)
	}
	if out != "ok" {
		t.Errorf("content = %q, want %q", out, "ok")
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Errorf("upstream calls = %d, want 3 (two retries then success)", got)
	}
}

// TestOpenAI_GivesUpAfterMaxRetries proves a persistent 500 is retried up to the
// cap and then returns an error (not an infinite loop).
func TestOpenAI_GivesUpAfterMaxRetries(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := NewOpenAIProvider("test-key")
	p.baseURL = srv.URL

	if _, err := p.GenerateCompletion(context.Background(), "sys", "user"); err == nil {
		t.Fatal("expected an error after exhausting retries, got nil")
	}
	// 1 initial + openAIMaxRetries retries.
	if got, want := atomic.LoadInt32(&calls), int32(openAIMaxRetries+1); got != want {
		t.Errorf("upstream calls = %d, want %d", got, want)
	}
}

// TestOpenAI_HasBoundedTimeout guards against the regression this fixes: the
// shared client must carry a timeout so a stalled connection can't hang forever.
func TestOpenAI_HasBoundedTimeout(t *testing.T) {
	if openAIHTTPClient.Timeout <= 0 || openAIHTTPClient.Timeout > 5*time.Minute {
		t.Errorf("openAI client timeout = %v, want a sane bounded value", openAIHTTPClient.Timeout)
	}
}
