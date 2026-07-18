package taxprovider

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/recurso-dev/recurso/internal/core/service/tax"
)

// Avalara AvaTax provider (Track D3, spec_lago_parity.md). Implements
// tax.SalesTaxProvider against POST /api/v2/transactions/create with
// type=SalesOrder — a rate *quote* that is never committed to the Avalara
// ledger, mirroring how the TaxJar provider only reads rates.
//
// EXPERIMENTAL: built against the AvaTax REST v2 reference; sandbox
// verification is founder-gated.

// DefaultAvalaraURL is AvaTax production; point AVALARA_API_URL at
// https://sandbox-rest.avatax.com for the sandbox.
const DefaultAvalaraURL = "https://rest.avatax.com"

const avalaraTimeout = 10 * time.Second

// Typed error kinds, mirroring the TaxJar taxonomy so callers share one
// degradation policy.
var (
	ErrAvalaraAuth        = errors.New("avalara: authentication failed")
	ErrAvalaraBadRequest  = errors.New("avalara: invalid request")
	ErrAvalaraUnavailable = errors.New("avalara: service unavailable")
)

// AvalaraError carries the HTTP detail behind one of the sentinels.
type AvalaraError struct {
	Kind       error
	StatusCode int
	Detail     string
}

func (e *AvalaraError) Error() string {
	if e.StatusCode == 0 {
		return fmt.Sprintf("%v: %s", e.Kind, e.Detail)
	}
	return fmt.Sprintf("%v (HTTP %d): %s", e.Kind, e.StatusCode, e.Detail)
}

func (e *AvalaraError) Unwrap() error { return e.Kind }

// AvalaraProvider implements tax.SalesTaxProvider.
type AvalaraProvider struct {
	accountID   string
	licenseKey  string
	companyCode string
	baseURL     string
	httpClient  *http.Client
}

// NewAvalaraProvider builds the provider. companyCode is the AvaTax
// company the quote runs under ("DEFAULT" when empty). baseURL "" uses
// production.
func NewAvalaraProvider(accountID, licenseKey, companyCode, baseURL string) *AvalaraProvider {
	if baseURL == "" {
		baseURL = DefaultAvalaraURL
	}
	if companyCode == "" {
		companyCode = "DEFAULT"
	}
	return &AvalaraProvider{
		accountID:   accountID,
		licenseKey:  licenseKey,
		companyCode: companyCode,
		baseURL:     strings.TrimRight(baseURL, "/"),
		httpClient:  &http.Client{Timeout: avalaraTimeout},
	}
}

func (p *AvalaraProvider) Name() string { return "avalara" }

// avalaraTxRequest is the SalesOrder quote request.
type avalaraTxRequest struct {
	Type         string             `json:"type"` // SalesOrder = quote only, never committed
	CompanyCode  string             `json:"companyCode"`
	Date         string             `json:"date"`
	CustomerCode string             `json:"customerCode"`
	CurrencyCode string             `json:"currencyCode"`
	Addresses    map[string]avaAddr `json:"addresses"`
	Lines        []avaLine          `json:"lines"`
}

type avaAddr struct {
	Country    string `json:"country"`
	Region     string `json:"region,omitempty"`
	PostalCode string `json:"postalCode,omitempty"`
}

type avaLine struct {
	Amount float64 `json:"amount"` // major units
}

type avalaraTxResponse struct {
	TotalTax float64 `json:"totalTax"`
	Summary  []struct {
		Region    string  `json:"region"`
		JurisName string  `json:"jurisName"`
		Rate      float64 `json:"rate"`
	} `json:"summary"`
}

// LookupSalesTax quotes the tax via an uncommitted SalesOrder.
func (p *AvalaraProvider) LookupSalesTax(ctx context.Context, q *tax.SalesTaxQuery) (*tax.SalesTaxResult, error) {
	reqBody := avalaraTxRequest{
		Type:         "SalesOrder",
		CompanyCode:  p.companyCode,
		Date:         time.Now().UTC().Format("2006-01-02"),
		CustomerCode: "recurso-quote",
		CurrencyCode: strings.ToUpper(q.Currency),
		Addresses: map[string]avaAddr{
			"shipFrom": {Country: q.FromCountry, Region: q.FromState, PostalCode: q.FromZip},
			"shipTo":   {Country: q.ToCountry, Region: q.ToState, PostalCode: q.ToZip},
		},
		Lines: []avaLine{{Amount: float64(q.Amount) / 100.0}},
	}
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/v2/transactions/create", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	creds := base64.StdEncoding.EncodeToString([]byte(p.accountID + ":" + p.licenseKey))
	req.Header.Set("Authorization", "Basic "+creds)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, &AvalaraError{Kind: ErrAvalaraUnavailable, Detail: err.Error()}
	}
	defer func() { _ = resp.Body.Close() }()
	payload, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return nil, &AvalaraError{Kind: ErrAvalaraAuth, StatusCode: resp.StatusCode, Detail: string(payload)}
	case resp.StatusCode >= 500:
		return nil, &AvalaraError{Kind: ErrAvalaraUnavailable, StatusCode: resp.StatusCode, Detail: string(payload)}
	case resp.StatusCode >= 400:
		return nil, &AvalaraError{Kind: ErrAvalaraBadRequest, StatusCode: resp.StatusCode, Detail: string(payload)}
	}

	var out avalaraTxResponse
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil, &AvalaraError{Kind: ErrAvalaraUnavailable, Detail: "bad response: " + err.Error()}
	}

	taxAmount := int64(math.Round(out.TotalTax * 100))
	rate := 0.0
	juris := make([]string, 0, len(out.Summary))
	for _, sum := range out.Summary {
		rate += sum.Rate
		if sum.JurisName != "" {
			juris = append(juris, sum.JurisName)
		}
	}
	jurisdiction := strings.ToUpper(q.ToCountry) + "/" + strings.ToUpper(q.ToState)
	if len(juris) > 0 {
		jurisdiction += "/" + strings.Join(juris, "+")
	}
	return &tax.SalesTaxResult{
		Rate:         rate,
		TaxAmount:    taxAmount,
		Jurisdiction: jurisdiction,
		// A SalesOrder quote with zero tax in a taxable state means Avalara
		// sees no registration/nexus for the company there.
		HasNexus: taxAmount > 0 || rate > 0,
	}, nil
}
