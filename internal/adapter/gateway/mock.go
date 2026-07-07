package gateway

import (
	"context"
	"fmt"
	"math/rand"
	"sync"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

type MockGateway struct {
	mu          sync.Mutex
	revokeCalls []MockRevokeCall
	RevokeErr   error // if set, RevokeMandate returns this error
	refundCalls []MockRefundCall
	RefundErr   error // if set, Refund returns this error
}

// MockRevokeCall records the arguments of a RevokeMandate invocation.
type MockRevokeCall struct {
	CustomerID string
	TokenID    string
}

// MockRefundCall records the arguments of a Refund invocation.
type MockRefundCall struct {
	PaymentID string
	Amount    int64
	Currency  string
}

func NewMockGateway() *MockGateway {
	return &MockGateway{}
}

func (g *MockGateway) CreateOrder(ctx context.Context, amount int64, currency string, receipt string, invoiceID string) (*port.PaymentOrder, error) {
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
		CustomerID:     "cust_mock_" + uuid.New().String(),
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

func (g *MockGateway) RevokeMandate(ctx context.Context, customerID, tokenID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.revokeCalls = append(g.revokeCalls, MockRevokeCall{CustomerID: customerID, TokenID: tokenID})
	return g.RevokeErr
}

// RevokeCalls returns a copy of the recorded RevokeMandate invocations.
func (g *MockGateway) RevokeCalls() []MockRevokeCall {
	g.mu.Lock()
	defer g.mu.Unlock()
	calls := make([]MockRevokeCall, len(g.revokeCalls))
	copy(calls, g.revokeCalls)
	return calls
}

func (g *MockGateway) Refund(ctx context.Context, paymentID string, amount int64, currency string) (*port.RefundResult, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.refundCalls = append(g.refundCalls, MockRefundCall{PaymentID: paymentID, Amount: amount, Currency: currency})
	if g.RefundErr != nil {
		return nil, g.RefundErr
	}
	return &port.RefundResult{
		RefundID: "rfnd_mock_" + uuid.New().String(),
		Status:   "processed",
	}, nil
}

// RefundCalls returns a copy of the recorded Refund invocations.
func (g *MockGateway) RefundCalls() []MockRefundCall {
	g.mu.Lock()
	defer g.mu.Unlock()
	calls := make([]MockRefundCall, len(g.refundCalls))
	copy(calls, g.refundCalls)
	return calls
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

func (g *MockGateway) CancelSubscription(ctx context.Context, subscriptionID string) error {
	return nil
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
