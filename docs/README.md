# Recurso — Source-of-Truth Docs

Durable product, design, and engineering philosophy. These outlast any single
feature and keep new work consistent as the product grows. Read the relevant one
**before** building, not after.

## The documents

| Doc | Answers | Read before… |
|---|---|---|
| [PRODUCT.md](PRODUCT.md) | What Recurso is, who it's for, what we believe | any product decision |
| [DESIGN.md](DESIGN.md) | How it looks and why (real tokens, not pixels) | building UI |
| [BRAND.md](BRAND.md) | How it sounds — voice, words to use/avoid | writing any copy |
| [UX_RULES.md](UX_RULES.md) | The behavioral contract every screen keeps | building a page |
| [ANTI_PATTERNS.md](ANTI_PATTERNS.md) | What never to build | anything money/UX |
| [ACCOUNTING_PRINCIPLES.md](ACCOUNTING_PRINCIPLES.md) | The ledger contract every money path obeys | any money path |
| [API_GUIDELINES.md](API_GUIDELINES.md) | The public API contract | adding an endpoint |
| [WEBSITE.md](WEBSITE.md) | Marketing-site goals, flow, and tone | website work |
| [COMPETITORS.md](COMPETITORS.md) | Where we win/lose and how we say it | positioning |
| [../REMEDIATION.md](../REMEDIATION.md) | Live audit-remediation status | picking up remediation work |

## The through-line

Recurso is **accounting-first**. The ledger is the product; billing is its
surface. Every doc here serves one promise: *every number is explainable, every
event is reversible, and the books always reconcile.* When a decision is
unclear, the option that keeps that promise wins.

## Precedence

The user's explicit instruction wins, then the project's existing system (these
docs + `CLAUDE.md` + ADRs), then defaults. When two of these docs disagree, that
is a bug — fix one deliberately and note why.
