package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/recurso-dev/recurso/internal/core/port"
)

// AdyenGateway is the global card/wallet gateway via Adyen Checkout —
// Track D1, spec_integrations_payments.md. It implements hosted checkout
// sessions, off-session charges on stored payment methods, and refunds;
// UPI mandates and virtual accounts return not-supported errors.
//
// EXPERIMENTAL: built against the Adyen Checkout v71 API reference;
// test-merchant verification is founder-gated.
type AdyenGateway struct {
	apiKey          string
	merchantAccount string
	baseURL         string
	httpClient      *http.Client
}

const adyenTestURL = "https://checkout-test.adyen.com/v71"

// NewAdyenGateway builds the gateway. environment "test" targets Adyen's
// test platform; live requires the account-specific URL prefix
// (https://<prefix>-checkout-live.adyenpayments.com/checkout/v71).
func NewAdyenGateway(apiKey, merchantAccount, environment, liveURLPrefix string) *AdyenGateway {
	baseURL := adyenTestURL
	if !strings.EqualFold(environment, "test") && liveURLPrefix != "" {
		baseURL = fmt.Sprintf("https://%s-checkout-live.adyenpayments.com/checkout/v71", liveURLPrefix)
	}
	return &AdyenGateway{
		apiKey:          apiKey,
		merchantAccount: merchantAccount,
		baseURL:         baseURL,
		httpClient:      &http.Client{Timeout: 30 * time.Second},
	}
}

func (g *AdyenGateway) do(ctx context.Context, path string, body any, idempotencyKey string, out any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("x-api-key", g.apiKey)
	req.Header.Set("Content-Type", "application/json")
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("adyen request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	payload, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode >= 400 {
		var ae struct {
			Message   string `json:"message"`
			ErrorCode string `json:"errorCode"`
		}
		_ = json.Unmarshal(payload, &ae)
		if ae.Message != "" {
			return fmt.Errorf("adyen %s: %s (code %s)", path, ae.Message, ae.ErrorCode)
		}
		return fmt.Errorf("adyen %s: HTTP %d", path, resp.StatusCode)
	}
	if out != nil {
		if err := json.Unmarshal(payload, out); err != nil {
			return fmt.Errorf("adyen %s: bad response: %w", path, err)
		}
	}
	return nil
}

// adyenAmount is Adyen's {value, currency} money shape (minor units).
type adyenAmount struct {
	Value    int64  `json:"value"`
	Currency string `json:"currency"`
}

// CreateOrder opens a Checkout Session: the returned ClientSecret carries
// sessionData for Adyen's Drop-in/Components on the frontend.
func (g *AdyenGateway) CreateOrder(ctx context.Context, amount int64, currency string, receipt string, invoiceID string) (*port.PaymentOrder, error) {
	var out struct {
		ID          string `json:"id"`
		SessionData string `json:"sessionData"`
	}
	err := g.do(ctx, "/sessions", map[string]any{
		"amount":          adyenAmount{Value: amount, Currency: strings.ToUpper(currency)},
		"merchantAccount": g.merchantAccount,
		"reference":       receipt,
		"returnUrl":       "https://checkout.recurso.dev/return", // overridden by frontend config
		"metadata":        map[string]string{"invoice_id": invoiceID},
	}, "session-"+invoiceID, &out)
	if err != nil {
		return nil, err
	}
	return &port.PaymentOrder{
		ID:           out.ID,
		Amount:       amount,
		Currency:     currency,
		Receipt:      receipt,
		ClientSecret: out.SessionData,
		Gateway:      "adyen",
	}, nil
}

// ChargeSavedPaymentMethod charges a stored payment method off-session
// (the renewal/auto-recharge path). storedPaymentMethodID is Adyen's
// stored method id; shopperReference is the Recurso customer id the method
// was stored under.
func (g *AdyenGateway) ChargeSavedPaymentMethod(ctx context.Context, shopperReference, storedPaymentMethodID string, amount int64, currency, invoiceID, idempotencyKey string) (*port.PaymentResult, error) {
	var out struct {
		PspReference  string `json:"pspReference"`
		ResultCode    string `json:"resultCode"`
		RefusalReason string `json:"refusalReason"`
	}
	err := g.do(ctx, "/payments", map[string]any{
		"amount":          adyenAmount{Value: amount, Currency: strings.ToUpper(currency)},
		"merchantAccount": g.merchantAccount,
		"reference":       invoiceID,
		"paymentMethod": map[string]string{
			"type":                  "scheme",
			"storedPaymentMethodId": storedPaymentMethodID,
		},
		"shopperReference":         shopperReference,
		"shopperInteraction":       "ContAuth",
		"recurringProcessingModel": "UnscheduledCardOnFile",
		"metadata":                 map[string]string{"invoice_id": invoiceID},
	}, idempotencyKey, &out)
	if err != nil {
		return &port.PaymentResult{Success: false, ErrorCode: "gateway_error", ErrorMsg: err.Error()}, err
	}
	if out.ResultCode != "Authorised" {
		return &port.PaymentResult{
			Success:   false,
			PaymentID: out.PspReference,
			ErrorCode: strings.ToLower(strings.ReplaceAll(out.ResultCode, " ", "_")),
			ErrorMsg:  out.RefusalReason,
		}, nil
	}
	return &port.PaymentResult{Success: true, PaymentID: out.PspReference}, nil
}

// RetryPayment is not directly supported: retries need the stored payment
// method + shopper reference, which the retry worker supplies through
// ChargeSavedPaymentMethod.
func (g *AdyenGateway) RetryPayment(ctx context.Context, invoiceID string, amount int64, currency string) (*port.PaymentResult, error) {
	return nil, fmt.Errorf("adyen retries require a stored payment method; use ChargeSavedPaymentMethod")
}

// Refund refunds a captured payment by pspReference.
func (g *AdyenGateway) Refund(ctx context.Context, paymentID string, amount int64, currency string) (*port.RefundResult, error) {
	var out struct {
		PspReference string `json:"pspReference"`
		Status       string `json:"status"`
	}
	err := g.do(ctx, "/payments/"+paymentID+"/refunds", map[string]any{
		"amount":          adyenAmount{Value: amount, Currency: strings.ToUpper(currency)},
		"merchantAccount": g.merchantAccount,
	}, "refund-"+paymentID, &out)
	if err != nil {
		return nil, err
	}
	status := out.Status
	if status == "" {
		status = "received"
	}
	return &port.RefundResult{RefundID: out.PspReference, Status: status}, nil
}

// --- Not supported ---

func (g *AdyenGateway) VerifyPayment(ctx context.Context, orderID, paymentID, signature string) error {
	return fmt.Errorf("signature verification is not used by adyen sessions; rely on webhooks")
}

func (g *AdyenGateway) CreateSubscription(ctx context.Context, planID string, totalCount int, customerEmail string, startAt *int64, currency string) (string, error) {
	return "", fmt.Errorf("gateway-managed subscriptions are not supported by adyen adapter (Recurso owns the cycle)")
}

func (g *AdyenGateway) CreateMandate(ctx context.Context, customerEmail, customerContact, vpa string, maxAmount int64, frequency string) (*port.MandateResult, error) {
	return nil, fmt.Errorf("UPI mandates are not supported by adyen")
}

func (g *AdyenGateway) ExecuteMandateDebit(ctx context.Context, req port.MandateDebitRequest) (*port.PaymentResult, error) {
	return nil, fmt.Errorf("UPI mandate debits are not supported by adyen")
}

func (g *AdyenGateway) RevokeMandate(ctx context.Context, customerID, tokenID string) error {
	return fmt.Errorf("UPI mandates are not supported by adyen")
}

func (g *AdyenGateway) CreateVirtualAccount(ctx context.Context, customerID, invoiceID string, amount int64, description string) (*port.VirtualAccountResult, error) {
	return nil, fmt.Errorf("virtual accounts are not supported by adyen")
}

func (g *AdyenGateway) CancelSubscription(ctx context.Context, subscriptionID string) error {
	return nil // Recurso owns the billing cycle; nothing exists gateway-side
}
