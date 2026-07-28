# Bugs Found

Issues surfaced during the test-engineering run. Fixed bugs link to their PR.

## Fixed (during this session's feature work, pre-test-run)
- **Subscription cancel was broken (400).** The cancel endpoint requires a
  `reason` (`binding:"required"`), but the dashboard posted an empty body, so
  every cancel failed validation. Fixed in PR #290 (cancel-with-reason dialog).
  Severity: high (core lifecycle action unusable). Verified: dashboard now sends
  the reason; frontend test asserts the payload.

## Open / under investigation
_(none yet from the test run — this file is updated as failing tests are triaged
into app-bug vs test-bug vs wrong-assumption.)_

## Triage protocol
For each failing test discovered:
1. Reproduce in isolation.
2. Classify: application bug / flaky test / incorrect assumption.
3. If app bug and safe to fix → fix + regression test + PR. Else document here
   with severity, root cause, repro, and fix status.
