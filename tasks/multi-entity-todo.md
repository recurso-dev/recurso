# Tasks: Multi-Entity Books

Plan: `tasks/multi-entity-plan.md`. Spec: `docs/spec_multi_entity_books.md`. One PR per increment.
**Blocked on founder decisions Q1–Q4 (see spec) before Inc 1 schema is finalized.**

## Inc 1 — entities foundation
- [ ] migration 000128: `entities` + `entity_invoice_sequences`; backfill one primary entity per tenant (tb_ledger_id=1, identity from existing configs).
- [ ] migration: nullable `entity_id` on invoices/subscriptions/credit_notes/quotes/ledger_accounts → backfill to primary → NOT NULL. (Customers/catalog only if Q1 says per-entity.)
- [ ] domain.Entity + EntityRepository + EntityService (nil-safe wiring).
- [ ] `/v1/entities` CRUD (+ openapi + drift) with "exactly one primary / can't delete last" guards.
- [ ] Dashboard: Entities settings page (list/create/edit).
- [ ] Verify: existing single-entity tenants byte-identical; 2nd entity creatable. `go test ./...` green.
- [ ] PR: "feat(entities): multi-entity foundation — Inc 1".

## Inc 2 — per-entity ledger
- [ ] Allocate tb_ledger_id per entity; per-entity `defaultAccounts`; postings resolve entity LedgerID.
- [ ] Reconciliation per entity; **invariant harness entity-aware** (seeds N entities; Σ=0 per entity + aggregate).
- [ ] Tests: per-entity isolation (posting in A never in B); reconciliation per entity.
- [ ] Gate: harness green per-entity + aggregate. Money-path — NOT self-merged. PR: Inc 2.

## Inc 3 — per-entity identity + invoice series
- [ ] entity_id on gst_config / tenant_eu_config / irp_config / tax_nexus (+ backfill).
- [ ] Gapless `entity_invoice_sequences` counter (atomic UPDATE…RETURNING); number = `{prefix}-{seq:06d}` at finalization.
- [ ] Invoice generation + IRN/Peppol resolve issuing entity identity.
- [ ] Tests: concurrent finalization → gapless/unique; per-entity identity on invoices. PR: Inc 3.

## Inc 4 — consolidation
- [ ] Consolidated trial balance / P&L across entity ledgers (plain sum).
- [ ] Per-entity filter on finance reports + dashboard consolidated view.
- [ ] Test: consolidated == Σ per-entity on a 2-entity tenant. PR: Inc 4.
