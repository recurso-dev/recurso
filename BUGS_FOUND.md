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
