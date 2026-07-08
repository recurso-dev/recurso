package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
	"github.com/swapnull-in/recur-so/internal/core/service/tax"
)

// GSTConfigProvider is the slice of the GST config repository the tax
// resolver needs. *db.GSTConfigRepository satisfies it.
type GSTConfigProvider interface {
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.TenantGSTConfig, error)
}

// InvoiceTax is the resolved tax for a single invoice. For non-GST
// jurisdictions (VAT, US sales tax) only Total is populated; the GST
// component fields stay zero.
type InvoiceTax struct {
	Total   int64
	IGST    int64
	CGST    int64
	SGST    int64
	TaxType string
	Note    string
	// Rate is the effective tax rate as a fraction (e.g. 0.18 for 18%). Used to
	// record per-line tax_rate on itemized invoices and the e-invoice GstRt.
	Rate float64
	// HSN is the HSN/SAC code the tax was resolved against (the tenant SAC for
	// GST). Empty for non-GST jurisdictions; callers default it to the tenant
	// SAC / 998314 when recording a line.
	HSN string
}

// TaxResolver picks the jurisdiction-appropriate tax engine for each invoice.
//
// Seller jurisdiction resolution order:
//  1. The tenant's GST configuration (gstConfigs): a GST registration implies
//     an Indian seller, with the seller state taken from the config.
//  2. Env-level company defaults (COMPANY_COUNTRY / COMPANY_STATE) when the
//     tenant has no config, the lookup fails, or no provider is wired.
//
// Engine selection follows tax.NewTaxEngine: IN -> GST, US -> sales tax,
// EU/GB -> VAT, anything else defaults to GST (India-focused product).
// A missing or broken tenant config never fails invoice generation.
type TaxResolver struct {
	gstConfigs     GSTConfigProvider
	defaultCountry string
	defaultState   string
	salesTax       tax.SalesTaxProvider // optional; nil keeps the US engine a 0% stub
	vatValidator   tax.VATValidator     // optional; nil keeps EU reverse charge presence-based
	logger         *slog.Logger
}

// NewTaxResolver creates a TaxResolver. gstConfigs may be nil (env defaults
// only). Empty defaults fall back to IN/TN, matching the historical behavior
// of the old calculateInvoiceGST helper.
func NewTaxResolver(gstConfigs GSTConfigProvider, defaultCountry, defaultState string) *TaxResolver {
	if strings.TrimSpace(defaultCountry) == "" {
		defaultCountry = "IN"
	}
	if strings.TrimSpace(defaultState) == "" {
		defaultState = "TN"
	}
	return &TaxResolver{
		gstConfigs:     gstConfigs,
		defaultCountry: strings.ToUpper(strings.TrimSpace(defaultCountry)),
		defaultState:   strings.ToUpper(strings.TrimSpace(defaultState)),
		logger:         slog.Default().With("service", "tax_resolver"),
	}
}

// WithSalesTaxProvider wires a live US sales-tax rate provider (TaxJar)
// into the resolver and returns the resolver for chaining. The provider is
// wrapped in a 24h per-(state,zip) rate cache here — engines are built per
// invoice by the factory, so the resolver-held provider is where cached
// rates survive. nil is a no-op (US engine stays the 0% stub).
func (r *TaxResolver) WithSalesTaxProvider(p tax.SalesTaxProvider) *TaxResolver {
	if p != nil {
		r.salesTax = tax.NewCachedSalesTaxProvider(p, tax.DefaultSalesTaxRateTTL)
	}
	return r
}

// WithVATValidator wires a live EU VAT-number validator (VIES) into the
// resolver and returns the resolver for chaining. When set, an intra-EU
// cross-border B2B invoice only gets reverse-charge treatment if the buyer's
// VAT number VALIDATES; an invalid number falls back to charging VAT. A VIES
// outage never fails the invoice — it degrades to the presence-based behaviour
// with an auditable note. nil is a no-op (reverse charge stays presence-based).
func (r *TaxResolver) WithVATValidator(v tax.VATValidator) *TaxResolver {
	r.vatValidator = v
	return r
}

// ResolveInvoiceTax computes the tax for one invoice line/amount (lowest
// currency unit). It never returns an error: tax resolution problems degrade to
// zero tax with a log line rather than blocking invoice generation.
//
// hsn is the per-line HSN/SAC code (Phase 2). For the India GST path a non-empty
// hsn is used as the code the rate is looked up against; an empty hsn falls back
// to the tenant SAC (then the 998314 default) — i.e. exactly the Phase-1
// behaviour. The US and EU engines ignore hsn.
func (r *TaxResolver) ResolveInvoiceTax(ctx context.Context, tenantID uuid.UUID, customer *domain.Customer, currency string, amount int64, hsn string) InvoiceTax {
	sellerCountry, sellerState, cfg := r.sellerJurisdiction(ctx, tenantID)

	engine := tax.NewTaxEngineWithSalesTaxProvider(sellerCountry, normalizeINState(sellerState), r.salesTax)

	switch engine.(type) {
	case *tax.USSalesTaxEngine:
		return r.resolveUSSalesTax(ctx, engine, customer, currency, amount)
	case *tax.EUVATEngine:
		return r.resolveEUVAT(ctx, engine, sellerCountry, customer, currency, amount)
	default:
		// *tax.GSTEngine — both for IN sellers and the factory's default for
		// unsupported seller countries (India-focused product).
		return r.resolveIndiaGST(ctx, engine, cfg, customer, currency, amount, hsn)
	}
}

// sellerJurisdiction resolves the seller's country/state. Tenant GST config
// wins; env defaults are the fallback. Lookup failures are logged (once per
// invoice generation) and never propagate.
func (r *TaxResolver) sellerJurisdiction(ctx context.Context, tenantID uuid.UUID) (country, state string, cfg *domain.TenantGSTConfig) {
	country, state = r.defaultCountry, r.defaultState

	if r.gstConfigs == nil || tenantID == uuid.Nil {
		return country, state, nil
	}

	c, err := r.gstConfigs.GetByTenantID(ctx, tenantID)
	if err != nil {
		r.logger.Warn("tenant GST config lookup failed; using env company defaults",
			"tenant_id", tenantID, "country", country, "state", state, "error", err)
		return country, state, nil
	}
	if c == nil || (c.GSTIN == "" && c.StateCode == "") {
		r.logger.Debug("no tenant GST config; using env company defaults",
			"tenant_id", tenantID, "country", country, "state", state)
		return country, state, nil
	}

	// A GST registration implies an Indian seller.
	country = "IN"
	state = c.StateCode
	if state == "" {
		state = domain.GetStateCodeFromGSTIN(c.GSTIN)
	}
	return country, state, c
}

// resolveIndiaGST applies India GST to INR invoices and zero-rates
// foreign-currency invoices as exports.
func (r *TaxResolver) resolveIndiaGST(ctx context.Context, engine port.TaxEngine, cfg *domain.TenantGSTConfig, customer *domain.Customer, currency string, amount int64, lineHSN string) InvoiceTax {
	// Resolve the HSN/SAC the rate is looked up against. The tenant SAC (then the
	// 998314 default) is the fallback; a non-empty per-line HSN overrides it.
	// An empty lineHSN therefore reproduces Phase-1 behaviour exactly.
	hsn := domain.DefaultSACCode
	if cfg != nil && cfg.SACCode != "" {
		hsn = cfg.SACCode
	}
	if lineHSN != "" {
		hsn = lineHSN
	}

	if !strings.EqualFold(strings.TrimSpace(currency), "INR") {
		// Non-INR invoice from an Indian seller = export of services.
		// With a Letter of Undertaking (LUT) the supply is zero-rated: 0% GST,
		// no upfront IGST. Without LUT on file the compliant alternative is
		// "pay IGST, claim refund" — a back-office workflow, not something to
		// add to the customer-facing invoice — so the invoice still carries
		// zero GST and the note records the missing LUT.
		note := "Zero-rated export (no LUT on file — IGST-with-refund route not applied to invoice)"
		if cfg != nil && cfg.HasLUT {
			note = "Zero-rated export under LUT"
		}
		return InvoiceTax{TaxType: "export", Note: note, HSN: hsn}
	}

	// Buyer state: PlaceOfSupply first, then the billing address state for
	// Indian buyers. Empty resolves to inter-state (IGST) in the engine.
	buyerState := normalizeINState(domain.PtrToString(customer.PlaceOfSupply))
	if buyerState == "" {
		buyerCountry := normalizeCountry(customer.BillingAddress.Country)
		if buyerCountry == "" || buyerCountry == "IN" {
			buyerState = normalizeINState(customer.BillingAddress.State)
		}
	}

	// The tenant's configured gst_rate (a percent) is the fallback the engine
	// applies only when the SAC/HSN code isn't in its rate map; 0 leaves the
	// engine's built-in default in place.
	var fallbackRate float64
	if cfg != nil && cfg.GSTRate > 0 {
		fallbackRate = cfg.GSTRate / 100.0
	}

	calc, err := engine.CalculateTax(ctx, &port.TaxRequest{
		Amount:       amount,
		Currency:     currency,
		HSNCode:      hsn,
		BuyerState:   buyerState,
		BuyerCountry: normalizeCountry(customer.BillingAddress.Country),
		IsBusiness:   isBusinessBuyer(customer),
		FallbackRate: fallbackRate,
	})
	if err != nil || calc == nil {
		r.logger.Warn("GST calculation failed; invoicing without tax", "error", err)
		return InvoiceTax{}
	}
	return InvoiceTax{
		Total:   calc.TotalTax,
		IGST:    calc.IGST,
		CGST:    calc.CGST,
		SGST:    calc.SGST,
		TaxType: calc.TaxType,
		Note:    calc.Note,
		Rate:    calc.TaxRate,
		HSN:     hsn,
	}
}

// resolveUSSalesTax applies US sales tax for US buyers. Sales to buyers
// outside the US carry no US sales tax. With a live provider wired
// (WithSalesTaxProvider) the engine returns real jurisdiction rates and
// marks invoices "sales_tax"; without one it stays the historical 0% stub
// ("sales_tax_stub"). A provider error at invoice time must never fail the
// invoice: it degrades to 0% with TaxType "sales_tax_error" and a warn log,
// so the invoice ships and the gap is auditable.
func (r *TaxResolver) resolveUSSalesTax(ctx context.Context, engine port.TaxEngine, customer *domain.Customer, currency string, amount int64) InvoiceTax {
	buyerCountry := normalizeCountry(customer.BillingAddress.Country)
	if buyerCountry != "" && buyerCountry != "US" {
		return InvoiceTax{TaxType: "export", Note: "Sale outside the US: no US sales tax"}
	}

	calc, err := engine.CalculateTax(ctx, &port.TaxRequest{
		Amount:        amount,
		Currency:      currency,
		BuyerState:    strings.ToUpper(strings.TrimSpace(customer.BillingAddress.State)),
		BuyerZip:      strings.TrimSpace(customer.BillingAddress.Zip),
		BuyerCountry:  buyerCountry,
		SellerCountry: "US",
		IsBusiness:    isBusinessBuyer(customer),
	})
	if err != nil || calc == nil {
		// Only the provider-backed engine can error (the stub never does).
		provider := "unknown"
		if us, ok := engine.(*tax.USSalesTaxEngine); ok && us.ProviderName() != "" {
			provider = us.ProviderName()
		}
		r.logger.Warn("US sales tax provider lookup failed; invoicing at 0%",
			"provider", provider, "error", err)
		return InvoiceTax{
			TaxType: "sales_tax_error",
			Note:    "US sales tax lookup via " + provider + " failed; invoiced at 0% (needs review)",
		}
	}
	return InvoiceTax{Total: calc.TotalTax, TaxType: calc.TaxType, Note: calc.Note, Rate: calc.TaxRate}
}

// resolveEUVAT applies EU/UK VAT. Reverse charge for B2B cross-border and
// zero-rating of exports are decided by the engine. When a VIES validator is
// wired, the buyer's VAT number is checked before an intra-EU cross-border
// B2B invoice gets reverse-charge treatment (see reverseChargeDecision).
func (r *TaxResolver) resolveEUVAT(ctx context.Context, engine port.TaxEngine, sellerCountry string, customer *domain.Customer, currency string, amount int64) InvoiceTax {
	buyerCountry := normalizeCountry(customer.BillingAddress.Country)
	isBusiness := isBusinessBuyer(customer)

	// Gate reverse charge on VIES validation for intra-EU cross-border B2B.
	// The decision only ever narrows a would-be reverse charge to charging
	// VAT (isBusiness->false) or annotates it; it never fails the invoice.
	var extraNote string
	if isBusiness {
		isBusiness, extraNote = r.reverseChargeDecision(ctx, sellerCountry, buyerCountry, customer)
	}

	calc, err := engine.CalculateTax(ctx, &port.TaxRequest{
		Amount:        amount,
		Currency:      currency,
		BuyerCountry:  buyerCountry,
		SellerCountry: sellerCountry,
		IsBusiness:    isBusiness,
	})
	if err != nil || calc == nil {
		r.logger.Warn("VAT calculation failed; invoicing without tax", "error", err)
		return InvoiceTax{}
	}
	return InvoiceTax{Total: calc.TotalTax, TaxType: calc.TaxType, Note: joinNotes(calc.Note, extraNote), Rate: calc.TaxRate}
}

// reverseChargeDecision returns the IsBusiness flag the VAT engine should see
// and an optional note explaining any VIES-driven override. It only acts on
// intra-EU cross-border B2B invoices (the reverse-charge candidates); every
// other case is returned unchanged as a business buyer.
//
// Behaviour when a validator is wired:
//   - VAT number validates      -> reverse charge (business=true), note records VIES confirmation
//   - VAT number invalid/format -> charge VAT     (business=false), note records the failure
//   - VIES unavailable/outage    -> degrade to presence-based reverse charge (business=true) with a note
//
// With no validator wired the buyer is left as a business (historical
// presence-based behaviour), so the B2B/B2C matrix is unchanged when VIES is
// disabled.
func (r *TaxResolver) reverseChargeDecision(ctx context.Context, sellerCountry, buyerCountry string, customer *domain.Customer) (isBusiness bool, note string) {
	// Not an intra-EU cross-border reverse-charge candidate: leave as-is.
	if r.vatValidator == nil ||
		buyerCountry == "" || buyerCountry == sellerCountry ||
		!tax.IsEUVATCountry(buyerCountry) {
		return true, ""
	}

	cc, num := splitEUVATNumber(domain.PtrToString(customer.TaxID), buyerCountry)
	if num == "" {
		// B2B buyer with no VAT number to check: cannot grant reverse charge
		// under validation. Charge VAT.
		return false, "no VAT number supplied; reverse charge not applied (VAT charged)"
	}

	res, err := r.vatValidator.ValidateVAT(ctx, cc, num)
	switch {
	case errors.Is(err, tax.ErrVATUnavailable):
		// Never fail the invoice on a VIES outage: degrade to presence-based.
		r.logger.Warn("VIES unavailable; applying presence-based reverse charge",
			"validator", r.vatValidator.Name(), "country", cc, "error", err)
		return true, "reverse charge applied on VAT-number presence — " + r.vatValidator.Name() + " unavailable, number unverified (needs review)"
	case err != nil:
		// Format/input rejection: definitively not eligible. Charge VAT.
		r.logger.Info("VAT number failed validation; charging VAT",
			"validator", r.vatValidator.Name(), "country", cc, "error", err)
		return false, "buyer VAT number failed validation via " + r.vatValidator.Name() + "; VAT charged"
	case res != nil && res.Valid:
		note = "reverse charge applied; buyer VAT number validated via " + r.vatValidator.Name()
		if res.Name != "" {
			note += " (" + res.Name + ")"
		}
		return true, note
	default:
		// Reached the registry and it says the number is not registered.
		return false, "buyer VAT number not valid per " + r.vatValidator.Name() + "; VAT charged"
	}
}

// splitEUVATNumber normalises a customer VAT identifier and splits off any
// leading ISO country prefix. "DE123456789" -> ("DE", "123456789"); a bare
// "123456789" -> (fallbackCountry, "123456789"). Spaces, dots, and dashes are
// stripped. The VIES "EL" code for Greece is mapped back to ISO "GR". An empty
// or too-short input returns an empty number.
func splitEUVATNumber(raw, fallbackCountry string) (cc, number string) {
	var b strings.Builder
	for _, r := range strings.ToUpper(raw) {
		if r == ' ' || r == '.' || r == '-' || r == '\t' {
			continue
		}
		b.WriteRune(r)
	}
	s := b.String()
	if s == "" {
		return fallbackCountry, ""
	}
	if len(s) >= 3 && isAlpha(s[0]) && isAlpha(s[1]) {
		iso := s[:2]
		if iso == "EL" {
			iso = "GR"
		}
		if tax.IsEUVATCountry(iso) {
			return iso, s[2:]
		}
	}
	return fallbackCountry, s
}

func isAlpha(b byte) bool { return b >= 'A' && b <= 'Z' }

// joinNotes concatenates two tax notes, skipping empties, so an engine note
// and a resolver override note read as one line on the invoice.
func joinNotes(a, b string) string {
	switch {
	case a == "":
		return b
	case b == "":
		return a
	default:
		return a + "; " + b
	}
}

// isBusinessBuyer reports whether the customer is a business (B2B) buyer.
func isBusinessBuyer(c *domain.Customer) bool {
	if c == nil {
		return false
	}
	if strings.EqualFold(c.TaxType, "business") {
		return true
	}
	return strings.TrimSpace(domain.PtrToString(c.TaxID)) != "" ||
		strings.TrimSpace(domain.PtrToString(c.GSTIN)) != ""
}

// gstNumericStateToAlpha maps numeric GST state codes (GSTIN prefix) to the
// standard two-letter state abbreviations used in PlaceOfSupply.
var gstNumericStateToAlpha = map[string]string{
	"01": "JK", "02": "HP", "03": "PB", "04": "CH", "05": "UK",
	"06": "HR", "07": "DL", "08": "RJ", "09": "UP", "10": "BR",
	"11": "SK", "12": "AR", "13": "NL", "14": "MN", "15": "MZ",
	"16": "TR", "17": "ML", "18": "AS", "19": "WB", "20": "JH",
	"21": "OD", "22": "CG", "23": "MP", "24": "GJ", "25": "DD",
	"26": "DN", "27": "MH", "28": "AP", "29": "KA", "30": "GA",
	"31": "LD", "32": "KL", "33": "TN", "34": "PY", "35": "AN",
	"36": "TS", "37": "AP", "38": "LA",
}

// normalizeINState canonicalizes Indian state identifiers to two-letter
// abbreviations so a tenant config storing the numeric GSTIN code ("33") and
// a customer PlaceOfSupply storing the abbreviation ("TN") compare equal.
// Accepts numeric GST codes, two-letter abbreviations, and full state names.
func normalizeINState(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	if alpha, ok := gstNumericStateToAlpha[s]; ok {
		return alpha
	}
	if len(s) == 2 {
		return s
	}
	// Full state name ("Tamil Nadu") -> numeric code -> abbreviation.
	key := strings.ReplaceAll(strings.ToLower(s), " ", "_")
	if num, ok := domain.IndianStateCode[key]; ok {
		if alpha, ok := gstNumericStateToAlpha[num]; ok {
			return alpha
		}
	}
	return s
}

// countryNameToISO2 maps common country names to ISO 3166-1 alpha-2 codes.
// Billing addresses in this codebase store free-form names (e.g. "India").
var countryNameToISO2 = map[string]string{
	"india":                    "IN",
	"united states":            "US",
	"united states of america": "US",
	"usa":                      "US",
	"america":                  "US",
	"united kingdom":           "GB",
	"great britain":            "GB",
	"uk":                       "GB",
	"germany":                  "DE",
	"france":                   "FR",
	"netherlands":              "NL",
	"ireland":                  "IE",
	"spain":                    "ES",
	"italy":                    "IT",
	"belgium":                  "BE",
	"austria":                  "AT",
	"poland":                   "PL",
	"portugal":                 "PT",
	"sweden":                   "SE",
	"denmark":                  "DK",
	"finland":                  "FI",
	"canada":                   "CA",
	"australia":                "AU",
	"singapore":                "SG",
	"united arab emirates":     "AE",
	"uae":                      "AE",
}

// normalizeCountry canonicalizes a billing-address country (name or code) to
// an ISO 3166-1 alpha-2 code. Unknown values are uppercased and passed
// through, which keeps them distinct from IN/US/EU codes.
func normalizeCountry(s string) string {
	t := strings.TrimSpace(s)
	if t == "" {
		return ""
	}
	if len(t) == 2 {
		return strings.ToUpper(t)
	}
	if iso, ok := countryNameToISO2[strings.ToLower(t)]; ok {
		return iso
	}
	return strings.ToUpper(t)
}
