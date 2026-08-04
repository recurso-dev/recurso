# ADR-007: Accounting policy is a first-class engine, separate from the tax and jurisdiction engines

## Status
Accepted

## Date
2026-08-04

## Context
Recurso's revenue-recognition and bad-debt behavior must vary by jurisdiction
(US GAAP recognizes accrual; India GST governs when tax relief on a write-off is
allowed; different countries answer "when is revenue earned?" and "can I reclaim
tax on bad debt?" differently). The first instinct is a country conditional in
the money path (`if country == "IN"`). That approach entangles three genuinely
separate concerns — *accounting* rules (when revenue is recognized, how bad debt
is treated), *tax* rules (rates, place-of-supply, exemptions), and
*jurisdiction* rules (which country's regime applies) — inside the ledger code,
and it does not compose: adding US GAAP vs IFRS, or a per-tenant policy override,
would mean touching the posting engine.

## Decision
Model accounting policy as its own value object and resolver, wired into the
money path as an optional dependency, kept strictly separate from the tax and
jurisdiction engines.

- `internal/service/accounting_policy.go` defines
  `AccountingPolicy{ RevenueRecognition RecognitionMethod; BadDebt
  BadDebtTreatment }`, where `RecognitionMethod` is `cash` (schedule at payment)
  or `accrual` (schedule at issuance), and `BadDebtTreatment` carries
  `AllowTaxRelief`, `RecognitionDelayDays`, and `RecoverableTaxes`.
- `PolicyResolver.For(country)` returns the policy for a jurisdiction;
  `Register(country, policy)` adds jurisdiction adapters over time. The resolver
  is **nil-safe** — a nil resolver or unknown country resolves to the
  conservative `cash` default, so the money path never depends on the engine
  being wired.
- The accounting engine never imports tax or jurisdiction logic. Tax amounts are
  computed by the tax engine; the accounting engine only decides *when* revenue
  is recognized and *how* an uncollectible is treated.

## Alternatives Considered

### Country conditionals inline in the ledger/revrec code
- Pros: nothing new to build
- Cons: three concerns entangled in the posting path; every new regime edits the
  money path; no per-tenant override seam; untestable in isolation
- Rejected: the ledger engine must not know GST rules (founder direction)

### One combined "TaxAndAccountingPolicy"
- Pros: one lookup
- Cons: fuses independent axes — a tenant can be US-accrual while its tax is
  handled by a BYO Avalara connection; collapsing them forces false coupling
- Rejected

### Per-tenant policy column only (no jurisdiction resolver)
- Pros: simplest storage
- Cons: loses the jurisdiction default that most tenants should inherit; every
  tenant must be configured explicitly
- Rejected: keep a jurisdiction default, allow a per-tenant/registered override

## Consequences
- Accounting behavior is selected through one seam; adding US GAAP / IFRS /
  India GST / UK VAT / EU VAT / Australia GST is a `Register` call plus a policy
  value, with no change to the ledger posting engine.
- The default is `cash` everywhere the resolver is absent, so production is
  unchanged until a policy is explicitly wired (see ADR-008 for the accrual
  rollout, ADR-009 for bad debt).
- This ADR sets the boundary; ADR-008 and ADR-009 are the two policy dimensions
  built on top of it. Superseding the *engine* boundary would require a new ADR.
- Proven by `internal/service/accounting_policy_test.go` (default cash, nil-safe
  resolution, per-country registration).
