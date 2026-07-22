package mcp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestClientGet_ForwardsKeyAndReturnsBody(t *testing.T) {
	var gotAuth, gotAccept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAccept = r.Header.Get("Accept")
		if r.URL.Path != "/v1/customers" || r.URL.Query().Get("limit") != "100" {
			t.Errorf("unexpected request: %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		_, _ = io.WriteString(w, `{"data":[{"id":"c1"}]}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	q := url.Values{}
	q.Set("limit", "100")
	body, apiErr := c.Get(context.Background(), "rsk_test_abc", "/v1/customers", q)
	if apiErr != nil {
		t.Fatalf("unexpected error: %v", apiErr)
	}
	if gotAuth != "Bearer rsk_test_abc" {
		t.Errorf("Authorization = %q, want Bearer rsk_test_abc", gotAuth)
	}
	if gotAccept != "application/json" {
		t.Errorf("Accept = %q", gotAccept)
	}
	if string(body) != `{"data":[{"id":"c1"}]}` {
		t.Errorf("body = %s", body)
	}
}

func TestClientGet_MapsErrorEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"error":{"message":"customer not found"}}`)
	}))
	defer srv.Close()

	_, apiErr := NewClient(srv.URL).Get(context.Background(), "k", "/v1/customers/x", nil)
	if apiErr == nil {
		t.Fatal("expected error")
	}
	if apiErr.Status != http.StatusNotFound {
		t.Errorf("status = %d", apiErr.Status)
	}
	if apiErr.Error() != "customer not found" {
		t.Errorf("message = %q, want customer not found", apiErr.Error())
	}
}

func TestClientGet_ErrorFallbackWhenNoEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `not json`)
	}))
	defer srv.Close()

	_, apiErr := NewClient(srv.URL).Get(context.Background(), "", "/v1/plans", nil)
	if apiErr == nil || apiErr.Error() != "unauthorized — check the API key" {
		t.Fatalf("got %v", apiErr)
	}
}

func TestClientPost_SendsIdempotencyKeyAndBody(t *testing.T) {
	var gotIdem, gotType string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotIdem = r.Header.Get("Idempotency-Key")
		gotType = r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = io.WriteString(w, `{"data":{"ok":true}}`)
	}))
	defer srv.Close()

	_, apiErr := NewClient(srv.URL).Post(context.Background(), "k", "/v1/customers",
		map[string]any{"name": "ACME"}, "idem-123")
	if apiErr != nil {
		t.Fatalf("unexpected error: %v", apiErr)
	}
	if gotIdem != "idem-123" {
		t.Errorf("Idempotency-Key = %q", gotIdem)
	}
	if gotType != "application/json" {
		t.Errorf("Content-Type = %q", gotType)
	}
	if gotBody["name"] != "ACME" {
		t.Errorf("body = %v", gotBody)
	}
}

func TestClientPost_NoIdempotencyKeyWhenEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Idempotency-Key"); got != "" {
			t.Errorf("Idempotency-Key should be absent, got %q", got)
		}
		_, _ = io.WriteString(w, `{}`)
	}))
	defer srv.Close()

	_, _ = NewClient(srv.URL).Post(context.Background(), "k", "/v1/plans/x/simulate-charges",
		map[string]any{}, "")
}
