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

// QuickBooksAdapter syncs entities to QuickBooks Online.
//
// Update gap: this adapter only creates objects. A true QBO update requires
// fetching the object first to obtain its current SyncToken and issuing a
// sparse update; that is not implemented yet. The service layer enforces
// create-once semantics via accounting_entity_mappings — once a mapping
// exists the adapter is not called again for that entity, so subsequent
// local changes are NOT pushed to QuickBooks.
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

// SyncCustomer creates the customer in QuickBooks and returns the provider
// Customer.Id. If a customer with the same email already exists, its Id is
// returned without creating a duplicate.
func (a *QuickBooksAdapter) SyncCustomer(ctx context.Context, customer *domain.Customer) (string, error) {
	// Check for existing customer by email to avoid duplicates
	existing, err := a.findCustomerByEmail(ctx, customer.Email)
	if err == nil && existing != "" {
		return existing, nil
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

	respBody, err := a.create(ctx, "customer", qbCustomer)
	if err != nil {
		return "", fmt.Errorf("QuickBooks customer sync failed: %w", err)
	}

	var created struct {
		Customer struct {
			Id string `json:"Id"`
		} `json:"Customer"`
	}
	if err := json.Unmarshal(respBody, &created); err != nil {
		return "", fmt.Errorf("failed to parse QuickBooks customer response: %w", err)
	}
	if created.Customer.Id == "" {
		return "", fmt.Errorf("QuickBooks customer response missing Id")
	}
	return created.Customer.Id, nil
}

// SyncInvoice creates the invoice in QuickBooks and returns the provider
// Invoice.Id. customerExternalID must be the QuickBooks Customer.Id (from a
// prior SyncCustomer), used as CustomerRef.value — QBO rejects unknown refs.
//
// Note: line items are sent as bare SalesItemLineDetail entries with a
// Description and no ItemRef; QuickBooks tolerates this by attributing the
// line to its default item. Mapping plan/product ItemRefs onto invoice lines
// is a known gap.
func (a *QuickBooksAdapter) SyncInvoice(ctx context.Context, invoice *domain.Invoice, customerExternalID string) (string, error) {
	if customerExternalID == "" {
		return "", fmt.Errorf("QuickBooks invoice sync requires the customer's QuickBooks Id (CustomerRef)")
	}

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
			"value": customerExternalID,
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

	respBody, err := a.create(ctx, "invoice", qbInvoice)
	if err != nil {
		return "", fmt.Errorf("QuickBooks invoice sync failed: %w", err)
	}

	var created struct {
		Invoice struct {
			Id string `json:"Id"`
		} `json:"Invoice"`
	}
	if err := json.Unmarshal(respBody, &created); err != nil {
		return "", fmt.Errorf("failed to parse QuickBooks invoice response: %w", err)
	}
	if created.Invoice.Id == "" {
		return "", fmt.Errorf("QuickBooks invoice response missing Id")
	}
	return created.Invoice.Id, nil
}

// SyncProduct creates the plan as a QuickBooks service Item and returns the
// provider Item.Id. If an item with the same name already exists, its Id is
// returned without creating a duplicate.
func (a *QuickBooksAdapter) SyncProduct(ctx context.Context, plan *domain.Plan) (string, error) {
	// Check for existing item by name to avoid duplicates
	existing, err := a.findItemByName(ctx, plan.Name)
	if err == nil && existing != "" {
		return existing, nil
	}

	qbItem := map[string]interface{}{
		"Name": plan.Name,
		"Type": "Service",
		"IncomeAccountRef": map[string]string{
			"value": a.incomeAccountRef,
		},
	}

	respBody, err := a.create(ctx, "item", qbItem)
	if err != nil {
		return "", fmt.Errorf("QuickBooks product sync failed: %w", err)
	}

	var created struct {
		Item struct {
			Id string `json:"Id"`
		} `json:"Item"`
	}
	if err := json.Unmarshal(respBody, &created); err != nil {
		return "", fmt.Errorf("failed to parse QuickBooks item response: %w", err)
	}
	if created.Item.Id == "" {
		return "", fmt.Errorf("QuickBooks item response missing Id")
	}
	return created.Item.Id, nil
}

// create POSTs a payload to the QBO create endpoint for the given object
// type and returns the raw response body.
func (a *QuickBooksAdapter) create(ctx context.Context, objectType string, payload map[string]interface{}) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal %s: %w", objectType, err)
	}

	url := fmt.Sprintf("%s/v3/company/%s/%s", a.baseURL, a.realmID, objectType)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
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
		return nil, fmt.Errorf("QuickBooks API error: status %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	return respBody, nil
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
