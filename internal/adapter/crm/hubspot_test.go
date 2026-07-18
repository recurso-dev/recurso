package crm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// hubspotFake serves search/create/patch with capture.
type hubspotFake struct {
	searchHits  int // contacts the search reports
	created     map[string]any
	patched     map[string]any
	patchedPath string
	auth        string
}

func (f *hubspotFake) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f.auth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/crm/v3/objects/contacts/search":
			if f.searchHits > 0 {
				_, _ = w.Write([]byte(`{"results":[{"id":"301"}]}`))
			} else {
				_, _ = w.Write([]byte(`{"results":[]}`))
			}
		case r.URL.Path == "/crm/v3/objects/contacts" && r.Method == "POST":
			_ = json.NewDecoder(r.Body).Decode(&f.created)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"501"}`))
		case r.Method == "PATCH":
			f.patchedPath = r.URL.Path
			_ = json.NewDecoder(r.Body).Decode(&f.patched)
			_, _ = w.Write([]byte(`{"id":"301"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

func testHubSpot(t *testing.T, fake *hubspotFake) *HubSpotClient {
	t.Helper()
	srv := httptest.NewServer(fake.handler())
	t.Cleanup(srv.Close)
	c := NewHubSpotClient("hs_pat_token")
	c.baseURL = srv.URL
	return c
}

func TestHubSpotUpsertCreatesWhenAbsent(t *testing.T) {
	fake := &hubspotFake{}
	c := testHubSpot(t, fake)

	id, err := c.UpsertContact(context.Background(), "jane@acme.com", map[string]string{
		"recurso_customer_id": "cus-1", "recurso_subscription_state": "active",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "501" {
		t.Fatalf("id = %q, want the created contact 501", id)
	}
	if fake.auth != "Bearer hs_pat_token" {
		t.Fatalf("auth = %q", fake.auth)
	}
	props := fake.created["properties"].(map[string]any)
	if props["email"] != "jane@acme.com" || props["recurso_subscription_state"] != "active" {
		t.Fatalf("created props = %+v", props)
	}
}

func TestHubSpotUpsertPatchesWhenPresent(t *testing.T) {
	fake := &hubspotFake{searchHits: 1}
	c := testHubSpot(t, fake)

	id, err := c.UpsertContact(context.Background(), "jane@acme.com", map[string]string{
		"recurso_subscription_state": "churned",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "301" || fake.patchedPath != "/crm/v3/objects/contacts/301" {
		t.Fatalf("id/path = %q/%q, want existing contact patched", id, fake.patchedPath)
	}
	if fake.created != nil {
		t.Fatal("no create may happen when the contact exists")
	}
	props := fake.patched["properties"].(map[string]any)
	if props["recurso_subscription_state"] != "churned" {
		t.Fatalf("patched props = %+v", props)
	}
}

func TestHubSpotErrorsSurface(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"invalid token"}`))
	}))
	defer srv.Close()
	c := NewHubSpotClient("bad")
	c.baseURL = srv.URL
	if _, err := c.UpsertContact(context.Background(), "x@y.com", nil); err == nil {
		t.Fatal("HTTP 401 must surface as an error")
	}
}
