package gateway

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/razorpay/razorpay-go"
	"github.com/recur-so/recurso/internal/core/port"
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
	if startAt != nil {
		data["start_at"] = *startAt
	} else {
		// Default to immediate (or rather, now + 1hr delay required by Razorpay sometimes? 
		// Actually Razorpay 'start_at' is optional, if omitted it starts immediately)
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
