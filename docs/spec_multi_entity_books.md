# Spec: Multi-Entity Books

## Objective

Let a single Recurso tenant operate **multiple legal entities** — e.g. "ACME Inc
(US)", "ACME Ltd (UK)", "ACME India Pvt Ltd" — each with its own isolated books,
tax registration, and invoice series, plus a **consolidated view** across all of
them. This is the "your finance team is ready for this" package: it pairs with the
RBAC/SoD + audit trail already shipped, and is the biggest upmarket unlock left.

**Users:** groups that bill under more than one legal entity (regional subsidiaries,
holding structures) and must keep statutory books + invoicing per entity while
seeing the group consolidated.

**Success looks like:**
1. An invoice, its ledger postings, its invoice number, and its seller/tax identity
   all resolve to the **issuing entity**.
2. A **consolidated trial balance / P&L** sums correctly across a tenant's entity
   ledgers.
3. **Every existing single-entity tenant is unchanged** — its data becomes a single
   "primary entity" with no behavioral difference.
4. The ledger invariant harness + zero-discrepancy reconciliation stay green, now
   entity-aware.

## Decisions (locked 2026-07-22 — see the decision-matrix artifact)

- **D1 — Tenancy:** an entity is a new **`entity_id` dimension under the tenant**
  (not its own tenant). One login, shared customers/catalog, per-entity books.
- **D2 — Ledger:** **a separate TigerBeetle ledger per entity** (`LedgerID = entity`).
  Structural isolation; consolidation sums across ledgers. Uses the `LedgerID` field
  that already exists (currently always `1`).
- **D3 — Intercompany:** **deferred to v2.** v1 = isolated per-entity books +
  consolidated view; no intercompany transfers / elimination.
- **D4 — Identity:** **full per-entity tax identity + a gapless per-entity invoice
  series.** Each entity issues under its own GSTIN/VAT/legal name and its own number
  series. (Also closes a latent gap: today's `INV-{unixnano}-{uuid}` is not sequential.)

## Grounding (current architecture, verified on `main@dc40f9b`)

- Everything is scoped by a single `tenant_id`. `organizations` groups tenants
  *above* the tenant line — an entity sits *below* it, a new concept.
- Ledger (ADR-002, TigerBeetle + Postgres): one chart of accounts per tenant via
  `defaultAccounts(tenantID)` (`internal/core/domain/ledger.go`), all on `LedgerID: 1`;
  AR is a per-customer sub-ledger. Codes: Cash 1000, AR 1100, TDS 1200, Deferred 2100,
  Tax Payable 2200, Customer Credit 2300, Revenue 4000, Recognized 4100, Refunds 5000,
  Credits 5100.
- Invoice numbers: `fmt.Sprintf("INV-%d-%s", now.UnixNano(), invID[:8])`
  (`internal/service/invoice.go`) — **not a sequence**.
- Per-tenant identity to make per-entity: GST config (GSTIN), `TenantEUConfig`
  (legal_name/vat_number/country), IRP config, US tax nexus.
- Next free migration number: **000128**.

## Model & schema (proposed)

```
entities
  id                   uuid pk
  tenant_id            uuid fk -> tenants
  name                 text                 -- display, e.g. "ACME India"
  legal_name           text
  is_primary           bool                 -- exactly one per tenant; the backfill target
  tb_ledger_id         int  unique-per-tenant -- TigerBeetle LedgerID (primary keeps 1)
  invoice_prefix       text                 -- e.g. "ACME-IN"
  country_code         text
  created_at, updated_at

entity_invoice_sequences            -- gapless, concurrency-safe per-entity counter
  entity_id            uuid pk fk -> entities
  next_number          bigint not null default 1

-- entity_id added (nullable -> backfill to primary -> NOT NULL) on:
invoices, subscriptions, credit_notes, quotes, ledger_accounts
-- per-entity identity: entity_id added to tenant_eu_config, gst_config,
-- irp_config, tax_nexus (or moved to be entity-scoped)
```

- **Per-entity ledger:** `defaultAccounts` is created **per entity** on that entity's
  `tb_ledger_id`. The primary entity reuses `LedgerID: 1`, so all existing ledger
  data stays valid untouched. AR remains a per-customer sub-ledger, now within an
  entity's ledger.
- **Per-entity invoice number:** `{entity.invoice_prefix}-{seq:06d}` (e.g.
  `ACME-IN-000042`), allocated from `entity_invoice_sequences` at **finalization**
  under a row lock (`UPDATE … SET next_number = next_number + 1 … RETURNING`), so the
  series is gapless and concurrency-safe.

## Migration & backward-compatibility

Migrations from **000128**, applied so single-entity tenants never change behavior:

1. `entities` + `entity_invoice_sequences`; **backfill one `is_primary` entity per
   existing tenant** (`tb_ledger_id = 1`, identity copied from the tenant's existing
   GST/EU/IRP config, `invoice_prefix` defaulted).
2. Add **nullable** `entity_id` to invoices/subscriptions/credit_notes/quotes/
   ledger_accounts; backfill every existing row to its tenant's primary entity; then
   set `NOT NULL`.
3. Move identity configs to per-entity (add `entity_id`, backfill to primary).
4. Wire the invoice-number sequence (existing invoices keep their historical numbers;
   only new invoices use the per-entity series).

## Invariant strategy (the non-negotiable gate)

The property-based ledger harness + zero-discrepancy reconciliation (ADR-002) become
**entity-aware**: reconciliation runs **per entity ledger**, and the invariant
"Σ debits == Σ credits" must hold **per entity AND in aggregate**. Harness seeds gain
an entity dimension (multiple entities per tenant, postings confined to each entity's
`LedgerID`). No increment that touches the ledger merges until this is green. Money-path
increments are **not self-merged**.

## Consolidation

A consolidated **trial balance / P&L** endpoint that aggregates account balances across
a tenant's entity ledgers, grouped by account code (v1 = plain summation; intercompany
elimination is D3/v2). Plus a per-entity filter on existing finance reports.

## Increments (each a reviewable PR; ledger/money-path never self-merged)

1. **Entities foundation** — `entities` table + primary-entity backfill + `entity_id`
   columns (nullable→backfill→NOT NULL) + entity CRUD API + dashboard. Everything still
   posts to the primary entity's `LedgerID: 1`. **Zero behavior change** — pure
   structure. Ships first, safest.
2. **Per-entity ledger** — allocate `tb_ledger_id` per entity, per-entity chart of
   accounts, postings scoped to the entity's ledger, reconciliation + invariant harness
   entity-aware. Money-path; invariant-gated.
3. **Per-entity identity + invoice series** — identity configs per entity, gapless
   per-entity sequence, invoice generation resolves entity → prefix + number + tax
   identity + e-invoicing (IRN/Peppol) identity.
4. **Consolidation** — consolidated trial balance / P&L across entities + a per-entity
   filter on finance reports + dashboard.

## Testing

- Unit: entity CRUD, primary-entity invariants (exactly one, can't delete the last),
  gapless-sequence concurrency (parallel finalizations produce a contiguous series, no
  gaps/dupes), entity resolution on invoice/subscription create.
- Postgres: per-entity ledger isolation (a posting in entity A never lands in entity B),
  reconciliation per entity, consolidated sum == Σ per-entity.
- **Invariant harness (entity-aware)** — the gate for increments 2–4.
- `go build ./... && go test ./...` green (Dockerfile builds gate on the suite).

## Boundaries

- **Always:** back-compat via a per-tenant primary entity; keep the invariant harness
  green; ledger changes on money-path PRs are not self-merged; money in minor units.
- **Ask first:** anything that changes the tenancy model itself; making customers or
  catalog per-entity (see open questions); changing historical invoice numbers.
- **Never:** cross-entity ledger postings in v1 (that's intercompany/D3); break existing
  single-entity behavior; renumber issued invoices.

## Success criteria

1. A new tenant with two entities issues invoices under each entity's own number series
   and tax identity, posting to separate ledgers.
2. Consolidated trial balance across the two entities balances and equals the sum of
   the per-entity trial balances.
3. An existing single-entity tenant sees no change (one primary entity; same numbers).
4. Concurrent invoice finalization yields a gapless, unique per-entity series.
5. Invariant harness green per-entity and in aggregate.

## Open questions (need founder decision — see chat)

- **Q1 — Shared vs per-entity resources.** Proposed: **customers shared** (a customer is
  a party; the entity is the *seller*), **catalog (plans/prices) shared**, but each
  **subscription and invoice belongs to exactly one entity**. Confirm — this is the
  biggest shaping choice.
- **Q2 — How does a subscription/invoice choose its entity?** Proposed: explicit
  `entity_id` at subscription creation (defaults to the primary entity); invoices inherit
  the subscription's entity; one-off invoices take an explicit entity.
- **Q3 — Gapless strictness + void policy.** Gapless per-entity series allocated at
  finalization; define what a void/cancel does to the series (leave a documented gap vs
  reuse). Some jurisdictions require strict gapless — confirm the target.
- **Q4 — e-invoicing per entity.** Confirm D4 extends to India IRN + EU Peppol resolving
  each entity's own GSTIN/VAT registration (it should).

## Increment plan & tasks

See `tasks/multi-entity-plan.md` and `tasks/multi-entity-todo.md` (namespaced —
do not touch `tasks/plan.md`/`todo.md`).
