# Bugs found & fixed

Running log of defects discovered during validation/hardening, newest first.
Each entry: severity, repro, root cause, fix, verification.

---

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
