# Recurso — Source-of-Truth Docs

Durable product, design, and engineering philosophy. These outlast any single
feature and keep new work consistent as the product grows. Read the relevant one
**before** building, not after.

## The documents

These are **code-derived**: every factual claim cites the file/package that
proves it, and the raw code-cited findings live in [`evidence/`](evidence/).
Where a doc and the code disagree, the code wins (see
[DOCUMENTATION_RULES.md](DOCUMENTATION_RULES.md)).

| Doc | Answers | Read before… |
|---|---|---|
| [PRODUCT.md](PRODUCT.md) | What Recurso is + a code-cited feature inventory | any product decision |
| [ARCHITECTURE.md](ARCHITECTURE.md) | Package map, flows, jobs, security (code-derived) | changing structure |
| [DESIGN.md](DESIGN.md) | Design intent vs. the real tokens + deltas | building UI |
| [BRAND.md](BRAND.md) | How it sounds — voice, words to use/avoid | writing any copy |
| [UX_RULES.md](UX_RULES.md) | The behavioral contract + UI audit | building a page |
| [ANTI_PATTERNS.md](ANTI_PATTERNS.md) | What never to build | anything money/UX |
| [ACCOUNTING_PRINCIPLES.md](ACCOUNTING_PRINCIPLES.md) | The ledger contract (posting codes 1–25) | any money path |
| [API_GUIDELINES.md](API_GUIDELINES.md) | The API contract + deviation audit | adding an endpoint |
| [WEBSITE.md](WEBSITE.md) | Marketing-site goals, flow, and tone | website work |
| [COMPETITORS.md](COMPETITORS.md) | Where we win/lose and how we say it | positioning |
| [DOCUMENTATION_RULES.md](DOCUMENTATION_RULES.md) | How these docs stay true; every-PR checklist | any PR |
| [evidence/](evidence/) | Raw code-cited inspection findings | verifying a claim |
| [../REMEDIATION.md](../REMEDIATION.md) | Live audit-remediation status | remediation work |

## The through-line

Recurso is **accounting-first**. The ledger is the product; billing is its
surface. Every doc here serves one promise: *every number is explainable, every
event is reversible, and the books always reconcile.* When a decision is
unclear, the option that keeps that promise wins.

## Precedence

The user's explicit instruction wins, then the project's existing system (these
docs + `CLAUDE.md` + ADRs), then defaults. When two of these docs disagree, that
is a bug — fix one deliberately and note why.
