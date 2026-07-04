package gateway

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

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

func (g *RazorpayGateway) CreateOrder(ctx context.Context, amount int64, currency string, receipt string) (*port.PaymentOrder, error) {
	data := map[string]interface{}{
		"amount":   amount,
		"currency": currency,
		"receipt":  receipt,
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

func (g *RazorpayGateway) CreateMandate(ctx context.Context, customerEmail, vpa string, maxAmount int64, frequency string) (*port.MandateResult, error) {
	data := map[string]interface{}{
		"type":        "link",
		"amount":      0,
		"currency":    "INR",
		"description": "UPI AutoPay Mandate",
		"subscription_registration": map[string]interface{}{
			"method":    "emandate",
			"auth_type": "netbanking",
			"bank_account": map[string]interface{}{
				"beneficiary_name": customerEmail,
			},
		},
		"customer": map[string]interface{}{
			"email": customerEmail,
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
	shortURL, _ := body["short_url"].(string)
	status, _ := body["status"].(string)

	return &port.MandateResult{
		TokenID:        tokenID,
		SubscriptionID: subID,
		AuthURL:        shortURL,
		Status:         status,
	}, nil
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

func (g *RazorpayGateway) RevokeMandate(ctx context.Context, tokenID string) error {
	// A silent no-op here would leave a mandate chargeable while the system
	// reports it revoked — fail loudly until token deletion is implemented.
	return fmt.Errorf("razorpay mandate revocation is not implemented yet; revoke token %s via the Razorpay dashboard", tokenID)
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
