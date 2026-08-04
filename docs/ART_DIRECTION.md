# Recurso — Art Direction

> The layer above the design system. **DESIGN.md** says which tokens are legal.
> **BRAND.md** says how Recurso sounds. This document says what Recurso *looks
> like* and *shows* — the creative direction that makes the site feel like one
> specific company and not "another SaaS." A design system constrains a site.
> Art direction gives it an identity. This is the missing layer.

---

## 1. The thesis

**Recurso is a financial operating system, not a SaaS product.** Everything on
the marketing surface exists to make a finance leader think, in the first five
seconds and before reading a single sentence:

> *"This company understands finance."*

Not "nice landing page." Not "another billing tool." The site earns that in one
way: **it shows the actual product doing real accounting, with real numbers that
tie out.** The product is the art. We do not decorate around it.

**The register:** a trusted accounting firm, built by world-class engineers.
Calm before clever. Precise before pretty. Users should feel *safe* before they
feel *impressed* — and safety, here, is visual: numbers that align, books that
balance, states that are honest.

## 2. The one decision that fixes everything

There is a tension in the current site, and it is the root cause of the
"generated" feel:

| Accounting software looks like | Developer marketing looks like |
|---|---|
| dense · precise · quiet · structured | big type · lots of whitespace · gradients · illustrations |
| a ledger, a trial balance, an invoice | a hero graphic, floating icons, feature cards |

**We choose accounting. Every time they conflict, accounting wins.** Recurso is a
developer *platform*, but its *soul* is the ledger. When a section could be a
clean marketing block or a dense, real accounting artifact, it becomes the
artifact. This single choice is what separates Recurso from every developer-SaaS
template on the internet.

Keep the developer-platform credibility (the API, the terminal, MCP) as
**punctuation** — a few dark "engineering glass" beats — not as the dominant
voice.

## 3. Show the product. Never make artwork.

The strongest asset Recurso has is that **its screens are already beautiful in a
way no illustration can fake** — because they're real double-entry accounting.

**Acceptable, and preferred, imagery:**
- A **real invoice** (GST/VAT-correct, IRN-signed, with the tax breakdown).
- A **real general ledger** — balanced double-entry postings, tabular figures,
  the BALANCED stamp.
- A **real reconciliation** screen — a count that matches.
- A **real month-end close** / trial balance / deferred-revenue waterfall.
- A **real dashboard** with representative-but-honest numbers.
- The **terminal** (`make demo`) and **code** — dark glass, real commands.

**Never:**
- Hero illustrations, 3D renders, isometric graphics, mascots.
- Floating/decorative icons; an icon per feature "because the grid wants one."
- Stock photography of people at laptops.
- Abstract gradient blobs, mesh gradients, "aurora" backgrounds.
- A generic dashboard mockup that isn't the real product.

If a section needs a visual, the answer is almost always **a real Recurso
screen**, cropped to the one thing that section is about.

## 4. Hero philosophy — the first five seconds

The hero is the thesis, and its job is to prove — not claim — that Recurso is
accounting-grade. **Lead with evidence, not a headline over a graphic.**

A hero should read like an operator's screen, not a billboard. Something like:

```
                                        ₹1,84,72,339   Revenue recognized   ↑ 14%
The books always                        3,821 / 3,821  payments reconciled   0 unmatched
reconcile.                              GST filed · VAT filed · nexus tracked
                                        ┌──────────────────────────────┐
[ Start free ]  [ Self-host ]           │  General journal — balanced   │
                                        └──────────────────────────────┘
```

The current hero already does the right thing on one axis (the live balanced
journal beside the claim). Push it further: **let real figures carry the first
impression**, and let the headline be a quiet, confident sentence a CFO would
say — not a feature name.

**Every hero must answer, in five seconds, visually:** *these people do real
accounting, and the numbers are correct.*

## 5. Sell confidence, not features

Feature names are what everyone writes. **Outcomes with proof** are what people
remember. Reframe every section from "what it is" to "what it gives you," and
back the claim with a real count.

| Don't write | Write instead |
|---|---|
| Automated Revenue Recognition | **Close your books in minutes, not days.** |
| Smart Reconciliation | **3,821 payments · 3,821 reconciled · 0 unmatched.** |
| Multi-region Tax Engine | **GST, VAT and US sales tax — filed-ready, from day one.** |
| Immutable Double-Entry Ledger | **Every number explains itself. Every entry reverses.** |
| Usage-Based Billing | **Meter anything. Bill it to the paisa.** |

The pattern: **a plain-language outcome, then a hard number as proof.** A count
that ends in `0 errors` or `0 unmatched` is worth more than any adjective.

## 6. How to visualize accounting concepts

Each core concept has one honest, memorable visual. Reuse these; don't invent an
icon for them.

- **Reconciliation** → a matched count: `3,821 payments · 3,821 reconciled · 0 unmatched`.
- **The ledger** → real DR/CR postings that sum equal, with a BALANCED stamp.
- **Revenue recognition** → a deferred → recognized curve, or a month-by-month
  waterfall.
- **Tax correctness** → a real invoice with the tax lines broken out (CGST/SGST,
  VAT reverse-charge, sales-tax nexus) and an IRN/UBL signature.
- **Trust / audit** → a trial balance that balances; an exported general ledger.
- **Multi-region** → three real invoices side by side (₹ / € / $), one ledger.
- **Agent-operability** → the MCP settings screen with money-path tools gated.

The metaphor is always **the accounting artifact itself**, rendered precisely —
never a symbolic illustration of it.

## 7. Composition — break the uniform rhythm

The tell of a generated site is that **every section has the same weight**:
heading, paragraph, three cards, CTA, repeat. Kill that. A page is a story with
beats of *different* size, density, and color.

**Rules:**
- **No section may repeat the previous section's shape.** If one was a 3-card
  grid, the next is a full-bleed number, or a single wide screenshot, or a dense
  table, or a dark code beat.
- **Vary the weight.** Some beats are loud (a single giant figure, a full-width
  ledger). Most are quiet. Contrast is the point — a quiet section only reads
  quiet next to a loud one.
- **Retire the feature-card grid as the default.** Where the current site uses a
  row of equal icon-cards (Products, Solutions, Compliance, Usage), replace the
  primary instance with **one real screen** and let two or three *supporting*
  facts sit beside it as plain lines — not boxed cards.
- **Asymmetry over symmetry.** A 60/40 split (argument + real artifact) reads
  more confident than three centered columns.
- **Density is a feature.** A dense, real trial-balance or ledger table is *more*
  premium here than whitespace — it signals "we handle the hard part."
- **Alignment is the aesthetic.** Columns of tabular figures, right-aligned money,
  hairline rules — precision is the decoration.

**The story arc** (each beat a distinct visual treatment):
thesis (real numbers) → the ledger underneath it all → what it meters/bills →
the books tie out (reconciliation, close) → tax, done right → operable by agents
→ open source you run → transparent pricing → the objections a CFO has → one last
proof and the ask.

## 8. Texture, color, motion, light

- **Palette:** warm-stone light for the marketing shell; **dark "engineering
  glass"** (`#08211C`) reserved for code, terminals, and product chrome; **cream
  ledger paper** reserved for accounting artifacts (the journal tape). Emerald is
  the one accent, used sparingly. *(Brand-color refresh emerald→blue is deferred
  post-GA — see DESIGN.md §13.)*
- **The one signature:** the living ledger tape — real postings that balance to
  the paisa, one red BALANCED stamp. One signature per page; everything else
  stays quiet. Don't add a second hero motif.
- **Motion:** functional and quiet — a reveal that never leaves content blank
  (fail-safe), the ledger posting its legs, the stamp landing. No parallax, no
  scroll-jacking, no numbers counting up on money, no floating decoration.
- **Light/texture:** flat and precise. The only "texture" is the ledger paper and
  the faint dot/line grid behind the hero. No noise, no glow, no glass-morphism.

## 9. The banned list

None of these appear on any Recurso surface — they are the vocabulary of the
generic SaaS template:

- Hero illustrations, 3D/isometric art, mascots, stock photos of people.
- The **feature-card grid as a section's primary content** (icon + heading +
  blurb × 3/4/6). Supporting facts may be plain lines; the hero of a section is a
  real screen.
- Decorative/floating icons; an icon assigned to every feature.
- Gradient blobs, mesh/aurora backgrounds, glass-morphism, drop-glow.
- Emoji, exclamation marks, playful microcopy ("Oops…", "Let's go! 🎉").
- Equal-weight sections repeating the same shape.
- Fake/placeholder screenshots; numbers that don't add up.
- A second hero motif competing with the ledger tape.

## 10. Establishing trust before a word is read

Trust here is visual and it is specific. Before any copy is read, the page should
already say "these people are careful with money" through:

1. **Numbers that tie out** — a visible count ending in `0 unmatched` / `0 errors`.
2. **A balanced ledger** — DR = CR, with the stamp.
3. **Tabular, aligned figures** — money right-aligned, monospaced, columns true.
4. **Real, precise artifacts** — an actual invoice, an actual trial balance.
5. **Restraint** — no noise, no gimmicks. Quiet is credible.

If a visitor screenshots the hero and it looks like a screenshot from an
accounting system, we've won.

## 11. How to use this document

- **Before designing a section, ask:** what real artifact proves this? Lead with
  it. What's the outcome (not the feature)? Say that. Does this section repeat the
  previous one's shape? Change it.
- **When the design system and the art direction seem to conflict,** the system
  tells you *which token*; this document tells you *what to build*. Build the
  accounting artifact; style it with the tokens.
- **This is direction, not audit.** It describes the target. The current site
  partially conforms (the hero's live journal, the real product screenshots) and
  partially doesn't (the feature-card grids, uniform section weight). Closing that
  gap is the work; track it in `../REMEDIATION.md`, not here.

## Related
- `DESIGN.md` — the token/system constraints ("which values are legal").
- `BRAND.md` — voice and personality ("how Recurso sounds").
- `WEBSITE.md` — the marketing-site goals and homepage flow.
- `COMPETITORS.md` — the differentiation this art direction must communicate.
