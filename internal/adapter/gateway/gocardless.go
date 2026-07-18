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

// GoCardlessGateway is the bank-debit gateway (SEPA/BACS/ACH) — Track D1,
// spec_integrations_payments.md. Bank debit is mandate-first, so it
// implements the mandate surface of port.PaymentGateway (the European/UK
// analog of Razorpay's UPI AutoPay) plus refunds; card-style order flows
// return explicit not-supported errors.
//
// EXPERIMENTAL: built against the GoCardless API reference (version
// 2015-07-06); sandbox verification is founder-gated.
type GoCardlessGateway struct {
	accessToken string
	baseURL     string
	httpClient  *http.Client
}

const (
	goCardlessLiveURL    = "https://api.gocardless.com"
	goCardlessSandboxURL = "https://api-sandbox.gocardless.com"
	goCardlessVersion    = "2015-07-06"
)

// NewGoCardlessGateway builds the gateway. environment "sandbox" targets
// the GoCardless sandbox; anything else targets live.
func NewGoCardlessGateway(accessToken, environment string) *GoCardlessGateway {
	baseURL := goCardlessLiveURL
	if strings.EqualFold(environment, "sandbox") {
		baseURL = goCardlessSandboxURL
	}
	return &GoCardlessGateway{
		accessToken: accessToken,
		baseURL:     baseURL,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

// gcError is GoCardless's error envelope.
type gcError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    int    `json:"code"`
	} `json:"error"`
}

// do sends a JSON request and decodes the response into out (may be nil).
// idempotencyKey is optional; GoCardless dedupes creates on it.
func (g *GoCardlessGateway) do(ctx context.Context, method, path string, body any, idempotencyKey string, out any) (int, error) {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, g.baseURL+path, reader)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+g.accessToken)
	req.Header.Set("GoCardless-Version", goCardlessVersion)
	req.Header.Set("Content-Type", "application/json")
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("gocardless request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode >= 400 {
		var ge gcError
		_ = json.Unmarshal(raw, &ge)
		if ge.Error.Message != "" {
			return resp.StatusCode, fmt.Errorf("gocardless %s: %s (%s)", path, ge.Error.Message, ge.Error.Type)
		}
		return resp.StatusCode, fmt.Errorf("gocardless %s: HTTP %d", path, resp.StatusCode)
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return resp.StatusCode, fmt.Errorf("gocardless %s: bad response: %w", path, err)
		}
	}
	return resp.StatusCode, nil
}

// CreateMandate starts the mandate-authorisation flow: a billing request
// (mandate_request) plus a hosted billing-request flow whose URL the
// customer visits to authorise the debit. vpa is ignored (UPI-specific);
// frequency/maxAmount ride the metadata for operator reference — bank
// debit schemes authorise open-ended mandates.
func (g *GoCardlessGateway) CreateMandate(ctx context.Context, customerEmail, customerContact, vpa string, maxAmount int64, frequency string) (*port.MandateResult, error) {
	var br struct {
		BillingRequests struct {
			ID string `json:"id"`
		} `json:"billing_requests"`
	}
	_, err := g.do(ctx, http.MethodPost, "/billing_requests", map[string]any{
		"billing_requests": map[string]any{
			"mandate_request": map[string]any{"scheme": "sepa_core"},
			"metadata": map[string]string{
				"email":      customerEmail,
				"max_amount": fmt.Sprintf("%d", maxAmount),
				"frequency":  frequency,
			},
		},
	}, "", &br)
	if err != nil {
		return nil, err
	}

	var flow struct {
		BillingRequestFlows struct {
			AuthorisationURL string `json:"authorisation_url"`
		} `json:"billing_request_flows"`
	}
	_, err = g.do(ctx, http.MethodPost, "/billing_request_flows", map[string]any{
		"billing_request_flows": map[string]any{
			"links": map[string]string{"billing_request": br.BillingRequests.ID},
			"prefilled_customer": map[string]string{
				"email": customerEmail,
			},
		},
	}, "", &flow)
	if err != nil {
		return nil, err
	}

	return &port.MandateResult{
		TokenID: br.BillingRequests.ID,
		AuthURL: flow.BillingRequestFlows.AuthorisationURL,
		Status:  "created",
	}, nil
}

// ExecuteMandateDebit collects a payment against an authorised mandate.
// req.TokenID must hold the GoCardless mandate id (MD...). The idempotency
// key makes retries of the same billing cycle collect at most once.
func (g *GoCardlessGateway) ExecuteMandateDebit(ctx context.Context, req port.MandateDebitRequest) (*port.PaymentResult, error) {
	var out struct {
		Payments struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"payments"`
	}
	_, err := g.do(ctx, http.MethodPost, "/payments", map[string]any{
		"payments": map[string]any{
			"amount":   req.Amount,
			"currency": strings.ToUpper(req.Currency),
			"links":    map[string]string{"mandate": req.TokenID},
			"metadata": map[string]string{"invoice_id": req.InvoiceID},
		},
	}, req.IdempotencyKey, &out)
	if err != nil {
		return &port.PaymentResult{Success: false, ErrorCode: "gateway_error", ErrorMsg: err.Error()}, err
	}
	// Bank debits settle asynchronously: pending_submission/submitted are the
	// healthy initial states. Settlement lands via webhook, like SEPA on
	// Stripe — the invoice stays open until then.
	return &port.PaymentResult{Success: true, PaymentID: out.Payments.ID}, nil
}

// RevokeMandate cancels the mandate. An already-cancelled mandate reports
// success, matching the port contract.
func (g *GoCardlessGateway) RevokeMandate(ctx context.Context, customerID, tokenID string) error {
	status, err := g.do(ctx, http.MethodPost, "/mandates/"+tokenID+"/actions/cancel", map[string]any{}, "", nil)
	if err != nil {
		// 422 cancellation_failed on an already-cancelled mandate = success.
		if status == http.StatusUnprocessableEntity {
			return nil
		}
		return err
	}
	return nil
}

// Refund returns money for a collected payment (paymentID: PM...).
func (g *GoCardlessGateway) Refund(ctx context.Context, paymentID string, amount int64, currency string) (*port.RefundResult, error) {
	var out struct {
		Refunds struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"refunds"`
	}
	_, err := g.do(ctx, http.MethodPost, "/refunds", map[string]any{
		"refunds": map[string]any{
			"amount": amount,
			"links":  map[string]string{"payment": paymentID},
			// GoCardless requires this guard against double refunds.
			"total_amount_confirmation": amount,
		},
	}, "refund-"+paymentID, &out)
	if err != nil {
		return nil, err
	}
	status := out.Refunds.Status
	if status == "" {
		status = "submitted"
	}
	return &port.RefundResult{RefundID: out.Refunds.ID, Status: status}, nil
}

// --- Not supported on bank-debit rails ---

func (g *GoCardlessGateway) CreateOrder(ctx context.Context, amount int64, currency string, receipt string, invoiceID string) (*port.PaymentOrder, error) {
	return nil, fmt.Errorf("one-off card orders are not supported by gocardless (mandate-first bank debit)")
}

func (g *GoCardlessGateway) VerifyPayment(ctx context.Context, orderID, paymentID, signature string) error {
	return fmt.Errorf("client-side payment verification is not supported by gocardless")
}

func (g *GoCardlessGateway) CreateSubscription(ctx context.Context, planID string, totalCount int, customerEmail string, startAt *int64, currency string) (string, error) {
	return "", fmt.Errorf("gateway-managed subscriptions are not supported by gocardless adapter (Recurso owns the cycle)")
}

func (g *GoCardlessGateway) RetryPayment(ctx context.Context, invoiceID string, amount int64, currency string) (*port.PaymentResult, error) {
	return nil, fmt.Errorf("card retry is not supported by gocardless; debit the mandate instead")
}

func (g *GoCardlessGateway) CreateVirtualAccount(ctx context.Context, customerID, invoiceID string, amount int64, description string) (*port.VirtualAccountResult, error) {
	return nil, fmt.Errorf("virtual accounts are not supported by gocardless")
}

func (g *GoCardlessGateway) CancelSubscription(ctx context.Context, subscriptionID string) error {
	return nil // Recurso owns the billing cycle; nothing exists gateway-side
}
