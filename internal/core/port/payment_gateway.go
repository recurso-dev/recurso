package port

import (
	"context"
)

type PaymentOrder struct {
	ID       string
	Amount   int64
	Currency string
	Receipt  string
	// ClientSecret is the gateway-side secret the frontend needs to confirm the
	// payment client-side (Stripe PaymentIntent client_secret). Empty for
	// gateways that don't use a client-confirmed flow (e.g. Razorpay orders).
	ClientSecret string
	// Gateway identifies which gateway created the order ("stripe", "razorpay",
	// "mock"), so callers never infer it from the order-ID format — mock orders
	// share Razorpay's "order_" prefix.
	Gateway string
}

// PaymentStatus is a read-back of a gateway payment/order, used to verify a
// checkout server-side before an invoice is marked paid. InvoiceID is the
// gateway metadata linking the payment back to a Recurso invoice.
type PaymentStatus struct {
	Status         string // gateway-reported, e.g. "succeeded", "processing", "requires_payment_method"
	InvoiceID      string // metadata invoice_id set at CreateOrder time
	PaymentID      string // gateway payment identifier (Stripe pi_*)
	AmountReceived int64  // minor units actually received (0 until settled)
}

// SavedCard is the result of finalizing a portal SetupIntent: the reusable
// payment method saved for a customer, plus the Recurso customer_id carried in
// the intent's metadata so the caller can bind it to the right customer.
type SavedCard struct {
	Status          string // SetupIntent status, e.g. "succeeded"
	CustomerID      string // Recurso customer id from the intent metadata
	PaymentMethodID string // pm_* to charge for future invoices
	Brand           string
	Last4           string
	ExpMonth        int
	ExpYear         int
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

// MandateDebitRequest carries everything a gateway needs to initiate a recurring
// auto-debit against a saved token. Razorpay's recurring-charge API requires the
// gateway-side customer id plus the customer's email/contact, not just the token.
type MandateDebitRequest struct {
	TokenID            string // gateway token (Razorpay token_*)
	RazorpayCustomerID string // gateway customer id (Razorpay cust_*)
	Email              string // customer email (required by create/recurring)
	Contact            string // customer phone/contact (required by create/recurring)
	Amount             int64
	Currency           string
	InvoiceID          string // Recurso invoice id — carried in order notes for webhook settlement
	// IdempotencyKey is stable across retries of the SAME billing cycle, so if a
	// debit succeeds at the gateway but a later local step fails and the cycle is
	// re-attempted, the gateway dedupes instead of debiting the customer twice.
	IdempotencyKey string
}

type PaymentGateway interface {
	CreateOrder(ctx context.Context, amount int64, currency string, receipt string, invoiceID string) (*PaymentOrder, error)
	VerifyPayment(ctx context.Context, orderID, paymentID, signature string) error
	CreateSubscription(ctx context.Context, planID string, totalCount int, customerEmail string, startAt *int64, currency string) (string, error)
	RetryPayment(ctx context.Context, invoiceID string, amount int64, currency string) (*PaymentResult, error)
	CreateMandate(ctx context.Context, customerEmail, customerContact, vpa string, maxAmount int64, frequency string) (*MandateResult, error)
	ExecuteMandateDebit(ctx context.Context, req MandateDebitRequest) (*PaymentResult, error)
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
