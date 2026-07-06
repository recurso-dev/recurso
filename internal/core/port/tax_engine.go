package port

import "context"

// TaxRequest contains the inputs needed to calculate tax
type TaxRequest struct {
	Amount        int64  // Amount in cents/paise (lowest currency unit)
	Currency      string // ISO 3-letter code
	HSNCode       string // Harmonized System of Nomenclature code (India)
	SellerState   string // Seller's state/province
	BuyerState    string // Buyer's state/province
	BuyerZip      string // Buyer's ZIP/postal code (US sales tax; ignored by the GST and VAT engines)
	BuyerCountry  string // Buyer's country (ISO 2-letter)
	SellerCountry string // Seller's country (ISO 2-letter)
	IsBusiness    bool   // B2B vs B2C
	IsSEZ         bool   // Special Economic Zone (India)
	IsExport      bool   // Export transaction
	IsRCM         bool   // Reverse Charge Mechanism
}

// TaxCalculation contains the computed tax breakdown
type TaxCalculation struct {
	TotalTax      int64   // Total tax in lowest currency unit
	TaxRate       float64 // Effective tax rate (0.0 to 1.0)
	TaxType       string  // e.g., "gst_intra", "gst_inter", "vat", "sales_tax", "exempt"
	ReverseCharge bool    // Whether RCM applies

	// GST-specific breakdowns
	IGST int64 `json:"igst,omitempty"`
	CGST int64 `json:"cgst,omitempty"`
	SGST int64 `json:"sgst,omitempty"`

	// VAT-specific
	VATRate   float64 `json:"vat_rate,omitempty"`
	VATAmount int64   `json:"vat_amount,omitempty"`

	Note string `json:"note,omitempty"` // Additional info (e.g., "Integrate TaxJar for real rates")
}

// TaxEngine calculates tax based on jurisdiction rules
type TaxEngine interface {
	CalculateTax(ctx context.Context, req *TaxRequest) (*TaxCalculation, error)
	GetApplicableRate(ctx context.Context, req *TaxRequest) (float64, error)
}
