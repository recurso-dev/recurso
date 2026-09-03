# Bugs Found

Issues surfaced during the test-engineering run. Fixed bugs link to their PR.

## Fixed (during this session's feature work, pre-test-run)
- **Subscription cancel was broken (400).** The cancel endpoint requires a
  `reason` (`binding:"required"`), but the dashboard posted an empty body, so
  every cancel failed validation. Fixed in PR #290 (cancel-with-reason dialog).
  Severity: high (core lifecycle action unusable). Verified: dashboard now sends
  the reason; frontend test asserts the payload.

## Fixed (during this test-engineering run)
- **HIGH: `golang.org/x/text` v0.38.0 — CVE-2026-56852** (`norm.Iter` can enter
  an infinite loop on crafted input). Surfaced by the **Security Scan (Trivy)**
  CI gate, which began failing on every open PR the moment the CVE hit the vuln
  DB — blocking *all* merges. Fixed by bumping the indirect dependency to
  **v0.39.0** (PR #300). `go build ./...` green; Security Scan clears. Severity:
  high (DoS via infinite loop on attacker-influenced text normalization).
  Verification: the same Trivy gate now passes.

- **LOW: Register form dropped fields under batched changes.** `handleChange`
  used a non-functional `setFormData({ ...formData, [name]: value })` reading a
  stale `formData` closure. Sequential human typing was fine, but a batched
  multi-field change (browser **autofill** / password managers filling several
  fields at once) could clobber all but the last field. Fixed to a functional
  update `setFormData(prev => ({ ...prev, [name]: value }))` (batch 10). Severity:
  low (autofill UX). Verification: `Register.test.jsx` now fills all fields and
  asserts the full payload reaches `registerAccount`. (Login was unaffected — it
  uses separate `useState` per field.)

- **Flaky test: AskAnalytics history-persistence.** `AskAnalytics.test.jsx`
  read `localStorage` synchronously right after the render assertion, but the
  write happens in a `useEffect` keyed on the history state — which can flush
  *after* the render. Passed locally, intermittently failed the CI **Frontend**
  job ("expected [] to have length 1"). Fixed by polling the localStorage read
  inside `waitFor`. Verified: 5/5 green locally after the fix. Class: test flake
  (not an app bug — the app persists correctly; the test raced the effect).

## Open / under investigation
_(none — every failing check discovered so far has been triaged and resolved:
the cancel-reason app bug (#290) and the x/text CVE (#300). Test failures during
authoring were all incorrect-assumption/selector issues in the new tests, fixed
before commit.)_

## Triage protocol
For each failing test discovered:
1. Reproduce in isolation.
2. Classify: application bug / flaky test / incorrect assumption.
3. If app bug and safe to fix → fix + regression test + PR. Else document here
   with severity, root cause, repro, and fix status.

---

# Validation & hardening log (BUG-001…008, merged from bugs-found.md)

### BUG-007 — Icon-only buttons without an accessible name (LOW / a11y)
- **Repro**: a screen reader on the Developers page (delete-endpoint,
  refresh-events, refresh-deliveries) and the Quotes row menu announces an
  unlabeled button — the `title` attribute is not reliably read.
- **Root cause**: four icon-only buttons carried only `title`, no `aria-label`.
- **Fix**: `aria-label` added to each (matching the title).
- **Verification**: frontend lint/build/vitest green; grep confirms every
  `size="icon"` and row-action icon button now has an accessible name.

### BUG-008 — Charts stack (968 kB) loaded on EVERY page, not just analytics (MED / perf)
- **Repro**: build the frontend; `dist/index.html` `<link rel=modulepreload>`s
  `charts-*.js` and the entry statically imports React from it — so every page
  (login, checkout, customers) downloads the 968 kB charting bundle.
- **Root cause**: Vite 8 bundles with **rolldown**, whose Rollup-compat
  `manualChunks` shim mis-places shared vendor code: React, clsx, tailwind-
  merge, @floating-ui, @headlessui, and prop-types all landed INSIDE the
  `charts` chunk. Since `cn()` (every component) needs clsx, and Radix/Stripe
  need @floating-ui/prop-types, and everything needs React, every page
  statically imported the charts chunk. `manualChunks` edits had no effect
  (rolldown ignores it for shared-vendor placement).
- **Fix (this PR)**: replaced `manualChunks` with rolldown's native
  `advancedChunks.groups`, ordered so React and the shared UI vendor libs are
  claimed BEFORE `charts`, plus a catch-all `vendor` group. Verified via
  unminified builds tracing each leaked symbol.
- **Result**: charts is no longer modulepreloaded; **0 non-analytics chunks
  import it** (full-sweep verified); it shrank 968 kB → 617 kB (vendor libs
  de-duplicated into shared chunks). The 617 kB charting stack now loads only
  on analytics routes.
- **Verification**: `grep charts- dist/index.html` shows no modulepreload;
  frontend lint/build/vitest green.
- **Credit**: performance investigation.

## 2026-07-28 — money-path correctness audit

### BUG-005 — Wallet auto-recharge double-credits under concurrent sweeps (HIGH)
- **Repro**: Redis-less deploy (supported), ≥2 API instances. Wallet balance
  20, threshold 100, recharge 500, saved card. Two auto-recharge ticks fire
  together; both read balance 20 → both charge the card with the SAME Stripe
  idempotency key (`wallet-recharge-<id>-20`) → **one** charge; but both call
  `TopUp` → balance 20→520→1020, two ledger legs. Customer paid $500, got
  $1000 of wallet credit; Cash + Customer-Credit each overstated $500.
- **Root cause**: `ListDueForRecharge` (wallet_repository.go) is a plain
  SELECT with no atomic claim — the one due-row worker lacking the ADR-003
  claim every sibling has — and `TopUp` was not idempotent. The gateway
  idempotency key deduped the CHARGE but nothing deduped the CREDIT.
- **Fix (this PR)**: `wallet_transactions.idempotency_key` + partial unique
  index (migration 000151); `TopUp` inserts `ON CONFLICT DO NOTHING` and
  returns `applied bool`; auto-recharge passes the same key it uses for the
  charge and skips the balance bump + ledger post on a duplicate. The wallet
  `FOR UPDATE` lock serializes the two sweeps; the unique key rejects the
  second credit.
- **Verification**: `TestWalletAutoRecharge_ConcurrentSweepCreditsOnce` — two
  sweeps at the same balance produce exactly one top-up and one ledger leg.
- **Credit**: found by the money-path correctness audit; three seeded leads
  (wallet drainer, credit applier, rev-rec) were checked and cleared.

## 2026-07-28 — full-product verification sweep

### BUG-001 — EU e-invoice endpoints 500 on every call (HIGH)
- **Repro**: `GET /v1/invoices/{id}/eu-einvoice` (or `/retry`) with a valid
  session → `500 {"code":"internal_error"}`. Server log: `tenant_id missing
  from context`.
- **Root cause**: `ownedInvoice` in `internal/adapter/handler/eu_einvoice.go`
  read the tenant-scoped invoice repository with the bare request context,
  never injecting `domain.TenantIDKey`. The repo fails closed on a missing
  tenant (tenant-context bug class), so every call errored.
- **Fix**: inject the authenticated tenant into the context before the repo
  read (PR #255).
- **Verification**: `TestOwnedInvoiceInjectsTenantContext` — a fake repo that
  errors without the tenant in context now returns 200.

### BUG-002 — INR mandate guard failures return 500 instead of 400 (MED)
- **Repro**: `POST /v1/mandates {currency:"INR", ...}` for a customer with no
  phone/VPA → `500 internal error` (generic).
- **Root cause**: the phone/VPA guards returned plain errors; the handler
  routed every error through `respondInternalError`, so a caller-input problem
  read as a server fault.
- **Fix**: `ErrVPARequired` sentinel added; the handler classifies phone/VPA
  guard errors as `400` with the guard's message (PR #255).
- **Verification**: `TestCreateMandateGuardFailuresAre400`.

### BUG-003 — Quotes page rendered raw customer UUIDs (LOW / UX)
- **Repro**: Quotes list → Customer column showed `31d00ffe…`.
- **Root cause**: cell rendered `customer_id.substring(0,8)` instead of the
  shared `CustomerName` component (violates the no-raw-UUIDs convention).
- **Fix**: `CustomerName` + `useCustomers` names map (PR #255).
- **Verification**: frontend gates green; prod render confirmed.

### BUG-004 — Card-level accounting sync forced a full re-push (MED / reliability)
- **Repro**: click **Sync** on a connected accounting card → forced full
  re-push (~3 provider calls/invoice) → exceeds the 15-min async budget on a
  real tenant → `sync_status=error`.
- **Root cause**: `TriggerSyncAsync` always passed `force=true`.
- **Fix**: scoped (card) syncs run incrementally (dirty-tracking skips
  unchanged); the all-provider button keeps `force=true` (PR #256).
- **Verification**: single-flight scope tests updated; full suite green.

