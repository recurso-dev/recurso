package handler

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/adapter/accounting"
	"github.com/recur-so/recurso/internal/core/domain"
	"github.com/recur-so/recurso/internal/core/port"
	"github.com/recur-so/recurso/internal/service"
)

type AccountingHandler struct {
	connRepo          port.AccountingConnectionRepository
	accountingService *service.AccountingService
}

func NewAccountingHandler(connRepo port.AccountingConnectionRepository, accountingSvc *service.AccountingService) *AccountingHandler {
	return &AccountingHandler{
		connRepo:          connRepo,
		accountingService: accountingSvc,
	}
}

func (h *AccountingHandler) ListConnections(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id missing"})
		return
	}

	conns, err := h.connRepo.ListByTenant(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if conns == nil {
		conns = []*domain.AccountingConnection{}
	}

	c.JSON(http.StatusOK, gin.H{"data": conns})
}

func (h *AccountingHandler) InitiateOAuth(c *gin.Context) {
	provider := c.Param("provider")

	var config *accounting.OAuthConfig
	baseURL := os.Getenv("APP_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	switch provider {
	case "quickbooks":
		config = &accounting.OAuthConfig{
			ClientID:     os.Getenv("QBO_CLIENT_ID"),
			ClientSecret: os.Getenv("QBO_CLIENT_SECRET"),
			RedirectURL:  baseURL + "/v1/accounting/callback/quickbooks",
			AuthURL:      "https://appcenter.intuit.com/connect/oauth2",
			TokenURL:     "https://oauth.platform.intuit.com/oauth2/v1/tokens/bearer",
			Scopes:       []string{"com.intuit.quickbooks.accounting"},
		}
	case "xero":
		config = &accounting.OAuthConfig{
			ClientID:     os.Getenv("XERO_CLIENT_ID"),
			ClientSecret: os.Getenv("XERO_CLIENT_SECRET"),
			RedirectURL:  baseURL + "/v1/accounting/callback/xero",
			AuthURL:      "https://login.xero.com/identity/connect/authorize",
			TokenURL:     "https://identity.xero.com/connect/token",
			Scopes:       []string{"openid", "profile", "email", "accounting.transactions", "accounting.contacts"},
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported provider"})
		return
	}

	tenantID, _ := c.MustGet("tenant_id").(uuid.UUID)
	state := tenantID.String() + ":" + provider

	authURL := accounting.BuildAuthURL(config, state)
	c.JSON(http.StatusOK, gin.H{"auth_url": authURL})
}

func (h *AccountingHandler) OAuthCallback(c *gin.Context) {
	provider := c.Param("provider")
	code := c.Query("code")
	state := c.Query("state")
	realmID := c.Query("realmId") // QuickBooks specific

	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing authorization code"})
		return
	}

	// Parse tenant ID from state
	var tenantID uuid.UUID
	if len(state) >= 36 {
		parsed, err := uuid.Parse(state[:36])
		if err == nil {
			tenantID = parsed
		}
	}
	if tenantID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state"})
		return
	}

	baseURL := os.Getenv("APP_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	var config *accounting.OAuthConfig
	switch provider {
	case "quickbooks":
		config = &accounting.OAuthConfig{
			ClientID:     os.Getenv("QBO_CLIENT_ID"),
			ClientSecret: os.Getenv("QBO_CLIENT_SECRET"),
			RedirectURL:  baseURL + "/v1/accounting/callback/quickbooks",
			TokenURL:     "https://oauth.platform.intuit.com/oauth2/v1/tokens/bearer",
		}
	case "xero":
		config = &accounting.OAuthConfig{
			ClientID:     os.Getenv("XERO_CLIENT_ID"),
			ClientSecret: os.Getenv("XERO_CLIENT_SECRET"),
			RedirectURL:  baseURL + "/v1/accounting/callback/xero",
			TokenURL:     "https://identity.xero.com/connect/token",
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported provider"})
		return
	}

	tokenResp, err := accounting.ExchangeCode(c.Request.Context(), config, code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token exchange failed: " + err.Error()})
		return
	}

	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	if realmID == "" {
		realmID = tokenResp.RealmID
	}

	conn := &domain.AccountingConnection{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Provider:       provider,
		AccessToken:    tokenResp.AccessToken,
		RefreshToken:   tokenResp.RefreshToken,
		TokenExpiresAt: &expiresAt,
		RealmID:        realmID,
		SyncStatus:     "idle",
		IsActive:       true,
		CreatedAt:      time.Now(),
	}

	if err := h.connRepo.Create(c.Request.Context(), conn); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save connection: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "connected", "connection_id": conn.ID})
}

func (h *AccountingHandler) Disconnect(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid connection id"})
		return
	}

	if err := h.connRepo.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "disconnected"})
}

func (h *AccountingHandler) TriggerSync(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id missing"})
		return
	}

	if err := h.accountingService.SyncAllForTenant(c.Request.Context(), tenantID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "sync_triggered"})
}

func (h *AccountingHandler) SyncStatus(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id missing"})
		return
	}

	logs, err := h.connRepo.ListSyncLogs(c.Request.Context(), tenantID, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if logs == nil {
		logs = []*domain.AccountingSyncLog{}
	}

	c.JSON(http.StatusOK, gin.H{"data": logs})
}
