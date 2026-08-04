package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/accounting"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/recurso-dev/recurso/internal/service"
)

type AccountingHandler struct {
	connRepo          port.AccountingConnectionRepository
	accountingService *service.AccountingService
	// oauthStateSecret signs the accounting OAuth `state` (HMAC-SHA256). It MUST
	// be a real secret — the caller supplies OAUTH_STATE_SECRET or an ephemeral
	// per-boot random key. A hardcoded key would let anyone forge a state and
	// bind an accounting connection to an arbitrary tenant.
	oauthStateSecret []byte
	// dashboardURL is the SPA base the OAuth callback redirects back to. The
	// callback is a top-level browser navigation (the provider redirects the
	// user here), so answering with JSON would strand the user on a raw JSON
	// page — every outcome 302s to {dashboardURL}/integrations instead.
	dashboardURL string
}

func NewAccountingHandler(connRepo port.AccountingConnectionRepository, accountingSvc *service.AccountingService, oauthStateSecret []byte, dashboardURL string) *AccountingHandler {
	return &AccountingHandler{
		connRepo:          connRepo,
		accountingService: accountingSvc,
		oauthStateSecret:  oauthStateSecret,
		dashboardURL:      strings.TrimRight(dashboardURL, "/"),
	}
}

// redirectToIntegrations 302s the browser back to the dashboard's Integrations
// page with a single outcome query param. Values are short stable codes (or a
// provider name) — raw error text never goes into a URL.
func (h *AccountingHandler) redirectToIntegrations(c *gin.Context, key, value string) {
	c.Redirect(http.StatusFound,
		h.dashboardURL+"/integrations?"+key+"="+url.QueryEscape(value))
}

// ConnectTokenBased creates (or refreshes) a connection for providers that
// don't fit the browser OAuth flow:
//   - netsuite: the admin supplies their NetSuite account id and a SuiteTalk
//     OAuth 2.0 access token minted in NetSuite's own UI (Track D2,
//     EXPERIMENTAL — sandbox verification is founder-gated). RealmID carries
//     the account id, mirroring QuickBooks.
//   - tally: no credentials at all — the adapter writes local JSONL export
//     files for Tally's import tooling, so "connecting" just enables the sync.
//
// Reconnecting an existing (e.g. deactivated) connection updates it in place,
// same as the OAuth callback's reconnect path.
func (h *AccountingHandler) ConnectTokenBased(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	provider := c.Param("provider")

	var req struct {
		AccountID   string `json:"account_id"`
		AccessToken string `json:"access_token"`
	}
	// Tally legitimately sends no body; ignore bind errors and validate below.
	_ = c.ShouldBindJSON(&req)

	switch provider {
	case "netsuite":
		if strings.TrimSpace(req.AccountID) == "" || strings.TrimSpace(req.AccessToken) == "" {
			respondError(c, http.StatusBadRequest, codeValidationFailed,
				"account_id and access_token are required for NetSuite")
			return
		}
	case "tally":
		// nothing to validate
	default:
		respondError(c, http.StatusBadRequest, codeValidationFailed,
			"provider does not support token-based connection")
		return
	}

	if existing, err := h.connRepo.GetByTenantAndProvider(c.Request.Context(), tenantID, provider); err == nil && existing != nil {
		existing.AccessToken = strings.TrimSpace(req.AccessToken)
		if rid := strings.TrimSpace(req.AccountID); rid != "" {
			existing.RealmID = rid
		}
		existing.SyncStatus = "idle"
		existing.LastError = ""
		existing.IsActive = true
		if err := h.connRepo.Update(c.Request.Context(), existing); err != nil {
			respondInternalError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": existing})
		return
	}

	conn := &domain.AccountingConnection{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Provider:    provider,
		AccessToken: strings.TrimSpace(req.AccessToken),
		RealmID:     strings.TrimSpace(req.AccountID),
		SyncStatus:  "idle",
		IsActive:    true,
		CreatedAt:   time.Now(),
	}
	if err := h.connRepo.Create(c.Request.Context(), conn); err != nil {
		respondInternalError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": conn})
}

func (h *AccountingHandler) ListConnections(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	conns, err := h.connRepo.ListByTenant(c.Request.Context(), tenantID)
	if err != nil {
		respondInternalError(c, err)
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
			// Granular scopes (mandatory for Xero apps created after 2 Mar 2026;
			// the broad accounting.transactions no longer exists for them).
			// offline_access is required for a refresh token — without it the
			// connection dies 30 minutes after consent. Items fall under
			// accounting.settings in the granular scheme.
			Scopes: []string{"openid", "profile", "email", "offline_access", "accounting.contacts", "accounting.invoices", "accounting.settings"},
		}
	default:
		respondError(c, http.StatusBadRequest, codeValidationFailed, "unsupported provider")
		return
	}

	// Without operator credentials the handoff would reach the provider with
	// an empty client_id and die on THEIR error page ("Invalid client_id") —
	// fail here with an actionable message instead.
	if config.ClientID == "" {
		envHint := map[string]string{
			"quickbooks": "QBO_CLIENT_ID / QBO_CLIENT_SECRET",
			"xero":       "XERO_CLIENT_ID / XERO_CLIENT_SECRET",
		}[provider]
		respondError(c, http.StatusServiceUnavailable, codeInternalError,
			provider+" OAuth is not configured on this server — the operator must register a "+provider+" developer app and set "+envHint)
		return
	}

	tenantID, _ := c.MustGet("tenant_id").(uuid.UUID)
	state := h.generateOAuthState(tenantID, provider)

	authURL := accounting.BuildAuthURL(config, state)
	c.JSON(http.StatusOK, gin.H{"auth_url": authURL})
}

func (h *AccountingHandler) OAuthCallback(c *gin.Context) {
	provider := c.Param("provider")
	code := c.Query("code")
	state := c.Query("state")
	realmID := c.Query("realmId") // QuickBooks specific

	if code == "" {
		h.redirectToIntegrations(c, "error", "missing_code")
		return
	}

	// Verify and parse tenant ID from HMAC-signed state
	tenantID, err := h.verifyOAuthState(state)
	if err != nil {
		h.redirectToIntegrations(c, "error", "invalid_state")
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
		h.redirectToIntegrations(c, "error", "unsupported_provider")
		return
	}

	tokenResp, err := accounting.ExchangeCode(c.Request.Context(), config, code)
	if err != nil {
		h.redirectToIntegrations(c, "error", "exchange_failed")
		return
	}

	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	if realmID == "" {
		realmID = tokenResp.RealmID
	}

	// Xero does not return an organisation ID from the token endpoint, but
	// every Xero API call requires it in the Xero-tenant-id header.
	if provider == "xero" && realmID == "" {
		realmID, err = accounting.FetchXeroTenantID(c.Request.Context(), tokenResp.AccessToken)
		if err != nil {
			h.redirectToIntegrations(c, "error", "org_lookup_failed")
			return
		}
	}

	// Reconnect flow: update the existing connection (e.g. one deactivated
	// after an invalid_grant) instead of creating a duplicate.
	if existing, err := h.connRepo.GetByTenantAndProvider(c.Request.Context(), tenantID, provider); err == nil && existing != nil {
		existing.AccessToken = tokenResp.AccessToken
		existing.RefreshToken = tokenResp.RefreshToken
		existing.TokenExpiresAt = &expiresAt
		if realmID != "" {
			existing.RealmID = realmID
		}
		existing.SyncStatus = "idle"
		existing.LastError = ""
		existing.IsActive = true

		if err := h.connRepo.Update(c.Request.Context(), existing); err != nil {
			h.redirectToIntegrations(c, "error", "save_failed")
			return
		}

		h.redirectToIntegrations(c, "connected", provider)
		return
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
		h.redirectToIntegrations(c, "error", "save_failed")
		return
	}

	h.redirectToIntegrations(c, "connected", provider)
}

func (h *AccountingHandler) Disconnect(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid connection id")
		return
	}

	// Verify the connection belongs to this tenant before deleting
	conn, err := h.connRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, http.StatusNotFound, codeNotFound, "connection not found")
		return
	}
	if conn.TenantID != tenantID {
		respondError(c, http.StatusNotFound, codeNotFound, "connection not found")
		return
	}

	if err := h.connRepo.Delete(c.Request.Context(), id); err != nil {
		respondInternalError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "disconnected"})
}

func (h *AccountingHandler) TriggerSync(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	// Optional ?provider= scopes the sweep to one connection (e.g. re-push
	// only Xero after fixing a bad record, without re-running QuickBooks).
	provider := strings.TrimSpace(c.Query("provider"))
	switch provider {
	case "", "quickbooks", "xero", "netsuite", "tally":
	default:
		respondError(c, http.StatusBadRequest, codeValidationFailed, "unknown provider")
		return
	}

	// Manual syncs run in the BACKGROUND: the sweep can run minutes against a
	// third-party API, far past what a proxied request survives. The
	// all-provider button is the explicit "reconcile everything" action and
	// forces a full re-push; a provider-scoped (card) sync is routine and
	// runs incrementally — dirty-tracking skips unchanged records.
	force := provider == ""
	if !h.accountingService.TriggerSyncAsync(tenantID, provider, force) {
		c.JSON(http.StatusOK, gin.H{"status": "sync_already_running"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"status": "sync_triggered", "provider": provider})
}

func (h *AccountingHandler) SyncStatus(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	// Filterable + paginated when the repo supports it (capability assertion
	// keeps port.AccountingConnectionRepository — and its mocks — untouched).
	type syncLogLister interface {
		ListSyncLogsFiltered(ctx context.Context, tenantID uuid.UUID, f db.SyncLogFilter) ([]*domain.AccountingSyncLog, int, error)
	}
	lister, ok := h.connRepo.(syncLogLister)
	if !ok {
		logs, err := h.connRepo.ListSyncLogs(c.Request.Context(), tenantID, 50)
		if err != nil {
			respondInternalError(c, err)
			return
		}
		if logs == nil {
			logs = []*domain.AccountingSyncLog{}
		}
		c.JSON(http.StatusOK, gin.H{"data": logs})
		return
	}

	// (25, 200) is what the OpenAPI spec has always documented; the args were
	// inverted here (200, 25), so any requested limit above 25 was silently
	// forced to 200.
	limit, offset := parseLimitOffset(c, 25, 200)
	logs, total, err := lister.ListSyncLogsFiltered(c.Request.Context(), tenantID, db.SyncLogFilter{
		Provider: strings.TrimSpace(c.Query("provider")),
		Status:   strings.TrimSpace(c.Query("status")),
		Search:   strings.TrimSpace(c.Query("search")),
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		respondInternalError(c, err)
		return
	}
	if logs == nil {
		logs = []*domain.AccountingSyncLog{}
	}

	c.JSON(http.StatusOK, gin.H{"data": logs, "total": total, "limit": limit, "offset": offset})
}

// generateOAuthState produces an HMAC-signed state: "tenantID:provider:signature"
func (h *AccountingHandler) generateOAuthState(tenantID uuid.UUID, provider string) string {
	payload := tenantID.String() + ":" + provider
	mac := hmac.New(sha256.New, h.oauthStateSecret)
	mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))
	return payload + ":" + sig
}

// verifyOAuthState verifies the HMAC signature and extracts the tenant ID.
func (h *AccountingHandler) verifyOAuthState(state string) (uuid.UUID, error) {
	parts := strings.SplitN(state, ":", 3)
	if len(parts) != 3 {
		return uuid.Nil, fmt.Errorf("malformed state")
	}

	tenantIDStr, provider, signature := parts[0], parts[1], parts[2]

	// Recompute expected signature
	payload := tenantIDStr + ":" + provider
	mac := hmac.New(sha256.New, h.oauthStateSecret)
	mac.Write([]byte(payload))
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return uuid.Nil, fmt.Errorf("invalid state signature")
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid tenant_id in state: %w", err)
	}

	return tenantID, nil
}
