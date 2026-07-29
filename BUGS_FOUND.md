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
