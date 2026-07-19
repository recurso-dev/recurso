# Implementation Plan: Demo Mode & Hosted Sandbox

Spec: `docs/spec_demo_mode.md` (APPROVED 2026-07-19).
Task list: `tasks/todo.md`. Prior program archived at
`tasks/done-lago-parity-*.md`.

## Overview

One `DEMO_MODE=true` flag turns an instance into a safe public sandbox:
auto-seeded rich data, every outward edge forced to mocks in code,
dangerous settings 403'd, hourly wipe-and-reseed, and a `/auth/demo`
endpoint so the website button lands visitors in a logged-in dashboard.
Delivered with `docker-compose.demo.yml`; DNS is the only founder step.

## Dependency graph

```
T1 internal/demo (mode + forced-safe wiring in main.go)
 ├── T2 guard middleware (blocked edges)          ← safety proven here
 ├── T4 /auth/demo + dashboard auto-login/banner
 └── T5 reset worker ──── uses ── T3 seed extension (independent build)
T6 compose file        (after T1–T5 integrate)
T7 website CTA (flagged; independent, ships dark)
```

Order: T1 → T2 → T3 → T4 → T5 → T6 → T7. High-risk-first: the forced-safe
wiring (T1) and guards (T2) are the security core, so they land and get
checkpointed before anything user-facing.

## Architecture decisions

- **Safety in code, not config**: when `demo.Enabled()`, main.go
  constructs mock gateways / console notifier / nil GSP and sets the
  webhook worker's delivery-disable flag — later env additions cannot
  reintroduce egress by accident.
- **Guards as middleware** (like the audit trail): a method+route
  blocklist keeps coverage uniform and testable in one place.
- **Reset = boot seed path**: the worker calls the same seeding used on
  first boot, so a reset instance is indistinguishable from a fresh one.
- **Sessions reuse the existing auth stack**: `/auth/demo` logs in a
  pre-seeded user via the normal session code; no parallel auth.

## Phases & checkpoints

### Phase 1 — Safety core (T1, T2)
Checkpoint: DEMO_MODE boots with mocks provably wired (unit-asserted);
every blocked edge 403s with code `demo_mode`; flag off = byte-identical
behavior; full suite green. **Nothing user-facing ships before this.**

### Phase 2 — Experience (T3, T4, T5)
Checkpoint: fresh DEMO_MODE boot → seeded v0.6.0 showcase data →
`/auth/demo` session works → banner shows → reset worker restores
pristine data on a fake-clock tick. Suite green.

### Phase 3 — Delivery (T6, T7)
Checkpoint: `docker compose -f docker-compose.demo.yml up` serves the
demo end-to-end locally; website builds with the CTA dark (flag off) and
light (flag on). Docs page updated; CHANGELOG entry.

## Risks & mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| A blocked edge is missed and demo emails/charges a human | High | Forced-mock construction (not guards) is the primary defense; guards are defense-in-depth; egress assertion tests |
| Reset races active requests into corrupt state | Med | Truncate+reseed in one transaction per group; demo tolerates a blip; worker test covers mid-reset reads |
| Demo endpoint leaks into production builds | Med | Registered only under `demo.Enabled()`; test asserts 404 when off |
| Shared tenant vandalism (offensive names on screen) | Low | Hourly reset bounds exposure; per-visitor tenancy is the recorded escalation path |
| Compose drift vs make demo | Low | Compose reuses the same images/seed binary |

## Open questions

- Hosting target (VPS / Fly / Railway) — founder; code-identical.
