# Bugs found & fixed

Running log of defects discovered during validation/hardening, newest first.
Each entry: severity, repro, root cause, fix, verification.

---

### BUG-007 — Icon-only buttons without an accessible name (LOW / a11y)
- **Repro**: a screen reader on the Developers page (delete-endpoint,
  refresh-events, refresh-deliveries) and the Quotes row menu announces an
  unlabeled button — the `title` attribute is not reliably read.
- **Root cause**: four icon-only buttons carried only `title`, no `aria-label`.
- **Fix**: `aria-label` added to each (matching the title).
- **Verification**: frontend lint/build/vitest green; grep confirms every
  `size="icon"` and row-action icon button now has an accessible name.

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
