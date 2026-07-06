package accounting

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

// --- Item Code linkage (Xero links invoice lines to items by Code) ---

func TestXeroInvoiceLineCarriesItemCode(t *testing.T) {
	var postBodies []map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "POST" && r.URL.Path == "/Invoices" {
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			postBodies = append(postBodies, body)
			_, _ = fmt.Fprint(w, `{"Invoices":[{"InvoiceID":"inv-1"}]}`)
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	adapter := newXeroTestAdapter(server.URL)
	invoice := &domain.Invoice{ID: uuid.New(), InvoiceNumber: "INV-001", Subtotal: 5000, Currency: "USD"}
	refs := port.InvoiceSyncRefs{
		CustomerExternalID: "c-1",
		ProductExternalID:  "item-uuid-9", // ignored by Xero (lines link by Code)
		ProductCode:        "pro-monthly",
	}

	id, err := adapter.SyncInvoice(context.Background(), invoice, refs, "")
	if err != nil {
		t.Fatalf("SyncInvoice returned error: %v", err)
	}
	if id != "inv-1" {
		t.Errorf("returned id = %q, want inv-1", id)
	}
	if len(postBodies) != 1 {
		t.Fatalf("got %d POSTs, want 1", len(postBodies))
	}
	inv := postBodies[0]["Invoices"].([]interface{})[0].(map[string]interface{})
	line := inv["LineItems"].([]interface{})[0].(map[string]interface{})
	if line["ItemCode"] != "pro-monthly" {
		t.Errorf("line ItemCode = %v, want pro-monthly", line["ItemCode"])
	}
}

func TestXeroInvoiceWithoutProductCodeOmitsItemCode(t *testing.T) {
	var postBodies []map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		postBodies = append(postBodies, body)
		_, _ = fmt.Fprint(w, `{"Invoices":[{"InvoiceID":"inv-2"}]}`)
	}))
	defer server.Close()

	adapter := newXeroTestAdapter(server.URL)
	invoice := &domain.Invoice{ID: uuid.New(), InvoiceNumber: "INV-002", Subtotal: 5000, Currency: "USD"}

	if _, err := adapter.SyncInvoice(context.Background(), invoice, port.InvoiceSyncRefs{CustomerExternalID: "c-1"}, ""); err != nil {
		t.Fatalf("SyncInvoice returned error: %v", err)
	}
	inv := postBodies[0]["Invoices"].([]interface{})[0].(map[string]interface{})
	line := inv["LineItems"].([]interface{})[0].(map[string]interface{})
	if _, has := line["ItemCode"]; has {
		t.Errorf("line without a product code carried ItemCode = %v, want it omitted", line["ItemCode"])
	}
}

func TestXeroItemCreatedWithPlanCode(t *testing.T) {
	var postBodies []map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "GET" && r.URL.Path == "/Items":
			_, _ = fmt.Fprint(w, `{"Items":[]}`) // name dedupe lookup: no match
		case r.Method == "POST" && r.URL.Path == "/Items":
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			postBodies = append(postBodies, body)
			_, _ = fmt.Fprint(w, `{"Items":[{"ItemID":"item-1"}]}`)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	adapter := newXeroTestAdapter(server.URL)
	plan := &domain.Plan{ID: uuid.New(), Name: "Pro Monthly", Code: "pro-monthly"}

	id, err := adapter.SyncProduct(context.Background(), plan, "")
	if err != nil {
		t.Fatalf("SyncProduct returned error: %v", err)
	}
	if id != "item-1" {
		t.Errorf("returned id = %q, want item-1", id)
	}
	if len(postBodies) != 1 {
		t.Fatalf("got %d POSTs, want 1", len(postBodies))
	}
	item := postBodies[0]["Items"].([]interface{})[0].(map[string]interface{})
	if item["Code"] != "pro-monthly" {
		t.Errorf("item Code = %v, want pro-monthly (must match the ItemCode invoice lines carry)", item["Code"])
	}
}

func TestXeroItemCodeTruncatedTo30Chars(t *testing.T) {
	long := strings.Repeat("a", 40)
	got := xeroItemCode(long)
	if got != strings.Repeat("a", 30) {
		t.Errorf("xeroItemCode(40 chars) = %q (len %d), want the first 30", got, len(got))
	}
	if xeroItemCode("short") != "short" {
		t.Errorf("xeroItemCode should leave short codes untouched")
	}
}
