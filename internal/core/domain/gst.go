package domain

// GSTConfig holds GST configuration for tax calculation
type GSTConfig struct {
	// Seller's GSTIN
	SellerGSTIN string `json:"seller_gstin"`
	// Seller's state code (first 2 digits of GSTIN)
	SellerStateCode string `json:"seller_state_code"`
	// SAC code for SaaS services
	SACCode string `json:"sac_code"` // Default: 998314 for cloud services
	// Tax rate percentage (e.g., 18 for 18%)
	TaxRatePercent float64 `json:"tax_rate_percent"`
}

// GSTBreakdown contains the calculated GST amounts
type GSTBreakdown struct {
	Subtotal      int64   `json:"subtotal"`        // Amount before tax
	TaxableAmount int64   `json:"taxable_amount"`  // Amount on which tax is calculated
	CGSTRate      float64 `json:"cgst_rate"`       // CGST rate (half of total)
	CGSTAmount    int64   `json:"cgst_amount"`     // Central GST
	SGSTRate      float64 `json:"sgst_rate"`       // SGST rate (half of total)
	SGSTAmount    int64   `json:"sgst_amount"`     // State GST
	IGSTRate      float64 `json:"igst_rate"`       // IGST rate (full rate)
	IGSTAmount    int64   `json:"igst_amount"`     // Integrated GST
	TotalTax      int64   `json:"total_tax"`       // Total tax amount
	GrandTotal    int64   `json:"grand_total"`     // Subtotal + tax
	IsInterState  bool    `json:"is_inter_state"`  // True if IGST, false if CGST+SGST
	PlaceOfSupply string  `json:"place_of_supply"` // State code where service is consumed
}

// CalculateGST computes GST breakdown based on seller and buyer state
func CalculateGST(subtotal int64, sellerStateCode, buyerStateCode string, taxRatePercent float64) GSTBreakdown {
	breakdown := GSTBreakdown{
		Subtotal:      subtotal,
		TaxableAmount: subtotal,
		PlaceOfSupply: buyerStateCode,
	}

	// Determine if inter-state (IGST) or intra-state (CGST+SGST)
	isInterState := sellerStateCode != buyerStateCode
	breakdown.IsInterState = isInterState

	// Calculate tax amounts (in paise/cents)
	totalTax := int64(float64(subtotal) * taxRatePercent / 100)

	if isInterState {
		// IGST for inter-state transactions
		breakdown.IGSTRate = taxRatePercent
		breakdown.IGSTAmount = totalTax
		breakdown.CGSTRate = 0
		breakdown.CGSTAmount = 0
		breakdown.SGSTRate = 0
		breakdown.SGSTAmount = 0
	} else {
		// CGST + SGST for intra-state transactions (split equally)
		halfRate := taxRatePercent / 2
		halfTax := totalTax / 2
		// Handle odd amounts - give extra paisa to CGST
		remainder := totalTax % 2

		breakdown.CGSTRate = halfRate
		breakdown.CGSTAmount = halfTax + remainder
		breakdown.SGSTRate = halfRate
		breakdown.SGSTAmount = halfTax
		breakdown.IGSTRate = 0
		breakdown.IGSTAmount = 0
	}

	breakdown.TotalTax = totalTax
	breakdown.GrandTotal = subtotal + totalTax

	return breakdown
}

// IndianStateCode maps state names to GST state codes
var IndianStateCode = map[string]string{
	"andhra_pradesh":     "37",
	"arunachal_pradesh":  "12",
	"assam":              "18",
	"bihar":              "10",
	"chhattisgarh":       "22",
	"delhi":              "07",
	"goa":                "30",
	"gujarat":            "24",
	"haryana":            "06",
	"himachal_pradesh":   "02",
	"jharkhand":          "20",
	"karnataka":          "29",
	"kerala":             "32",
	"madhya_pradesh":     "23",
	"maharashtra":        "27",
	"manipur":            "14",
	"meghalaya":          "17",
	"mizoram":            "15",
	"nagaland":           "13",
	"odisha":             "21",
	"punjab":             "03",
	"rajasthan":          "08",
	"sikkim":             "11",
	"tamil_nadu":         "33",
	"telangana":          "36",
	"tripura":            "16",
	"uttar_pradesh":      "09",
	"uttarakhand":        "05",
	"west_bengal":        "19",
	"andaman_nicobar":    "35",
	"chandigarh":         "04",
	"dadra_nagar_haveli": "26",
	"daman_diu":          "25",
	"jammu_kashmir":      "01",
	"ladakh":             "38",
	"lakshadweep":        "31",
	"puducherry":         "34",
}

// GetStateCodeFromGSTIN extracts state code from GSTIN (first 2 digits)
func GetStateCodeFromGSTIN(gstin string) string {
	if len(gstin) >= 2 {
		return gstin[:2]
	}
	return ""
}

// ValidateGSTIN basic validation for GSTIN format
func ValidateGSTIN(gstin string) bool {
	// GSTIN is 15 characters: 2 digits state code + 10 char PAN + 1 entity code + 1 check digit + 1 default 'Z'
	if len(gstin) != 15 {
		return false
	}
	// First 2 must be digits (state code)
	if gstin[0] < '0' || gstin[0] > '9' || gstin[1] < '0' || gstin[1] > '9' {
		return false
	}
	return true
}

// DefaultSACCode for SaaS/Cloud services
const DefaultSACCode = "998314"

// DefaultGSTRate for software services in India
const DefaultGSTRate = 18.0

// TenantGSTConfig holds persisted GST configuration for a tenant
type TenantGSTConfig struct {
	TenantID  string  `json:"tenant_id" db:"tenant_id"`
	GSTIN     string  `json:"gstin" db:"gstin"`
	StateCode string  `json:"state_code" db:"state_code"`
	StateName string  `json:"state_name" db:"state_name"`
	SACCode   string  `json:"sac_code" db:"sac_code"`
	GSTRate   float64 `json:"gst_rate" db:"gst_rate"`
	PAN       string  `json:"pan" db:"pan"`
	LegalName string  `json:"legal_name" db:"legal_name"`
	TradeName string  `json:"trade_name" db:"trade_name"`
	Address   string  `json:"address" db:"address"`
	HasLUT    bool    `json:"has_lut" db:"has_lut"` // Letter of Undertaking for exports
}
