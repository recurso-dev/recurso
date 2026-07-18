package accounting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// NetSuiteAdapter syncs entities to NetSuite via the SuiteTalk REST record
// API (Track D2, spec_lago_parity.md).
//
// Upsert semantics: create = POST (NetSuite answers 204 with the new
// record's id in the Location header); update = PATCH /{type}/{id}. A 404
// on a carried external ID returns port.ErrExternalGone so the sync
// service clears the stale mapping and re-creates — the same contract as
// QuickBooks/Xero.
//
// Auth: a bearer access token supplied by the existing accounting OAuth
// flow (NetSuite OAuth 2.0). accountID is the NetSuite account, which
// shapes the API host.
//
// EXPERIMENTAL: built against the SuiteTalk REST reference; sandbox
// verification is founder-gated.
type NetSuiteAdapter struct {
	baseURL     string
	accessToken string
	httpClient  *http.Client
}

// NewNetSuiteAdapter builds the adapter for one NetSuite account.
func NewNetSuiteAdapter(accessToken, accountID string) *NetSuiteAdapter {
	host := strings.ToLower(strings.ReplaceAll(accountID, "_", "-"))
	return &NetSuiteAdapter{
		baseURL:     fmt.Sprintf("https://%s.suitetalk.api.netsuite.com/services/rest/record/v1", host),
		accessToken: accessToken,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

// SetCredentials refreshes the bearer token (called by the sync worker
// after an OAuth refresh).
func (a *NetSuiteAdapter) SetCredentials(accessToken, accountID string) {
	a.accessToken = accessToken
	if accountID != "" {
		host := strings.ToLower(strings.ReplaceAll(accountID, "_", "-"))
		a.baseURL = fmt.Sprintf("https://%s.suitetalk.api.netsuite.com/services/rest/record/v1", host)
	}
}

// upsert POSTs (create) or PATCHes (update) a record, returning the
// provider id: the carried externalID on update, or the id NetSuite hands
// back in the Location header on create.
func (a *NetSuiteAdapter) upsert(ctx context.Context, recordType, externalID string, payload any) (string, error) {
	method := http.MethodPost
	path := "/" + recordType
	if externalID != "" {
		method = http.MethodPatch
		path += "/" + externalID
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, method, a.baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+a.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("netsuite %s %s: %w", method, recordType, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	switch {
	case resp.StatusCode == http.StatusNotFound && externalID != "":
		return "", fmt.Errorf("netsuite %s %s: %w", recordType, externalID, port.ErrExternalGone)
	case resp.StatusCode >= 400:
		return "", fmt.Errorf("netsuite %s %s: HTTP %d: %s", method, recordType, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if externalID != "" {
		return externalID, nil
	}
	// Create: the id is the last segment of the Location header
	// (.../record/v1/customer/1234).
	loc := resp.Header.Get("Location")
	if loc == "" {
		return "", fmt.Errorf("netsuite create %s: response missing Location header", recordType)
	}
	segments := strings.Split(strings.TrimRight(loc, "/"), "/")
	return segments[len(segments)-1], nil
}

// SyncCustomer upserts the customer as a NetSuite customer record.
func (a *NetSuiteAdapter) SyncCustomer(ctx context.Context, customer *domain.Customer, externalID string) (string, error) {
	name := domain.PtrToString(customer.Name)
	if name == "" {
		name = customer.Email
	}
	payload := map[string]any{
		"companyName": name,
		"email":       customer.Email,
	}
	return a.upsert(ctx, "customer", externalID, payload)
}

// SyncInvoice upserts the invoice with one line per Recurso invoice line.
// Lines reference the synced item when refs carry one; NetSuite requires
// an item per line, so without a product mapping the sync fails loudly
// rather than writing an unbalanced record.
func (a *NetSuiteAdapter) SyncInvoice(ctx context.Context, invoice *domain.Invoice, refs port.InvoiceSyncRefs, externalID string) (string, error) {
	if refs.CustomerExternalID == "" {
		return "", fmt.Errorf("netsuite invoice sync requires a synced customer")
	}
	if refs.ProductExternalID == "" {
		return "", fmt.Errorf("netsuite invoice sync requires a synced item (NetSuite lines must reference an item)")
	}

	items := make([]map[string]any, 0, len(invoice.LineItems))
	for _, line := range invoice.LineItems {
		items = append(items, map[string]any{
			"item":        map[string]string{"id": refs.ProductExternalID},
			"description": line.Description,
			"quantity":    line.Quantity,
			"amount":      float64(line.Amount) / 100.0, // major units
		})
	}
	if len(items) == 0 {
		items = append(items, map[string]any{
			"item":        map[string]string{"id": refs.ProductExternalID},
			"description": "Invoice " + invoice.InvoiceNumber,
			"quantity":    1,
			"amount":      float64(invoice.Subtotal) / 100.0,
		})
	}

	payload := map[string]any{
		"entity":      map[string]string{"id": refs.CustomerExternalID},
		"tranDate":    invoice.CreatedAt.UTC().Format("2006-01-02"),
		"memo":        "Recurso " + invoice.InvoiceNumber,
		"otherRefNum": invoice.InvoiceNumber,
		"item":        map[string]any{"items": items},
	}
	return a.upsert(ctx, "invoice", externalID, payload)
}

// SyncProduct upserts the plan as a service-sale item.
func (a *NetSuiteAdapter) SyncProduct(ctx context.Context, plan *domain.Plan, externalID string) (string, error) {
	payload := map[string]any{
		"itemId":      plan.Code,
		"displayName": plan.Name,
	}
	return a.upsert(ctx, "serviceSaleItem", externalID, payload)
}
