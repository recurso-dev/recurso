package gateway

import (
	"context"
	"fmt"

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
