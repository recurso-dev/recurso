# Plan: Lago Parity Program

Source spec: `docs/spec_lago_parity.md` (APPROVED 2026-07-18).
Task list: `tasks/todo.md`. Order: **A → B1 → B2 → B3 → C1 → C2** (D8).

## Components and dependency order

```
A1 billing-cycle scheduler ──┐
A2 mandate metered lines ────┼── independent of each other; A1 first
A3 SDK metering methods ─────┘   (defines renewal semantics B relies on)
        │
B1 wallets ── domain → service+ledger → invoice drain → auto-recharge → API
        │         (drain hooks the invoice flow A1 exercises)
B2 commitments ── true-up line in GenerateInvoice (after B1 so ordering
        │          wallet → credits → gateway is settled)
B3 alerts ── needs nothing from B1/B2 except commitment %-thresholds;
        │     scheduled sweep reuses A1 patterns
C1 ingestion ── batch + transaction_id; independent, any time after A
C2 audit log ── touches every mutation path; LAST so it wraps the final
                surface area once (avoids re-instrumenting)
```

## Why this order

- A1 is the keystone: unattended renewal is what makes every Track B
  feature *automatic* rather than API-triggered. It also surfaces any
  latent issue in v1 rating under real scheduling before we stack wallets
  on top.
- B1 before B2: the payment-application order (D3: wallet → credit notes
  → gateway) must exist before commitment true-up lines change invoice
  totals, or we'd re-test the ordering twice.
- C2 last by design, not laziness: the audit helper instruments mutation
  paths; doing it after B means wallets/commitments/alerts get coverage
  in one pass.

## Risks and mitigations

| Risk | Mitigation |
| --- | --- |
| Scheduler double-bills a period under concurrent ticks | Conditional-UPDATE claim on `current_period_end` (mandate cycle-key pattern) + the existing `usage_ratings` window guard as second layer; concurrency test required |
| Scheduler surprises existing self-hosters on upgrade (D7 on-by-default) | CHANGELOG breaking-change callout + `BILLING_CYCLE_INTERVAL=0` kill switch + skips subs with mandates/gateway-managed cycles |
| Wallet drain breaks invoice/ledger invariants | Drain applies AFTER invoice commit as a payment-like application (mirrors credit notes), never mutates lines; invariant tests extended |
| GST-on-advances exposure (D4) | Shipping default: tax at consumption; explicitly listed in the CA-review packet |
| SDK drift across three repos | All methods generated/written against the same synced openapi.yaml; each SDK's test suite extended in the same PR as the methods |
| Audit log write amplification on hot paths | Audit only config-grade mutations (no per-event/per-invoice writes); single INSERT, no reads on the hot path |

## Verification checkpoints

1. **After A1/A2:** integration-style test proving unattended renewal with
   metered lines + mandate invoice carries usage; full suite green.
2. **After A3:** all three SDK suites green; publish commands documented.
3. **After B1:** wallet end-to-end test (top-up → drain → ledger balanced);
   payment-order test (wallet before credit notes before gateway).
4. **After B2:** true-up boundary tests (exactly-at, below, above commitment).
5. **After B3:** alert dedup test (once per threshold per period).
6. **After C1:** batch idempotency test (500 events, duplicates collapse).
7. **After C2:** every mutating endpoint writes an audit row (checked by a
   sweep test over the router table).

Each checkpoint = commit + push; docs/OpenAPI/CHANGELOG sync per track,
not per task.
