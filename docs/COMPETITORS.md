# Recurso — Competitive Landscape

> Where we win, where we don't, and how we talk about it. Honesty here keeps us
> from over-claiming (see `BRAND.md`). Update as the market moves.

## The one-line positioning

Most billing tools bolt accounting on as an export. Recurso is **accounting-first
billing** — the ledger is the source of truth, not a downstream sync. That's the
axis we compete on.

## Stripe Billing

- **Their strengths:** ubiquitous, superb payments, huge ecosystem, trusted
  brand, great docs.
- **Their gaps:** billing is payments-first; revenue recognition and a true
  double-entry ledger are add-ons/exports, not the core. Reconciliation is your
  problem.
- **We win on:** an auditable ledger where every number ties out; month-end
  close as a first-class artifact; open-source/self-hostable.
- **We're weaker on:** payments breadth, brand trust, ecosystem.
- **How we say it:** "Stripe moves the money; Recurso proves the books." Never
  disparage — we integrate with Stripe.

## Chargebee

- **Strengths:** mature subscription management, wide integrations, established
  in mid-market.
- **Gaps:** accounting is integration-driven; correctness is trust-me, not
  prove-it.
- **We win on:** built-in reconciliation and the invariant/ledger guarantees;
  API-first developer experience.
- **Weaker on:** breadth of billing edge cases they've accumulated over years.

## RevenueCat

- **Strengths:** the default for mobile-app subscription entitlements; excellent
  at app-store billing.
- **Gaps:** entitlement-focused, not a general billing + accounting engine; no
  ledger.
- **We win on:** being a full billing/accounting engine (we even import from
  them); web billing, invoicing, tax, revrec.
- **Weaker on:** native app-store billing depth.

## Orb / Metronome

- **Strengths:** excellent usage metering built for high-volume consumption
  pricing (the AI-billing wave).
- **Gaps:** metering-first; lighter on double-entry accounting, revrec close,
  and multi-tax compliance.
- **We win on:** metering *plus* the accounting spine and statutory compliance
  (GST/VAT/US) in one system.
- **Weaker on:** raw metering scale/throughput at the very top end (a resilience
  + load-test priority — see REMEDIATION.md M4).

## Lago

- **Strengths:** open-source billing, developer-friendly, strong usage billing.
- **Gaps:** accounting depth, revrec, and statutory e-invoicing are less
  developed.
- **We win on:** the accounting-first ledger, reconciliation guarantees, and
  compliance breadth — the "beat Lago" program is explicitly about matching its
  metering while exceeding it on correctness and compliance.
- **Weaker on:** community size/maturity.

## Where Recurso is genuinely weaker (say it honestly)

- Payments breadth and brand trust vs Stripe.
- Metering throughput at extreme scale vs Metronome/Orb (until the resilience +
  load work lands).
- Ecosystem/integration count vs the incumbents.
- Operational maturity — the current frontier is proving resilience and
  continuous quality, not adding features.

## How we communicate differences

- Lead with the provable claim (the ledger ties out), not a feature checklist.
- Never say "better than X" — show the artifact (a balanced close pack, a
  zero-discrepancy reconciliation) and let it speak.
- Integrate, don't attack — we import from Stripe/Chargebee/RevenueCat; they're
  migration sources, not just rivals.

## Related

- `PRODUCT.md` — the mission this positioning serves
- `WEBSITE.md` — how this shows up in marketing
- `BRAND.md` — the no-hype constraint
