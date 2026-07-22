# Plan: Multi-Entity Books

Spec: `docs/spec_multi_entity_books.md` (decisions D1–D4 locked). Tasks:
`tasks/multi-entity-todo.md`. **Namespaced `multi-entity-*` — do NOT touch
`tasks/plan.md`/`todo.md` (Demo Mode).**

## Dependency graph & order

```
Inc 1  entities table + entity_id backfill        (structure only, LedgerID still 1)
   │
Inc 2  per-entity ledger (LedgerID per entity)     (money-path, invariant-gated)
   │
Inc 3  per-entity identity + gapless invoice series
   │
Inc 4  consolidated trial balance / P&L + reports
```

Inc 1 is a hard prerequisite for 2–4 (the `entity_id` dimension must exist first).
Within Inc 1 the migration is sequenced nullable → backfill → NOT NULL so it's
online-safe. Inc 3 and Inc 4 are independent of each other once Inc 2 lands.

## Increments (migrations from 000128; ledger/money-path PRs not self-merged)

### Inc 1 — entities foundation (zero behavior change)
- migration: `entities`, `entity_invoice_sequences`; backfill one `is_primary`
  entity per tenant (tb_ledger_id=1, identity copied from existing configs).
- migration: nullable `entity_id` on invoices/subscriptions/credit_notes/quotes/
  ledger_accounts → backfill to primary → NOT NULL.
- domain.Entity + repo + service (nil-safe wiring idiom); entity CRUD endpoints
  (`/v1/entities*`) + openapi; dashboard entities page.
- Everything still posts to LedgerID:1. **Acceptance:** existing tenants unchanged;
  a tenant can create a 2nd entity (books not yet split).

### Inc 2 — per-entity ledger (invariant-gated)
- Allocate `tb_ledger_id` per entity; `defaultAccounts` created per entity on its
  ledger; postings resolve the entity's LedgerID.
- Reconciliation runs per entity; **invariant harness becomes entity-aware** (seeds
  create N entities, postings confined per ledger; Σ=0 per entity + aggregate).
- **Gate:** harness green per-entity + aggregate before merge. Not self-merged.

### Inc 3 — per-entity identity + gapless invoice series
- Move GST/EU/IRP/nexus configs to per-entity (entity_id + backfill).
- `entity_invoice_sequences` gapless counter (atomic `UPDATE … RETURNING` under row
  lock); invoice number = `{prefix}-{seq:06d}`, allocated at finalization.
- Invoice generation + e-invoicing (IRN/Peppol) resolve the issuing entity's identity.
- Tests: concurrency (no gaps/dupes), per-entity identity on generated invoices.

### Inc 4 — consolidation
- Consolidated trial balance / P&L across entity ledgers (plain sum, no elimination).
- Per-entity filter on existing finance reports + dashboard consolidated view.

## Risks & mitigations

- **Ledger reshape breaks invariants** → Inc 2 is gated on the entity-aware harness;
  primary entity keeps LedgerID:1 so existing data is untouched; money-path not self-merged.
- **Big `entity_id` migration** → online-safe sequence (nullable→backfill→NOT NULL);
  every existing row maps to the primary entity.
- **Gapless sequence under concurrency** → atomic DB counter, allocated at finalization
  (not draft), covered by a concurrency test.
- **Shared-vs-per-entity ambiguity (Q1)** → resolve before Inc 1 schema is finalized;
  it decides whether customers/catalog get an entity_id.
- **iCloud mid-merge reverts / stacked-PR base deletion** → verify origin/main via
  git ls-tree after merges (prior lessons).

## Verification checkpoints

- After Inc 1: existing single-entity behavior byte-identical; entity CRUD works.
- After Inc 2: invariant harness green per-entity + aggregate; per-entity isolation test.
- After Inc 3: gapless-series concurrency test; per-entity identity on invoices.
- After Inc 4: consolidated == Σ per-entity on a real 2-entity tenant.
