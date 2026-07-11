package accounting

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// XeroAdapter syncs entities to Xero.
//
// Create-or-update semantics: Xero's POST endpoints act as upserts when the
// payload carries the provider ID (ContactID, InvoiceID, ItemID), so when the
// caller passes a known externalID the adapter simply POSTs with that ID
// embedded. When Xero responds 404 for a carried ID (object deleted at the
// provider), port.ErrExternalGone is returned so the service can clear the
// stale mapping and re-create.
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

// SyncCustomer creates or updates the customer as a Xero Contact and returns
// the provider ContactID. With an externalID the payload carries that
// ContactID and Xero updates in place; without one a contact is created
// (deduping by email — if a contact with the same email already exists, its
// ContactID is returned instead).
func (a *XeroAdapter) SyncCustomer(ctx context.Context, customer *domain.Customer, externalID string) (string, error) {
	if externalID == "" {
		// Check for existing contact by email to avoid duplicates
		if existing, err := a.findContactByEmail(ctx, customer.Email); err == nil && existing != "" {
			return existing, nil
		}
	}

	name := ""
	if customer.Name != nil {
		name = *customer.Name
	}

	contactData := map[string]interface{}{
		"Name":         name,
		"EmailAddress": customer.Email,
	}
	if externalID != "" {
		contactData["ContactID"] = externalID
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
		if externalID != "" && isXeroGone(err) {
			return "", fmt.Errorf("xero contact %s: %w", externalID, port.ErrExternalGone)
		}
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

// SyncInvoice creates or updates the invoice in Xero and returns the provider
// InvoiceID. refs.CustomerExternalID must be the Xero ContactID (from a prior
// SyncCustomer) — Xero rejects invoices referencing unknown contacts. With an
// externalID the payload carries that InvoiceID and Xero updates in place.
//
// Xero invoice lines reference items by item Code, not ItemID, so
// refs.ProductExternalID is ignored; when refs.ProductCode is set it is
// attached to the line as ItemCode (matching the Code that SyncProduct gives
// the item). Without it the line is sent bare with description + AccountCode.
func (a *XeroAdapter) SyncInvoice(ctx context.Context, invoice *domain.Invoice, refs port.InvoiceSyncRefs, externalID string) (string, error) {
	if refs.CustomerExternalID == "" {
		return "", fmt.Errorf("xero invoice sync requires the customer's Xero ContactID")
	}

	lineItem := map[string]interface{}{
		"Description": fmt.Sprintf("Invoice %s", invoice.InvoiceNumber),
		"Quantity":    1,
		"UnitAmount":  float64(invoice.Subtotal) / 100,
		"AccountCode": "200", // Default sales account
	}
	if refs.ProductCode != "" {
		lineItem["ItemCode"] = xeroItemCode(refs.ProductCode)
	}
	lineItems := []map[string]interface{}{lineItem}

	// Add tax line if applicable
	if invoice.TaxAmount > 0 {
		lineItems[0]["TaxAmount"] = float64(invoice.TaxAmount) / 100
	}

	invoiceData := map[string]interface{}{
		"Type": "ACCREC",
		"Contact": map[string]string{
			"ContactID": refs.CustomerExternalID,
		},
		"LineItems":    lineItems,
		"CurrencyCode": invoice.Currency,
		"Status":       "AUTHORISED",
		"Reference":    invoice.InvoiceNumber,
	}
	if externalID != "" {
		invoiceData["InvoiceID"] = externalID
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
		if externalID != "" && isXeroGone(err) {
			return "", fmt.Errorf("xero invoice %s: %w", externalID, port.ErrExternalGone)
		}
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

// SyncProduct creates or updates the plan as a Xero Item and returns the
// provider ItemID. With an externalID the payload carries that ItemID and
// Xero updates in place; without one an item is created (deduping by name —
// if an item with the same name already exists, its ItemID is returned
// instead).
//
// The item always carries a Code derived from the plan's code — Xero links
// invoice lines to items by that Code (see SyncInvoice), so it must match
// the ItemCode the invoice lines carry.
func (a *XeroAdapter) SyncProduct(ctx context.Context, plan *domain.Plan, externalID string) (string, error) {
	if externalID == "" {
		// Check for existing item by name to avoid duplicates
		if existing, err := a.findItemByName(ctx, plan.Name); err == nil && existing != "" {
			return existing, nil
		}
	}

	code := plan.Code
	if code == "" {
		code = plan.ID.String()[:8] // plans always have a code; defensive fallback
	}
	itemData := map[string]interface{}{
		"Code":        xeroItemCode(code),
		"Name":        plan.Name,
		"Description": plan.Name,
	}
	if externalID != "" {
		itemData["ItemID"] = externalID
	}

	item := map[string]interface{}{
		"Items": []map[string]interface{}{itemData},
	}

	respBody, err := a.post(ctx, "/Items", item)
	if err != nil {
		if externalID != "" && isXeroGone(err) {
			return "", fmt.Errorf("xero item %s: %w", externalID, port.ErrExternalGone)
		}
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

// xeroItemCode normalizes a plan code into a Xero Item Code. Xero caps item
// codes at 30 characters; applying the same truncation to the item's Code
// (SyncProduct) and the invoice line's ItemCode (SyncInvoice) keeps them
// linked even for long plan codes.
func xeroItemCode(code string) string {
	const maxLen = 30
	if len(code) > maxLen {
		return code[:maxLen]
	}
	return code
}

// xeroAPIError is an HTTP-level failure from the Xero API.
type xeroAPIError struct {
	status int
}

func (e *xeroAPIError) Error() string {
	return fmt.Sprintf("xero API error: status %d", e.status)
}

// isXeroGone reports whether the error is Xero saying the object referenced
// by the payload's provider ID does not exist (404 on an ID-carrying POST).
func isXeroGone(err error) bool {
	var apiErr *xeroAPIError
	return errors.As(err, &apiErr) && apiErr.status == http.StatusNotFound
}

// post sends a JSON payload to the Xero API and returns the raw response
// body. HTTP >=400 responses are returned as *xeroAPIError.
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
		return nil, &xeroAPIError{status: resp.StatusCode}
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
