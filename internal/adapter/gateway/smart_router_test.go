package gateway

import (
	"context"
	"strings"
	"testing"

	"github.com/recurso-dev/recurso/internal/core/port"
)

// routeSpy tracks which gateway methods were called and with what arguments.
type routeSpy struct {
	MockGateway
	name string

	createOrderCalls int
	lastOrderAmount  int64
	lastOrderCcy     string

	createSubCalls int
	lastSubCcy     string

	retryCalls   int
	lastRetryCcy string

	verifyCalls int
	lastOrderID string
}

func (s *routeSpy) CreateOrder(ctx context.Context, amount int64, currency string, receipt string, invoiceID string) (*port.PaymentOrder, error) {
	s.createOrderCalls++
	s.lastOrderAmount = amount
	s.lastOrderCcy = currency
	return &port.PaymentOrder{ID: s.name + "_order", Amount: amount, Currency: currency, Receipt: receipt}, nil
}

func (s *routeSpy) CreateSubscription(ctx context.Context, planID string, totalCount int, customerEmail string, startAt *int64, currency string) (string, error) {
	s.createSubCalls++
	s.lastSubCcy = currency
	return "sub_" + s.name, nil
}

func (s *routeSpy) RetryPayment(ctx context.Context, invoiceID string, amount int64, currency string) (*port.PaymentResult, error) {
	s.retryCalls++
	s.lastRetryCcy = currency
	return &port.PaymentResult{Success: true, PaymentID: "pay_" + s.name}, nil
}

func (s *routeSpy) VerifyPayment(ctx context.Context, orderID, paymentID, signature string) error {
	s.verifyCalls++
	s.lastOrderID = orderID
	return nil
}

func TestSmartRouter_CreateOrder_CurrencyRouting(t *testing.T) {
	tests := []struct {
		name        string
		currency    string
		wantGateway string // "razorpay" or "stripe"
	}{
		{"INR routes to Razorpay", "INR", "razorpay"},
		{"lowercase inr routes to Razorpay", "inr", "razorpay"},
		{"USD routes to Stripe", "USD", "stripe"},
		{"EUR routes to Stripe", "EUR", "stripe"},
		{"empty currency routes to Stripe", "", "stripe"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			razorpay := &routeSpy{name: "razorpay"}
			stripe := &routeSpy{name: "stripe"}
			router := NewSmartRouter(razorpay, stripe)

			order, err := router.CreateOrder(context.Background(), 118000, tc.currency, "rcpt_1", "inv_1")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var hit, miss *routeSpy
			if tc.wantGateway == "razorpay" {
				hit, miss = razorpay, stripe
			} else {
				hit, miss = stripe, razorpay
			}

			if hit.createOrderCalls != 1 {
				t.Errorf("%s CreateOrder calls = %d, want 1", tc.wantGateway, hit.createOrderCalls)
			}
			if miss.createOrderCalls != 0 {
				t.Errorf("%s CreateOrder calls = %d, want 0", miss.name, miss.createOrderCalls)
			}
			// Amount and currency must pass through unchanged.
			if hit.lastOrderAmount != 118000 {
				t.Errorf("amount passed = %d, want 118000", hit.lastOrderAmount)
			}
			if hit.lastOrderCcy != tc.currency {
				t.Errorf("currency passed = %q, want %q", hit.lastOrderCcy, tc.currency)
			}
			if order == nil || order.ID != tc.wantGateway+"_order" {
				t.Errorf("order = %+v, want from %s", order, tc.wantGateway)
			}
		})
	}
}

func TestSmartRouter_CreateSubscription_CurrencyRouting(t *testing.T) {
	tests := []struct {
		currency    string
		wantGateway string
	}{
		{"INR", "razorpay"},
		{"USD", "stripe"},
		{"GBP", "stripe"},
	}

	for _, tc := range tests {
		t.Run(tc.currency, func(t *testing.T) {
			razorpay := &routeSpy{name: "razorpay"}
			stripe := &routeSpy{name: "stripe"}
			router := NewSmartRouter(razorpay, stripe)

			subID, err := router.CreateSubscription(context.Background(), "plan_1", 12, "a@b.com", nil, tc.currency)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if subID != "sub_"+tc.wantGateway {
				t.Errorf("subscription ID = %q, want from %s", subID, tc.wantGateway)
			}
			if tc.wantGateway == "razorpay" && (razorpay.createSubCalls != 1 || stripe.createSubCalls != 0) {
				t.Errorf("calls razorpay=%d stripe=%d, want 1/0", razorpay.createSubCalls, stripe.createSubCalls)
			}
			if tc.wantGateway == "stripe" && (stripe.createSubCalls != 1 || razorpay.createSubCalls != 0) {
				t.Errorf("calls razorpay=%d stripe=%d, want 0/1", razorpay.createSubCalls, stripe.createSubCalls)
			}
		})
	}
}

func TestSmartRouter_RetryPayment_CurrencyRouting(t *testing.T) {
	razorpay := &routeSpy{name: "razorpay"}
	stripe := &routeSpy{name: "stripe"}
	router := NewSmartRouter(razorpay, stripe)

	if _, err := router.RetryPayment(context.Background(), "inv_1", 5000, "INR"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if razorpay.retryCalls != 1 || stripe.retryCalls != 0 {
		t.Errorf("INR retry: razorpay=%d stripe=%d, want 1/0", razorpay.retryCalls, stripe.retryCalls)
	}

	if _, err := router.RetryPayment(context.Background(), "inv_2", 5000, "USD"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stripe.retryCalls != 1 {
		t.Errorf("USD retry: stripe=%d, want 1", stripe.retryCalls)
	}
}

func TestSmartRouter_VerifyPayment_RoutesByOrderIDPrefix(t *testing.T) {
	razorpay := &routeSpy{name: "razorpay"}
	stripe := &routeSpy{name: "stripe"}
	router := NewSmartRouter(razorpay, stripe)

	// Razorpay order IDs start with "order_"
	if err := router.VerifyPayment(context.Background(), "order_abc123", "pay_1", "sig"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if razorpay.verifyCalls != 1 || stripe.verifyCalls != 0 {
		t.Errorf("order_ prefix: razorpay=%d stripe=%d, want 1/0", razorpay.verifyCalls, stripe.verifyCalls)
	}
	if razorpay.lastOrderID != "order_abc123" {
		t.Errorf("razorpay verify called with %q, want 'order_abc123'", razorpay.lastOrderID)
	}

	// Stripe PaymentIntents start with "pi_"
	if err := router.VerifyPayment(context.Background(), "pi_xyz789", "pay_2", "sig"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stripe.verifyCalls != 1 {
		t.Errorf("pi_ prefix: stripe=%d, want 1", stripe.verifyCalls)
	}
}

func TestSmartRouter_VerifyPayment_UnknownPrefix_Error(t *testing.T) {
	razorpay := &routeSpy{name: "razorpay"}
	stripe := &routeSpy{name: "stripe"}
	router := NewSmartRouter(razorpay, stripe)

	err := router.VerifyPayment(context.Background(), "ch_unknown_gateway", "pay_3", "sig")
	if err == nil {
		t.Fatal("expected error for unrecognized order ID prefix")
	}
	if !strings.Contains(err.Error(), "unknown gateway") {
		t.Errorf("error = %q, want it to mention 'unknown gateway'", err.Error())
	}
	if razorpay.verifyCalls != 0 || stripe.verifyCalls != 0 {
		t.Errorf("no gateway should be called for unknown prefix: razorpay=%d stripe=%d", razorpay.verifyCalls, stripe.verifyCalls)
	}
}

// --- Currency overrides (Track D1) ---

func TestSmartRouterCurrencyOverrides(t *testing.T) {
	razorpay := NewMockGateway()
	stripe := NewMockGateway()
	gc := NewMockGateway()
	r := NewSmartRouter(razorpay, stripe)
	r.RegisterGateway("gocardless", gc)

	if err := r.SetCurrencyOverrides("EUR=gocardless, GBP=gocardless"); err != nil {
		t.Fatalf("SetCurrencyOverrides: %v", err)
	}

	gw, err := r.gatewayFor("eur")
	if err != nil || gw != port.PaymentGateway(gc) {
		t.Fatalf("EUR should route to gocardless, got %v (%v)", gw, err)
	}
	// Non-overridden currencies keep the existing rule.
	gw, _ = r.gatewayFor("INR")
	if gw != port.PaymentGateway(razorpay) {
		t.Fatal("INR must still route to razorpay")
	}
	gw, _ = r.gatewayFor("USD")
	if gw != port.PaymentGateway(stripe) {
		t.Fatal("USD must still default to stripe")
	}
}

func TestSmartRouterOverridesRejectUnknownGateway(t *testing.T) {
	r := NewSmartRouter(NewMockGateway(), NewMockGateway())
	if err := r.SetCurrencyOverrides("EUR=gocardless"); err == nil {
		t.Fatal("naming an unconfigured gateway must fail at boot, not misroute at charge time")
	}
	if err := r.SetCurrencyOverrides("EUR"); err == nil {
		t.Fatal("malformed entries must be rejected")
	}
	if err := r.SetCurrencyOverrides(""); err != nil {
		t.Fatalf("empty spec is valid: %v", err)
	}
}

// mandateProbe records which gateway a mandate call landed on.
type mandateProbe struct {
	MockGateway
	created []string // currencies seen by CreateMandate
}

func (p *mandateProbe) CreateMandate(_ context.Context, _, _, _ string, _ int64, _, currency string) (*port.MandateResult, error) {
	p.created = append(p.created, currency)
	return &port.MandateResult{TokenID: "probe", Status: "created"}, nil
}

// Mandates must route by currency like orders: an overridden currency reaches
// its gateway, everything else (including the historical empty currency)
// stays on Razorpay. Before this, ALL mandate calls were hardwired to
// Razorpay and the GoCardless adapter was unreachable (live-verified gap).
func TestSmartRouter_MandateRoutesByCurrency(t *testing.T) {
	rzp := &mandateProbe{}
	gc := &mandateProbe{}
	r := NewSmartRouter(rzp, NewMockGateway())
	r.RegisterGateway("gocardless", gc)
	if err := r.SetCurrencyOverrides("EUR=gocardless"); err != nil {
		t.Fatalf("overrides: %v", err)
	}

	if _, err := r.CreateMandate(context.Background(), "e@x.com", "+1", "", 1000, "monthly", "EUR"); err != nil {
		t.Fatalf("EUR mandate: %v", err)
	}
	if len(gc.created) != 1 || len(rzp.created) != 0 {
		t.Fatalf("EUR mandate must reach gocardless (gc=%v rzp=%v)", gc.created, rzp.created)
	}

	// INR and the historical empty currency stay on the UPI rail.
	for _, cur := range []string{"INR", ""} {
		if _, err := r.CreateMandate(context.Background(), "e@x.com", "+1", "v@upi", 1000, "monthly", cur); err != nil {
			t.Fatalf("%q mandate: %v", cur, err)
		}
	}
	if len(rzp.created) != 2 {
		t.Fatalf("INR/empty mandates must reach razorpay, got %v", rzp.created)
	}

	// Debit routes by the request currency the same way.
	if _, err := r.ExecuteMandateDebit(context.Background(), port.MandateDebitRequest{TokenID: "MD1", Amount: 500, Currency: "EUR"}); err != nil {
		t.Fatalf("EUR debit: %v", err)
	}

	// An override naming an unconfigured gateway is rejected at CONFIG time
	// (boot fails fast), so a mandate can never silently reach the wrong rail.
	r2 := NewSmartRouter(&mandateProbe{}, NewMockGateway())
	if err := r2.SetCurrencyOverrides("EUR=gocardless"); err == nil {
		t.Fatal("override naming an unregistered gateway must be rejected")
	}
}

// GoCardless picks the direct-debit scheme from the mandate currency.
func TestGoCardlessSchemeForCurrency(t *testing.T) {
	if got := schemeForCurrency("EUR"); got != "sepa_core" {
		t.Errorf("EUR scheme = %q", got)
	}
	if got := schemeForCurrency("gbp"); got != "bacs" {
		t.Errorf("GBP scheme = %q", got)
	}
	if got := schemeForCurrency("USD"); got != "sepa_core" {
		t.Errorf("USD default scheme = %q", got)
	}
}
