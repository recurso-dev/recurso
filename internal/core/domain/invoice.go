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

// BillingReason records why an invoice was created (Stripe-aligned). Set at every
// creation site and persisted to invoices.billing_reason; empty only for rows
// created before the column existed.
const (
	BillingReasonSubscriptionCreate = "subscription_create" // first invoice when a subscription starts
	BillingReasonSubscriptionCycle  = "subscription_cycle"  // recurring renewal / trial conversion / advance
	BillingReasonSubscriptionUpdate = "subscription_update" // mid-cycle change (proration)
	BillingReasonMandateDebit       = "mandate_debit"       // UPI-mandate auto-debit
	BillingReasonGiftPurchase       = "gift_purchase"       // prepaid gift
	BillingReasonManual             = "manual"              // one-off / quote conversion
	BillingReasonProgressiveUsage   = "progressive_usage"   // interim usage bill when accrued usage crosses a threshold (A5)
)

type Invoice struct {
	ID       uuid.UUID `json:"id"`
	TenantID uuid.UUID `json:"tenant_id"`
	// EntityID is the issuing legal entity (Multi-Entity Books). Nil resolves to
	// the tenant's primary entity, so single-entity tenants are unchanged.
	EntityID       *uuid.UUID `json:"entity_id,omitempty"`
	SubscriptionID *uuid.UUID `json:"subscription_id,omitempty"` // Nullable for one-off invoices
	CustomerID     uuid.UUID  `json:"customer_id"`
	InvoiceNumber  string     `json:"invoice_number"`
	BillingReason  string     `json:"billing_reason" db:"billing_reason"`
	AmountDue      int64      `json:"amount_due" db:"amount_due"`
	AmountPaid     int64      `json:"amount_paid" db:"amount_paid"`
	// CreditApplied is account credit (adjustment credit notes) applied to this
	// invoice at billing time (ENG-153). Total stays gross; amount due =
	// total - amount_paid - credit_applied.
	CreditApplied int64 `json:"credit_applied" db:"credit_applied"`
	// Financials
	Currency  string `json:"currency" db:"currency"`
	Subtotal  int64  `json:"subtotal" db:"subtotal"`
	TaxAmount int64  `json:"tax_amount" db:"tax_amount"` // Renamed from Tax
	// TaxType is the resolved tax evaluation reason (e.g. "sales_tax",
	// "sales_tax_exempt", "no_nexus", "gst_intra", "vat"). Set once at invoice
	// creation and read by the liability report via a direct column query; it is
	// NOT hydrated when an invoice is scanned back (reads leave it empty). See
	// Track D · D3c.
	TaxType string `json:"tax_type" db:"tax_type"`
	Total   int64  `json:"total" db:"total"`

	// Compliance P24
	IGSTAmount           int64      `json:"igst_amount" db:"igst_amount"`
	CGSTAmount           int64      `json:"cgst_amount" db:"cgst_amount"`
	SGSTAmount           int64      `json:"sgst_amount" db:"sgst_amount"`
	HSNCode              string     `json:"hsn_code" db:"hsn_code"`
	IRN                  string     `json:"irn" db:"irn"`
	AckNo                string     `json:"ack_no" db:"ack_no"`
	SignedQRCode         string     `json:"signed_qr_code" db:"signed_qr_code"`                   // P25
	EInvoiceStatus       string     `json:"e_invoice_status" db:"e_invoice_status"`               // P25
	AckDate              string     `json:"ack_date" db:"ack_date"`                               // P25: IRP acknowledgement date
	EInvoiceRetryCount   int        `json:"e_invoice_retry_count" db:"e_invoice_retry_count"`     // P25: E-invoice retry attempts
	EInvoiceNextRetryAt  *time.Time `json:"e_invoice_next_retry_at" db:"e_invoice_next_retry_at"` // P25: Next e-invoice retry time
	EInvoiceErrorMessage string     `json:"e_invoice_error_message" db:"e_invoice_error_message"` // P25: Last error from IRP
	TDSAmount            int64      `json:"tds_amount" db:"tds_amount"`                           // P25: Deducted by customer

	Status InvoiceStatus `json:"status" db:"status"`

	// GatewayPaymentID is the gateway-side identifier of the payment that
	// settled this invoice (Stripe pi_*/ch_*, Razorpay pay_*). Populated by
	// the payment-success webhook handlers; empty for offline payments and
	// invoices paid before this field existed. Required for API refunds.
	GatewayPaymentID string `json:"gateway_payment_id,omitempty" db:"gateway_payment_id"`

	// MandateCycleKey is the per-cycle claim key for mandate auto-debits
	// ("md-<mandate_id>-<cycle>"). A UNIQUE index makes the OPEN invoice the
	// durable at-most-once claim: a re-attempt of the same cycle can't create a
	// second invoice, so it can't charge again (ENG-164). Empty for all
	// non-mandate invoices, which the partial unique index leaves unconstrained.
	MandateCycleKey string `json:"-" db:"mandate_cycle_key"`

	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is bumped by InvoiceRepository.Update. The accounting sync
	// compares it against the invoice's mapping to skip unchanged invoices;
	// a zero value means "unknown" and always syncs.
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
	DueDate      time.Time  `json:"due_date"`
	PaidAt       *time.Time `json:"paid_at,omitempty"`
	PaymentTerms string     `json:"payment_terms"` // P15

	// Multi-Currency (FX)
	ExchangeRate      float64 `json:"exchange_rate,omitempty" db:"exchange_rate"`
	BaseCurrencyTotal int64   `json:"base_currency_total,omitempty" db:"base_currency_total"`
	BaseCurrency      string  `json:"base_currency,omitempty" db:"base_currency"`

	// Retry Logic
	NextRetryAt *time.Time `json:"next_retry_at,omitempty"`
	RetryCount  int        `json:"retry_count"`

	// Smart Dunning (RL feedback loop)
	DunningActionID   string `json:"dunning_action_id,omitempty" db:"dunning_action_id"`
	DunningContextKey string `json:"dunning_context_key,omitempty" db:"dunning_context_key"`
	LastPaymentError  string `json:"last_payment_error,omitempty" db:"last_payment_error"`
	DunningManagedBy  string `json:"dunning_managed_by,omitempty" db:"dunning_managed_by"`

	// Dunning Campaign (payment wall)
	PaymentWallActive bool `json:"payment_wall_active" db:"payment_wall_active"`

	// Itemization (per-product HSN codes & itemized invoice tax, Phase 1).
	// Populated on generation (before persistence) and hydrated on read.
	// Each line carries its own HSN/SAC code and GST breakdown; the line
	// amounts and per-line taxes reconcile exactly to Subtotal/TaxAmount.
	LineItems []InvoiceItem `json:"line_items,omitempty" db:"-"`
}

// InvoiceItem is a single itemized line on an invoice. Amounts are in the
// invoice's lowest currency unit (paise/cents). TaxRate is the effective GST
// rate as a percent (e.g. 18.0). In Phase 1 every line uses the tenant SAC as
// its HSNCode; per-product HSN arrives in Phase 2.
type InvoiceItem struct {
	ID            uuid.UUID `json:"id" db:"id"`
	InvoiceID     uuid.UUID `json:"invoice_id" db:"invoice_id"`
	Description   string    `json:"description" db:"description"`
	HSNCode       string    `json:"hsn_code" db:"hsn_code"`
	Quantity      int       `json:"quantity" db:"quantity"`
	UnitAmount    int64     `json:"unit_amount" db:"unit_amount"`
	Amount        int64     `json:"amount" db:"amount"`
	TaxRate       float64   `json:"tax_rate" db:"tax_rate"`
	CGSTAmount    int64     `json:"cgst_amount" db:"cgst_amount"`
	SGSTAmount    int64     `json:"sgst_amount" db:"sgst_amount"`
	IGSTAmount    int64     `json:"igst_amount" db:"igst_amount"`
	TaxableAmount int64     `json:"taxable_amount" db:"taxable_amount"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
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
	// IsMandate marks a UPI-mandate auto-debit invoice (it carries a
	// mandate_cycle_key). The dunning worker's gateway retry can't collect one, so
	// the scheduler keeps it on email dunning instead (ENG-168).
	IsMandate bool
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
