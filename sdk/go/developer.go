package recurso

import (
	"context"
	"net/http"
	"time"
)

// APIKey is a tenant API key. KeyValue (the raw secret) is populated only in
// the creation response.
type APIKey struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	KeyValue  string    `json:"key_value"`
	KeyPrefix string    `json:"key_prefix"`
	Type      string    `json:"type"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// DeveloperService groups the API-key management endpoints.
type DeveloperService struct{ client *Client }

// ListKeys returns the tenant's API keys (raw key values are never returned).
func (s *DeveloperService) ListKeys(ctx context.Context) ([]APIKey, error) {
	return getData[[]APIKey](ctx, s.client, http.MethodGet, "/developer/keys", nil)
}

// CreateKey generates a new secret API key. The raw key_value is returned only
// in this response.
func (s *DeveloperService) CreateKey(ctx context.Context) (*APIKey, error) {
	var out APIKey
	if err := s.client.do(ctx, http.MethodPost, "/developer/keys", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
