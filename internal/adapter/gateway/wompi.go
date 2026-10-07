package gateway

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/recurso-dev/recurso/internal/core/port"
)

const (
	wompiSandboxURL    = "https://sandbox.wompi.co/v1"
	wompiProductionURL = "https://production.wompi.co/v1"
)

// WompiGateway implements port.PaymentGateway and service.SavedCardCharger for
// Bancolombia / Wompi (Colombia).
type WompiGateway struct {
	publicKey       string
	privateKey      string
	eventsSecret    string
	integritySecret string
	baseURL         string
	httpClient      *http.Client

	// Acceptance token cache
	mu              sync.RWMutex
	acceptanceToken string
	tokenFetchedAt  time.Time
}

// NewWompiGateway creates a new WompiGateway. It targets sandbox if privateKey
// starts with "prv_test_" or publicKey starts with "pub_test_".
func NewWompiGateway(publicKey, privateKey, eventsSecret, integritySecret string) *WompiGateway {
	baseURL := wompiProductionURL
	if strings.HasPrefix(privateKey, "prv_test_") || strings.HasPrefix(publicKey, "pub_test_") {
		baseURL = wompiSandboxURL
	}

	return &WompiGateway{
		publicKey:       publicKey,
		privateKey:      privateKey,
		eventsSecret:    eventsSecret,
		integritySecret: integritySecret,
		baseURL:         baseURL,
		httpClient:      &http.Client{Timeout: 30 * time.Second},
	}
}

func (g *WompiGateway) PublicKey() string {
	return g.publicKey
}

func (g *WompiGateway) BaseURL() string {
	return g.baseURL
}

// ComputeIntegritySignature calculates sha256(reference + amount_in_cents + currency + integritySecret).
func (g *WompiGateway) ComputeIntegritySignature(reference string, amountInCents int64, currency string) string {
	raw := fmt.Sprintf("%s%d%s%s", reference, amountInCents, strings.ToUpper(currency), g.integritySecret)
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}

// GetAcceptanceToken fetches the legal acceptance token required by Colombian regulations.
func (g *WompiGateway) GetAcceptanceToken(ctx context.Context) (string, error) {
	g.mu.RLock()
	if g.acceptanceToken != "" && time.Since(g.tokenFetchedAt) < 12*time.Hour {
		tok := g.acceptanceToken
		g.mu.RUnlock()
		return tok, nil
	}
	g.mu.RUnlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.baseURL+"/merchants/info", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("x-merchant-public-key", g.publicKey)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("wompi merchants info: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	payload, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("wompi merchants info: HTTP %d: %s", resp.StatusCode, string(payload))
	}

	var res struct {
		Data struct {
			PresignedAcceptance struct {
				AcceptanceToken string `json:"acceptance_token"`
			} `json:"presigned_acceptance"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &res); err != nil {
		return "", fmt.Errorf("wompi merchants info unmarshal: %w", err)
	}
	token := res.Data.PresignedAcceptance.AcceptanceToken
	if token == "" {
		return "", fmt.Errorf("wompi merchants info returned empty acceptance token")
	}

	g.mu.Lock()
	g.acceptanceToken = token
	g.tokenFetchedAt = time.Now()
	g.mu.Unlock()

	return token, nil
}

func (g *WompiGateway) doPrivate(ctx context.Context, method, path string, body any, out any) error {
	var bodyReader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, g.baseURL+path, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+g.privateKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("wompi request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	payload, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 400 {
		var we struct {
			Error struct {
				Type   string              `json:"type"`
				Reason string              `json:"reason"`
				Fields map[string][]string `json:"fields"`
			} `json:"error"`
			Message string `json:"message"`
		}
		_ = json.Unmarshal(payload, &we)
		if we.Error.Reason != "" {
			return fmt.Errorf("wompi %s: %s (type %s)", path, we.Error.Reason, we.Error.Type)
		}
		if we.Message != "" {
			return fmt.Errorf("wompi %s: %s", path, we.Message)
		}
		return fmt.Errorf("wompi %s: HTTP %d: %s", path, resp.StatusCode, string(payload))
	}

	if out != nil {
		if err := json.Unmarshal(payload, out); err != nil {
			return fmt.Errorf("wompi %s: unmarshal response: %w", path, err)
		}
	}
	return nil
}

// CreateOrder prepares a payment order for checkout.
func (g *WompiGateway) CreateOrder(ctx context.Context, amount int64, currency string, receipt string, invoiceID string) (*port.PaymentOrder, error) {
	if strings.ToUpper(currency) != "COP" {
		return nil, fmt.Errorf("wompi only supports COP currency, got %s", currency)
	}

	ref := fmt.Sprintf("inv_%s_%d", invoiceID, time.Now().Unix())
	sig := g.ComputeIntegritySignature(ref, amount, "COP")

	return &port.PaymentOrder{
		ID:           ref,
		Amount:       amount,
		Currency:     currency,
		Receipt:      receipt,
		ClientSecret: sig,
		Gateway:      "wompi",
	}, nil
}

// CreatePaymentSource saves a tokenized card into a reusable payment source ID.
func (g *WompiGateway) CreatePaymentSource(ctx context.Context, customerEmail, cardToken string) (string, error) {
	accToken, err := g.GetAcceptanceToken(ctx)
	if err != nil {
		return "", err
	}

	body := map[string]any{
		"type":             "CARD",
		"token":            cardToken,
		"customer_email":   customerEmail,
		"acceptance_token": accToken,
	}

	var res struct {
		Data struct {
			ID     any    `json:"id"`
			Status string `json:"status"`
		} `json:"data"`
	}

	if err := g.doPrivate(ctx, http.MethodPost, "/payment_sources", body, &res); err != nil {
		return "", err
	}

	sourceID := fmt.Sprintf("%v", res.Data.ID)
	return sourceID, nil
}

// ChargeSavedPaymentMethod satisfies service.SavedCardCharger for recurring off-session charges.
func (g *WompiGateway) ChargeSavedPaymentMethod(ctx context.Context, customerEmail, paymentMethodID string, amount int64, currency, invoiceID, idempotencyKey string) (*port.PaymentResult, error) {
	accToken, err := g.GetAcceptanceToken(ctx)
	if err != nil {
		return &port.PaymentResult{Success: false, ErrorCode: "wompi_acceptance_error", ErrorMsg: err.Error()}, err
	}

	sourceIDInt, err := strconv.ParseInt(paymentMethodID, 10, 64)
	if err != nil {
		return &port.PaymentResult{Success: false, ErrorCode: "invalid_payment_source_id", ErrorMsg: err.Error()}, err
	}

	ref := fmt.Sprintf("rec_%s_%d", invoiceID, time.Now().Unix())
	sig := g.ComputeIntegritySignature(ref, amount, "COP")

	body := map[string]any{
		"amount_in_cents":   amount,
		"currency":          "COP",
		"customer_email":    customerEmail,
		"reference":         ref,
		"signature":         sig,
		"payment_source_id": sourceIDInt,
		"payment_method": map[string]any{
			"installments": 1,
		},
		"acceptance_token": accToken,
	}

	var res struct {
		Data struct {
			ID            string `json:"id"`
			Status        string `json:"status"`
			StatusMessage string `json:"status_message"`
		} `json:"data"`
	}

	if err := g.doPrivate(ctx, http.MethodPost, "/transactions", body, &res); err != nil {
		return &port.PaymentResult{Success: false, ErrorCode: "gateway_error", ErrorMsg: err.Error()}, err
	}

	switch res.Data.Status {
	case "APPROVED":
		return &port.PaymentResult{Success: true, PaymentID: res.Data.ID}, nil
	case "PENDING":
		// PENDING transactions settle asynchronously via webhook.
		return &port.PaymentResult{Success: true, PaymentID: res.Data.ID}, nil
	default:
		return &port.PaymentResult{
			Success:   false,
			PaymentID: res.Data.ID,
			ErrorCode: strings.ToLower(res.Data.Status),
			ErrorMsg:  res.Data.StatusMessage,
		}, nil
	}
}

// ChargeToken charges a card token (tok_...) directly on checkout.
func (g *WompiGateway) ChargeToken(ctx context.Context, customerEmail, cardToken string, installments int, amount int64, currency, invoiceID string) (*port.PaymentResult, error) {
	accToken, err := g.GetAcceptanceToken(ctx)
	if err != nil {
		return &port.PaymentResult{Success: false, ErrorCode: "wompi_acceptance_error", ErrorMsg: err.Error()}, err
	}

	if installments < 1 {
		installments = 1
	}

	ref := fmt.Sprintf("chk_%s_%d", invoiceID, time.Now().Unix())
	sig := g.ComputeIntegritySignature(ref, amount, "COP")

	body := map[string]any{
		"amount_in_cents": amount,
		"currency":        "COP",
		"customer_email":  customerEmail,
		"reference":       ref,
		"signature":       sig,
		"payment_method": map[string]any{
			"type":         "CARD",
			"token":        cardToken,
			"installments": installments,
		},
		"acceptance_token": accToken,
	}

	var res struct {
		Data struct {
			ID            string `json:"id"`
			Status        string `json:"status"`
			StatusMessage string `json:"status_message"`
		} `json:"data"`
	}

	if err := g.doPrivate(ctx, http.MethodPost, "/transactions", body, &res); err != nil {
		return &port.PaymentResult{Success: false, ErrorCode: "gateway_error", ErrorMsg: err.Error()}, err
	}

	switch res.Data.Status {
	case "APPROVED":
		return &port.PaymentResult{Success: true, PaymentID: res.Data.ID}, nil
	case "PENDING":
		return &port.PaymentResult{Success: true, PaymentID: res.Data.ID}, nil
	default:
		return &port.PaymentResult{
			Success:   false,
			PaymentID: res.Data.ID,
			ErrorCode: strings.ToLower(res.Data.Status),
			ErrorMsg:  res.Data.StatusMessage,
		}, nil
	}
}

// VerifyPayment checks whether a transaction succeeded by ID.
func (g *WompiGateway) VerifyPayment(ctx context.Context, orderID, paymentID, signature string) error {
	txID := paymentID
	if txID == "" {
		txID = orderID
	}
	if txID == "" {
		return fmt.Errorf("wompi verify requires paymentID or orderID")
	}

	var res struct {
		Data struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"data"`
	}

	if err := g.doPrivate(ctx, http.MethodGet, "/transactions/"+txID, nil, &res); err != nil {
		return err
	}

	if res.Data.Status != "APPROVED" {
		return fmt.Errorf("wompi transaction %s status is %s (expected APPROVED)", txID, res.Data.Status)
	}

	return nil
}

// GetPaymentStatus inspects a transaction status and minor units received.
func (g *WompiGateway) GetPaymentStatus(ctx context.Context, transactionID string) (*port.PaymentStatus, error) {
	var res struct {
		Data struct {
			ID            string `json:"id"`
			Reference     string `json:"reference"`
			Status        string `json:"status"`
			AmountInCents int64  `json:"amount_in_cents"`
		} `json:"data"`
	}

	if err := g.doPrivate(ctx, http.MethodGet, "/transactions/"+transactionID, nil, &res); err != nil {
		return nil, err
	}

	status := "processing"
	var received int64
	if res.Data.Status == "APPROVED" {
		status = "succeeded"
		received = res.Data.AmountInCents
	} else if res.Data.Status == "DECLINED" || res.Data.Status == "ERROR" || res.Data.Status == "VOIDED" {
		status = "failed"
	}

	// Extract invoice ID from reference if formatted as inv_{invoice_id}_{timestamp}
	invoiceID := res.Data.Reference
	parts := strings.Split(res.Data.Reference, "_")
	if len(parts) >= 2 {
		invoiceID = parts[1]
	}

	return &port.PaymentStatus{
		Status:         status,
		InvoiceID:      invoiceID,
		PaymentID:      res.Data.ID,
		AmountReceived: received,
	}, nil
}

// Refund refunds a captured transaction via V2 refunds endpoint.
func (g *WompiGateway) Refund(ctx context.Context, paymentID string, amount int64, currency string) (*port.RefundResult, error) {
	body := map[string]any{
		"transaction_id":  paymentID,
		"amount_in_cents": amount,
		"reason":          "Customer refund",
	}

	var res struct {
		Data struct {
			ID     any    `json:"id"`
			Status string `json:"status"`
		} `json:"data"`
	}

	if err := g.doPrivate(ctx, http.MethodPost, "/refunds", body, &res); err != nil {
		return nil, err
	}

	return &port.RefundResult{
		RefundID: fmt.Sprintf("%v", res.Data.ID),
		Status:   strings.ToLower(res.Data.Status),
	}, nil
}

// VerifyWebhookChecksum validates X-Event-Checksum or signature.checksum.
func (g *WompiGateway) VerifyWebhookChecksum(parsed map[string]any, signature string) bool {
	if g.eventsSecret == "" {
		return false
	}

	sigObj, ok := parsed["signature"].(map[string]any)
	if !ok {
		return false
	}
	props, ok := sigObj["properties"].([]any)
	if !ok {
		return false
	}

	dataObj, _ := parsed["data"].(map[string]any)
	var concat strings.Builder
	for _, p := range props {
		path, ok := p.(string)
		if !ok {
			continue
		}
		// Properties are usually "transaction.id", "transaction.status", etc.
		// Strip "transaction." prefix if dataObj already has transaction key
		val := resolveNestedProperty(dataObj, path)
		concat.WriteString(fmt.Sprint(val))
	}

	var tsStr string
	if ts, ok := parsed["timestamp"].(float64); ok {
		tsStr = strconv.FormatInt(int64(ts), 10)
	} else if ts, ok := parsed["timestamp"].(int64); ok {
		tsStr = strconv.FormatInt(ts, 10)
	} else {
		tsStr = fmt.Sprint(parsed["timestamp"])
	}
	concat.WriteString(tsStr)
	concat.WriteString(g.eventsSecret)

	expected := sha256.Sum256([]byte(concat.String()))
	expectedHex := hex.EncodeToString(expected[:])

	if signature == "" {
		if c, ok := sigObj["checksum"].(string); ok {
			signature = c
		}
	}

	return subtle.ConstantTimeCompare([]byte(strings.ToLower(expectedHex)), []byte(strings.ToLower(signature))) == 1
}

func resolveNestedProperty(m map[string]any, path string) any {
	parts := strings.Split(path, ".")
	var current any = m
	for _, part := range parts {
		if currMap, ok := current.(map[string]any); ok {
			current = currMap[part]
		} else {
			return nil
		}
	}
	return current
}

// --- Stubs for unused port.PaymentGateway methods ---

func (g *WompiGateway) CreateSubscription(ctx context.Context, planID string, totalCount int, customerEmail string, startAt *int64, currency string) (string, error) {
	return "", fmt.Errorf("wompi does not support gateway-managed subscriptions; Recurso manages renewal cycles")
}

func (g *WompiGateway) CancelSubscription(ctx context.Context, subscriptionID string) error {
	return fmt.Errorf("wompi does not support gateway-managed subscriptions; Recurso manages renewal cycles")
}

func (g *WompiGateway) CreateMandate(ctx context.Context, customerEmail, customerContact, vpa string, maxAmount int64, frequency, currency string) (*port.MandateResult, error) {
	return nil, fmt.Errorf("upi mandates are not supported by wompi")
}

func (g *WompiGateway) ExecuteMandateDebit(ctx context.Context, req port.MandateDebitRequest) (*port.PaymentResult, error) {
	return nil, fmt.Errorf("mandate debit is not supported by wompi; use ChargeSavedPaymentMethod")
}

func (g *WompiGateway) RevokeMandate(ctx context.Context, customerID, tokenID, currency string) error {
	return nil
}

func (g *WompiGateway) CreateVirtualAccount(ctx context.Context, customerID, invoiceID string, amount int64, description string) (*port.VirtualAccountResult, error) {
	return nil, fmt.Errorf("virtual accounts are not supported by wompi")
}

func (g *WompiGateway) RetryPayment(ctx context.Context, invoiceID string, amount int64, currency string) (*port.PaymentResult, error) {
	return nil, fmt.Errorf("retry payment requires a stored payment method; use ChargeSavedPaymentMethod")
}
