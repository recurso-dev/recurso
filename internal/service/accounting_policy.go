package service

import "strings"

// AccountingPolicy is the jurisdiction-resolved set of accounting-behavior
// decisions the ledger consults — deliberately SEPARATE from the accounting
// engine (which knows debits/credits, not tax law) and the tax engine (which
// knows rates). The accounting engine must not hardcode GST/VAT rules; anything
// jurisdiction-specific lives behind this interface so international expansion
// is adding an adapter, not editing the ledger (founder direction, 2026-08-04;
// see docs/ACCOUNTING_PRINCIPLES.md "Policy-driven accounting").
//
// This is the seam for the accrual epic (#466): the write-off split
// (recognized → Bad Debt Expense vs unrecognized → Deferred) and its tax
// treatment are policy decisions, not universal accounting rules.
type AccountingPolicy struct {
	// RevenueRecognition is when recognition schedules are built. "accrual"
	// builds them at invoice issuance (the #466 target); "cash" (today's
	// behavior) builds them at payment.
	RevenueRecognition RecognitionMethod
	// BadDebt governs how a write-off of a partially-recognized invoice is
	// booked.
	BadDebt BadDebtTreatment
}

// RecognitionMethod enumerates when revenue recognition schedules are created.
type RecognitionMethod string

const (
	RecognitionCash    RecognitionMethod = "cash"    // schedule at payment (current default)
	RecognitionAccrual RecognitionMethod = "accrual" // schedule at issuance (#466 target)
)

// BadDebtTreatment captures the jurisdiction-specific rules for writing off an
// uncollectible invoice whose revenue was (partly) recognized.
type BadDebtTreatment struct {
	// AllowTaxRelief reports whether the output tax on the written-off amount
	// can be reclaimed (reversed out of Tax Payable). Jurisdictions differ: US
	// sales tax has no output-tax-on-revenue to relieve; UK/EU VAT bad-debt
	// relief has conditions and a waiting period; India GST does not generally
	// allow output-tax reversal on bad debts.
	AllowTaxRelief bool
	// RecognitionDelayDays is how long an invoice must be overdue before its
	// recognized revenue may be treated as bad debt for tax purposes (0 = no
	// statutory wait). Advisory metadata for reporting; the ledger split itself
	// does not gate on it.
	RecognitionDelayDays int
	// RecoverableTaxes lists the tax kinds (e.g. "vat") whose relief this
	// jurisdiction permits, when AllowTaxRelief is true.
	RecoverableTaxes []string
}

// PolicyResolver returns the AccountingPolicy for a seller jurisdiction. It is
// the registry the founder's per-jurisdiction adapters (US / IndiaGST / UKVAT /
// EUVAT / AustraliaGST) plug into. Increment 1 ships the US default and the
// resolver seam; the other adapters register as the epic proceeds.
type PolicyResolver struct {
	adapters map[string]AccountingPolicy // keyed by ISO country (upper)
	fallback AccountingPolicy
}

// NewPolicyResolver returns a resolver seeded with the US default and an accrual
// fallback. RevenueRecognition defaults to accrual across jurisdictions — the
// #466 target — while the switch that actually moves schedule creation to
// issuance is gated separately (increment 2), so wiring this resolver changes no
// behavior on its own.
func NewPolicyResolver() *PolicyResolver {
	us := AccountingPolicy{
		RevenueRecognition: RecognitionAccrual,
		// US sales tax is collected as a liability, not output tax on revenue,
		// so there is nothing to relieve on a bad debt.
		BadDebt: BadDebtTreatment{AllowTaxRelief: false},
	}
	return &PolicyResolver{
		adapters: map[string]AccountingPolicy{"US": us},
		fallback: AccountingPolicy{
			RevenueRecognition: RecognitionAccrual,
			BadDebt:            BadDebtTreatment{AllowTaxRelief: false},
		},
	}
}

// Register adds or overrides a jurisdiction's policy (used by the per-country
// adapters as they are built).
func (r *PolicyResolver) Register(country string, p AccountingPolicy) {
	r.adapters[strings.ToUpper(strings.TrimSpace(country))] = p
}

// For returns the policy for a seller country, or the fallback.
func (r *PolicyResolver) For(country string) AccountingPolicy {
	if r == nil {
		return AccountingPolicy{RevenueRecognition: RecognitionCash}
	}
	if p, ok := r.adapters[strings.ToUpper(strings.TrimSpace(country))]; ok {
		return p
	}
	return r.fallback
}
