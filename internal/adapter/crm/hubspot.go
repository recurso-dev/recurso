// Package crm pushes billing state into CRM systems (Track D4,
// spec_lago_parity.md). CRM SaaS egress is residency-guarded at the
// wiring site, exactly like the accounting SaaS adapters.
package crm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HubSpotClient upserts Recurso customers as HubSpot contacts with billing
// properties (customer id, subscription status), keyed by email.
//
// Auth: a HubSpot private-app access token (Bearer). EXPERIMENTAL: built
// against the HubSpot CRM v3 reference; live verification founder-gated.
type HubSpotClient struct {
	accessToken string
	baseURL     string
	httpClient  *http.Client
}

const hubSpotBaseURL = "https://api.hubapi.com"

func NewHubSpotClient(accessToken string) *HubSpotClient {
	return &HubSpotClient{
		accessToken: accessToken,
		baseURL:     hubSpotBaseURL,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *HubSpotClient) do(ctx context.Context, method, path string, body any, out any) (int, error) {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("hubspot request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	payload, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 400 && resp.StatusCode != http.StatusNotFound {
		return resp.StatusCode, fmt.Errorf("hubspot %s: HTTP %d: %s", path, resp.StatusCode, strings.TrimSpace(string(payload)))
	}
	if out != nil && resp.StatusCode < 400 {
		if err := json.Unmarshal(payload, out); err != nil {
			return resp.StatusCode, fmt.Errorf("hubspot %s: bad response: %w", path, err)
		}
	}
	return resp.StatusCode, nil
}

// findContactByEmail returns the HubSpot contact id for an email, or ""
// when none exists.
func (c *HubSpotClient) findContactByEmail(ctx context.Context, email string) (string, error) {
	var out struct {
		Results []struct {
			ID string `json:"id"`
		} `json:"results"`
	}
	_, err := c.do(ctx, http.MethodPost, "/crm/v3/objects/contacts/search", map[string]any{
		"filterGroups": []any{map[string]any{
			"filters": []any{map[string]any{
				"propertyName": "email", "operator": "EQ", "value": email,
			}},
		}},
		"limit": 1,
	}, &out)
	if err != nil {
		return "", err
	}
	if len(out.Results) == 0 {
		return "", nil
	}
	return out.Results[0].ID, nil
}

// UpsertContact creates or updates the contact for email with the given
// properties (email is always included). Returns the HubSpot contact id.
func (c *HubSpotClient) UpsertContact(ctx context.Context, email string, properties map[string]string) (string, error) {
	props := map[string]string{"email": email}
	for k, v := range properties {
		props[k] = v
	}

	id, err := c.findContactByEmail(ctx, email)
	if err != nil {
		return "", err
	}
	if id == "" {
		var created struct {
			ID string `json:"id"`
		}
		if _, err := c.do(ctx, http.MethodPost, "/crm/v3/objects/contacts",
			map[string]any{"properties": props}, &created); err != nil {
			return "", err
		}
		return created.ID, nil
	}
	if _, err := c.do(ctx, http.MethodPatch, "/crm/v3/objects/contacts/"+id,
		map[string]any{"properties": props}, nil); err != nil {
		return "", err
	}
	return id, nil
}
