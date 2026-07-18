package accounting

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

func netsuiteTestAdapter(srvURL string) *NetSuiteAdapter {
	a := NewNetSuiteAdapter("ns_token", "123456-sb1")
	a.baseURL = srvURL
	return a
}

func TestNetSuiteSyncCustomerCreateReadsLocation(t *testing.T) {
	var method, path, auth string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		auth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Location", "https://x.suitetalk.api.netsuite.com/services/rest/record/v1/customer/4711")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	a := netsuiteTestAdapter(srv.URL)

	name := "Acme Corp"
	id, err := a.SyncCustomer(context.Background(), &domain.Customer{Name: &name, Email: "a@acme.com"}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "4711" {
		t.Fatalf("id = %q, want 4711 (from Location header)", id)
	}
	if method != "POST" || path != "/customer" || auth != "Bearer ns_token" {
		t.Fatalf("request = %s %s auth %q", method, path, auth)
	}
	if body["companyName"] != "Acme Corp" || body["email"] != "a@acme.com" {
		t.Fatalf("body = %+v", body)
	}
}

func TestNetSuiteSyncCustomerUpdatePatches(t *testing.T) {
	var method, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	a := netsuiteTestAdapter(srv.URL)

	id, err := a.SyncCustomer(context.Background(), &domain.Customer{Email: "a@acme.com"}, "4711")
	if err != nil || id != "4711" {
		t.Fatalf("id/err = %q/%v", id, err)
	}
	if method != "PATCH" || path != "/customer/4711" {
		t.Fatalf("request = %s %s, want PATCH /customer/4711", method, path)
	}
}

func TestNetSuiteGoneMapsToErrExternalGone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	a := netsuiteTestAdapter(srv.URL)

	_, err := a.SyncCustomer(context.Background(), &domain.Customer{Email: "a@x.com"}, "gone1")
	if !errors.Is(err, port.ErrExternalGone) {
		t.Fatalf("err = %v, want ErrExternalGone so the mapping is cleared and re-created", err)
	}
}

func TestNetSuiteSyncInvoiceShape(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Location", srvLocation(r, "invoice/9001"))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	a := netsuiteTestAdapter(srv.URL)

	inv := &domain.Invoice{
		ID:            uuid.New(),
		InvoiceNumber: "INV-42",
		Subtotal:      100000,
		LineItems: []domain.InvoiceItem{
			{Description: "Pro Plan", Quantity: 1, Amount: 100000},
			{Description: "API calls — usage", Quantity: 1, Amount: 52500},
		},
	}
	id, err := a.SyncInvoice(context.Background(), inv, port.InvoiceSyncRefs{
		CustomerExternalID: "4711", ProductExternalID: "77",
	}, "")
	if err != nil || id != "9001" {
		t.Fatalf("id/err = %q/%v", id, err)
	}

	if body["entity"].(map[string]any)["id"] != "4711" {
		t.Fatalf("entity = %+v", body["entity"])
	}
	items := body["item"].(map[string]any)["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("items = %d, want one per invoice line", len(items))
	}
	first := items[0].(map[string]any)
	if first["amount"].(float64) != 1000.0 { // 100000 minor -> 1000.00 major
		t.Fatalf("line amount = %v, want major units", first["amount"])
	}
	if first["item"].(map[string]any)["id"] != "77" {
		t.Fatalf("line item ref = %+v", first["item"])
	}
}

func TestNetSuiteInvoiceRequiresRefs(t *testing.T) {
	a := netsuiteTestAdapter("http://unused")
	inv := &domain.Invoice{ID: uuid.New()}
	if _, err := a.SyncInvoice(context.Background(), inv, port.InvoiceSyncRefs{}, ""); err == nil {
		t.Fatal("missing customer/item refs must fail loudly, not write unbalanced records")
	}
}

// srvLocation builds an absolute Location echoing the test server's host.
func srvLocation(r *http.Request, tail string) string {
	return "http://" + r.Host + "/services/rest/record/v1/" + tail
}
