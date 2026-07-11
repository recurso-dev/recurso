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

// QBO fault codes we react to. Intuit signals most business errors as HTTP
// 400 with a Fault body; the code disambiguates.
const (
	qboFaultObjectNotFound = "610"  // Object Not Found — entity deleted/never existed
	qboFaultStaleObject    = "5010" // Stale Object Error — SyncToken out of date
)

// QuickBooksAdapter syncs entities to QuickBooks Online.
//
// Create-or-update semantics: when the caller passes a known externalID the
// adapter GETs the entity to obtain its current SyncToken, then issues a
// sparse update (POST with Id + SyncToken + sparse:true). A stale-token
// rejection is retried once with a refetched token. When QBO reports the
// object gone (HTTP 404 or fault 610), port.ErrExternalGone is returned so
// the service can clear the stale mapping and re-create.
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

// SyncCustomer creates or updates the customer in QuickBooks and returns the
// provider Customer.Id. With an externalID it performs a sparse update of
// that object; without one it creates the customer (deduping by email — if a
// customer with the same email already exists, its Id is returned instead).
func (a *QuickBooksAdapter) SyncCustomer(ctx context.Context, customer *domain.Customer, externalID string) (string, error) {
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

	if externalID != "" {
		id, err := a.update(ctx, "customer", "Customer", externalID, qbCustomer)
		if err != nil {
			return "", fmt.Errorf("QuickBooks customer sync failed: %w", err)
		}
		return id, nil
	}

	// Check for existing customer by email to avoid duplicates
	if existing, err := a.findCustomerByEmail(ctx, customer.Email); err == nil && existing != "" {
		return existing, nil
	}

	respBody, err := a.post(ctx, "customer", qbCustomer)
	if err != nil {
		return "", fmt.Errorf("QuickBooks customer sync failed: %w", err)
	}
	id, _, err := parseQBOEntity("Customer", respBody)
	if err != nil {
		return "", err
	}
	return id, nil
}

// SyncInvoice creates or updates the invoice in QuickBooks and returns the
// provider Invoice.Id. refs.CustomerExternalID must be the QuickBooks
// Customer.Id (from a prior SyncCustomer), used as CustomerRef.value — QBO
// rejects unknown refs. When refs.ProductExternalID is set (the QuickBooks
// Item.Id of the plan backing the invoice) it is attached to the line as
// SalesItemLineDetail.ItemRef; otherwise the line is sent bare and QBO
// attributes it to its default item.
func (a *QuickBooksAdapter) SyncInvoice(ctx context.Context, invoice *domain.Invoice, refs port.InvoiceSyncRefs, externalID string) (string, error) {
	if refs.CustomerExternalID == "" {
		return "", fmt.Errorf("QuickBooks invoice sync requires the customer's QuickBooks Id (CustomerRef)")
	}

	lineDetail := map[string]interface{}{
		"UnitPrice": float64(invoice.Subtotal) / 100,
		"Qty":       1,
	}
	if refs.ProductExternalID != "" {
		lineDetail["ItemRef"] = map[string]string{"value": refs.ProductExternalID}
	}

	lineItems := []map[string]interface{}{
		{
			"Amount":              float64(invoice.Subtotal) / 100,
			"DetailType":          "SalesItemLineDetail",
			"Description":         fmt.Sprintf("Invoice %s", invoice.InvoiceNumber),
			"SalesItemLineDetail": lineDetail,
		},
	}

	qbInvoice := map[string]interface{}{
		"CustomerRef": map[string]string{
			"value": refs.CustomerExternalID,
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

	if externalID != "" {
		id, err := a.update(ctx, "invoice", "Invoice", externalID, qbInvoice)
		if err != nil {
			return "", fmt.Errorf("QuickBooks invoice sync failed: %w", err)
		}
		return id, nil
	}

	respBody, err := a.post(ctx, "invoice", qbInvoice)
	if err != nil {
		return "", fmt.Errorf("QuickBooks invoice sync failed: %w", err)
	}
	id, _, err := parseQBOEntity("Invoice", respBody)
	if err != nil {
		return "", err
	}
	return id, nil
}

// SyncProduct creates or updates the plan as a QuickBooks service Item and
// returns the provider Item.Id. With an externalID it performs a sparse
// update; without one it creates the item (deduping by name — if an item
// with the same name already exists, its Id is returned instead).
func (a *QuickBooksAdapter) SyncProduct(ctx context.Context, plan *domain.Plan, externalID string) (string, error) {
	qbItem := map[string]interface{}{
		"Name": plan.Name,
		"Type": "Service",
		"IncomeAccountRef": map[string]string{
			"value": a.incomeAccountRef,
		},
	}

	if externalID != "" {
		id, err := a.update(ctx, "item", "Item", externalID, qbItem)
		if err != nil {
			return "", fmt.Errorf("QuickBooks product sync failed: %w", err)
		}
		return id, nil
	}

	// Check for existing item by name to avoid duplicates
	if existing, err := a.findItemByName(ctx, plan.Name); err == nil && existing != "" {
		return existing, nil
	}

	respBody, err := a.post(ctx, "item", qbItem)
	if err != nil {
		return "", fmt.Errorf("QuickBooks product sync failed: %w", err)
	}
	id, _, err := parseQBOEntity("Item", respBody)
	if err != nil {
		return "", err
	}
	return id, nil
}

// update performs a QBO sparse update: GET the entity for its current
// SyncToken, then POST the payload with Id + SyncToken + sparse:true. A
// stale-token rejection (fault 5010) is retried once with a refetched token.
// When the object is gone at the provider (HTTP 404 / fault 610) the returned
// error wraps port.ErrExternalGone.
func (a *QuickBooksAdapter) update(ctx context.Context, objectType, jsonKey, externalID string, payload map[string]interface{}) (string, error) {
	syncToken, err := a.fetchSyncToken(ctx, objectType, jsonKey, externalID)
	if err != nil {
		return "", err
	}

	for attempt := 0; ; attempt++ {
		payload["Id"] = externalID
		payload["SyncToken"] = syncToken
		payload["sparse"] = true

		respBody, err := a.post(ctx, objectType, payload)
		if err == nil {
			id, _, perr := parseQBOEntity(jsonKey, respBody)
			if perr != nil {
				return "", perr
			}
			return id, nil
		}

		var apiErr *qboAPIError
		if errors.As(err, &apiErr) {
			if apiErr.gone() {
				return "", fmt.Errorf("quickbooks %s %s: %w", objectType, externalID, port.ErrExternalGone)
			}
			if apiErr.stale() && attempt == 0 {
				syncToken, err = a.fetchSyncToken(ctx, objectType, jsonKey, externalID)
				if err != nil {
					return "", err
				}
				continue
			}
		}
		return "", err
	}
}

// fetchSyncToken GETs the entity and returns its current SyncToken. A 404 or
// QBO fault 610 (Object Not Found) is mapped to port.ErrExternalGone.
func (a *QuickBooksAdapter) fetchSyncToken(ctx context.Context, objectType, jsonKey, externalID string) (string, error) {
	reqURL := fmt.Sprintf("%s/v3/company/%s/%s/%s", a.baseURL, a.realmID, objectType, url.PathEscape(externalID))
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	a.setHeaders(req)

	body, err := a.do(req)
	if err != nil {
		var apiErr *qboAPIError
		if errors.As(err, &apiErr) && apiErr.gone() {
			return "", fmt.Errorf("quickbooks %s %s: %w", objectType, externalID, port.ErrExternalGone)
		}
		return "", fmt.Errorf("failed to fetch %s %s for update: %w", objectType, externalID, err)
	}

	_, syncToken, err := parseQBOEntity(jsonKey, body)
	if err != nil {
		return "", err
	}
	return syncToken, nil
}

// post POSTs a payload to the QBO entity endpoint (create, or update when the
// payload carries Id+SyncToken) and returns the raw response body. HTTP >=400
// responses are returned as *qboAPIError with the fault codes parsed out.
func (a *QuickBooksAdapter) post(ctx context.Context, objectType string, payload map[string]interface{}) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal %s: %w", objectType, err)
	}

	reqURL := fmt.Sprintf("%s/v3/company/%s/%s", a.baseURL, a.realmID, objectType)
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	a.setHeaders(req)
	return a.do(req)
}

// do executes the request and returns the response body. HTTP >=400
// responses become a *qboAPIError carrying the status and any QBO fault
// codes so callers can classify stale-token and object-gone failures.
func (a *QuickBooksAdapter) do(req *http.Request) ([]byte, error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, newQBOAPIError(resp.StatusCode, respBody)
	}
	return respBody, nil
}

// qboAPIError is an HTTP-level failure from the QBO API, enriched with the
// fault codes Intuit embeds in the response body.
type qboAPIError struct {
	status int
	codes  []string
}

func newQBOAPIError(status int, body []byte) *qboAPIError {
	var fault struct {
		Fault struct {
			Error []struct {
				Code string `json:"code"`
			} `json:"Error"`
		} `json:"Fault"`
	}
	_ = json.Unmarshal(body, &fault) // best effort; non-JSON bodies leave codes empty

	e := &qboAPIError{status: status}
	for _, fe := range fault.Fault.Error {
		e.codes = append(e.codes, fe.Code)
	}
	return e
}

func (e *qboAPIError) Error() string {
	if len(e.codes) > 0 {
		return fmt.Sprintf("QuickBooks API error: status %d (fault codes %s)", e.status, strings.Join(e.codes, ","))
	}
	return fmt.Sprintf("QuickBooks API error: status %d", e.status)
}

func (e *qboAPIError) hasCode(code string) bool {
	for _, c := range e.codes {
		if c == code {
			return true
		}
	}
	return false
}

// gone reports whether the error means the referenced object no longer
// exists at QBO. Intuit uses both plain 404s and 400+fault-610 for this.
func (e *qboAPIError) gone() bool {
	return e.status == http.StatusNotFound || e.hasCode(qboFaultObjectNotFound)
}

// stale reports whether the update was rejected for an out-of-date SyncToken.
func (e *qboAPIError) stale() bool {
	return e.hasCode(qboFaultStaleObject)
}

// parseQBOEntity extracts Id and SyncToken from a QBO entity envelope such
// as {"Customer": {"Id": "42", "SyncToken": "3", ...}}.
func parseQBOEntity(jsonKey string, body []byte) (id, syncToken string, err error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return "", "", fmt.Errorf("failed to parse QuickBooks %s response: %w", jsonKey, err)
	}
	raw, ok := envelope[jsonKey]
	if !ok {
		return "", "", fmt.Errorf("QuickBooks response missing %q", jsonKey)
	}
	var entity struct {
		Id        string `json:"Id"`
		SyncToken string `json:"SyncToken"`
	}
	if err := json.Unmarshal(raw, &entity); err != nil {
		return "", "", fmt.Errorf("failed to parse QuickBooks %s response: %w", jsonKey, err)
	}
	if entity.Id == "" {
		return "", "", fmt.Errorf("QuickBooks %s response missing Id", jsonKey)
	}
	return entity.Id, entity.SyncToken, nil
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
