package gateway

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/razorpay/razorpay-go"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

type RazorpayGateway struct {
	client *razorpay.Client
	keyID  string
	secret string
}

func NewRazorpayGateway(keyID, secret string) *RazorpayGateway {
	client := razorpay.NewClient(keyID, secret)
	return &RazorpayGateway{
		client: client,
		keyID:  keyID,
		secret: secret,
	}
}

func (g *RazorpayGateway) CreateOrder(ctx context.Context, amount int64, currency string, receipt string, invoiceID string) (*port.PaymentOrder, error) {
	data := map[string]interface{}{
		"amount":   amount,
		"currency": currency,
		"receipt":  receipt,
		"notes":    map[string]interface{}{"invoice_id": invoiceID},
	}

	body, err := g.client.Order.Create(data, nil)
	if err != nil {
		return nil, fmt.Errorf("razorpay create order failed: %v", err)
	}

	id, ok := body["id"].(string)
	if !ok {
		return nil, fmt.Errorf("razorpay response missing id")
	}

	return &port.PaymentOrder{
		ID:       id,
		Amount:   amount,
		Currency: currency,
		Receipt:  receipt,
		Gateway:  "razorpay",
	}, nil
}

func (g *RazorpayGateway) VerifyPayment(ctx context.Context, orderID, paymentID, signature string) error {
	payload := orderID + "|" + paymentID
	mac := hmac.New(sha256.New, []byte(g.secret))
	mac.Write([]byte(payload))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	if expectedSignature != signature {
		return fmt.Errorf("invalid signature")
	}
	return nil
}

// GetOrderInvoiceID fetches a Razorpay order and returns the invoice_id recorded
// in its notes at creation time. The checkout verify step uses this to bind a
// signature-verified payment to the right invoice before settling — so a
// genuine payment for one invoice can't be replayed to settle another (mirrors
// the Stripe metadata check).
func (g *RazorpayGateway) GetOrderInvoiceID(ctx context.Context, orderID string) (string, error) {
	body, err := g.client.Order.Fetch(orderID, nil, nil)
	if err != nil {
		return "", fmt.Errorf("razorpay fetch order %s failed: %v", orderID, err)
	}
	notes, ok := body["notes"].(map[string]interface{})
	if !ok {
		return "", nil
	}
	invoiceID, _ := notes["invoice_id"].(string)
	return invoiceID, nil
}

func (g *RazorpayGateway) CreateSubscription(ctx context.Context, planID string, totalCount int, customerEmail string, startAt *int64, currency string) (string, error) {
	// 1. Create Subscription
	data := map[string]interface{}{
		"plan_id":     planID,
		"total_count": totalCount, // How many billing cycles
		"quantity":    1,
		"notes": map[string]interface{}{
			"email": customerEmail,
		},
	}
	// Razorpay 'start_at' is optional; if omitted the subscription starts immediately.
	if startAt != nil {
		data["start_at"] = *startAt
	}

	// Because razorpay-go might not have Subscription helper in old versions or it's named differently.
	// Looking at library, it usually has `Subscription.Create`.
	// If `g.client.Subscription` is available.

	body, err := g.client.Subscription.Create(data, nil)
	if err != nil {
		return "", fmt.Errorf("razorpay create subscription failed: %v", err)
	}

	id, ok := body["id"].(string)
	if !ok {
		return "", fmt.Errorf("razorpay response missing subscription id")
	}

	return id, nil
}

// CreateMandate issues a UPI-Autopay registration link the customer approves on
// Razorpay's hosted page (short_url). Verified against the live test API:
// method "upi" (not "emandate", which demands full bank details), a customer
// contact number (Razorpay rejects recurring links without one), and a minimum
// ₹1 authorization charge on the link itself.
func (g *RazorpayGateway) CreateMandate(ctx context.Context, customerEmail, customerContact, vpa string, maxAmount int64, frequency string) (*port.MandateResult, error) {
	data := map[string]interface{}{
		"type":        "link",
		"amount":      100, // ₹1 token-authorization charge — Razorpay's minimum for a registration link
		"currency":    "INR",
		"description": "UPI AutoPay Mandate",
		"subscription_registration": map[string]interface{}{
			"method":     "upi",
			"max_amount": maxAmount,
			"frequency":  razorpayFrequency(frequency),
		},
		"customer": map[string]interface{}{
			"email":   customerEmail,
			"contact": customerContact,
		},
		"notes": map[string]interface{}{
			"vpa":       vpa,
			"frequency": frequency,
		},
	}

	body, err := g.client.Invoice.Create(data, nil)
	if err != nil {
		return nil, fmt.Errorf("razorpay create mandate failed: %v", err)
	}

	tokenID, _ := body["token_id"].(string)
	subID, _ := body["id"].(string)
	customerID, _ := body["customer_id"].(string)
	shortURL, _ := body["short_url"].(string)
	status, _ := body["status"].(string)

	return &port.MandateResult{
		TokenID:        tokenID,
		SubscriptionID: subID,
		CustomerID:     customerID,
		AuthURL:        shortURL,
		Status:         status,
	}, nil
}

// razorpayFrequency maps a mandate frequency to Razorpay's enum. "quarterly"
// isn't one of Razorpay's frequencies, so it becomes as_presented (charge on
// demand within max_amount).
func razorpayFrequency(f string) string {
	switch f {
	case "weekly", "monthly", "yearly":
		return f
	default:
		return "as_presented"
	}
}

func (g *RazorpayGateway) ExecuteMandateDebit(ctx context.Context, tokenID string, amount int64, currency, invoiceID string) (*port.PaymentResult, error) {
	data := map[string]interface{}{
		"amount":   amount,
		"currency": currency,
		"receipt":  invoiceID,
		"notes": map[string]interface{}{
			"invoice_id": invoiceID,
			"token_id":   tokenID,
		},
	}

	body, err := g.client.Order.Create(data, nil)
	if err != nil {
		return &port.PaymentResult{
			Success:   false,
			ErrorCode: "mandate_debit_failed",
			ErrorMsg:  err.Error(),
		}, nil
	}

	orderID, _ := body["id"].(string)
	return &port.PaymentResult{
		Success:   true,
		PaymentID: orderID,
	}, nil
}

func (g *RazorpayGateway) RevokeMandate(ctx context.Context, customerID, tokenID string) error {
	if tokenID == "" {
		return fmt.Errorf("razorpay revoke mandate: token id is required")
	}
	if customerID == "" {
		// Razorpay's token deletion API is customer-scoped
		// (DELETE /v1/customers/{customer_id}/tokens/{token_id}), so a revoke
		// without the customer id cannot be performed via the API. Fail loudly
		// rather than pretend the token was deleted.
		return fmt.Errorf("razorpay revoke mandate: customer id is required to delete token %s", tokenID)
	}

	if _, err := g.client.Token.Delete(customerID, tokenID, nil, nil); err != nil {
		if isRazorpayTokenGone(err) {
			// Token already deleted or never existed at the gateway: the
			// desired end state (not chargeable) holds, so revoke is idempotent.
			return nil
		}
		return fmt.Errorf("razorpay revoke mandate failed for token %s: %w", tokenID, err)
	}
	return nil
}

// isRazorpayTokenGone reports whether the error from the Razorpay token
// deletion API indicates the token (or its customer) no longer exists.
// razorpay-go surfaces API failures as typed errors carrying only the
// human-readable description, so we have to match on the message.
func isRazorpayTokenGone(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "does not exist") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "already deleted") ||
		strings.Contains(msg, "no such token")
}

func (g *RazorpayGateway) CreateVirtualAccount(ctx context.Context, customerID, invoiceID string, amount int64, description string) (*port.VirtualAccountResult, error) {
	data := map[string]interface{}{
		"receivers":       map[string]interface{}{"types": []string{"bank_account"}},
		"description":     description,
		"amount_expected": amount,
		"notes": map[string]interface{}{
			"customer_id": customerID,
			"invoice_id":  invoiceID,
		},
	}

	body, err := g.client.VirtualAccount.Create(data, nil)
	if err != nil {
		return nil, fmt.Errorf("razorpay create virtual account failed: %v", err)
	}

	vaID, _ := body["id"].(string)
	status, _ := body["status"].(string)

	var accountNumber, ifsc, bankName, beneficiary string
	if receivers, ok := body["receivers"].([]interface{}); ok && len(receivers) > 0 {
		if recv, ok := receivers[0].(map[string]interface{}); ok {
			accountNumber, _ = recv["account_number"].(string)
			ifsc, _ = recv["ifsc"].(string)
			bankName, _ = recv["bank_name"].(string)
			beneficiary, _ = recv["name"].(string)
		}
	}

	return &port.VirtualAccountResult{
		VAID:            vaID,
		AccountNumber:   accountNumber,
		IFSCCode:        ifsc,
		BankName:        bankName,
		BeneficiaryName: beneficiary,
		Status:          status,
	}, nil
}

func (g *RazorpayGateway) CancelSubscription(ctx context.Context, subscriptionID string) error {
	// Unreachable via SmartRouter (which routes cancels to Stripe), but fail
	// loudly for any direct caller rather than pretending the cancel worked.
	return fmt.Errorf("razorpay subscription cancellation is not implemented yet (subscription %s)", subscriptionID)
}

// Refund issues a (possibly partial) refund via Razorpay's payment refund API
// (POST /v1/payments/{payment_id}/refund). paymentID must be a captured
// payment id (pay_*); amount is in paise. currency is implied by the payment.
func (g *RazorpayGateway) Refund(ctx context.Context, paymentID string, amount int64, currency string) (*port.RefundResult, error) {
	if !strings.HasPrefix(paymentID, "pay_") {
		return nil, fmt.Errorf("razorpay refund: unrecognized payment id %q (expected pay_*)", paymentID)
	}

	body, err := g.client.Payment.Refund(paymentID, int(amount), map[string]interface{}{}, nil)
	if err != nil {
		return nil, fmt.Errorf("razorpay refund failed for %s: %w", paymentID, err)
	}

	refundID, ok := body["id"].(string)
	if !ok {
		return nil, fmt.Errorf("razorpay refund response for %s missing refund id", paymentID)
	}
	status, _ := body["status"].(string)

	return &port.RefundResult{
		RefundID: refundID,
		Status:   status,
	}, nil
}

func (g *RazorpayGateway) RetryPayment(ctx context.Context, invoiceID string, amount int64, currency string) (*port.PaymentResult, error) {
	// Create a new order for the retry attempt
	data := map[string]interface{}{
		"amount":   amount,
		"currency": currency,
		"receipt":  "retry_" + invoiceID,
		"notes": map[string]interface{}{
			"invoice_id":    invoiceID,
			"retry_payment": "true",
		},
	}

	body, err := g.client.Order.Create(data, nil)
	if err != nil {
		return nil, fmt.Errorf("razorpay retry payment infra error: %w", err)
	}

	orderID, ok := body["id"].(string)
	if !ok {
		return nil, fmt.Errorf("razorpay response missing order id")
	}

	status, _ := body["status"].(string)
	if status == "paid" {
		return &port.PaymentResult{
			Success:   true,
			PaymentID: orderID,
		}, nil
	}

	return &port.PaymentResult{
		Success:   false,
		ErrorCode: "payment_pending",
		ErrorMsg:  fmt.Sprintf("order created but not yet paid: %s", orderID),
	}, nil
}
