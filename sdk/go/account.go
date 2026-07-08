package recurso

import (
	"context"
	"net/http"
	"time"
)

// Tenant is the account (tenant) that owns the API key.
type Tenant struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	DataRegion     string    `json:"data_region"`
	BaseCurrency   string    `json:"base_currency"`
	OrganizationID string    `json:"organization_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// AccountUpdateParams updates the tenant account.
type AccountUpdateParams struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// AccountService groups the tenant-account endpoints.
type AccountService struct{ client *Client }

// Get returns the tenant account.
func (s *AccountService) Get(ctx context.Context) (*Tenant, error) {
	out, err := getData[Tenant](ctx, s.client, http.MethodGet, "/account", nil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Update updates the tenant account.
func (s *AccountService) Update(ctx context.Context, params *AccountUpdateParams) (*Tenant, error) {
	out, err := getData[Tenant](ctx, s.client, http.MethodPut, "/account", params)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
