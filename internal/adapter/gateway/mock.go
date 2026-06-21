package gateway

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/port"
)

type MockGateway struct{}

func NewMockGateway() *MockGateway {
	return &MockGateway{}
}

func (g *MockGateway) CreateOrder(ctx context.Context, amount int64, currency string, receipt string) (*port.PaymentOrder, error) {
	return &port.PaymentOrder{
		ID:       "order_" + uuid.New().String(),
		Amount:   amount,
		Currency: currency,
		Receipt:  receipt,
	}, nil
}

func (g *MockGateway) VerifyPayment(ctx context.Context, orderID, paymentID, signature string) error {
	if signature == "fail" {
		return fmt.Errorf("mock verification failed")
	}
	return nil
}

func (g *MockGateway) CreateSubscription(ctx context.Context, planID string, totalCount int, customerEmail string, startAt *int64, currency string) (string, error) {
	return "sub_mock_" + uuid.New().String(), nil
}

func (g *MockGateway) CreateMandate(ctx context.Context, customerEmail, vpa string, maxAmount int64, frequency string) (*port.MandateResult, error) {
	return &port.MandateResult{
		TokenID:        "tok_mock_" + uuid.New().String(),
		SubscriptionID: "sub_mock_" + uuid.New().String(),
		AuthURL:        "https://mock.razorpay.com/authorize/" + uuid.New().String(),
		Status:         "created",
	}, nil
}

func (g *MockGateway) ExecuteMandateDebit(ctx context.Context, tokenID string, amount int64, currency, invoiceID string) (*port.PaymentResult, error) {
	return &port.PaymentResult{
		Success:   true,
		PaymentID: "pay_mock_" + uuid.New().String(),
	}, nil
}

func (g *MockGateway) RevokeMandate(ctx context.Context, tokenID string) error {
	return nil
}

func (g *MockGateway) CreateVirtualAccount(ctx context.Context, customerID, invoiceID string, amount int64, description string) (*port.VirtualAccountResult, error) {
	return &port.VirtualAccountResult{
		VAID:            "va_mock_" + uuid.New().String(),
		AccountNumber:   "1112109002233556",
		IFSCCode:        "RATN0VAAPIS",
		BankName:        "RBL Bank",
		BeneficiaryName: "Recurso Payments",
		Status:          "active",
	}, nil
}

func (g *MockGateway) RetryPayment(ctx context.Context, invoiceID string, amount int64, currency string) (*port.PaymentResult, error) {
	// ~40% success rate for mock simulation
	if rand.Float64() < 0.4 {
		return &port.PaymentResult{
			Success:   true,
			PaymentID: "pay_mock_" + uuid.New().String(),
		}, nil
	}

	errorCodes := []string{"insufficient_funds", "card_expired", "card_declined", "processing_error"}
	code := errorCodes[rand.Intn(len(errorCodes))]
	return &port.PaymentResult{
		Success:   false,
		ErrorCode: code,
		ErrorMsg:  "mock payment failed: " + code,
	}, nil
}
