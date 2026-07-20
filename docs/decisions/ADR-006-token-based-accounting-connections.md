# ADR-006: Token-based connection flow for non-OAuth accounting providers

## Status
Accepted

## Date
2026-07-20

## Context
Accounting sync supports four providers. QuickBooks and Xero use the browser
OAuth flow (`/accounting/connect/:provider` → consent screen → callback).
NetSuite and Tally had working sync adapters but **no way to establish a
connection**, so the Integrations page could never offer them:
- NetSuite's OAuth needs account-specific authorize/token URLs (the account id
  is part of the hostname), so a generic redirect flow can't start without
  user input anyway; admins can mint SuiteTalk OAuth 2.0 tokens in NetSuite's
  own UI.
- Tally is a local JSONL export — there is no remote party to authenticate.

## Decision
One additional endpoint, `POST /v1/accounting/connect-token/:provider`, for
providers outside browser OAuth:
- `netsuite`: requires `account_id` + `access_token` (pasted from NetSuite's
  UI). `RealmID` carries the account id, mirroring how QBO uses it, so
  `getAdapterForConnection` needed no changes.
- `tally`: no credentials; connecting simply enables the export sync.
Reconnecting updates the existing row in place (same as the OAuth callback's
reconnect path). The UI collects credentials in a right-side sheet and marks
NetSuite EXPERIMENTAL (sandbox verification is founder-gated).

## Alternatives Considered

### Implement full NetSuite OAuth 2.0
- Pros: no token pasting; refresh handled automatically
- Cons: needs a registered integration record per NetSuite account,
  account-specific URLs mid-flow, and founder-gated sandbox verification the
  adapter hasn't had yet; heavy lift for an EXPERIMENTAL integration
- Deferred: the token flow validates demand first; OAuth can supersede this
  ADR later without breaking the connection model

### Overload the existing /connect endpoint
- Cons: same route returning either a redirect URL or a created connection
  depending on provider is a confusing contract for SDKs
- Rejected: separate operation, separate OpenAPI semantics

## Consequences
- Pasted NetSuite tokens expire and are NOT auto-refreshed (no refresh token
  is collected); syncs will fail with auth errors until the admin reconnects.
  Acceptable for EXPERIMENTAL; revisit with OAuth if NetSuite graduates.
- Tally connections respect `RESIDENCY_MODE=self_hosted` (it is the one
  provider allowed there — file export, no egress).
- Tokens are stored server-side (`access_token` column, `json:"-"`) and never
  displayed after entry.
