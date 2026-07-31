package marketing

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBrevoContactSync_AddSignupContact(t *testing.T) {
	var gotAuth, gotBody, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("api-key")
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := &BrevoContactSync{apiKey: "key123", listID: 7, baseURL: srv.URL, client: srv.Client()}
	if err := c.AddSignupContact(context.Background(), "owner@acme.com", "Acme", "US"); err != nil {
		t.Fatalf("AddSignupContact: %v", err)
	}
	if gotPath != "/contacts" {
		t.Errorf("path = %q, want /contacts", gotPath)
	}
	if gotAuth != "key123" {
		t.Errorf("api-key header = %q, want key123", gotAuth)
	}
	for _, want := range []string{`"owner@acme.com"`, `"Acme"`, `"US"`, `"updateEnabled":true`, `"listIds":[7]`} {
		if !strings.Contains(gotBody, want) {
			t.Errorf("request body missing %q\nbody: %s", want, gotBody)
		}
	}
}

// A non-2xx from Brevo surfaces as an error (so the caller logs it), and no list
// is sent when listID is 0.
func TestBrevoContactSync_ErrorAndNoList(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"bad key"}`))
	}))
	defer srv.Close()

	c := &BrevoContactSync{apiKey: "x", listID: 0, baseURL: srv.URL, client: srv.Client()}
	err := c.AddSignupContact(context.Background(), "a@b.com", "Co", "US")
	if err == nil {
		t.Fatal("expected an error on a 401 response")
	}
	if strings.Contains(gotBody, "listIds") {
		t.Errorf("listIds should be omitted when listID is 0: %s", gotBody)
	}
}
