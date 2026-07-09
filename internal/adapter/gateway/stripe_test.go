package gateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/client"
)

func TestStripePaymentMethodTypes(t *testing.T) {
	tests := []struct {
		name     string
		currency string
		want     []string
	}{
		{
			name:     "EUR offers card plus European local methods",
			currency: "EUR",
			want:     []string{"bancontact", "card", "ideal", "sepa_debit"},
		},
		{
			name:     "lowercase eur is treated the same",
			currency: "eur",
			want:     []string{"bancontact", "card", "ideal", "sepa_debit"},
		},
		{
			name:     "USD offers card plus ACH (us_bank_account)",
			currency: "USD",
			want:     []string{"card", "us_bank_account"},
		},
		{
			name:     "GBP is card only",
			currency: "GBP",
			want:     []string{"card"},
		},
		{
			name:     "INR is card only",
			currency: "INR",
			want:     []string{"card"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := append([]string(nil), stripePaymentMethodTypes(tt.currency)...)
			sort.Strings(got)
			if len(got) != len(tt.want) {
				t.Fatalf("stripePaymentMethodTypes(%q) = %v, want %v", tt.currency, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("stripePaymentMethodTypes(%q) = %v, want %v", tt.currency, got, tt.want)
				}
			}
		})
	}
}

// newTestStripeGateway wires a StripeGateway to a mock Stripe backend served by
// srv, so we can assert on the outgoing request shape without real API calls.
func newTestStripeGateway(t *testing.T, srv *httptest.Server) *StripeGateway {
	t.Helper()
	backend := stripe.GetBackendWithConfig(stripe.APIBackend, &stripe.BackendConfig{
		URL:        stripe.String(srv.URL),
		HTTPClient: srv.Client(),
	})
	sc := &client.API{}
	sc.Init("sk_test_dummy", &stripe.Backends{API: backend, Connect: backend, Uploads: backend})
	return &StripeGateway{sc: sc}
}

// capturePaymentMethodTypes pulls the payment_method_types[] values out of the
// form-encoded PaymentIntent create request.
func capturePaymentMethodTypes(r *http.Request) []string {
	_ = r.ParseForm()
	var methods []string
	for key, vals := range r.Form {
		if strings.HasPrefix(key, "payment_method_types") {
			methods = append(methods, vals...)
		}
	}
	sort.Strings(methods)
	return methods
}

func TestStripeCreateOrder_EURRequestShape(t *testing.T) {
	var captured []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = capturePaymentMethodTypes(r)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"pi_test_eur","amount":1000,"currency":"eur","status":"requires_payment_method"}`))
	}))
	defer srv.Close()

	gw := newTestStripeGateway(t, srv)
	order, err := gw.CreateOrder(context.Background(), 1000, "eur", "INV-001", "inv-uuid-1")
	if err != nil {
		t.Fatalf("CreateOrder returned error: %v", err)
	}
	if order.ID != "pi_test_eur" {
		t.Fatalf("order.ID = %q, want pi_test_eur", order.ID)
	}

	want := []string{"bancontact", "card", "ideal", "sepa_debit"}
	if strings.Join(captured, ",") != strings.Join(want, ",") {
		t.Fatalf("EUR payment_method_types = %v, want %v", captured, want)
	}
}

func TestStripeCreateOrder_USDCardAndACH(t *testing.T) {
	var captured []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = capturePaymentMethodTypes(r)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"pi_test_usd","amount":1000,"currency":"usd","status":"requires_payment_method"}`))
	}))
	defer srv.Close()

	gw := newTestStripeGateway(t, srv)
	if _, err := gw.CreateOrder(context.Background(), 1000, "usd", "INV-002", "inv-uuid-2"); err != nil {
		t.Fatalf("CreateOrder returned error: %v", err)
	}

	if len(captured) != 2 || captured[0] != "card" || captured[1] != "us_bank_account" {
		t.Fatalf("USD payment_method_types = %v, want [card us_bank_account]", captured)
	}
}

// TestStripeChargeSavedPaymentMethod asserts the off-session charge sends the
// customer + saved payment method and confirms unattended — the shape that
// actually collects from a saved card (ENG-5 Phase 2).
func TestStripeChargeSavedPaymentMethod(t *testing.T) {
	var got struct{ customer, pm, offSession, confirm, invoiceID string }
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		got.customer = r.Form.Get("customer")
		got.pm = r.Form.Get("payment_method")
		got.offSession = r.Form.Get("off_session")
		got.confirm = r.Form.Get("confirm")
		got.invoiceID = r.Form.Get("metadata[invoice_id]")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"pi_offsession","status":"succeeded"}`))
	}))
	defer srv.Close()

	gw := newTestStripeGateway(t, srv)
	res, err := gw.ChargeSavedPaymentMethod(context.Background(), "cus_123", "pm_456", 4900, "usd", "inv-uuid-9", "retry-inv-uuid-9-2")
	if err != nil {
		t.Fatalf("ChargeSavedPaymentMethod returned error: %v", err)
	}
	if !res.Success || res.PaymentID != "pi_offsession" {
		t.Fatalf("result = %+v, want success pi_offsession", res)
	}
	if got.customer != "cus_123" || got.pm != "pm_456" {
		t.Errorf("customer/payment_method = %q/%q, want cus_123/pm_456", got.customer, got.pm)
	}
	if got.offSession != "true" || got.confirm != "true" {
		t.Errorf("off_session/confirm = %q/%q, want true/true", got.offSession, got.confirm)
	}
	if got.invoiceID != "inv-uuid-9" {
		t.Errorf("metadata[invoice_id] = %q, want inv-uuid-9", got.invoiceID)
	}
}

// A card decline must come back as a business failure (for dunning), never as a
// transport error.
func TestStripeChargeSavedPaymentMethod_Decline(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = w.Write([]byte(`{"error":{"type":"card_error","code":"card_declined","message":"Your card was declined."}}`))
	}))
	defer srv.Close()

	gw := newTestStripeGateway(t, srv)
	res, err := gw.ChargeSavedPaymentMethod(context.Background(), "cus_1", "pm_1", 1000, "usd", "inv-1", "k1")
	if err != nil {
		t.Fatalf("a decline must be a business failure, not an error: %v", err)
	}
	if res.Success || res.ErrorCode != "card_declined" {
		t.Fatalf("result = %+v, want failure card_declined", res)
	}
}
