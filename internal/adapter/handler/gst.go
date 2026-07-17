package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

// GSTConfig represents GST configuration for a tenant
type GSTConfig struct {
	GSTIN     string  `json:"gstin"`
	StateCode string  `json:"state_code"`
	StateName string  `json:"state_name"`
	SACCode   string  `json:"sac_code"`
	GSTRate   float64 `json:"gst_rate"`
	PAN       string  `json:"pan"`
	LegalName string  `json:"legal_name"`
	TradeName string  `json:"trade_name"`
	Address   string  `json:"address"`
	HasLUT    bool    `json:"has_lut"` // Letter of Undertaking for exports
}

// GSTHandler handles GST configuration endpoints
type GSTHandler struct {
	gstConfigRepo *db.GSTConfigRepository
	gstrSvc       *service.GSTRService
}

// NewGSTHandler creates a new GST handler
func NewGSTHandler(gstConfigRepo *db.GSTConfigRepository, gstrSvc *service.GSTRService) *GSTHandler {
	return &GSTHandler{gstConfigRepo: gstConfigRepo, gstrSvc: gstrSvc}
}

// GetGSTR1 returns the GSTR-1 outward-supply return for a tax period, both as
// readable sections/totals ("data") and as the GSTN upload JSON ("gov_schema").
// GET /v1/india/gstr1?month=&year=
func (h *GSTHandler) GetGSTR1(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	if h.gstrSvc == nil {
		respondError(c, http.StatusServiceUnavailable, codeInternalError, "GSTR-1 export not configured")
		return
	}

	month, err := strconv.Atoi(c.Query("month"))
	if err != nil || month < 1 || month > 12 {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "month must be 1-12")
		return
	}
	year, err := strconv.Atoi(c.Query("year"))
	if err != nil || year < 2017 || year > 2100 {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "year must be a valid GST-era year")
		return
	}

	ret, err := h.gstrSvc.GetGSTR1(c.Request.Context(), tenantID, month, year)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "Failed to build GSTR-1")
		return
	}

	// Seller GSTIN for the government JSON header (best-effort; empty if unset).
	sellerGSTIN := ""
	if h.gstConfigRepo != nil {
		if cfg, cerr := h.gstConfigRepo.GetByTenantID(c.Request.Context(), tenantID); cerr == nil && cfg != nil {
			sellerGSTIN = cfg.GSTIN
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       ret,
		"gov_schema": service.BuildGSTR1GovDocument(sellerGSTIN, ret),
	})
}

// GetGSTR3B returns the GSTR-3B summary return for a tax period, both as
// readable sections ("data") and as the GSTN upload JSON ("gov_schema").
// GET /v1/india/gstr3b?month=&year=
func (h *GSTHandler) GetGSTR3B(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	if h.gstrSvc == nil {
		respondError(c, http.StatusServiceUnavailable, codeInternalError, "GSTR-3B export not configured")
		return
	}

	month, err := strconv.Atoi(c.Query("month"))
	if err != nil || month < 1 || month > 12 {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "month must be 1-12")
		return
	}
	year, err := strconv.Atoi(c.Query("year"))
	if err != nil || year < 2017 || year > 2100 {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "year must be a valid GST-era year")
		return
	}

	ret, err := h.gstrSvc.GetGSTR3B(c.Request.Context(), tenantID, month, year)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "Failed to build GSTR-3B")
		return
	}

	// Seller GSTIN for the government JSON header (best-effort; empty if unset).
	sellerGSTIN := ""
	if h.gstConfigRepo != nil {
		if cfg, cerr := h.gstConfigRepo.GetByTenantID(c.Request.Context(), tenantID); cerr == nil && cfg != nil {
			sellerGSTIN = cfg.GSTIN
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       ret,
		"gov_schema": service.BuildGSTR3BGovDocument(sellerGSTIN, ret),
	})
}

// GetConfig returns the current GST configuration
// GET /settings/gst
func (h *GSTHandler) GetConfig(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	if h.gstConfigRepo != nil {
		config, err := h.gstConfigRepo.GetByTenantID(c.Request.Context(), tenantID)
		if err == nil && config != nil {
			c.JSON(http.StatusOK, gin.H{"data": GSTConfig{
				GSTIN:     config.GSTIN,
				StateCode: config.StateCode,
				StateName: config.StateName,
				SACCode:   config.SACCode,
				GSTRate:   config.GSTRate,
				PAN:       config.PAN,
				LegalName: config.LegalName,
				TradeName: config.TradeName,
				Address:   config.Address,
				HasLUT:    config.HasLUT,
			}})
			return
		}
	}

	// Return default config if not found
	config := GSTConfig{
		SACCode:   "998314",
		GSTRate:   18.0,
		StateName: "Not configured",
	}
	c.JSON(http.StatusOK, gin.H{"data": config})
}

// UpdateConfig updates GST configuration
// PUT /settings/gst
func (h *GSTHandler) UpdateConfig(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	var config GSTConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	// Validate GSTIN format
	if config.GSTIN != "" && !validateGSTIN(config.GSTIN) {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "Invalid GSTIN format")
		return
	}

	// Extract state code from GSTIN if not provided
	if config.GSTIN != "" && config.StateCode == "" {
		config.StateCode = config.GSTIN[:2]
		config.StateName = getStateName(config.StateCode)
	}

	// Save to database
	if h.gstConfigRepo != nil {
		dbConfig := &domain.TenantGSTConfig{
			GSTIN:     config.GSTIN,
			StateCode: config.StateCode,
			StateName: config.StateName,
			SACCode:   config.SACCode,
			GSTRate:   config.GSTRate,
			PAN:       config.PAN,
			LegalName: config.LegalName,
			TradeName: config.TradeName,
			Address:   config.Address,
			HasLUT:    config.HasLUT,
		}
		if err := h.gstConfigRepo.Upsert(c.Request.Context(), tenantID, dbConfig); err != nil {
			respondError(c, http.StatusInternalServerError, codeInternalError, "Failed to save GST configuration")
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": config, "message": "GST configuration updated"})
}

// ValidateGSTIN validates a GSTIN and returns details
// POST /settings/gst/validate
func (h *GSTHandler) ValidateGSTIN(c *gin.Context) {
	var req struct {
		GSTIN string `json:"gstin"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	if !validateGSTIN(req.GSTIN) {
		c.JSON(http.StatusOK, gin.H{
			"valid":   false,
			"message": "Invalid GSTIN format. GSTIN must be 15 characters.",
		})
		return
	}

	// Parse GSTIN components
	stateCode := req.GSTIN[:2]
	pan := req.GSTIN[2:12]
	stateName := getStateName(stateCode)

	c.JSON(http.StatusOK, gin.H{
		"valid":      true,
		"state_code": stateCode,
		"state_name": stateName,
		"pan":        pan,
		"message":    "GSTIN format is valid",
	})
}

// validateGSTIN performs basic GSTIN validation
func validateGSTIN(gstin string) bool {
	if len(gstin) != 15 {
		return false
	}
	// First 2 chars must be digits (state code)
	if gstin[0] < '0' || gstin[0] > '9' || gstin[1] < '0' || gstin[1] > '9' {
		return false
	}
	return true
}

// getStateName returns state name from code
func getStateName(code string) string {
	states := map[string]string{
		"01": "Jammu & Kashmir",
		"02": "Himachal Pradesh",
		"03": "Punjab",
		"04": "Chandigarh",
		"05": "Uttarakhand",
		"06": "Haryana",
		"07": "Delhi",
		"08": "Rajasthan",
		"09": "Uttar Pradesh",
		"10": "Bihar",
		"11": "Sikkim",
		"12": "Arunachal Pradesh",
		"13": "Nagaland",
		"14": "Manipur",
		"15": "Mizoram",
		"16": "Tripura",
		"17": "Meghalaya",
		"18": "Assam",
		"19": "West Bengal",
		"20": "Jharkhand",
		"21": "Odisha",
		"22": "Chhattisgarh",
		"23": "Madhya Pradesh",
		"24": "Gujarat",
		"26": "Dadra & Nagar Haveli",
		"27": "Maharashtra",
		"29": "Karnataka",
		"30": "Goa",
		"31": "Lakshadweep",
		"32": "Kerala",
		"33": "Tamil Nadu",
		"34": "Puducherry",
		"35": "Andaman & Nicobar",
		"36": "Telangana",
		"37": "Andhra Pradesh",
		"38": "Ladakh",
	}
	if name, ok := states[code]; ok {
		return name
	}
	return "Unknown"
}
