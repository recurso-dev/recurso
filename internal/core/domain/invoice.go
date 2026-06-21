package domain

import (
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type InvoiceStatus string

const (
	InvoiceStatusDraft         InvoiceStatus = "draft"
	InvoiceStatusOpen          InvoiceStatus = "open"
	InvoiceStatusPaid          InvoiceStatus = "paid"
	InvoiceStatusVoid          InvoiceStatus = "void"
	InvoiceStatusUncollectible InvoiceStatus = "uncollectible"
	InvoiceStatusPastDue       InvoiceStatus = "past_due"
)

type Invoice struct {
	ID             uuid.UUID  `json:"id"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	SubscriptionID *uuid.UUID `json:"subscription_id,omitempty"` // Nullable for one-off invoices
	CustomerID     uuid.UUID  `json:"customer_id"`
	InvoiceNumber  string     `json:"invoice_number"`
	BillingReason  string     `json:"billing_reason" db:"billing_reason"`
	AmountDue      int64      `json:"amount_due" db:"amount_due"`
	AmountPaid     int64      `json:"amount_paid" db:"amount_paid"`
	// Financials
	Currency  string `json:"currency" db:"currency"`
	Subtotal  int64  `json:"subtotal" db:"subtotal"`
	TaxAmount int64  `json:"tax_amount" db:"tax_amount"` // Renamed from Tax
	Total     int64  `json:"total" db:"total"`

	// Compliance P24
	IGSTAmount     int64  `json:"igst_amount" db:"igst_amount"`
	CGSTAmount     int64  `json:"cgst_amount" db:"cgst_amount"`
	SGSTAmount     int64  `json:"sgst_amount" db:"sgst_amount"`
	HSNCode        string `json:"hsn_code" db:"hsn_code"`
	IRN            string `json:"irn" db:"irn"`
	AckNo          string `json:"ack_no" db:"ack_no"`
	SignedQRCode   string `json:"signed_qr_code" db:"signed_qr_code"`     // P25
	EInvoiceStatus       string     `json:"e_invoice_status" db:"e_invoice_status"`               // P25
	AckDate              string     `json:"ack_date" db:"ack_date"`                               // P25: IRP acknowledgement date
	EInvoiceRetryCount   int        `json:"e_invoice_retry_count" db:"e_invoice_retry_count"`     // P25: E-invoice retry attempts
	EInvoiceNextRetryAt  *time.Time `json:"e_invoice_next_retry_at" db:"e_invoice_next_retry_at"` // P25: Next e-invoice retry time
	EInvoiceErrorMessage string     `json:"e_invoice_error_message" db:"e_invoice_error_message"` // P25: Last error from IRP
	TDSAmount            int64      `json:"tds_amount" db:"tds_amount"`                           // P25: Deducted by customer

	Status InvoiceStatus `json:"status" db:"status"`

	CreatedAt    time.Time  `json:"created_at"`
	DueDate      time.Time  `json:"due_date"`
	PaidAt       *time.Time `json:"paid_at,omitempty"`
	PaymentTerms string     `json:"payment_terms"` // P15

	// Retry Logic
	NextRetryAt *time.Time `json:"next_retry_at,omitempty"`
	RetryCount  int        `json:"retry_count"`

	// Smart Dunning (RL feedback loop)
	DunningActionID   string `json:"dunning_action_id,omitempty" db:"dunning_action_id"`
	DunningContextKey string `json:"dunning_context_key,omitempty" db:"dunning_context_key"`
	LastPaymentError  string `json:"last_payment_error,omitempty" db:"last_payment_error"`
	DunningManagedBy  string `json:"dunning_managed_by,omitempty" db:"dunning_managed_by"`
}

// OverdueInvoice contains invoice info with customer details for dunning
type OverdueInvoice struct {
	ID            uuid.UUID
	TenantID      uuid.UUID
	CustomerID    uuid.UUID
	CustomerName  string
	CustomerEmail string
	InvoiceNumber string
	Amount        int64
	Currency      string
	DueDate       time.Time
	RetryCount    int
	NextRetryAt   *time.Time
}

// CalculateDueDate returns the due date based on payment terms (e.g., "net15", "net30")
func CalculateDueDate(start time.Time, terms string) time.Time {
	if terms == "" || terms == "net0" || terms == "due_on_receipt" {
		return start
	}

	// Parse "netX"
	if strings.HasPrefix(terms, "net") {
		daysStr := strings.TrimPrefix(terms, "net")
		days, err := strconv.Atoi(daysStr)
		if err == nil {
			return start.AddDate(0, 0, days)
		}
	}

	return start
}
