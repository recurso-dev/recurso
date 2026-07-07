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

// RefundResult represents the outcome of a gateway refund call.
type RefundResult struct {
	RefundID string // gateway-side refund id (Stripe re_*, Razorpay rfnd_*)
	Status   string // gateway-reported status, e.g. "succeeded", "processed", "pending"
}

type MandateResult struct {
	TokenID        string
	SubscriptionID string
	CustomerID     string // gateway-side customer id (e.g. Razorpay cust_*), needed for token revocation
	AuthURL        string
	Status         string
}

type PaymentGateway interface {
	CreateOrder(ctx context.Context, amount int64, currency string, receipt string, invoiceID string) (*PaymentOrder, error)
	VerifyPayment(ctx context.Context, orderID, paymentID, signature string) error
	CreateSubscription(ctx context.Context, planID string, totalCount int, customerEmail string, startAt *int64, currency string) (string, error)
	RetryPayment(ctx context.Context, invoiceID string, amount int64, currency string) (*PaymentResult, error)
	CreateMandate(ctx context.Context, customerEmail, vpa string, maxAmount int64, frequency string) (*MandateResult, error)
	ExecuteMandateDebit(ctx context.Context, tokenID string, amount int64, currency, invoiceID string) (*PaymentResult, error)
	// RevokeMandate deletes the recurring-payment token at the gateway.
	// customerID is the gateway-side customer id (required by Razorpay's
	// DELETE /v1/customers/{customer_id}/tokens/{token_id} API).
	// Implementations must treat an already-deleted token as success.
	RevokeMandate(ctx context.Context, customerID, tokenID string) error
	CreateVirtualAccount(ctx context.Context, customerID, invoiceID string, amount int64, description string) (*VirtualAccountResult, error)
	CancelSubscription(ctx context.Context, subscriptionID string) error
	// Refund returns money for a previously captured payment. paymentID is the
	// gateway-side payment identifier recorded when the invoice was paid
	// (Stripe pi_*/ch_*, Razorpay pay_*). amount is in minor units and may be
	// less than the original charge (partial refund).
	Refund(ctx context.Context, paymentID string, amount int64, currency string) (*RefundResult, error)
}

type VirtualAccountResult struct {
	VAID            string
	AccountNumber   string
	IFSCCode        string
	BankName        string
	BeneficiaryName string
	Status          string
}
