package accounting

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	AuthURL      string
	TokenURL     string
	Scopes       []string
}

type OAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	RealmID      string `json:"realmId"`
}

// TokenError represents an error response from an OAuth token endpoint.
type TokenError struct {
	StatusCode  int
	Code        string // OAuth error code, e.g. "invalid_grant"
	Description string
}

func (e *TokenError) Error() string {
	if e.Code != "" {
		if e.Description != "" {
			return fmt.Sprintf("token endpoint returned %d: %s (%s)", e.StatusCode, e.Code, e.Description)
		}
		return fmt.Sprintf("token endpoint returned %d: %s", e.StatusCode, e.Code)
	}
	return fmt.Sprintf("token endpoint returned status %d", e.StatusCode)
}

// IsInvalidGrant reports whether err indicates the refresh token (or auth
// code) was definitively rejected by the provider. When this happens the
// stored refresh token is dead and the merchant must reconnect.
func IsInvalidGrant(err error) bool {
	var te *TokenError
	return errors.As(err, &te) && te.Code == "invalid_grant"
}

// newTokenError builds a TokenError from a non-200 token endpoint response,
// extracting the standard OAuth error fields when present.
func newTokenError(resp *http.Response) *TokenError {
	te := &TokenError{StatusCode: resp.StatusCode}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return te
	}
	var parsed struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if json.Unmarshal(body, &parsed) == nil {
		te.Code = parsed.Error
		te.Description = parsed.ErrorDescription
	}
	return te
}

// RefreshAccessToken exchanges a refresh token for a new access token.
// Both QuickBooks and Xero rotate refresh tokens, so the rotated pair is
// persisted via the repository before returning.
func RefreshAccessToken(ctx context.Context, config *OAuthConfig, conn *domain.AccountingConnection, repo port.AccountingConnectionRepository) error {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", conn.RefreshToken)

	req, err := http.NewRequestWithContext(ctx, "POST", config.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create token request: %w", err)
	}

	req.SetBasicAuth(config.ClientID, config.ClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("token refresh request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token refresh failed: %w", newTokenError(resp))
	}

	var tokenResp OAuthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}

	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	conn.AccessToken = tokenResp.AccessToken
	if tokenResp.RefreshToken != "" {
		conn.RefreshToken = tokenResp.RefreshToken
	}
	conn.TokenExpiresAt = &expiresAt

	return repo.Update(ctx, conn)
}

// ExchangeCode exchanges an authorization code for tokens.
func ExchangeCode(ctx context.Context, config *OAuthConfig, code string) (*OAuthTokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", config.RedirectURL)

	req, err := http.NewRequestWithContext(ctx, "POST", config.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	req.SetBasicAuth(config.ClientID, config.ClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed: %w", newTokenError(resp))
	}

	var tokenResp OAuthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &tokenResp, nil
}

// XeroConnectionsURL lists the Xero organisations an access token can reach.
// Package-level so tests can point it at a stub server.
var XeroConnectionsURL = "https://api.xero.com/connections"

// FetchXeroTenantID resolves the Xero tenant (organisation) ID for an access
// token. Xero does not return it from the token endpoint; every API call
// requires it in the Xero-tenant-id header, so it must be captured at
// connect time.
func FetchXeroTenantID(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", XeroConnectionsURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create connections request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("xero connections request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("xero connections request failed with status %d", resp.StatusCode)
	}

	var connections []struct {
		TenantID   string `json:"tenantId"`
		TenantType string `json:"tenantType"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&connections); err != nil {
		return "", fmt.Errorf("failed to decode connections response: %w", err)
	}

	for _, c := range connections {
		if c.TenantType == "ORGANISATION" {
			return c.TenantID, nil
		}
	}
	if len(connections) > 0 {
		return connections[0].TenantID, nil
	}
	return "", fmt.Errorf("no xero organisations authorized for this token")
}

// BuildAuthURL constructs the OAuth authorization URL.
func BuildAuthURL(config *OAuthConfig, state string) string {
	params := url.Values{}
	params.Set("client_id", config.ClientID)
	params.Set("redirect_uri", config.RedirectURL)
	params.Set("response_type", "code")
	params.Set("scope", strings.Join(config.Scopes, " "))
	params.Set("state", state)

	return config.AuthURL + "?" + params.Encode()
}
