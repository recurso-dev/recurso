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

// XeroAdapter syncs entities to Xero.
//
// Update gap: this adapter only creates objects. Xero's POST endpoints act
// as upserts when the payload carries the provider ID (e.g. ContactID), but
// we do not resend already-synced entities: the service layer enforces
// create-once semantics via accounting_entity_mappings, so subsequent local
// changes are NOT pushed to Xero.
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

// SyncCustomer creates the customer as a Xero Contact and returns the
// provider ContactID. If a contact with the same email already exists, its
// ContactID is returned without creating a duplicate.
func (a *XeroAdapter) SyncCustomer(ctx context.Context, customer *domain.Customer) (string, error) {
	// Check for existing contact by email to avoid duplicates
	existing, err := a.findContactByEmail(ctx, customer.Email)
	if err == nil && existing != "" {
		return existing, nil
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

	respBody, err := a.post(ctx, "/Contacts", contact)
	if err != nil {
		return "", fmt.Errorf("xero customer sync failed: %w", err)
	}

	var created struct {
		Contacts []struct {
			ContactID string `json:"ContactID"`
		} `json:"Contacts"`
	}
	if err := json.Unmarshal(respBody, &created); err != nil {
		return "", fmt.Errorf("failed to parse xero contact response: %w", err)
	}
	if len(created.Contacts) == 0 || created.Contacts[0].ContactID == "" {
		return "", fmt.Errorf("xero contact response missing ContactID")
	}
	return created.Contacts[0].ContactID, nil
}

// SyncInvoice creates the invoice in Xero and returns the provider
// InvoiceID. customerExternalID must be the Xero ContactID (from a prior
// SyncCustomer) — Xero rejects invoices referencing unknown contacts.
func (a *XeroAdapter) SyncInvoice(ctx context.Context, invoice *domain.Invoice, customerExternalID string) (string, error) {
	if customerExternalID == "" {
		return "", fmt.Errorf("xero invoice sync requires the customer's Xero ContactID")
	}

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
			"ContactID": customerExternalID,
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

	respBody, err := a.post(ctx, "/Invoices", xeroInvoice)
	if err != nil {
		return "", fmt.Errorf("xero invoice sync failed: %w", err)
	}

	var created struct {
		Invoices []struct {
			InvoiceID string `json:"InvoiceID"`
		} `json:"Invoices"`
	}
	if err := json.Unmarshal(respBody, &created); err != nil {
		return "", fmt.Errorf("failed to parse xero invoice response: %w", err)
	}
	if len(created.Invoices) == 0 || created.Invoices[0].InvoiceID == "" {
		return "", fmt.Errorf("xero invoice response missing InvoiceID")
	}
	return created.Invoices[0].InvoiceID, nil
}

// SyncProduct creates the plan as a Xero Item and returns the provider
// ItemID. If an item with the same name already exists, its ItemID is
// returned without creating a duplicate.
func (a *XeroAdapter) SyncProduct(ctx context.Context, plan *domain.Plan) (string, error) {
	// Check for existing item by name to avoid duplicates
	existing, err := a.findItemByName(ctx, plan.Name)
	if err == nil && existing != "" {
		return existing, nil
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

	respBody, err := a.post(ctx, "/Items", item)
	if err != nil {
		return "", fmt.Errorf("xero product sync failed: %w", err)
	}

	var created struct {
		Items []struct {
			ItemID string `json:"ItemID"`
		} `json:"Items"`
	}
	if err := json.Unmarshal(respBody, &created); err != nil {
		return "", fmt.Errorf("failed to parse xero item response: %w", err)
	}
	if len(created.Items) == 0 || created.Items[0].ItemID == "" {
		return "", fmt.Errorf("xero item response missing ItemID")
	}
	return created.Items[0].ItemID, nil
}

// post sends a JSON payload to the Xero API and returns the raw response
// body.
func (a *XeroAdapter) post(ctx context.Context, path string, payload interface{}) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	a.setHeaders(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("xero API error: status %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	return respBody, nil
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
