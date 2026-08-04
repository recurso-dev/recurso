# Recurso — Website (Marketing)

> For the marketing site (`recurso-website`). The goal is trust, not dazzle.
> Every choice answers one question: *would a CFO trust this company with their
> revenue?*

## Goal

Make **finance leaders and CFOs trust us.** Not impress designers. Authority,
enterprise credibility, professionalism. If a section makes us look clever but
not trustworthy, cut it.

## Identity (settled)

The **light identity is the keeper** (a dark rebuild was tried and reverted).
Emerald accent, generous whitespace, tokens-only styling. Do not reintroduce a
dark marketing theme without a deliberate decision.

## Homepage flow

1. **Hero** — the provable claim, plainly stated. Not "revolutionary billing" —
   something like "Billing where the books always reconcile."
2. **Trusted by / social proof** — logos, numbers, or a concrete proof artifact.
3. **Why Recurso** — the accounting-first thesis in one screen.
4. **Features**, each explained not hyped:
   - Accounting Engine (the double-entry ledger)
   - Revenue Recognition (deferred revenue, month-end close)
   - Usage Billing (metering, charge models)
   - Customer Portal (self-service)
   - Developer API (API-first, SDKs)
   - Enterprise Security (residency, audit trail, RBAC)
5. **Testimonials** — real, specific, finance-voiced.
6. **Pricing** — transparent.
7. **FAQ** — the objections a finance buyer actually has.
8. **CTA** — try the demo / talk to us.

## Writing style

- Short sentences. No marketing fluff.
- No "revolutionary," "AI-powered," "best platform," "seamless." (See
  `BRAND.md`.)
- **Explain, don't hype.** Show the artifact — a balanced close pack, a
  zero-discrepancy reconciliation, a real invoice — and let it carry the claim.
- Numbers are specific and real.
- Every feature section leads with *what it does for the finance team*, then how.

## Proof over promises

The strongest marketing asset Recurso has is that **the books demonstrably tie
out.** Screenshots of a balanced trial balance, a reconciliation reading "0
discrepancies," a real month-end close pack — these outperform any adjective.
Use them.

## Technical constraints

- Static site (Vite), direct-push to main, auto-deploys via Cloudflare.
- Two SSG gotchas documented in project memory; tokens-only styling rule.
- `/llms.txt` + `/llms-full.txt` are maintained for AI discovery.

## Related

- `BRAND.md` — voice; `DESIGN.md` — visual language
- `COMPETITORS.md` — the differentiation to communicate
- `PRODUCT.md` — the substance behind the marketing
