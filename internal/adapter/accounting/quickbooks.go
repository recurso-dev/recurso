package accounting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/recur-so/recurso/internal/core/domain"
)

type QuickBooksAdapter struct {
	baseURL     string
	accessToken string
	realmID     string
}

func NewQuickBooksAdapter(accessToken, realmID string, sandbox bool) *QuickBooksAdapter {
	baseURL := "https://quickbooks.api.intuit.com"
	if sandbox {
		baseURL = "https://sandbox-quickbooks.api.intuit.com"
	}

	return &QuickBooksAdapter{
		baseURL:     baseURL,
		accessToken: accessToken,
		realmID:     realmID,
	}
}

func (a *QuickBooksAdapter) SetCredentials(accessToken, realmID string) {
	a.accessToken = accessToken
	a.realmID = realmID
}

func (a *QuickBooksAdapter) SyncCustomer(ctx context.Context, customer *domain.Customer) error {
	name := ""
	if customer.Name != nil {
		name = *customer.Name
	}

	qbCustomer := map[string]interface{}{
		"DisplayName":    name,
		"PrimaryEmailAddr": map[string]string{
			"Address": customer.Email,
		},
	}

	body, err := json.Marshal(qbCustomer)
	if err != nil {
		return fmt.Errorf("failed to marshal customer: %w", err)
	}

	url := fmt.Sprintf("%s/v3/company/%s/customer", a.baseURL, a.realmID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+a.accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("QuickBooks customer sync failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("QuickBooks API error: status %d", resp.StatusCode)
	}

	return nil
}

func (a *QuickBooksAdapter) SyncInvoice(ctx context.Context, invoice *domain.Invoice) error {
	qbInvoice := map[string]interface{}{
		"CustomerRef": map[string]string{
			"value": invoice.CustomerID.String(),
		},
		"Line": []map[string]interface{}{
			{
				"Amount":         float64(invoice.Total) / 100,
				"DetailType":     "SalesItemLineDetail",
				"SalesItemLineDetail": map[string]interface{}{
					"UnitPrice": float64(invoice.Total) / 100,
					"Qty":       1,
				},
			},
		},
	}

	body, err := json.Marshal(qbInvoice)
	if err != nil {
		return fmt.Errorf("failed to marshal invoice: %w", err)
	}

	url := fmt.Sprintf("%s/v3/company/%s/invoice", a.baseURL, a.realmID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+a.accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("QuickBooks invoice sync failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("QuickBooks API error: status %d", resp.StatusCode)
	}

	return nil
}

func (a *QuickBooksAdapter) SyncProduct(ctx context.Context, plan *domain.Plan) error {
	qbItem := map[string]interface{}{
		"Name": plan.Name,
		"Type": "Service",
		"IncomeAccountRef": map[string]string{
			"value": "1",
		},
	}

	body, err := json.Marshal(qbItem)
	if err != nil {
		return fmt.Errorf("failed to marshal product: %w", err)
	}

	url := fmt.Sprintf("%s/v3/company/%s/item", a.baseURL, a.realmID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+a.accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("QuickBooks product sync failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("QuickBooks API error: status %d", resp.StatusCode)
	}

	return nil
}
