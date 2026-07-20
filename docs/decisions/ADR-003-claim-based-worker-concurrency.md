# ADR-003: Claim-based concurrency for money-path workers

## Status
Accepted

## Date
2026-07-20

## Context
Background workers act on due rows: the mandate-debit scheduler charges due
mandates; the rev-rec worker posts due recognition events. Cloud Run can run
multiple instances, and the distributed scheduler lock is best-effort — so two
workers can process the same due set concurrently. Observed failure (audit
F2): both workers SELECTed the same pending recognition events; the loser's
duplicate ledger post hit the idempotency unique index and its error path
overwrote the winner's `recognized` status with `failed`, understating
recognized revenue.

## Decision
Workers **claim** their work atomically with a single
`UPDATE … SET status='processing' WHERE … status='pending' RETURNING …`.
Postgres row locks serialize concurrent claims; the loser re-evaluates the
WHERE against the flipped status and receives a disjoint set. Status
transitions out of `processing` (`recognized`/`failed`) are guarded with
`AND status='processing'`, so a late loser can never demote a completed row.
Crash recovery: claims older than a grace window are requeued
(`claimed_at` column; 1h for rev-rec, the claim-window lease for mandates).

This is one idiom, used in both `MandateRepository.ClaimDueForDebit` and
`RevRecRepository.ClaimDueEvents`.

## Alternatives Considered

### SELECT … FOR UPDATE SKIP LOCKED
- Pros: classic queue pattern
- Cons: locks live only for the transaction — but these workers make external
  calls (gateway charges, ledger posts) that must not run inside a long DB
  transaction; once committed, the lock is gone and a second worker can grab
  the row
- Rejected: the status flip *is* the persistent lock

### Distributed lock around the whole worker tick
- Pros: simple to reason about
- Cons: single point of contention; the existing scheduler lock is already
  best-effort by design (ENG-161) and a lock failure must degrade to
  correctness, not double-charging
- Rejected as the *only* mechanism: claims make correctness independent of
  lock health

### Rely on ledger idempotency alone
- Pros: no schema change
- Cons: exactly the F2 bug — idempotency prevents double *posting* but the
  duplicate-post error still corrupts row status
- Rejected: detection without status safety is insufficient

## Consequences
- New workers over due rows should copy this idiom (claim → side effect →
  guarded transition), not invent locking.
- `recognition_events` gained `claimed_at` (migration 000105) and a
  `processing` status value; monitoring can watch for stuck `processing` rows
  older than the requeue window.
- Proven by `TestClaimDueEvents_DisjointAndGuarded` (disjoint claims; late
  failure-mark is a no-op) and the mandate cycle-claim tests.
