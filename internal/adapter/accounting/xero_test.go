package accounting

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

func newXeroTestAdapter(serverURL string) *XeroAdapter {
	return &XeroAdapter{baseURL: serverURL, accessToken: "test-token", tenantID: "xero-tenant"}
}

func TestXeroUpdatePostsWithContactID(t *testing.T) {
	var postBodies []map[string]interface{}
	var queried bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "POST" && r.URL.Path == "/Contacts":
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			postBodies = append(postBodies, body)
			_, _ = fmt.Fprint(w, `{"Contacts":[{"ContactID":"c-1"}]}`)
		case r.Method == "GET" && r.URL.Path == "/Contacts":
			queried = true // email dedupe lookup — must not run on updates
			_, _ = fmt.Fprint(w, `{"Contacts":[]}`)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	adapter := newXeroTestAdapter(server.URL)
	name := "Acme"
	customer := &domain.Customer{ID: uuid.New(), Email: "acme@example.com", Name: &name}

	id, err := adapter.SyncCustomer(context.Background(), customer, "c-1")
	if err != nil {
		t.Fatalf("SyncCustomer(update) returned error: %v", err)
	}
	if id != "c-1" {
		t.Errorf("returned id = %q, want c-1", id)
	}
	if queried {
		t.Error("email dedupe lookup ran on the update path")
	}
	if len(postBodies) != 1 {
		t.Fatalf("got %d POSTs, want 1", len(postBodies))
	}
	contact := postBodies[0]["Contacts"].([]interface{})[0].(map[string]interface{})
	if contact["ContactID"] != "c-1" {
		t.Errorf("POST body ContactID = %v, want c-1 (Xero upserts on ID)", contact["ContactID"])
	}
}

func TestXeroUpdate404ReturnsErrExternalGone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := newXeroTestAdapter(server.URL)
	name := "Acme"
	customer := &domain.Customer{ID: uuid.New(), Email: "acme@example.com", Name: &name}

	_, err := adapter.SyncCustomer(context.Background(), customer, "c-gone")
	if !errors.Is(err, port.ErrExternalGone) {
		t.Errorf("err = %v, want ErrExternalGone", err)
	}
}

func TestXeroCreate404IsNotErrExternalGone(t *testing.T) {
	// Without a carried ID a 404 is just an API failure, not a stale
	// mapping — the service must not clear anything.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := newXeroTestAdapter(server.URL)
	name := "Acme"
	customer := &domain.Customer{ID: uuid.New(), Email: "acme@example.com", Name: &name}

	_, err := adapter.SyncCustomer(context.Background(), customer, "")
	if err == nil {
		t.Fatal("SyncCustomer returned nil, want error")
	}
	if errors.Is(err, port.ErrExternalGone) {
		t.Errorf("create-path 404 misclassified as ErrExternalGone: %v", err)
	}
}
