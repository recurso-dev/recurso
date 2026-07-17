package domain

import (
	"time"

	"github.com/google/uuid"
)

type VirtualAccount struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	TenantID        uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	CustomerID      uuid.UUID  `json:"customer_id" db:"customer_id"`
	InvoiceID       *uuid.UUID `json:"invoice_id,omitempty" db:"invoice_id"`
	AccountNumber   string     `json:"account_number" db:"account_number"`
	IFSCCode        string     `json:"ifsc_code" db:"ifsc_code"`
	BankName        string     `json:"bank_name" db:"bank_name"`
	BeneficiaryName string     `json:"beneficiary_name" db:"beneficiary_name"`
	RazorpayVAID    string     `json:"razorpay_va_id" db:"razorpay_va_id"`
	Status          string     `json:"status" db:"status"`
	AmountExpected  int64      `json:"amount_expected" db:"amount_expected"`
	AmountReceived  int64      `json:"amount_received" db:"amount_received"`
	ClosedAt        *time.Time `json:"closed_at,omitempty" db:"closed_at"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
}

type OfflinePayment struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	TenantID        uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	CustomerID      uuid.UUID  `json:"customer_id" db:"customer_id"`
	InvoiceID       *uuid.UUID `json:"invoice_id,omitempty" db:"invoice_id"`
	PaymentType     string     `json:"payment_type" db:"payment_type"`
	Amount          int64      `json:"amount" db:"amount"`
	TDSAmount       int64      `json:"tds_amount" db:"tds_amount"` // tax deducted at source by the customer — settles the invoice but is not cash received
	Currency        string     `json:"currency" db:"currency"`
	ReferenceNumber string     `json:"reference_number,omitempty" db:"reference_number"`
	Notes           string     `json:"notes,omitempty" db:"notes"`
	RecordedBy      string     `json:"recorded_by,omitempty" db:"recorded_by"`
	RecordedAt      time.Time  `json:"recorded_at" db:"recorded_at"`
}
