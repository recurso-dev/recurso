---
name: Recurso Design Language
description: >-
  Recurso's accounting-first design language — optimized for correctness, trust,
  auditability, and data density. Every UI decision answers "would a CFO trust
  this with their revenue?". Warm-stone neutrals, Inter with tabular figures for
  money, soft shadows, 8px radius, light-only. Tokens below are the REAL shipped
  values from src/index.css + tailwind.config.js (the accent is emerald today).
tokens_source:
  - frontend/src/index.css
  - frontend/tailwind.config.js
colors:
  # accent — emerald (implemented). --primary: 161 94% 30%
  primary: "#05946A"            # deep emerald, interactive elements
  primary-hover: "#059669"      # emerald-600, tailwind.config.js:59
  chart-accent: "#10B981"       # emerald-500, charts (index.css:20 comment)
  ring: "#05946A"               # focus ring = --ring (emerald)
  # neutrals — warm stone, not cool gray
  background: "#FFFFFF"         # --background 0 0% 100%
  foreground: "#1C1A17"        # --foreground 24 10% 10% (warm near-black)
  muted: "#F5F5F4"             # --muted 60 5% 96%
  muted-foreground: "#78736E"  # --muted-foreground 25 5% 45%
  border: "#E7E5E1"            # --border/--input 20 6% 90% (hairline)
  body-canvas: "#FAFAF9"       # body bg-stone-50
  # semantic
  success: "#047857"           # emerald-700 (badge success)
  warning: "#B45309"           # amber-700 (badge warning)
  destructive: "#DC4444"       # --destructive 0 72% 51%
typography:
  font-family: "Inter, ui-sans-serif, system-ui, sans-serif"   # tailwind.config.js:96-99
  mono: "ui-monospace, 'SF Mono', 'Cascadia Mono', Menlo"      # .money / kbd
  scale:                       # Tremor scale, tailwind.config.js:109-114
    label:  { size: 12px, line: 16px }
    body:   { size: 14px, line: 20px }   # default
    title:  { size: 18px, line: 28px }
    metric: { size: 30px, line: 36px }
  money: { feature: "tnum", family: "mono" }   # tabular, monospace (index.css:69-79)
spacing:                       # Tailwind scale (rem × 16)
  scale: [4, 8, 12, 16, 24, 32, 48, 64]   # px — the only step values used
radius:
  base: 8px      # --radius 0.5rem
  md: 6px        # calc(radius - 2px), buttons
  sm: 4px        # calc(radius - 4px)
shadows:         # tailwind.config.js:115-119
  input: "0 1px 2px 0 rgb(0 0 0 / 0.05)"
  card: "0 1px 3px 0 rgb(0 0 0 / 0.1), 0 1px 2px -1px rgb(0 0 0 / 0.1)"
  dropdown: "0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -2px rgb(0 0 0 / 0.1)"
motion:          # target standard (§8); reduced-motion always respected
  hover: 100ms
  press: 80ms
  page: 150ms
  modal: 200ms
dark-mode: disabled   # light-only, index.css:8
---

# Recurso — Design Language

> **Code-derived.** The frontmatter tokens are the real values from the cited
> files; implementation is the source of truth for "what is." Sections that
> describe a *target* (motion timings, the 44px touch target, the accent) are
> marked as **Direction** — adopt them going forward; don't assume the whole app
> already conforms. Point-in-time implementation gaps live in a **separate audit**
> (§14), so this document stays timeless.

---

## §1. Philosophy

Recurso is an **accounting-first** product. Under every screen is a real
double-entry ledger, and the interface exists to make that ledger legible,
trustworthy, and fast to operate. The single filter for every UI decision:

> **"Would a CFO, finance manager, or engineering leader trust this product with
> their revenue?"**

Recurso is not trying to be the most visually striking SaaS. It is trying to be
the most **trustworthy** billing and accounting platform. Every choice — a
number's alignment, a confirm dialog, an empty state — either earns that trust or
spends it.

**Personality:** enterprise · professional · calm · precise · trustworthy ·
minimal · fast. Never playful, never consumer, never "startup landing page."

**Desired qualities** (describe the intent directly — do not imitate any brand):
enterprise-first · accounting-focused · calm · information-dense · trustworthy ·
keyboard-friendly · quiet-by-default. Where an external product does one of these
well it can inform a decision, but the target is the *quality*, not the aesthetic
of any company.

## §2. Core principles (priority order)

When two goals conflict, the higher-priority one wins. This ordering is the
tie-breaker an engineer or an AI agent should apply.

1. **Correctness** — the number, the state, and the money path must be right. A
   beautiful screen that shows a wrong or unexplained figure is a failure.
2. **Trust** — auditability, reversibility, honest states (error/loading/empty),
   and explainability. Never hide uncertainty; surface it.
3. **Readability** — the primary figure is instantly findable; hierarchy is
   obvious; tables scan cleanly.
4. **Density** — high information-per-screen without clutter. Finance users
   compare; give them enough on one screen to compare.
5. **Beauty** — polish and restraint, *after* the four above are satisfied.

## §3. Visual language

- **Light-only.** White ground (`--background`), warm-stone body canvas
  (`bg-stone-50`). No dark mode (`index.css:8`); never write `dark:` variants.
- **Warm neutrals, not cool gray.** Foreground is a warm near-black
  (`--foreground: 24 10% 10%`); borders are a hairline warm stone
  (`--border: 20 6% 90%`). The warmth reads "enterprise light," not "cold tech."
- **One accent, used sparingly.** Emerald marks interactive elements and the
  single primary action per view — never as a decorative fill or a body-text
  color.
- **Quiet elevation.** Three soft shadows only (`input`/`card`/`dropdown`); no
  heavy layered shadows, no glow. Depth is a hairline border plus a soft shadow.
- **No decoration for its own sake.** No gradients unless they illustrate real
  value, no illustrations, no emoji, no background patterns behind data.

## §4. Tokens

The frontmatter is the token spec (colors, type, spacing, radius, shadows,
motion) drawn from `src/index.css` and `tailwind.config.js`. Rules of use:

- **Color:** use the semantic tokens (`primary`, `foreground`, `muted`, `border`,
  `success`, `warning`, `destructive`) — never a raw hex. Emerald is
  accent + primary action; amber = warning; red = destructive only.
- **Money is exponent-aware.** Format through `currencyDecimals` (`utils.js:17-26`)
  — never divide by `/100`. JPY has 0 decimals, KWD/BHD have 3.
- **Semantic color is never the sole signal** — always pair with a text label or
  icon (`ui/badge.jsx:12-17`).
- **Known token debt:** emerald currently does double duty as accent *and*
  success. When it's resolved, success may shift; until then, keep pairing
  success with its label so meaning never rests on hue alone.

## §5. Layout & spacing

### Spacing system

Use only the token steps — **4 · 8 · 12 · 16 · 24 · 32 · 48 · 64 px** (Tailwind
`1/2/3/4/6/8/12/16`). Never invent an off-scale value (`13px`, `mt-[22px]`).

| Step | Use |
|---|---|
| 4 / 8 | Between an icon and its label; inside a chip/badge; tight control gaps |
| 12 / 16 | Inside a card/cell; between a label and its field; between stacked controls |
| 24 | Between related blocks within a section; card grid gap |
| 32 | Between distinct sections of a page |
| 48 / 64 | Page top/bottom breathing room; major section separation |

Lay out sibling groups with **flex/grid + `gap`**, never per-element margins that
collapse or double.

### Canonical page hierarchy

Every dashboard page reads top-to-bottom in this order — **numbers first, charts
second, tables third, actions in place**:

```
Page
 └ PageHeader        title + one-line description + primary action (right)
 └ StatCards         the key figures — the answer before the detail
 └ Chart (optional)  the trend behind the figures
 └ DataTable         the rows, with row-level actions
 └ Detail (Sheet)    right-side panel for a single record
```

A finance user should get the answer (StatCards) before scrolling, then the
evidence (table). Actions live where the object is (row action, header action) —
not in a separate toolbar the eye has to hunt for.

### Responsive

Works at **320 / 768 / 1024 / 1440**. Wide content (tables, code, charts) scrolls
inside its own `overflow-x-auto` container; the page body never scrolls sideways.
Mobile-first: stack, then expand.

## §6. Typography

- **Inter** for everything (`tailwind.config.js:96-99`), system fallback.
- **Type scale** (Tremor): label 12 / body 14 (default) / title 18 / metric 30.
  Don't invent sizes; don't skip heading levels.
- **Money is tabular + monospace.** `tabular-nums` on every `td` (`index.css:69-71`)
  and the `.money` mono stack (`:74-79`) so digits align in columns. IDs are
  shortened via `shortId()` (`utils.js:94-96`).
- Right-align every numeric/money column; left-align text.

## §7. Components (house library)

Compose from these; **do not invent new base styles** (no new button variant, no
one-off card). Each entry: purpose · when to use · when NOT · notes.

- **Button** (`ui/button.jsx`) — *purpose:* trigger an action. *Use:* one primary
  (emerald/filled) per view; everything else secondary/ghost/outline. *Don't:*
  make a destructive action the default; put two primaries on one screen; create
  a new variant. *Notes:* `rounded-md`; visible focus ring
  `ring-2 ring-ring ring-offset-2`; sizes sm `h-8` / default `h-9` / lg `h-10`.
- **Card** (`ui/card.jsx`) — *purpose:* group one coherent unit. *Use:* a stat, a
  form, a table wrapper. *Don't:* **nest a card inside a card**; stack heavy
  shadows. *Notes:* hairline border + `card` shadow, `rounded-lg`, generous
  padding.
- **DataTable** (`patterns/DataTable.jsx`) — *purpose:* the accounting-first
  surface. *Use:* every list. *Always* provide error / loading / empty / data
  states (`:91-104`); keep rows keyboard-operable (`:124-139`); right-align money.
  *Don't:* hand-roll a `<table>` without the wrapper; paginate an internal
  processing sweep.
- **Sheet** (`ui/sheet.jsx:35`) — *purpose:* view/edit one record without leaving
  the list. *Use:* right-side `sm:max-w-md`, header + description, pinned footer.
  *Don't:* use it for a bulk/multi-record flow.
- **ConfirmDialog** (`ui/confirm-dialog.jsx`) — *purpose:* guard an irreversible
  or money-moving action. *Use:* Radix dialog with a `busy` state and destructive
  styling for destructive intent. *Don't:* confirm trivial, reversible actions.
- **StatCard / PageHeader / EmptyState / ErrorState / LoadingSkeleton /
  CustomerSelect+Name** (`patterns/`) — the standard page furniture. Use the
  pickers, never raw UUID inputs.
- **Charts** (`charts/ChartTooltip.jsx`) — shared premium tooltip; animation is
  gated on visibility + `prefers-reduced-motion` (`:77-87`).

## §8. Interaction & motion  *(Direction — adopt going forward)*

Motion is functional, never decorative. Fast, quiet, and always cancellable by
reduced-motion.

| Interaction | Duration | Notes |
|---|---|---|
| Hover state | **100ms** | color/border only; no movement on data rows |
| Press / active | **80ms** | subtle; confirms the tap |
| Page / route transition | **150ms** | fade; no slide that shifts data |
| Modal / sheet open | **200ms** | fade + small translate |
| Skeleton → content | fade | never a spinner for content that has a skeleton |

- **`prefers-reduced-motion: reduce` disables all of the above** — content is
  visible and static, never gated behind an animation.
- No parallax, no scroll-jacking, no animated numbers counting up on money.
- A control's motion must never delay the user reading a figure.

## §9. Accessibility (contract)

Non-negotiable for every interactive surface (WCAG 2.1 AA baseline):

- **Touch target ≥ 44×44px** on touch; ≥ 24px always *(Direction — some controls
  are h-9/36px today; size up on touch)*.
- **Visible focus** on every focusable element (`ring-2 ring-ring ring-offset-2`)
  — never remove the outline.
- **Keyboard-reachable and operable** — every action works without a mouse;
  DataTable rows are keyboard-operable.
- **No hover-only interactions** — anything revealed on hover is also reachable by
  focus/click.
- **Contrast** ≥ 4.5:1 body, ≥ 3:1 large text; small semantic text uses the
  darker token (`brand.dark`, `emerald-700`).
- **Dialogs:** focus trapped, **Esc closes**, **Enter confirms** (Radix).
- **Labels:** every input has a `<label>` or `aria-label`; icon-only buttons have
  an `aria-label`.
- **Never rely on color alone** — pair with text/icon.

## §10. Copywriting

Words are UI. Write from the user's side of the screen, in the register of a
finance tool.

- **Short, specific, professional.** No marketing fluff, no exclamation marks, no
  emoji, no "Oops."
- **Name the action, not the vibe.** A button says exactly what happens; the toast
  confirms it happened.
- **Use accounting terminology** users recognize — *invoice, credit note,
  reconcile, write-off, deferred revenue* — not invented product-speak.
- **Errors** explain what went wrong and how to fix it; no apology, no blame.

| Prefer | Over |
|---|---|
| Create invoice | Let's get started! |
| Retry | Oops… something went wrong |
| No invoices yet | Nothing to see here 🎉 |
| Mark uncollectible | Say goodbye 👋 |
| Reconciliation found 0 discrepancies | All good! |

## §11. AI generation rules

When an AI agent generates or edits Recurso UI:

- **Never invent a color** — use the semantic tokens only.
- **Never invent spacing** — use the 4/8/12/16/24/32/48/64 scale only.
- **Reuse existing components; prefer composition** over configuration. If a
  pattern exists (`DataTable`, `Sheet`, `StatCard`, `ConfirmDialog`), use it.
- **Never create a new button style, shadow, or radius.**
- **Never add** gradients, illustrations, emoji, playful copy, background
  patterns behind data, or a second primary action.
- **Money** goes through the exponent-aware formatter; **numbers** are tabular and
  right-aligned.
- **Every list** ships error + loading + empty + data states.
- **Every irreversible/money action** is guarded by `ConfirmDialog`.
- **Use accounting terminology** (§10); write no `dark:` variants (§3).
- When unsure, **match the nearest existing page**, and prefer the more
  conservative, quieter option.

## §12. Do / Don't

**Do:** use emerald sparingly for accent + the one primary action; right-align
tabular money; give every list its four states; name the exact figure;
keyboard-operate everything; guard money-moving actions.

**Don't:** write `dark:` variants; hardcode `/100`; nest cards; add a second
primary; make a destructive action the default; render a blank page on error;
paginate a billing sweep; use color as the only signal. (Full list:
`ANTI_PATTERNS.md`.)

## §13. Implementation notes

- **Accent = emerald** (`index.css:20`), not blue. **Brand refresh (emerald→blue)
  is DEFERRED until post-GA** (founder decision, 2026-08-04): recoloring now would
  inject noise into screenshots, docs, marketing, and visual-regression baselines
  during the accrual-accounting work. Emerald is the shipped and documented
  identity — do not begin the rebrand.
- **Card radius** is 8px today; the target is ~12px — a one-token change to make
  when the accent decision is made.
- **Tables** are not yet virtualized; large lists use server pagination
  (`clampLimitOffset`).

## §14. Audit (separate document)

Point-in-time implementation gaps — missing responsive wrappers, state-handling
drift, oversized components, specific file:line violations — are **not** kept
here (they'd date this document). They live in **`docs/evidence/design-and-ux.md`**
and are tracked in **`../REMEDIATION.md`**. Update those, not §1–§13.

---

## Source of truth

- **Code:** `frontend/src/index.css`, `frontend/tailwind.config.js`,
  `frontend/src/lib/utils.js`, `frontend/src/components/{ui,patterns,charts}/`,
  `frontend/src/App.jsx`.
- **Evidence / audit:** `docs/evidence/design-and-ux.md`.
- **Related:** `BRAND.md`, `UX_RULES.md`, `ANTI_PATTERNS.md`,
  `DOCUMENTATION_RULES.md`.
- **Format:** token-spec frontmatter (the `awesome-design-md` convention)
  populated with Recurso's real values; body organized philosophy → principles →
  language → tokens → layout → type → components → motion → a11y → copy → AI
  rules → do/don't → notes → audit.
