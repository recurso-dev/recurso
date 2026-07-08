package port

import (
	"context"

	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// EInvoiceRequest wraps all data needed to generate an IRN via NIC/IRP
type EInvoiceRequest struct {
	Invoice *domain.Invoice
	Seller  EInvoiceSeller
	Buyer   EInvoiceBuyer
	Items   []EInvoiceItem
}

type EInvoiceSeller struct {
	GSTIN     string `json:"gstin"`
	LegalName string `json:"legal_name"`
	TradeName string `json:"trade_name"`
	Address   string `json:"address"`
	Location  string `json:"location"`
	PinCode   string `json:"pin_code"`
	StateCode string `json:"state_code"`
}

type EInvoiceBuyer struct {
	GSTIN     string `json:"gstin"`
	LegalName string `json:"legal_name"`
	TradeName string `json:"trade_name"`
	Address   string `json:"address"`
	Location  string `json:"location"`
	PinCode   string `json:"pin_code"`
	StateCode string `json:"state_code"`
	Place     string `json:"place"` // Place of supply state code
}

type EInvoiceItem struct {
	SlNo        int     `json:"sl_no"`
	Description string  `json:"description"`
	HSNCode     string  `json:"hsn_code"`
	Quantity    float64 `json:"quantity"`
	Unit        string  `json:"unit"`
	UnitPrice   int64   `json:"unit_price"`
	TotalAmount int64   `json:"total_amount"`
	// TaxableAmount is the post-discount assessable value the GST was computed
	// on (AssAmt in the IRP schema). Equals TotalAmount when no discount applies.
	TaxableAmount int64   `json:"taxable_amount"`
	TaxRate       float64 `json:"tax_rate"`
	IGSTAmount    int64   `json:"igst_amount"`
	CGSTAmount    int64   `json:"cgst_amount"`
	SGSTAmount    int64   `json:"sgst_amount"`
}

type EInvoiceResponse struct {
	IRN           string `json:"irn"`
	SignedQRCode  string `json:"signed_qr_code"`
	Status        string `json:"status"` // GENERATED, FAILED
	AckNo         string `json:"ack_no"`
	AckDate       string `json:"ack_date"`
	SignedInvoice string `json:"signed_invoice"`
	ErrorCode     string `json:"error_code"`
	ErrorMessage  string `json:"error_message"`
}

// CancelIRNRequest contains data for cancelling an IRN
type CancelIRNRequest struct {
	IRN        string `json:"irn"`
	CancelCode int    `json:"cancel_code"` // 1=Duplicate, 2=Data Entry Mistake, 3=Order Cancelled, 4=Others
	Reason     string `json:"reason"`
}

type GSPAdapter interface {
	GenerateIRN(ctx context.Context, invoice *domain.Invoice) (*EInvoiceResponse, error)
	GenerateIRNFull(ctx context.Context, req *EInvoiceRequest) (*EInvoiceResponse, error)
	CancelIRN(ctx context.Context, irn string, reason string) error
	GetIRNByDocDetails(ctx context.Context, docType, docNum, docDate string) (*EInvoiceResponse, error)
}
