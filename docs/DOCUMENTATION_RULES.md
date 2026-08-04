# Recurso — Documentation Rules

> Documentation is part of the product. It is maintained with the same rigor as
> production code. This file governs how the `docs/` source-of-truth folder
> stays true.

## The prime directive

**The implementation is the source of truth.** When a document and the code
disagree, the code wins and the document is a bug to be fixed. Prefer
documenting **what exists today** over describing an ideal future state. When a
doc mentions planned work, it is clearly marked as planned (e.g. "🧭 planned" or
a link to `../REMEDIATION.md`), never stated as if shipped.

## How every document is written

1. **Inspect the code first.** No statement of fact goes in a doc without a
   file/package that supports it. Reference the implementing module inline
   (e.g. "posted by `internal/service/ledger.go`").
2. **Mark assumptions explicitly.** If something can't be verified from code,
   label it `ASSUMPTION:` and say what would confirm it.
3. **Separate current from planned.** Implemented capabilities and
   aspirational/roadmap items live in clearly distinct sections.
4. **Audit docs carry a violations section.** DESIGN, UX_RULES, and
   API_GUIDELINES each end with "current state → violations → recommendations,"
   grounded in real files. They are both a spec and an audit.

## Every PR must

A PR is **incomplete** if it changes behavior and does not also:

1. **Update the relevant doc(s)** in `docs/` (and `CLAUDE.md` if a convention
   changed).
2. **Verify the feature inventory** in `PRODUCT.md` — a new capability is added
   with its implementing package; a removed one is deleted.
3. **Verify API documentation** — a new/changed endpoint updates
   `cmd/api/openapi.yaml` (the drift gate enforces existence) and
   `API_GUIDELINES.md` if a convention changed.
4. **Verify architecture** — a new package, scheduler, worker, external
   integration, or flow updates `ARCHITECTURE.md`.
5. **Verify screenshots** — a UI change that invalidates a docs screenshot
   re-captures it (see the `images/setup/` and `images/dashboard/` conventions
   in the recurso-docs repo).
6. **Update the changelog** — `CHANGELOG.md` gets an `[Unreleased]` entry.
7. **Update migration notes** — a new migration is the next sequential number
   with `.up.sql` + `.down.sql`, and any behavioral consequence is noted.
8. **Keep REMEDIATION.md current** — remediation PRs update their finding's
   status.

## Review posture

When reviewing a document, **assume it is wrong until verified against the
code.** For any statement you can't trace to a file, either find the file and
cite it, mark it an assumption, or delete it.

## Ownership

These docs are owned by whoever last touched the code they describe. "The docs
are someone else's job" is not a valid state — the PR that changed the behavior
owns the doc update.

## Index

See `docs/README.md` for the document map. The set: PRODUCT, ARCHITECTURE,
DESIGN, BRAND, UX_RULES, ANTI_PATTERNS, ACCOUNTING_PRINCIPLES, API_GUIDELINES,
WEBSITE, COMPETITORS, DOCUMENTATION_RULES, plus `../REMEDIATION.md`.
