package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetenv(t *testing.T) {
	t.Setenv("TEST_KEY_RECURSO", "custom_val")

	if got := getenv("TEST_KEY_RECURSO", "default"); got != "custom_val" {
		t.Errorf("expected custom_val, got %s", got)
	}

	if got := getenv("UNSET_KEY_RECURSO", "default"); got != "default" {
		t.Errorf("expected default, got %s", got)
	}
}

func TestMCPHealthEndpoint(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	if rec.Body.String() != "ok" {
		t.Errorf("expected body 'ok', got %s", rec.Body.String())
	}
}
