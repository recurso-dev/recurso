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

	"github.com/recur-so/recurso/internal/core/domain"
)

type QuickBooksAdapter struct {
	baseURL          string
	accessToken      string
	realmID          string
	incomeAccountRef string
}

func NewQuickBooksAdapter(accessToken, realmID string, sandbox bool) *QuickBooksAdapter {
	baseURL := "https://quickbooks.api.intuit.com"
	if sandbox {
		baseURL = "https://sandbox-quickbooks.api.intuit.com"
	}

	return &QuickBooksAdapter{
		baseURL:          baseURL,
		accessToken:      accessToken,
		realmID:          realmID,
		incomeAccountRef: "1", // Default, configurable
	}
}

func (a *QuickBooksAdapter) SetCredentials(accessToken, realmID string) {
	a.accessToken = accessToken
	a.realmID = realmID
}

func (a *QuickBooksAdapter) SetIncomeAccountRef(ref string) {
	a.incomeAccountRef = ref
}

func (a *QuickBooksAdapter) SyncCustomer(ctx context.Context, customer *domain.Customer) error {
	// Check for existing customer by email to avoid duplicates
	existing, err := a.findCustomerByEmail(ctx, customer.Email)
	if err == nil && existing != "" {
		// Customer already exists, skip creation
		return nil
	}

	name := ""
	if customer.Name != nil {
		name = *customer.Name
	}

	qbCustomer := map[string]interface{}{
		"DisplayName": name,
		"PrimaryEmailAddr": map[string]string{
			"Address": customer.Email,
		},
	}

	// Add phone if available
	if customer.Phone != "" {
		qbCustomer["PrimaryPhone"] = map[string]string{
			"FreeFormNumber": customer.Phone,
		}
	}

	// Add billing address
	if customer.BillingAddress.Line1 != "" {
		qbCustomer["BillAddr"] = map[string]string{
			"Line1":                  customer.BillingAddress.Line1,
			"City":                   customer.BillingAddress.City,
			"CountrySubDivisionCode": customer.BillingAddress.State,
			"PostalCode":             customer.BillingAddress.Zip,
			"Country":                customer.BillingAddress.Country,
		}
	}

	// Add tax ID
	if customer.TaxID != nil && *customer.TaxID != "" {
		qbCustomer["ResaleNum"] = *customer.TaxID
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

	a.setHeaders(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("QuickBooks customer sync failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("QuickBooks API error: status %d", resp.StatusCode)
	}

	return nil
}

func (a *QuickBooksAdapter) SyncInvoice(ctx context.Context, invoice *domain.Invoice) error {
	lineItems := []map[string]interface{}{
		{
			"Amount":      float64(invoice.Subtotal) / 100,
			"DetailType":  "SalesItemLineDetail",
			"Description": fmt.Sprintf("Invoice %s", invoice.InvoiceNumber),
			"SalesItemLineDetail": map[string]interface{}{
				"UnitPrice": float64(invoice.Subtotal) / 100,
				"Qty":       1,
			},
		},
	}

	qbInvoice := map[string]interface{}{
		"CustomerRef": map[string]string{
			"value": invoice.CustomerID.String(),
		},
		"Line":        lineItems,
		"DocNumber":   invoice.InvoiceNumber,
		"CurrencyRef": map[string]string{"value": invoice.Currency},
	}

	// Add tax detail if tax exists
	if invoice.TaxAmount > 0 {
		qbInvoice["TxnTaxDetail"] = map[string]interface{}{
			"TotalTax": float64(invoice.TaxAmount) / 100,
		}
	}

	// Add due date
	if !invoice.DueDate.IsZero() {
		qbInvoice["DueDate"] = invoice.DueDate.Format("2006-01-02")
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

	a.setHeaders(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("QuickBooks invoice sync failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("QuickBooks API error: status %d", resp.StatusCode)
	}

	return nil
}

func (a *QuickBooksAdapter) SyncProduct(ctx context.Context, plan *domain.Plan) error {
	// Check for existing item by name to avoid duplicates
	existing, err := a.findItemByName(ctx, plan.Name)
	if err == nil && existing != "" {
		return nil
	}

	qbItem := map[string]interface{}{
		"Name": plan.Name,
		"Type": "Service",
		"IncomeAccountRef": map[string]string{
			"value": a.incomeAccountRef,
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

	a.setHeaders(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("QuickBooks product sync failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("QuickBooks API error: status %d", resp.StatusCode)
	}

	return nil
}

// escapeQBOValue escapes single quotes for QuickBooks query language
func escapeQBOValue(s string) string {
	return strings.ReplaceAll(s, "'", "\\'")
}

// findCustomerByEmail queries QuickBooks for an existing customer by email
func (a *QuickBooksAdapter) findCustomerByEmail(ctx context.Context, email string) (string, error) {
	query := fmt.Sprintf("SELECT Id FROM Customer WHERE PrimaryEmailAddr = '%s'", escapeQBOValue(email))
	reqURL := fmt.Sprintf("%s/v3/company/%s/query?query=%s", a.baseURL, a.realmID, url.QueryEscape(query))

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
		QueryResponse struct {
			Customer []struct {
				Id string `json:"Id"`
			} `json:"Customer"`
		} `json:"QueryResponse"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if len(result.QueryResponse.Customer) > 0 {
		return result.QueryResponse.Customer[0].Id, nil
	}
	return "", fmt.Errorf("not found")
}

// findItemByName queries QuickBooks for an existing item by name
func (a *QuickBooksAdapter) findItemByName(ctx context.Context, name string) (string, error) {
	query := fmt.Sprintf("SELECT Id FROM Item WHERE Name = '%s'", escapeQBOValue(name))
	reqURL := fmt.Sprintf("%s/v3/company/%s/query?query=%s", a.baseURL, a.realmID, url.QueryEscape(query))

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
		QueryResponse struct {
			Item []struct {
				Id string `json:"Id"`
			} `json:"Item"`
		} `json:"QueryResponse"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if len(result.QueryResponse.Item) > 0 {
		return result.QueryResponse.Item[0].Id, nil
	}
	return "", fmt.Errorf("not found")
}

func (a *QuickBooksAdapter) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+a.accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
}
