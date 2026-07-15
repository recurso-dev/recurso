package recurso

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestClient spins up an httptest server and returns a client pointed at it
// plus a pointer to the last request the handler saw.
func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *http.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewClient("rsk_test_key", WithBaseURL(srv.URL+"/v1")), nil
}

func TestCreate_SendsAuthAndBody_DecodesBareResource(t *testing.T) {
	var gotAuth, gotPath, gotMethod, gotCT string
	var gotBody map[string]any
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotCT = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		// createSubscription returns a bare resource (no data envelope).
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"sub_1","status":"active","customer_id":"cus_1"}`))
	})

	sub, err := client.Subscriptions.Create(context.Background(), &SubscriptionCreateParams{
		CustomerID: "cus_1", PlanID: "plan_1",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if gotAuth != "Bearer rsk_test_key" {
		t.Errorf("auth header = %q", gotAuth)
	}
	if gotMethod != "POST" || gotPath != "/v1/subscriptions" {
		t.Errorf("request = %s %s, want POST /v1/subscriptions", gotMethod, gotPath)
	}
	if gotCT != "application/json" {
		t.Errorf("content-type = %q", gotCT)
	}
	if gotBody["customer_id"] != "cus_1" || gotBody["plan_id"] != "plan_1" {
		t.Errorf("body = %v", gotBody)
	}
	if sub.ID != "sub_1" || sub.Status != "active" {
		t.Errorf("decoded sub = %+v", sub)
	}
}

func TestGet_UnwrapsDataEnvelope(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/mandates/m_1" {
			t.Errorf("path = %s", r.URL.Path)
		}
		// getMandate wraps the resource in a {"data": ...} envelope.
		_, _ = w.Write([]byte(`{"data":{"id":"m_1","status":"active","max_amount":50000}}`))
	})
	m, err := client.Mandates.Get(context.Background(), "m_1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if m.ID != "m_1" || m.MaxAmount != 50000 {
		t.Errorf("decoded mandate = %+v (envelope not unwrapped?)", m)
	}
}

func TestList_DecodesDataArray_AndQuery(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("limit"); got != "2" {
			t.Errorf("limit query = %q, want 2", got)
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"cus_1","email":"a@x.com"},{"id":"cus_2"}]}`))
	})
	customers, err := client.Customers.List(context.Background(), &ListParams{Limit: 2})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(customers) != 2 || customers[0].ID != "cus_1" || customers[0].Email != "a@x.com" {
		t.Errorf("decoded customers = %+v", customers)
	}
}

func TestNon2xx_ReturnsAPIError(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"validation_failed","message":"plan_id is required"}}`))
	})
	_, err := client.Subscriptions.Create(context.Background(), &SubscriptionCreateParams{CustomerID: "cus_1"})
	if err == nil {
		t.Fatal("expected an error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error is not *APIError: %v", err)
	}
	if apiErr.StatusCode != 400 || apiErr.Code != "validation_failed" || apiErr.Message != "plan_id is required" {
		t.Errorf("APIError = %+v", apiErr)
	}
}

func TestCustomersCreate_DefaultsCountryUS(t *testing.T) {
	var gotBody map[string]any
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		_, _ = w.Write([]byte(`{"id":"cus_1"}`))
	})
	if _, err := client.Customers.Create(context.Background(), &CustomerCreateParams{
		Email: "a@x.com", Name: "A",
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if gotBody["country"] != "US" {
		t.Errorf("country default = %v, want US", gotBody["country"])
	}
}

func TestPDFURL(t *testing.T) {
	client := NewClient("k", WithBaseURL("https://api.example.com/v1"))
	if got := client.Invoices.PDFURL("inv_1"); got != "https://api.example.com/v1/invoices/inv_1/pdf" {
		t.Errorf("PDFURL = %q", got)
	}
}

func TestDefaultBaseURL(t *testing.T) {
	client := NewClient("k")
	if client.baseURL != DefaultBaseURL {
		t.Errorf("default base URL = %q, want %q", client.baseURL, DefaultBaseURL)
	}
}
