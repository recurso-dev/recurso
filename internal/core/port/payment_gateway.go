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

// PaymentResult represents the outcome of a payment retry attempt
type PaymentResult struct {
	Success   bool
	PaymentID string
	ErrorCode string // "insufficient_funds", "card_expired", "network_error", etc.
	ErrorMsg  string
}

type PaymentGateway interface {
	CreateOrder(ctx context.Context, amount int64, currency string, receipt string) (*PaymentOrder, error)
	VerifyPayment(ctx context.Context, orderID, paymentID, signature string) error
	CreateSubscription(ctx context.Context, planID string, totalCount int, customerEmail string, startAt *int64, currency string) (string, error)
	RetryPayment(ctx context.Context, invoiceID string, amount int64, currency string) (*PaymentResult, error)
}
