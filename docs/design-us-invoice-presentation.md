# Design: US invoice presentation (per-seller regime)

**Status:** SHIPPED (US Market Readiness · Inc 1a) · **Scope:** invoice presentation only — no tax math, no ledger, no migration
**Related:** [[design-us-nexus]], [[design-per-product-hsn]] · roadmap "US invoice presentation"

## Summary

A non-Indian seller's invoice must not show GST artifacts. The **PDF** was already
jurisdiction-aware (`BuildInvoiceData` → `ShowGST`), but its seller-country signal
was the **env-global** `PDF_COMPANY_COUNTRY`, and the **dashboard** `InvoiceDetail`
had **no regime gate at all** — it printed `HSN … · …% GST`, the CGST/SGST/IGST
breakdown, TDS, and the IRP e-invoice section for every invoice, US ones included.

This makes presentation follow the **seller's jurisdiction per tenant** — the same
signal tax already uses — and gates the dashboard to match the PDF.

## The regime signal (and why not `TaxType`)

Each invoice's `TaxType` (`gst_intra`/`sales_tax`/`vat`/…) faithfully records the
regime it was taxed under — but `invoice.go` documents that **`TaxType` is not
hydrated on read** (scans leave it empty). So it's unusable on the list/read path.

Instead we reuse `TaxResolver.sellerJurisdiction` (GST config ⇒ IN, else the
configured company default) via a new public `TaxResolver.SellerCountry(ctx,
tenantID)`, mapped to a presentation regime by `RegimeForCountry`:

| Seller country | `tax_regime` | Shows |
|---|---|---|
| IN / empty | `gst` | GSTIN, HSN, CGST/SGST/IGST, IRN/e-invoice |
| US | `sales_tax` | EIN, one "Sales Tax" line |
| EU / GB | `vat` | VAT id, one "VAT" line |
| other | `plain` | one generic "Tax" line |

The regime the invoice is **displayed** under thus matches the regime it was
**taxed** under, by construction.

## What changed

- **`domain.Invoice.TaxRegime`** — a computed, non-persisted (`db:"-"`) field +
  regime constants + `RegimeFallback()` (a data-only heuristic: a GST split or an
  INR invoice ⇒ `gst`, else `plain`) for when no resolver is wired / legacy rows.
- **`TaxResolver.SellerCountry` + `RegimeForCountry`** (service) — expose the
  existing seller-jurisdiction logic for presentation.
- **`InvoicePDFService.BuildInvoiceDataFor(inv, cust, sellerCountry)`** — the
  regime decision now takes an explicit per-invoice seller country; the old
  `BuildInvoiceData` delegates with the env default, so behavior is unchanged
  where the resolver isn't wired.
- **Handlers** (`ListInvoices`, PDF download/portal) gain a **nil-safe**
  `SetSellerResolver` (the repo's optional-dependency idiom). `ListInvoices`
  resolves the seller country **once per request** (all rows share a tenant) and
  stamps `tax_regime`; the PDF handler resolves per invoice (portal path uses
  `inv.TenantID`, which has no request tenant).
- **`main.go`** wires `taxResolver` into both handlers.
- **Dashboard `InvoiceDetail.jsx`** gates on `tax_regime` (falling back to the
  same data heuristic): non-GST hides HSN/GST line labels, CGST/SGST/IGST, TDS,
  and the IRP e-invoice section, and labels the tax line "Sales Tax"/"VAT"/"Tax".
- **OpenAPI** documents `Invoice.tax_regime`.

## Safety

- **No tax math, no ledger postings, no migration** → the invariant harness and
  reconciliation are untouched. Pure presentation.
- Existing GST/US regime PDF tests are unchanged (the common single-tenant paths
  behave identically); new tests cover the per-call country override, the country
  → regime map, and the dashboard gate.

## Known limitation / follow-ups

- **Tenant-level, not entity-level.** `SellerCountry` mirrors
  `sellerJurisdiction`, which is per-**tenant**. A multi-entity tenant with both a
  US and an IN entity gets one regime for all its invoices — the same limitation
  tax resolution has today. Refining both tax and presentation to the invoice's
  issuing `EntityID` (entities already carry `country_code`) is a shared follow-up.
- **"Sales-tax-by-jurisdiction lines"** (state/county breakdown) is not modeled —
  a single "Sales Tax" line is shown. Deeper breakdown depends on the tax result
  carrying per-jurisdiction detail; deferred.
- **US-first onboarding defaults** (country ⇒ USD / Stripe / EIN-W9) is Inc 1b,
  which will also give `SellerCountry` a first-class per-tenant country setting
  rather than leaning on the env default.
