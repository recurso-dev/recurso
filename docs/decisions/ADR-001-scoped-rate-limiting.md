# ADR-001: Scope-namespaced fixed-window rate limiting

## Status
Accepted

## Date
2026-07-20

## Context
Production users were intermittently locked out: any ~20 API requests within a
minute made every `/auth/*` call return 429, bouncing logged-in users to the
login screen, where login itself 429'd ("Could not reach the API"). Root cause:
the global limiter (500/min, mounted on every route) and the strict public
auth limiter (20/min) both counted into the same Redis key
(`ratelimit:<client-ip>`), so the strictest limiter judged the *combined*
traffic of all limiters.

Requirements: brute-forceable endpoints (login, register, password reset)
need a low ceiling; session-state endpoints (`/auth/me`, fired on every page
load) must never lock out normal browsing; the general API needs a high
tenant-fair ceiling.

## Decision
`RateLimitMiddleware` takes a mandatory `scope` string that namespaces the
counter key (`ratelimit:<scope>:<ip>` / `ratelimit:<scope>:tenant:<id>`).
Three scopes exist: `api` (500/min, global), `public` (20/min, credential and
payment-initiation endpoints), `session` (120/min — `/auth/me`, logout, oauth
provider listing). Fixed-window via Redis `INCR`+`EXPIRE` is kept.

## Alternatives Considered

### One shared limiter with per-route thresholds
- Pros: single counter, simple mental model
- Cons: exactly the bug we had — any limiter reading a shared counter judges
  everyone else's traffic
- Rejected: correctness requires isolation between budgets

### Sliding-window or token-bucket algorithms
- Pros: smoother behavior at window edges
- Cons: more Redis round-trips or Lua scripting; the failure mode we fixed was
  key collision, not window shape
- Rejected for now: fixed-window is adequate; revisit only with evidence of
  edge-burst abuse

### Per-endpoint scopes (one bucket per route)
- Pros: maximal isolation
- Cons: an attacker gets a fresh 20/min budget per credential endpoint;
  operators lose any aggregate view
- Rejected: scope-per-*budget-class* is the useful granularity

## Consequences
- Limiters with different limits MUST use different scopes — enforced by the
  signature (there is no default scope).
- A user's dashboard browsing can no longer exhaust the auth budget; automated
  crawlers still can (three retries over ~2.25s in `AuthProvider` absorb
  transient 429s).
- The E2E-visible symptom class ("login randomly says API down") is gone;
  regression coverage lives in the login-bounce retry logic and rate-limit
  tests.
