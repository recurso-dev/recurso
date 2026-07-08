# Design: Per-product HSN codes & itemized invoice tax

**Status:** Proposed · **Author:** engineering · **Scope:** India GST correctness
**Related:** [[cloud-dogfooding-runbook]], the correctness-sweep fixes (MRR, invoice amounts)

## Summary

Today Recurso taxes each invoice at a single, tenant-level GST rate derived from
one SAC code. This is correct for a tenant selling one thing (SaaS at 18%) but
**wrong — and non-compliant — the moment a tenant sells mixed-rate products**
(e.g. SaaS 18% + e-books 5% + consulting). The fix is to attach an HSN/SAC code
to each catalog item, itemize invoices, and tax each line at its own rate.

This doc scopes that work and breaks **Phase 1** into concrete tasks.

## Background: current state (grounded in code)

- **Invoices are single-amount.** `domain.Invoice` has one `subtotal/tax_amount/total`;
  there is no `invoice_items` table.
- **The catalog has no HSN.** Plans, add-on plans, and one-time charges carry no
  SAC/HSN code. The rate comes from the tenant's `tenant_gst_configs.sac_code`
  (via the HSN→rate map in `internal/core/service/tax/gst.go`).
- **Tax is resolved once per invoice.** `TaxResolver.ResolveInvoiceTax(ctx,
  tenantID, customer, currency, amount)` takes a single amount.
- **…but the recurring path already loops lines.** `internal/service/invoice.go`
  (~L115–138) iterates base plan + each add-on, taxes **each line independently**
  (to avoid rounding drift), and accumulates totals — then **discards the per-line
  detail**, persisting only aggregates. All lines use the same tenant SAC.
- **The e-invoice schema already supports per-line HSN.** `internal/adapter/gsp/schema.go`
  builds `ItemList []ItemDtls` with `HsnCd` per item — but because invoices have no
  line items, it **falls back to a synthetic single `998314` line** from the invoice
  totals (L161–176). So a mixed-rate invoice is reported to the government IRP as
  all-18%. **This is the core compliance gap.**

## Goals / non-goals

**Goals**
- Each plan / add-on / one-time charge carries an HSN/SAC code.
- Invoices are itemized (persisted line items), each line taxed at its own rate.
- GST breakdown (CGST/SGST/IGST) aggregates from real lines.
- The e-invoice `ItemList` and the PDF reflect real per-line HSN + rate.

**Non-goals (for this work)**
- Changing the HSN→rate map itself (a separate, ongoing data concern).
- Per-line coupons UI (invoice-level discount distribution is in-scope as a rule).
- US/EU line-level tax (those engines already take a single amount; itemization
  can extend to them later — GST is the driver).

## Design (layers)

1. **Catalog HSN** — `hsn_code` on `plans`, add-on plans, and one-time charges.
   Defaults to the tenant SAC (then `998314`) when unset → backward-compatible.
2. **Invoice line items** — a new `invoice_items` table + domain + repo. Invoices
   go from single-amount to itemized.
3. **Per-line tax** — extend the resolver to accept an HSN per line
   (`ResolveInvoiceTax(..., hsn)`), threading to `TaxRequest.HSNCode`. Generalize
   the existing recurring loop and **persist** each line with its rate/tax.
4. **Generation rework** — the ~6 invoice-construction sites build & persist line
   items instead of a bare total, then aggregate: `subscription.go` initial/proration
   (×2), `invoice.go` recurring/advance (×2), `gift.go`, `mandate.go`, `quote.go` convert.
5. **Outputs** — e-invoice `ItemList` reads real lines (drop the synthetic fallback
   for itemized invoices); PDF shows per-line HSN/rate; the invoice API returns
   `line_items` (parity with quotes, which already do).

## Phasing

- **Phase 1 — Itemize (no rate change).** Add `invoice_items`, persist lines from
  the *existing* loop (still at the tenant SAC), return them via the API, and wire
  the real lines into the e-invoice + PDF. No rate behavior changes; this is pure
  structure. Fixes the synthetic-single-line for the common case and de-risks the rest.
- **Phase 2 — Catalog HSN + per-line rates.** Add `hsn_code` to the catalog, thread
  it per line, rates vary correctly. Delivers the multi-rate compliance.
- **Phase 3 — Polish.** PDF per-line detail, old-invoice backfill decision, discount
  distribution edge cases, IRP sandbox validation.

## Phase 1 — concrete tasks

Each task lists acceptance criteria (AC). Money-path work → every task ships with tests.

- [ ] **1.1 — `invoice_items` table + domain.** Migration `invoice_items`
      (id, invoice_id FK CASCADE, description, hsn_code, quantity, unit_amount,
      amount, tax_rate, cgst_amount, sgst_amount, igst_amount, taxable_amount,
      created_at). `domain.InvoiceItem` + `[]InvoiceItem` on `domain.Invoice`
      (json `line_items`, omitempty).
      **AC:** up+down migration valid; domain compiles; no behavior change yet.
- [ ] **1.2 — `InvoiceItemRepository`.** Port + SQL repo: bulk `Create([]*InvoiceItem)`
      in the invoice's tx, `ListByInvoiceID`. Tenant-scoped via the parent invoice.
      **AC:** DB-backed test (create N items, read back, cascade-delete with invoice).
- [ ] **1.3 — Line accumulator in generation.** Introduce a small in-memory
      `lineBuilder` the gen paths append to (description, hsn, qty, unit, amount,
      and the per-line tax already computed). Refactor `invoice.go`'s existing
      base+add-on loop to record lines instead of only summing.
      **AC:** recurring invoice for base + 2 add-ons yields 3 line rows whose
      amounts+tax sum exactly to the invoice `subtotal`/`tax_amount`/`total`.
- [ ] **1.4 — Persist lines on invoice create.** In each of the ~6 gen sites, write
      the accumulated items in the same tx as the invoice. Single-amount paths
      (gift/mandate/quote) emit one line. HSN = tenant SAC (Phase-1 default).
      **AC:** every newly-created invoice has ≥1 line item; totals reconcile;
      existing invoice tests stay green.
- [ ] **1.5 — Return line items in the API.** Invoice GET/list hydrate `line_items`
      (like quotes). New reads via the repo; keep it lazy/optional if list perf matters.
      **AC:** `GET /v1/invoices` and the single-invoice response include `line_items`;
      the dashboard InvoiceDetail can render them (follow-up UI, not blocking).
- [ ] **1.6 — Wire real lines into the e-invoice.** In `gsp/schema.go`, build
      `ItemList` from the invoice's line items when present; keep the synthetic
      single-line fallback ONLY for legacy invoices with no items.
      **AC:** an itemized invoice produces an `ItemList` with one `ItemDtls` per
      line (real `HsnCd`), and per-item tax sums to the header tax.
- [ ] **1.7 — Regression + reconciliation tests.** A test asserting, across
      subscription/proration/gift/mandate paths, the invariant
      `Σ line.amount == subtotal` and `Σ line.tax == tax_amount` (no rounding drift),
      plus the e-invoice item-sum invariant.
      **AC:** all pass; `go test ./internal/...`, gofmt, golangci-lint clean.

**Phase 1 explicitly excludes:** per-item HSN on the catalog and varying rates —
every line still uses the tenant SAC, so no invoice's *numbers* change. Only the
*structure* (persisted lines + accurate e-invoice ItemList) does.

## Design decisions / open questions

1. **Discount distribution (Phase 2/3).** An invoice-level coupon must distribute
   across lines to compute each line's `taxable_amount`. Proposed: pro-rata by line
   amount, largest-remainder rounding so the parts sum to the discount exactly.
2. **Rounding.** Keep the existing per-line tax rounding (already avoids aggregate
   drift). CGST = SGST = round(lineTax/2) with the odd paisa to CGST (current rule).
3. **Backward compatibility.** Old invoices have no items → the e-invoice/PDF
   synthetic-single-line fallback stays for them; no backfill required for Phase 1.
   Phase 3 can optionally backfill a one-line item per legacy invoice.
4. **Catalog default (Phase 2).** When a plan has no `hsn_code`, fall back to the
   tenant SAC, then `998314`. Never emit an empty HSN to the IRP.

## Risks

- **Money-path blast radius.** Reworks ~6 invoice-generation sites — the same code
  the correctness sweep found fragile (`amount_due=0`, dropped tax fields). Mitigate
  with the reconciliation invariant tests (1.7) on every path.
- **IRP certification.** The e-invoice `ItemList` becomes accurate, but validating
  the payload shape needs the **IRP sandbox** — same "needs a live external system
  to certify" caveat as SAML SSO. Unit-test the mapping; certify against the sandbox
  before claiming compliance.
- **List performance.** Hydrating line items on invoice *list* endpoints could add
  N queries — batch-load by invoice IDs, or omit items from list responses (detail only).

## Rough size

Multi-session feature. Phase 1 alone ≈ one focused session (new table + repo +
refactor the gen loop + persist across 6 sites + e-invoice wiring + tests). Phases
2–3 are comparable. Not launch-blocking for single-product SaaS tenants; becomes
important for mixed-rate catalogs or e-invoicing across multiple products.
