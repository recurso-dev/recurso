package accounting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/recur-so/recurso/internal/core/domain"
)

type XeroAdapter struct {
	baseURL     string
	accessToken string
	tenantID    string // Xero tenant ID (different from Recurso tenant)
}

func NewXeroAdapter(accessToken, xeroTenantID string) *XeroAdapter {
	return &XeroAdapter{
		baseURL:     "https://api.xero.com/api.xro/2.0",
		accessToken: accessToken,
		tenantID:    xeroTenantID,
	}
}

func (a *XeroAdapter) SetCredentials(accessToken, xeroTenantID string) {
	a.accessToken = accessToken
	a.tenantID = xeroTenantID
}

func (a *XeroAdapter) SyncCustomer(ctx context.Context, customer *domain.Customer) error {
	name := ""
	if customer.Name != nil {
		name = *customer.Name
	}

	contact := map[string]interface{}{
		"Contacts": []map[string]interface{}{
			{
				"Name":         name,
				"EmailAddress": customer.Email,
			},
		},
	}

	body, err := json.Marshal(contact)
	if err != nil {
		return fmt.Errorf("failed to marshal customer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/Contacts", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	a.setHeaders(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("Xero customer sync failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Xero API error: status %d", resp.StatusCode)
	}

	return nil
}

func (a *XeroAdapter) SyncInvoice(ctx context.Context, invoice *domain.Invoice) error {
	xeroInvoice := map[string]interface{}{
		"Invoices": []map[string]interface{}{
			{
				"Type": "ACCREC",
				"Contact": map[string]string{
					"ContactID": invoice.CustomerID.String(),
				},
				"LineItems": []map[string]interface{}{
					{
						"Description": fmt.Sprintf("Invoice %s", invoice.InvoiceNumber),
						"Quantity":    1,
						"UnitAmount":  float64(invoice.Total) / 100,
					},
				},
				"CurrencyCode": invoice.Currency,
				"Status":       "AUTHORISED",
			},
		},
	}

	body, err := json.Marshal(xeroInvoice)
	if err != nil {
		return fmt.Errorf("failed to marshal invoice: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/Invoices", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	a.setHeaders(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("Xero invoice sync failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Xero API error: status %d", resp.StatusCode)
	}

	return nil
}

func (a *XeroAdapter) SyncProduct(ctx context.Context, plan *domain.Plan) error {
	item := map[string]interface{}{
		"Items": []map[string]interface{}{
			{
				"Code":        plan.ID.String()[:8],
				"Name":        plan.Name,
				"Description": plan.Name,
			},
		},
	}

	body, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to marshal product: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/Items", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	a.setHeaders(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("Xero product sync failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Xero API error: status %d", resp.StatusCode)
	}

	return nil
}

func (a *XeroAdapter) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+a.accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Xero-tenant-id", a.tenantID)
}
