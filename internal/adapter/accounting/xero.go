package accounting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/swapnull-in/recur-so/internal/core/domain"
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
	// Check for existing contact by email to avoid duplicates
	existing, err := a.findContactByEmail(ctx, customer.Email)
	if err == nil && existing != "" {
		return nil
	}

	name := ""
	if customer.Name != nil {
		name = *customer.Name
	}

	contactData := map[string]interface{}{
		"Name":         name,
		"EmailAddress": customer.Email,
	}

	// Add phone
	if customer.Phone != "" {
		contactData["Phones"] = []map[string]string{
			{"PhoneType": "DEFAULT", "PhoneNumber": customer.Phone},
		}
	}

	// Add billing address
	if customer.BillingAddress.Line1 != "" {
		contactData["Addresses"] = []map[string]string{
			{
				"AddressType":  "POBOX",
				"AddressLine1": customer.BillingAddress.Line1,
				"City":         customer.BillingAddress.City,
				"Region":       customer.BillingAddress.State,
				"PostalCode":   customer.BillingAddress.Zip,
				"Country":      customer.BillingAddress.Country,
			},
		}
	}

	// Add tax number
	if customer.TaxID != nil && *customer.TaxID != "" {
		contactData["TaxNumber"] = *customer.TaxID
	}

	contact := map[string]interface{}{
		"Contacts": []map[string]interface{}{contactData},
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
		return fmt.Errorf("xero customer sync failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("xero API error: status %d", resp.StatusCode)
	}

	return nil
}

func (a *XeroAdapter) SyncInvoice(ctx context.Context, invoice *domain.Invoice) error {
	lineItems := []map[string]interface{}{
		{
			"Description": fmt.Sprintf("Invoice %s", invoice.InvoiceNumber),
			"Quantity":    1,
			"UnitAmount":  float64(invoice.Subtotal) / 100,
			"AccountCode": "200", // Default sales account
		},
	}

	// Add tax line if applicable
	if invoice.TaxAmount > 0 {
		lineItems[0]["TaxAmount"] = float64(invoice.TaxAmount) / 100
	}

	invoiceData := map[string]interface{}{
		"Type": "ACCREC",
		"Contact": map[string]string{
			"ContactID": invoice.CustomerID.String(),
		},
		"LineItems":    lineItems,
		"CurrencyCode": invoice.Currency,
		"Status":       "AUTHORISED",
		"Reference":    invoice.InvoiceNumber,
	}

	// Add due date
	if !invoice.DueDate.IsZero() {
		invoiceData["DueDate"] = invoice.DueDate.Format("2006-01-02")
	}

	xeroInvoice := map[string]interface{}{
		"Invoices": []map[string]interface{}{invoiceData},
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
		return fmt.Errorf("xero invoice sync failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("xero API error: status %d", resp.StatusCode)
	}

	return nil
}

func (a *XeroAdapter) SyncProduct(ctx context.Context, plan *domain.Plan) error {
	// Check for existing item by name to avoid duplicates
	existing, err := a.findItemByName(ctx, plan.Name)
	if err == nil && existing != "" {
		return nil
	}

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
		return fmt.Errorf("xero product sync failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("xero API error: status %d", resp.StatusCode)
	}

	return nil
}

// escapeXeroFilter escapes double quotes in Xero OData filter values
func escapeXeroFilter(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}

// findContactByEmail queries Xero for existing contact by email
func (a *XeroAdapter) findContactByEmail(ctx context.Context, email string) (string, error) {
	filter := fmt.Sprintf(`EmailAddress=="%s"`, escapeXeroFilter(email))
	reqURL := fmt.Sprintf("%s/Contacts?where=%s", a.baseURL, url.QueryEscape(filter))

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return "", err
	}
	a.setHeaders(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("query failed: status %d", resp.StatusCode)
	}

	var result struct {
		Contacts []struct {
			ContactID string `json:"ContactID"`
		} `json:"Contacts"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if len(result.Contacts) > 0 {
		return result.Contacts[0].ContactID, nil
	}
	return "", fmt.Errorf("not found")
}

// findItemByName queries Xero for existing item by name
func (a *XeroAdapter) findItemByName(ctx context.Context, name string) (string, error) {
	filter := fmt.Sprintf(`Name=="%s"`, escapeXeroFilter(name))
	reqURL := fmt.Sprintf("%s/Items?where=%s", a.baseURL, url.QueryEscape(filter))

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return "", err
	}
	a.setHeaders(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("query failed: status %d", resp.StatusCode)
	}

	var result struct {
		Items []struct {
			ItemID string `json:"ItemID"`
		} `json:"Items"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if len(result.Items) > 0 {
		return result.Items[0].ItemID, nil
	}
	return "", fmt.Errorf("not found")
}

func (a *XeroAdapter) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+a.accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Xero-tenant-id", a.tenantID)
}
