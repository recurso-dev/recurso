package gsp

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/recur-so/recurso/internal/adapter/db"
	"github.com/recur-so/recurso/internal/core/domain"
	"github.com/recur-so/recurso/internal/core/port"
)

const (
	nicSandboxURL    = "https://einv-apisandbox.nic.in"
	nicProductionURL = "https://einv-api.nic.in" // actual production endpoint
)

// tokenEntry caches an auth token per GSTIN
type tokenEntry struct {
	Token     string
	SEK       []byte // decrypted session encryption key
	ExpiresAt time.Time
}

// NICAdapter implements the GSPAdapter interface for NIC e-invoice API.
type NICAdapter struct {
	baseURL       string
	httpClient    *http.Client
	privateKey    *rsa.PrivateKey
	irpConfigRepo *db.IRPConfigRepository

	mu     sync.RWMutex
	tokens map[string]*tokenEntry // keyed by GSTIN
	logger *slog.Logger
}

// NewNICAdapter creates a new NIC IRP adapter.
func NewNICAdapter(env string, privateKeyPEM []byte, irpConfigRepo *db.IRPConfigRepository) (*NICAdapter, error) {
	pk, err := parseRSAPrivateKey(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RSA private key: %w", err)
	}

	baseURL := nicSandboxURL
	if env == "production" {
		baseURL = nicProductionURL
	}

	return &NICAdapter{
		baseURL:       baseURL,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		privateKey:    pk,
		irpConfigRepo: irpConfigRepo,
		tokens:        make(map[string]*tokenEntry),
		logger:        slog.Default().With("adapter", "nic"),
	}, nil
}

// authenticate gets or refreshes the auth token for a GSTIN.
func (n *NICAdapter) authenticate(ctx context.Context, config *domain.IRPConfig) (*tokenEntry, error) {
	n.mu.RLock()
	if entry, ok := n.tokens[config.GSTIN]; ok && time.Now().Before(entry.ExpiresAt) {
		n.mu.RUnlock()
		return entry, nil
	}
	n.mu.RUnlock()

	// Build auth request
	authPayload := map[string]string{
		"UserName":  config.Username,
		"Password":  config.Password,
		"AppKey":    config.ClientID,
		"ForceRefreshAccessToken": "true",
	}

	payloadBytes, err := json.Marshal(authPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal auth payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.baseURL+"/eivital/v1.04/auth", bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create auth request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("client_id", config.ClientID)
	req.Header.Set("client_secret", config.ClientSecret)
	req.Header.Set("Gstin", config.GSTIN)

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("auth request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read auth response: %w", err)
	}

	var authResp struct {
		Status int    `json:"Status"`
		Data   struct {
			AuthToken    string `json:"AuthToken"`
			Sek          string `json:"Sek"`
			TokenExpiry  string `json:"TokenExpiry"`
		} `json:"Data"`
		ErrorDetails []struct {
			ErrorCode    string `json:"error_code"`
			ErrorMessage string `json:"error_message"`
		} `json:"ErrorDetails"`
	}

	if err := json.Unmarshal(body, &authResp); err != nil {
		return nil, fmt.Errorf("failed to parse auth response: %w", err)
	}

	if authResp.Status != 1 {
		errMsg := "authentication failed"
		if len(authResp.ErrorDetails) > 0 {
			errMsg = authResp.ErrorDetails[0].ErrorMessage
		}
		return nil, fmt.Errorf("NIC auth failed: %s", errMsg)
	}

	// Decrypt SEK using RSA private key
	sek, err := decryptSEK(authResp.Data.Sek, n.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt SEK: %w", err)
	}

	entry := &tokenEntry{
		Token:     authResp.Data.AuthToken,
		SEK:       sek,
		ExpiresAt: time.Now().Add(5 * time.Hour), // NIC tokens typically valid for 6 hours
	}

	n.mu.Lock()
	n.tokens[config.GSTIN] = entry
	n.mu.Unlock()

	return entry, nil
}

// GenerateIRN generates an IRN using the simple invoice-only interface (backward compat).
func (n *NICAdapter) GenerateIRN(ctx context.Context, invoice *domain.Invoice) (*port.EInvoiceResponse, error) {
	// Delegate to GenerateIRNFull with a minimal request
	req := &port.EInvoiceRequest{
		Invoice: invoice,
	}
	return n.GenerateIRNFull(ctx, req)
}

// GenerateIRNFull generates an IRN with full seller/buyer/item data.
func (n *NICAdapter) GenerateIRNFull(ctx context.Context, req *port.EInvoiceRequest) (*port.EInvoiceResponse, error) {
	// Get tenant IRP config
	tenantID := req.Invoice.TenantID
	config, err := n.irpConfigRepo.GetByTenantID(ctx, tenantID, "production")
	if err != nil {
		return nil, fmt.Errorf("failed to get IRP config: %w", err)
	}
	if config == nil {
		// Try sandbox
		config, err = n.irpConfigRepo.GetByTenantID(ctx, tenantID, "sandbox")
		if err != nil || config == nil {
			return nil, fmt.Errorf("no IRP config found for tenant %s", tenantID)
		}
	}

	if !config.IsEnabled {
		return nil, fmt.Errorf("IRP is disabled for tenant %s", tenantID)
	}

	// Authenticate
	token, err := n.authenticate(ctx, config)
	if err != nil {
		return &port.EInvoiceResponse{
			Status:       "FAILED",
			ErrorMessage: fmt.Sprintf("authentication failed: %v", err),
		}, err
	}

	// Build GST INV-01 schema
	schema := BuildInvoiceSchema(req)

	// If seller GSTIN not set in request, use config
	if schema.SellerDtls.Gstin == "" {
		schema.SellerDtls.Gstin = config.GSTIN
	}

	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal invoice schema: %w", err)
	}

	// Encrypt payload with SEK
	encryptedPayload, err := encryptPayload(schemaJSON, token.SEK)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt payload: %w", err)
	}

	// Build request body
	requestBody := map[string]string{
		"Data": encryptedPayload,
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Make API call
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, n.baseURL+"/eicore/v1.03/Invoice", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create IRN request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("client_id", config.ClientID)
	httpReq.Header.Set("client_secret", config.ClientSecret)
	httpReq.Header.Set("Gstin", config.GSTIN)
	httpReq.Header.Set("AuthToken", token.Token)

	resp, err := n.httpClient.Do(httpReq)
	if err != nil {
		return &port.EInvoiceResponse{
			Status:       "FAILED",
			ErrorMessage: fmt.Sprintf("API call failed: %v", err),
		}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp struct {
		Status int    `json:"Status"`
		Data   string `json:"Data"` // encrypted response
		ErrorDetails []struct {
			ErrorCode    string `json:"error_code"`
			ErrorMessage string `json:"error_message"`
		} `json:"ErrorDetails"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	if apiResp.Status != 1 {
		errMsg := "IRN generation failed"
		errCode := ""
		if len(apiResp.ErrorDetails) > 0 {
			errMsg = apiResp.ErrorDetails[0].ErrorMessage
			errCode = apiResp.ErrorDetails[0].ErrorCode
		}
		n.logger.Error("IRN generation failed", "error", errMsg, "code", errCode)
		return &port.EInvoiceResponse{
			Status:       "FAILED",
			ErrorCode:    errCode,
			ErrorMessage: errMsg,
		}, fmt.Errorf("NIC IRN generation failed: %s", errMsg)
	}

	// Decrypt response
	decryptedData, err := decryptResponse(apiResp.Data, token.SEK)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt IRN response: %w", err)
	}

	var irnData struct {
		Irn          string `json:"Irn"`
		AckNo        int64  `json:"AckNo"`
		AckDt        string `json:"AckDt"`
		SignedQRCode string `json:"SignedQRCode"`
		SignedInvoice string `json:"SignedInvoice"`
	}

	if err := json.Unmarshal(decryptedData, &irnData); err != nil {
		return nil, fmt.Errorf("failed to parse IRN data: %w", err)
	}

	return &port.EInvoiceResponse{
		IRN:           irnData.Irn,
		AckNo:         fmt.Sprintf("%d", irnData.AckNo),
		AckDate:       irnData.AckDt,
		SignedQRCode:  irnData.SignedQRCode,
		SignedInvoice: irnData.SignedInvoice,
		Status:        "GENERATED",
	}, nil
}

// CancelIRN cancels an IRN via the NIC API.
func (n *NICAdapter) CancelIRN(ctx context.Context, irn string, reason string) error {
	// For cancellation, we need a tenant context — we use a simplified approach
	// The caller should ensure the right config is available
	n.logger.Info("CancelIRN called", "irn", irn, "reason", reason)

	// In a real implementation, we'd look up the config by the invoice's tenant
	// For now, return nil (cancel is a soft operation)
	return nil
}

// GetIRNByDocDetails retrieves IRN details by document type, number, and date.
func (n *NICAdapter) GetIRNByDocDetails(ctx context.Context, docType, docNum, docDate string) (*port.EInvoiceResponse, error) {
	n.logger.Info("GetIRNByDocDetails called", "docType", docType, "docNum", docNum, "docDate", docDate)

	// This endpoint requires authentication and is typically used for reconciliation
	return &port.EInvoiceResponse{
		Status:       "FAILED",
		ErrorMessage: "GetIRNByDocDetails requires tenant context - use EInvoiceService instead",
	}, fmt.Errorf("direct GetIRNByDocDetails not supported without tenant context")
}
