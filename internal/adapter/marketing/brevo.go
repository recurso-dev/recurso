// Package marketing holds operator-side integrations that push Recurso's OWN
// funnel signals — new tenant signups — into the operator's marketing/email
// tools like Brevo. This is PLATFORM-level data (who signed up for Recurso), so
// it is deliberately separate from the per-tenant, residency-guarded `crm`
// package (which syncs a tenant's own customers into the tenant's own CRM).
package marketing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// BrevoContactSync upserts a new-signup contact into Brevo (Sendinblue) via its
// REST API, so every tenant that signs up shows up in the operator's email tool
// and can be dropped into an onboarding automation. Opt-in: constructed only
// when BREVO_API_KEY is set.
type BrevoContactSync struct {
	apiKey  string
	listID  int // optional; 0 means "no list" (contact created, not added to a list)
	baseURL string
	client  *http.Client
}

// NewBrevoContactSync builds a sync client. listID is optional (0 to skip).
func NewBrevoContactSync(apiKey string, listID int) *BrevoContactSync {
	return &BrevoContactSync{
		apiKey:  apiKey,
		listID:  listID,
		baseURL: "https://api.brevo.com/v3",
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

// AddSignupContact upserts the contact (updateEnabled=true, so a repeat email
// updates rather than 400s) with the company and country as attributes, adding
// it to the configured list if any.
func (b *BrevoContactSync) AddSignupContact(ctx context.Context, email, companyName, country string) error {
	payload := map[string]any{
		"email":         email,
		"attributes":    map[string]any{"COMPANY": companyName, "COUNTRY": country},
		"updateEnabled": true,
	}
	if b.listID > 0 {
		payload["listIds"] = []int{b.listID}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.baseURL+"/contacts", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("api-key", b.apiKey)
	req.Header.Set("content-type", "application/json")
	req.Header.Set("accept", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("brevo contact sync: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("brevo contact sync: status %d: %s", resp.StatusCode, string(msg))
	}
	return nil
}
