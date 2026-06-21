package port

import (
	"context"
)

type PaymentOrder struct {
	ID       string
	Amount   int64
	Currency string
	Receipt  string
}

type PaymentGateway interface {
	CreateOrder(ctx context.Context, amount int64, currency string, receipt string) (*PaymentOrder, error)
	VerifyPayment(ctx context.Context, orderID, paymentID, signature string) error
	CreateSubscription(ctx context.Context, planID string, totalCount int, customerEmail string, startAt *int64, currency string) (string, error)
}
