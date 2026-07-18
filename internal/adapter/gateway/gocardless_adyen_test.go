package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/recurso-dev/recurso/internal/core/port"
)

// --- Test harness: capture one request, serve a canned response ---

type capturedReq struct {
	method, path, auth, version, apiKey, idem string
	body                                      map[string]any
}

func gatewayTestServer(t *testing.T, status int, response string) (*httptest.Server, *capturedReq) {
	t.Helper()
	cap := &capturedReq{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cap.method, cap.path = r.Method, r.URL.Path
		cap.auth = r.Header.Get("Authorization")
		cap.version = r.Header.Get("GoCardless-Version")
		cap.apiKey = r.Header.Get("x-api-key")
		cap.idem = r.Header.Get("Idempotency-Key")
		_ = json.NewDecoder(r.Body).Decode(&cap.body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(srv.Close)
	return srv, cap
}

func testGoCardless(srvURL string) *GoCardlessGateway {
	g := NewGoCardlessGateway("gc_token", "sandbox")
	g.baseURL = srvURL
	return g
}

func testAdyen(srvURL string) *AdyenGateway {
	g := NewAdyenGateway("adyen_key", "MerchantCo", "test", "")
	g.baseURL = srvURL
	return g
}

// --- GoCardless ---

func TestGoCardlessExecuteMandateDebit(t *testing.T) {
	srv, cap := gatewayTestServer(t, http.StatusCreated,
		`{"payments":{"id":"PM123","status":"pending_submission"}}`)
	g := testGoCardless(srv.URL)

	res, err := g.ExecuteMandateDebit(context.Background(), port.MandateDebitRequest{
		TokenID:        "MD001",
		Amount:         118000,
		Currency:       "eur",
		InvoiceID:      "inv-1",
		IdempotencyKey: "cycle-42",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success || res.PaymentID != "PM123" {
		t.Fatalf("res = %+v, want success PM123", res)
	}
	if cap.method != "POST" || cap.path != "/payments" {
		t.Fatalf("request = %s %s, want POST /payments", cap.method, cap.path)
	}
	if cap.auth != "Bearer gc_token" || cap.version == "" {
		t.Fatalf("auth/version headers = %q/%q", cap.auth, cap.version)
	}
	if cap.idem != "cycle-42" {
		t.Fatalf("idempotency key = %q, want cycle-42 (same billing cycle must dedupe)", cap.idem)
	}
	payments := cap.body["payments"].(map[string]any)
	if payments["amount"].(float64) != 118000 || payments["currency"] != "EUR" {
		t.Fatalf("payment body = %+v", payments)
	}
	if payments["links"].(map[string]any)["mandate"] != "MD001" {
		t.Fatalf("mandate link missing: %+v", payments)
	}
}

func TestGoCardlessCreateMandateFlow(t *testing.T) {
	// Two sequential calls: billing request, then flow. Serve both from one
	// handler switching on path.
	var flowsBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/billing_requests":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"billing_requests":{"id":"BRQ1"}}`))
		case "/billing_request_flows":
			_ = json.NewDecoder(r.Body).Decode(&flowsBody)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"billing_request_flows":{"authorisation_url":"https://pay.gocardless.com/flow/1"}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	g := testGoCardless(srv.URL)

	res, err := g.CreateMandate(context.Background(), "c@example.com", "+44123", "", 500000, "monthly")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.TokenID != "BRQ1" || res.AuthURL != "https://pay.gocardless.com/flow/1" {
		t.Fatalf("res = %+v", res)
	}
	links := flowsBody["billing_request_flows"].(map[string]any)["links"].(map[string]any)
	if links["billing_request"] != "BRQ1" {
		t.Fatalf("flow not linked to billing request: %+v", flowsBody)
	}
}

func TestGoCardlessRevokeMandateAlreadyCancelled(t *testing.T) {
	srv, _ := gatewayTestServer(t, http.StatusUnprocessableEntity,
		`{"error":{"message":"Mandate is already cancelled","type":"invalid_state","code":422}}`)
	g := testGoCardless(srv.URL)

	if err := g.RevokeMandate(context.Background(), "", "MD001"); err != nil {
		t.Fatalf("already-cancelled mandate must be treated as success, got %v", err)
	}
}

func TestGoCardlessRefundConfirmationGuard(t *testing.T) {
	srv, cap := gatewayTestServer(t, http.StatusCreated,
		`{"refunds":{"id":"RF1","status":"submitted"}}`)
	g := testGoCardless(srv.URL)

	res, err := g.Refund(context.Background(), "PM123", 5000, "EUR")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.RefundID != "RF1" {
		t.Fatalf("res = %+v", res)
	}
	refunds := cap.body["refunds"].(map[string]any)
	if refunds["total_amount_confirmation"].(float64) != 5000 {
		t.Fatal("total_amount_confirmation guard missing — double refunds possible")
	}
}

func TestGoCardlessUnsupportedSurfaces(t *testing.T) {
	g := testGoCardless("http://unused")
	if _, err := g.CreateOrder(context.Background(), 1, "EUR", "r", "i"); err == nil {
		t.Fatal("CreateOrder must be not-supported")
	}
	if _, err := g.RetryPayment(context.Background(), "i", 1, "EUR"); err == nil {
		t.Fatal("RetryPayment must be not-supported")
	}
}

// --- Adyen ---

func TestAdyenCreateOrderSession(t *testing.T) {
	srv, cap := gatewayTestServer(t, http.StatusCreated,
		`{"id":"CS_1","sessionData":"Ab02b4c..."}`)
	g := testAdyen(srv.URL)

	order, err := g.CreateOrder(context.Background(), 9900, "usd", "rcpt-1", "inv-9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if order.ID != "CS_1" || order.ClientSecret != "Ab02b4c..." || order.Gateway != "adyen" {
		t.Fatalf("order = %+v", order)
	}
	if cap.path != "/sessions" || cap.apiKey != "adyen_key" {
		t.Fatalf("request path/key = %s/%s", cap.path, cap.apiKey)
	}
	amount := cap.body["amount"].(map[string]any)
	if amount["value"].(float64) != 9900 || amount["currency"] != "USD" {
		t.Fatalf("amount = %+v", amount)
	}
	if cap.body["merchantAccount"] != "MerchantCo" {
		t.Fatalf("merchantAccount missing: %+v", cap.body)
	}
}

func TestAdyenChargeSavedMethodAuthorised(t *testing.T) {
	srv, cap := gatewayTestServer(t, http.StatusOK,
		`{"pspReference":"883Z","resultCode":"Authorised"}`)
	g := testAdyen(srv.URL)

	res, err := g.ChargeSavedPaymentMethod(context.Background(), "cus_1", "spm_9", 2900, "USD", "inv-1", "renewal-inv-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success || res.PaymentID != "883Z" {
		t.Fatalf("res = %+v", res)
	}
	if cap.idem != "renewal-inv-1" {
		t.Fatalf("idempotency key = %q", cap.idem)
	}
	pm := cap.body["paymentMethod"].(map[string]any)
	if pm["storedPaymentMethodId"] != "spm_9" || pm["type"] != "scheme" {
		t.Fatalf("paymentMethod = %+v", pm)
	}
	if cap.body["shopperInteraction"] != "ContAuth" || cap.body["recurringProcessingModel"] != "UnscheduledCardOnFile" {
		t.Fatalf("off-session flags wrong: %+v", cap.body)
	}
}

func TestAdyenChargeSavedMethodRefused(t *testing.T) {
	srv, _ := gatewayTestServer(t, http.StatusOK,
		`{"pspReference":"884Z","resultCode":"Refused","refusalReason":"NOT_ENOUGH_BALANCE"}`)
	g := testAdyen(srv.URL)

	res, err := g.ChargeSavedPaymentMethod(context.Background(), "cus_1", "spm_9", 2900, "USD", "inv-1", "k")
	if err != nil {
		t.Fatalf("a refusal is a result, not an error: %v", err)
	}
	if res.Success || res.ErrorCode != "refused" || res.ErrorMsg != "NOT_ENOUGH_BALANCE" {
		t.Fatalf("res = %+v, want refused with reason", res)
	}
}

func TestAdyenRefund(t *testing.T) {
	srv, cap := gatewayTestServer(t, http.StatusCreated,
		`{"pspReference":"885Z","status":"received"}`)
	g := testAdyen(srv.URL)

	res, err := g.Refund(context.Background(), "883Z", 500, "USD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.RefundID != "885Z" || res.Status != "received" {
		t.Fatalf("res = %+v", res)
	}
	if cap.path != "/payments/883Z/refunds" {
		t.Fatalf("path = %s", cap.path)
	}
}

func TestAdyenErrorMapping(t *testing.T) {
	srv, _ := gatewayTestServer(t, http.StatusUnprocessableEntity,
		`{"status":422,"errorCode":"14_012","message":"The amount is not allowed"}`)
	g := testAdyen(srv.URL)

	if _, err := g.CreateOrder(context.Background(), -1, "USD", "r", "i"); err == nil {
		t.Fatal("HTTP 422 must surface as an error")
	}
}
