package service

import (
	"encoding/xml"
	"fmt"
	"sort"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// EN 16931 UBL 2.1 generation (Track C, increment 1). BuildUBLInvoice maps a
// Recurso invoice + supplier/customer parties to a Peppol BIS Billing 3.0
// UBL 2.1 Invoice document. Amounts are Recurso's int64 minor units rendered as
// EN 16931 decimals (2 fraction digits); the line net amounts sum to the
// invoice subtotal and the tax subtotals sum to the invoice tax, so the
// document's monetary totals reconcile exactly.
//
// Namespaces are declared on the root and prefixes (cbc:/cac:) are emitted as
// literal tag names — the pragmatic Go-encoding/xml approach that yields valid
// prefixed UBL.

const (
	ublCustomizationID = "urn:cen.eu:en16931:2017#compliant#urn:fdc:peppol.eu:2017:poacc:billing:3.0"
	ublProfileID       = "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0"
	ublInvoiceTypeCode = "380" // commercial invoice
	ublVATScheme       = "VAT"
	ublUnitCode        = "C62" // UN/ECE Rec 20: "one" (dimensionless unit)
)

type ublAmount struct {
	Currency string `xml:"currencyID,attr"`
	Value    string `xml:",chardata"`
}

type ublQuantity struct {
	UnitCode string `xml:"unitCode,attr"`
	Value    string `xml:",chardata"`
}

type ublCountry struct {
	Code string `xml:"cbc:IdentificationCode"`
}

type ublAddress struct {
	Street     string     `xml:"cbc:StreetName,omitempty"`
	City       string     `xml:"cbc:CityName,omitempty"`
	PostalZone string     `xml:"cbc:PostalZone,omitempty"`
	Country    ublCountry `xml:"cac:Country"`
}

type ublPartyTaxScheme struct {
	CompanyID string         `xml:"cbc:CompanyID"`
	TaxScheme ublTaxSchemeID `xml:"cac:TaxScheme"`
}

type ublTaxSchemeID struct {
	ID string `xml:"cbc:ID"`
}

type ublLegalEntity struct {
	RegistrationName string `xml:"cbc:RegistrationName"`
}

type ublPartyName struct {
	Name string `xml:"cbc:Name"`
}

type ublParty struct {
	PartyName      ublPartyName       `xml:"cac:PartyName"`
	PostalAddress  ublAddress         `xml:"cac:PostalAddress"`
	PartyTaxScheme *ublPartyTaxScheme `xml:"cac:PartyTaxScheme,omitempty"`
	LegalEntity    ublLegalEntity     `xml:"cac:PartyLegalEntity"`
}

type ublSupplier struct {
	Party ublParty `xml:"cac:Party"`
}

type ublTaxCategory struct {
	ID        string         `xml:"cbc:ID"`
	Percent   string         `xml:"cbc:Percent"`
	TaxScheme ublTaxSchemeID `xml:"cac:TaxScheme"`
}

type ublTaxSubtotal struct {
	TaxableAmount ublAmount      `xml:"cbc:TaxableAmount"`
	TaxAmount     ublAmount      `xml:"cbc:TaxAmount"`
	TaxCategory   ublTaxCategory `xml:"cac:TaxCategory"`
}

type ublTaxTotal struct {
	TaxAmount   ublAmount        `xml:"cbc:TaxAmount"`
	TaxSubtotal []ublTaxSubtotal `xml:"cac:TaxSubtotal"`
}

type ublMonetaryTotal struct {
	LineExtensionAmount ublAmount `xml:"cbc:LineExtensionAmount"`
	TaxExclusiveAmount  ublAmount `xml:"cbc:TaxExclusiveAmount"`
	TaxInclusiveAmount  ublAmount `xml:"cbc:TaxInclusiveAmount"`
	PayableAmount       ublAmount `xml:"cbc:PayableAmount"`
}

type ublItem struct {
	Name                  string         `xml:"cbc:Name"`
	ClassifiedTaxCategory ublTaxCategory `xml:"cac:ClassifiedTaxCategory"`
}

type ublPrice struct {
	PriceAmount ublAmount `xml:"cbc:PriceAmount"`
}

type ublInvoiceLine struct {
	ID                  string      `xml:"cbc:ID"`
	InvoicedQuantity    ublQuantity `xml:"cbc:InvoicedQuantity"`
	LineExtensionAmount ublAmount   `xml:"cbc:LineExtensionAmount"`
	Item                ublItem     `xml:"cac:Item"`
	Price               ublPrice    `xml:"cac:Price"`
}

type ublInvoice struct {
	XMLName xml.Name `xml:"Invoice"`
	Xmlns   string   `xml:"xmlns,attr"`
	Cac     string   `xml:"xmlns:cac,attr"`
	Cbc     string   `xml:"xmlns:cbc,attr"`

	CustomizationID string `xml:"cbc:CustomizationID"`
	ProfileID       string `xml:"cbc:ProfileID"`
	ID              string `xml:"cbc:ID"`
	IssueDate       string `xml:"cbc:IssueDate"`
	DueDate         string `xml:"cbc:DueDate,omitempty"`
	InvoiceTypeCode string `xml:"cbc:InvoiceTypeCode"`
	CurrencyCode    string `xml:"cbc:DocumentCurrencyCode"`

	Supplier    ublSupplier      `xml:"cac:AccountingSupplierParty"`
	Customer    ublSupplier      `xml:"cac:AccountingCustomerParty"`
	TaxTotal    ublTaxTotal      `xml:"cac:TaxTotal"`
	Monetary    ublMonetaryTotal `xml:"cac:LegalMonetaryTotal"`
	InvoiceLine []ublInvoiceLine `xml:"cac:InvoiceLine"`
}

// ublMoney renders int64 minor units as an EN 16931 decimal with 2 fraction
// digits (e.g. 12345 -> "123.45").
func ublMoney(minor int64) string {
	neg := minor < 0
	if neg {
		minor = -minor
	}
	s := fmt.Sprintf("%d.%02d", minor/100, minor%100)
	if neg {
		return "-" + s
	}
	return s
}

// taxCategoryFor picks the EN 16931 tax-category code for a rate. A positive
// rate is standard-rated ("S"); a zero rate with a buyer VAT id is treated as an
// intra-EU reverse charge ("AE"); otherwise zero-rated ("Z").
func taxCategoryFor(ratePercent float64, buyerHasVAT bool) string {
	if ratePercent > 0 {
		return "S"
	}
	if buyerHasVAT {
		return "AE"
	}
	return "Z"
}

func ublParty2(p domain.EUParty) ublParty {
	party := ublParty{
		PartyName:     ublPartyName{Name: p.Name},
		PostalAddress: ublAddress{Street: p.Street, City: p.City, PostalZone: p.PostalZone, Country: ublCountry{Code: p.CountryCode}},
		LegalEntity:   ublLegalEntity{RegistrationName: p.Name},
	}
	if p.VATID != "" {
		party.PartyTaxScheme = &ublPartyTaxScheme{CompanyID: p.VATID, TaxScheme: ublTaxSchemeID{ID: ublVATScheme}}
	}
	return party
}

// BuildUBLInvoice produces the EN 16931 UBL 2.1 XML for an invoice. seller and
// buyer are the resolved parties. It errors if a mandatory field (invoice
// number, currency, seller/buyer name + country) is missing.
func BuildUBLInvoice(inv *domain.Invoice, seller, buyer domain.EUParty) ([]byte, error) {
	if inv == nil {
		return nil, fmt.Errorf("eu e-invoice: nil invoice")
	}
	if inv.InvoiceNumber == "" {
		return nil, fmt.Errorf("eu e-invoice: invoice number is required (BT-1)")
	}
	if len(inv.Currency) != 3 {
		return nil, fmt.Errorf("eu e-invoice: a 3-letter currency is required (BT-5)")
	}
	for label, p := range map[string]domain.EUParty{"seller": seller, "buyer": buyer} {
		if p.Name == "" {
			return nil, fmt.Errorf("eu e-invoice: %s name is required", label)
		}
		if len(p.CountryCode) != 2 {
			return nil, fmt.Errorf("eu e-invoice: %s country code (ISO 3166-1 alpha-2) is required", label)
		}
	}

	cur := inv.Currency
	buyerHasVAT := buyer.VATID != ""

	// Invoice lines + per-rate tax grouping. Line net = InvoiceItem.Amount; the
	// sum equals the invoice subtotal (the itemization reconciles by
	// construction).
	lines := make([]ublInvoiceLine, 0, len(inv.LineItems))
	type grp struct {
		taxable int64
		rate    float64
	}
	byRate := map[float64]*grp{}
	for i, li := range inv.LineItems {
		qty := li.Quantity
		if qty == 0 {
			qty = 1
		}
		cat := taxCategoryFor(li.TaxRate, buyerHasVAT)
		lines = append(lines, ublInvoiceLine{
			ID:                  fmt.Sprintf("%d", i+1),
			InvoicedQuantity:    ublQuantity{UnitCode: ublUnitCode, Value: fmt.Sprintf("%d", qty)},
			LineExtensionAmount: ublAmount{Currency: cur, Value: ublMoney(li.Amount)},
			Item: ublItem{
				Name:                  itemName(li.Description),
				ClassifiedTaxCategory: ublTaxCategory{ID: cat, Percent: ublPercent(li.TaxRate), TaxScheme: ublTaxSchemeID{ID: ublVATScheme}},
			},
			Price: ublPrice{PriceAmount: ublAmount{Currency: cur, Value: ublMoney(li.UnitAmount)}},
		})
		g := byRate[li.TaxRate]
		if g == nil {
			g = &grp{rate: li.TaxRate}
			byRate[li.TaxRate] = g
		}
		g.taxable += li.Amount
	}

	// Tax subtotals per rate, tax = round(taxable * rate/100); reconcile the
	// aggregate to the invoice's tax amount by adjusting the last (highest-rate)
	// subtotal so Σ subtotal tax == BT-110.
	rates := make([]float64, 0, len(byRate))
	for r := range byRate {
		rates = append(rates, r)
	}
	sort.Float64s(rates)
	subtotals := make([]ublTaxSubtotal, 0, len(rates))
	var taxSum int64
	for _, r := range rates {
		g := byRate[r]
		lineTax := int64(float64(g.taxable)*r/100.0 + 0.5)
		taxSum += lineTax
		subtotals = append(subtotals, ublTaxSubtotal{
			TaxableAmount: ublAmount{Currency: cur, Value: ublMoney(g.taxable)},
			TaxAmount:     ublAmount{Currency: cur, Value: ublMoney(lineTax)},
			TaxCategory:   ublTaxCategory{ID: taxCategoryFor(r, buyerHasVAT), Percent: ublPercent(r), TaxScheme: ublTaxSchemeID{ID: ublVATScheme}},
		})
	}
	// Reconcile: fold any rounding delta into the last subtotal's tax so the
	// document's tax total equals the invoice's exactly.
	if n := len(subtotals); n > 0 && taxSum != inv.TaxAmount {
		last := &subtotals[n-1]
		adj := inv.TaxAmount - taxSum
		g := byRate[rates[n-1]]
		last.TaxAmount = ublAmount{Currency: cur, Value: ublMoney(int64(float64(g.taxable)*rates[n-1]/100.0+0.5) + adj)}
	}
	if len(subtotals) == 0 {
		// No lines (defensive): a single zero subtotal keeps the document valid.
		subtotals = append(subtotals, ublTaxSubtotal{
			TaxableAmount: ublAmount{Currency: cur, Value: ublMoney(inv.Subtotal)},
			TaxAmount:     ublAmount{Currency: cur, Value: ublMoney(inv.TaxAmount)},
			TaxCategory:   ublTaxCategory{ID: taxCategoryFor(0, buyerHasVAT), Percent: "0", TaxScheme: ublTaxSchemeID{ID: ublVATScheme}},
		})
	}

	doc := ublInvoice{
		Xmlns:           "urn:oasis:names:specification:ubl:schema:xsd:Invoice-2",
		Cac:             "urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2",
		Cbc:             "urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2",
		CustomizationID: ublCustomizationID,
		ProfileID:       ublProfileID,
		ID:              inv.InvoiceNumber,
		IssueDate:       inv.CreatedAt.UTC().Format("2006-01-02"),
		InvoiceTypeCode: ublInvoiceTypeCode,
		CurrencyCode:    cur,
		Supplier:        ublSupplier{Party: ublParty2(seller)},
		Customer:        ublSupplier{Party: ublParty2(buyer)},
		TaxTotal: ublTaxTotal{
			TaxAmount:   ublAmount{Currency: cur, Value: ublMoney(inv.TaxAmount)},
			TaxSubtotal: subtotals,
		},
		Monetary: ublMonetaryTotal{
			LineExtensionAmount: ublAmount{Currency: cur, Value: ublMoney(inv.Subtotal)},
			TaxExclusiveAmount:  ublAmount{Currency: cur, Value: ublMoney(inv.Subtotal)},
			TaxInclusiveAmount:  ublAmount{Currency: cur, Value: ublMoney(inv.Total)},
			PayableAmount:       ublAmount{Currency: cur, Value: ublMoney(inv.Total)},
		},
		InvoiceLine: lines,
	}
	if !inv.DueDate.IsZero() {
		doc.DueDate = inv.DueDate.UTC().Format("2006-01-02")
	}

	body, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("eu e-invoice: marshal UBL: %w", err)
	}
	return append([]byte(xml.Header), body...), nil
}

func itemName(desc string) string {
	if desc == "" {
		return "Item"
	}
	return desc
}

// ublPercent renders a VAT rate percentage without trailing noise (20 -> "20",
// 8.5 -> "8.5").
func ublPercent(rate float64) string {
	return fmt.Sprintf("%g", rate)
}
