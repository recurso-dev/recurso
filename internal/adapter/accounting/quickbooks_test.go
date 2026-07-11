package accounting

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// qboTestServer routes GET entity reads and POST writes for one QBO object
// type and records what it saw.
type qboTestServer struct {
	t *testing.T

	getSyncTokens []string                 // SyncToken returned per GET, consumed in order (last repeats)
	getCalls      int                      //
	postResponses []func() (int, string)   // response per POST, consumed in order (last repeats)
	postCalls     int                      //
	postBodies    []map[string]interface{} // decoded body of each POST
}

func (s *qboTestServer) handler(objectType, jsonKey, id string) http.HandlerFunc {
	entityPath := fmt.Sprintf("/v3/company/realm-1/%s/%s", objectType, id)
	collectionPath := fmt.Sprintf("/v3/company/realm-1/%s", objectType)

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "GET" && r.URL.Path == entityPath:
			idx := s.getCalls
			if idx >= len(s.getSyncTokens) {
				idx = len(s.getSyncTokens) - 1
			}
			s.getCalls++
			_, _ = fmt.Fprintf(w, `{"%s":{"Id":"%s","SyncToken":"%s"}}`, jsonKey, id, s.getSyncTokens[idx])
		case r.Method == "POST" && r.URL.Path == collectionPath:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				s.t.Errorf("failed to decode POST body: %v", err)
			}
			s.postBodies = append(s.postBodies, body)
			idx := s.postCalls
			if idx >= len(s.postResponses) {
				idx = len(s.postResponses) - 1
			}
			s.postCalls++
			status, resp := s.postResponses[idx]()
			w.WriteHeader(status)
			_, _ = fmt.Fprint(w, resp)
		default:
			s.t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func newQBOTestAdapter(serverURL string) *QuickBooksAdapter {
	return &QuickBooksAdapter{
		baseURL:          serverURL,
		accessToken:      "test-token",
		realmID:          "realm-1",
		incomeAccountRef: "1",
	}
}

func qboOK(jsonKey, id string) func() (int, string) {
	return func() (int, string) {
		return http.StatusOK, fmt.Sprintf(`{"%s":{"Id":"%s","SyncToken":"4"}}`, jsonKey, id)
	}
}

func qboFault(code string) func() (int, string) {
	return func() (int, string) {
		return http.StatusBadRequest, fmt.Sprintf(`{"Fault":{"Error":[{"Message":"boom","code":"%s"}],"type":"ValidationFault"}}`, code)
	}
}

func testCustomer() *domain.Customer {
	name := "Acme"
	return &domain.Customer{ID: uuid.New(), Email: "acme@example.com", Name: &name}
}

func TestQuickBooksUpdateFetchesSyncTokenAndSendsSparseUpdate(t *testing.T) {
	ts := &qboTestServer{t: t, getSyncTokens: []string{"3"}, postResponses: []func() (int, string){qboOK("Customer", "42")}}
	server := httptest.NewServer(ts.handler("customer", "Customer", "42"))
	defer server.Close()

	adapter := newQBOTestAdapter(server.URL)
	id, err := adapter.SyncCustomer(context.Background(), testCustomer(), "42")
	if err != nil {
		t.Fatalf("SyncCustomer(update) returned error: %v", err)
	}
	if id != "42" {
		t.Errorf("returned id = %q, want 42", id)
	}
	if ts.getCalls != 1 {
		t.Errorf("entity GET called %d times, want 1 (SyncToken fetch)", ts.getCalls)
	}
	if ts.postCalls != 1 {
		t.Fatalf("POST called %d times, want 1", ts.postCalls)
	}

	body := ts.postBodies[0]
	if body["Id"] != "42" {
		t.Errorf("update body Id = %v, want 42", body["Id"])
	}
	if body["SyncToken"] != "3" {
		t.Errorf("update body SyncToken = %v, want 3 (fetched before update)", body["SyncToken"])
	}
	if body["sparse"] != true {
		t.Errorf("update body sparse = %v, want true", body["sparse"])
	}
}

func TestQuickBooksUpdateStaleSyncTokenRefetchesAndRetriesOnce(t *testing.T) {
	ts := &qboTestServer{
		t:             t,
		getSyncTokens: []string{"3", "7"}, // token moved between fetch and update
		postResponses: []func() (int, string){qboFault("5010"), qboOK("Customer", "42")},
	}
	server := httptest.NewServer(ts.handler("customer", "Customer", "42"))
	defer server.Close()

	adapter := newQBOTestAdapter(server.URL)
	id, err := adapter.SyncCustomer(context.Background(), testCustomer(), "42")
	if err != nil {
		t.Fatalf("SyncCustomer(update) returned error after stale-token retry: %v", err)
	}
	if id != "42" {
		t.Errorf("returned id = %q, want 42", id)
	}
	if ts.getCalls != 2 {
		t.Errorf("entity GET called %d times, want 2 (initial + stale refetch)", ts.getCalls)
	}
	if ts.postCalls != 2 {
		t.Fatalf("POST called %d times, want 2 (stale + retry)", ts.postCalls)
	}
	if ts.postBodies[0]["SyncToken"] != "3" || ts.postBodies[1]["SyncToken"] != "7" {
		t.Errorf("SyncTokens sent = %v/%v, want 3 then refetched 7", ts.postBodies[0]["SyncToken"], ts.postBodies[1]["SyncToken"])
	}
}

func TestQuickBooksUpdateStaleTokenRetriesOnlyOnce(t *testing.T) {
	ts := &qboTestServer{
		t:             t,
		getSyncTokens: []string{"3"},
		postResponses: []func() (int, string){qboFault("5010")}, // always stale
	}
	server := httptest.NewServer(ts.handler("customer", "Customer", "42"))
	defer server.Close()

	adapter := newQBOTestAdapter(server.URL)
	_, err := adapter.SyncCustomer(context.Background(), testCustomer(), "42")
	if err == nil {
		t.Fatal("SyncCustomer(update) returned nil, want error after exhausted retry")
	}
	if errors.Is(err, port.ErrExternalGone) {
		t.Errorf("stale-token failure misclassified as ErrExternalGone: %v", err)
	}
	if ts.postCalls != 2 {
		t.Errorf("POST called %d times, want exactly 2 (one retry)", ts.postCalls)
	}
}

func TestQuickBooksUpdateGoneOnFetchReturnsErrExternalGone(t *testing.T) {
	for name, respond := range map[string]http.HandlerFunc{
		"http 404": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		},
		"fault 610": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprint(w, `{"Fault":{"Error":[{"Message":"Object Not Found","code":"610"}],"type":"ValidationFault"}}`)
		},
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(respond)
			defer server.Close()

			adapter := newQBOTestAdapter(server.URL)
			_, err := adapter.SyncCustomer(context.Background(), testCustomer(), "42")
			if !errors.Is(err, port.ErrExternalGone) {
				t.Errorf("err = %v, want ErrExternalGone", err)
			}
		})
	}
}

func TestQuickBooksUpdateGoneOnPostReturnsErrExternalGone(t *testing.T) {
	// GET succeeds (entity readable) but the update POST reports 610 —
	// e.g. the object was deleted between fetch and update.
	ts := &qboTestServer{t: t, getSyncTokens: []string{"3"}, postResponses: []func() (int, string){qboFault("610")}}
	server := httptest.NewServer(ts.handler("invoice", "Invoice", "17"))
	defer server.Close()

	adapter := newQBOTestAdapter(server.URL)
	invoice := &domain.Invoice{ID: uuid.New(), InvoiceNumber: "INV-001", Currency: "USD", Subtotal: 5000, DueDate: time.Now()}
	_, err := adapter.SyncInvoice(context.Background(), invoice, port.InvoiceSyncRefs{CustomerExternalID: "9"}, "17")
	if !errors.Is(err, port.ErrExternalGone) {
		t.Errorf("err = %v, want ErrExternalGone", err)
	}
}

func TestQuickBooksInvoiceLineCarriesItemRefWhenProductMapped(t *testing.T) {
	ts := &qboTestServer{t: t, postResponses: []func() (int, string){qboOK("Invoice", "17")}}
	server := httptest.NewServer(ts.handler("invoice", "Invoice", "17"))
	defer server.Close()

	adapter := newQBOTestAdapter(server.URL)
	invoice := &domain.Invoice{ID: uuid.New(), InvoiceNumber: "INV-001", Currency: "USD", Subtotal: 5000}

	if _, err := adapter.SyncInvoice(context.Background(), invoice,
		port.InvoiceSyncRefs{CustomerExternalID: "9", ProductExternalID: "77"}, ""); err != nil {
		t.Fatalf("SyncInvoice returned error: %v", err)
	}

	line := ts.postBodies[0]["Line"].([]interface{})[0].(map[string]interface{})
	detail := line["SalesItemLineDetail"].(map[string]interface{})
	itemRef, ok := detail["ItemRef"].(map[string]interface{})
	if !ok {
		t.Fatalf("SalesItemLineDetail.ItemRef missing: %v", detail)
	}
	if itemRef["value"] != "77" {
		t.Errorf("ItemRef.value = %v, want 77", itemRef["value"])
	}
}

func TestQuickBooksInvoiceLineOmitsItemRefWithoutProductMapping(t *testing.T) {
	ts := &qboTestServer{t: t, postResponses: []func() (int, string){qboOK("Invoice", "17")}}
	server := httptest.NewServer(ts.handler("invoice", "Invoice", "17"))
	defer server.Close()

	adapter := newQBOTestAdapter(server.URL)
	invoice := &domain.Invoice{ID: uuid.New(), InvoiceNumber: "INV-001", Currency: "USD", Subtotal: 5000}

	if _, err := adapter.SyncInvoice(context.Background(), invoice,
		port.InvoiceSyncRefs{CustomerExternalID: "9"}, ""); err != nil {
		t.Fatalf("SyncInvoice returned error: %v", err)
	}

	line := ts.postBodies[0]["Line"].([]interface{})[0].(map[string]interface{})
	detail := line["SalesItemLineDetail"].(map[string]interface{})
	if _, ok := detail["ItemRef"]; ok {
		t.Errorf("SalesItemLineDetail.ItemRef present without a product mapping: %v", detail)
	}
	if line["Description"] == "" {
		t.Error("bare line lost its Description")
	}
}
