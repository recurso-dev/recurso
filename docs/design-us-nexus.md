# Design: US sales-tax nexus depth

**Status:** Phase 1 proposed · **Scope:** US sales tax (post-Wayfair economic nexus)
**Related:** [[design-per-product-hsn]] (same "engine native, data needs certification" shape)

## Summary

Post-*Wayfair* (2018), a US seller must collect sales tax in a state only where it
has **nexus** — physical presence, or **economic nexus** (crossing a state's sales
$ / transaction-count threshold). Today Recurso outsources this entirely to the
TaxJar provider (`SalesTaxResult.HasNexus`) and is a 0% stub without it. There is
**no Recurso-native nexus config and no economic-nexus tracking**. This adds them,
so Recurso collects tax where — and only where — the seller is obligated, and warns
before a threshold is crossed.

## Current state (grounded in code)

- `internal/core/service/tax/sales_tax.go` — `USSalesTaxEngine`: with a provider,
  returns the provider's rate and honors `res.HasNexus` ("no nexus in buyer state
  per provider — nothing to collect"); without a provider, a 0% stub.
- `internal/service/tax_resolver.go` `resolveUSSalesTax` — sends the buyer
  state/zip to the engine; no seller-side nexus concept.
- No `tenant_tax_nexus` table, no per-state thresholds, no cumulative sales tracking.

## The compliance boundary (read this before Phase 2)

Encoding each US state's **economic-nexus thresholds** is a *data* problem, not just
code: thresholds vary by state (commonly $100k or 200 transactions, but many
differ), change over time, and getting them wrong is a real liability. So the
**engine** is ours to build, but the **threshold dataset** must be treated like the
HSN rate map — seeded, versioned, and **flagged for review by a US sales-tax
professional / a maintained data source (e.g. TaxJar's own nexus API) before it's
relied on for filing.** Same caveat class as the GST CA-review and the IRP sandbox.
Phase 1 below needs NONE of this data — it's pure config + gating.

## Phasing

- **Phase 1 — Nexus config + gating (no threshold data).** The tenant declares the
  states it has nexus in; US sales tax is collected only in those states, else 0%
  with an auditable "no nexus" note. Native, works with or without TaxJar.
- **Phase 2 — Economic-nexus tracking.** A per-state threshold dataset (flagged per
  the boundary above) + cumulative per-(tenant, state, period) taxable-sales and
  transaction counts, updated as invoices post; auto-establish economic nexus when
  a threshold is crossed; a nexus-status endpoint (per state: registered? economic?
  how close to the threshold?).
- **Phase 3 — Alerts + dashboard.** "Approaching threshold in TX" warnings and a
  nexus dashboard.

## Phase 1 — concrete tasks

- [ ] **1.1 — `tenant_tax_nexus` table + domain.** Migration: `tenant_tax_nexus`
      (id, tenant_id FK, state_code CHAR(2), nexus_type TEXT CHECK in
      'physical'|'voluntary'|'economic', established_at, created_at; UNIQUE
      (tenant_id, state_code)). `domain.TaxNexus`.
      **AC:** up+down valid; domain compiles.
- [ ] **1.2 — Repository.** Port + SQL repo: `ListByTenant`, `Upsert`, `Delete`,
      and a fast `HasNexus(ctx, tenantID, state) bool`. Tenant-scoped.
      **AC:** DB-backed test (upsert/list/has/delete).
- [ ] **1.3 — Config API.** `GET /v1/settings/tax/nexus` (list) and
      `PUT /v1/settings/tax/nexus` (set the full list of physical/voluntary nexus
      states), owner/admin. Append to openapi.
      **AC:** round-trips; tenant-isolated.
- [ ] **1.4 — Gate the US tax path.** In `resolveUSSalesTax`, before charging:
      look up whether the seller has nexus in the buyer's state (the new repo).
      No nexus → `InvoiceTax{Total:0, TaxType:"no_nexus", Note:"No sales-tax nexus
      in <ST> — not collected"}`. Nexus → today's behavior (provider rate / stub).
      Keep the provider's `HasNexus` as a secondary gate (both must allow) so a
      live TaxJar can't over-collect beyond declared nexus.
      **AC:** a buyer in a non-nexus state yields 0% + the note; a buyer in a nexus
      state is unchanged. Tests for both.
- [ ] **1.5 — Tests + wire-up.** Resolver tests (nexus vs non-nexus state); repo
      test; handler test. `go test`, gofmt, golangci-lint clean.

**Phase 1 explicitly excludes** threshold data and cumulative tracking (Phase 2) —
so nothing in Phase 1 encodes a state tax rule that could be wrong; it only gates on
what the tenant declares.

## Design decisions / open questions

1. **Interaction with TaxJar (Phase 1).** When a live provider is wired, use
   `native_nexus AND provider_nexus` (AND) so declared nexus is required and the
   provider can still say "registered but this specific address is exempt". Never
   collect where the tenant hasn't declared nexus.
2. **Marketplace facilitator laws** (a marketplace collects on the seller's behalf)
   — out of scope; note it.
3. **Threshold measurement period** (calendar year vs rolling 12 months, prior vs
   current year) varies by state — a Phase-2 data-model concern.

## Risks

- **Money/compliance:** gating changes *what tax is collected*. Phase 1 is safe
  (only reduces collection to declared-nexus states, the conservative direction),
  but tests must prove a nexus-state sale is byte-identical to today.
- **Phase 2 data liability** — see the compliance boundary above.
