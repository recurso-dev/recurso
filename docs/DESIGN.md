---
name: Recurso Design Language
description: >-
  Recurso's own accounting-first design language — a hybrid (~45% Linear / 35%
  Stripe / 20% Vercel, principles not copies) optimized for trust, clarity,
  auditability, and data density. Every UI decision answers "would a CFO trust
  this with their revenue?". Warm-stone neutrals, Inter with tabular figures for
  money, soft shadows, 8px radius, light-only. Tokens below are the REAL shipped
  values from src/index.css + tailwind.config.js (the accent is emerald today);
  the founder-set target direction (blue accent) and its deltas are in §A/§I.
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
radius:
  base: 8px      # --radius 0.5rem
  md: 6px        # calc(radius - 2px), buttons
  sm: 4px        # calc(radius - 4px)
shadows:         # tailwind.config.js:115-119
  input: "0 1px 2px 0 rgb(0 0 0 / 0.05)"
  card: "0 1px 3px 0 rgb(0 0 0 / 0.1), 0 1px 2px -1px rgb(0 0 0 / 0.1)"
  dropdown: "0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -2px rgb(0 0 0 / 0.1)"
dark-mode: disabled   # light-only, index.css:8
---

# Recurso — Design Language

> **Code-derived.** The frontmatter tokens above are the real values from the
> cited files; the prose cites files too. Implementation is the source of truth
> for "what is." §A is the founder-set *intent*; §C lists where intent ≠ code.

---

## §A. Design intent (target language)

**Recurso is its own system — a hybrid, not a copy of any brand.** Extract the
principles below, don't imitate the aesthetic. The blend that fits an
accounting-first developer platform:

| Surface | Blend (principles, not brand copies) |
|---|---|
| **Dashboard** | 70% Linear · 20% Stripe · 10% GitHub — data density, spacing, keyboard-first, enterprise tables |
| **Website** | 50% Stripe · 30% Vercel · 20% Mercury — financial trust + clean marketing |
| **Docs** | 80% Mintlify · 20% Stripe Docs |
| **Customer portal** | 70% Stripe Billing · 30% Linear |
| **This doc's basis** | ~45% Linear · 35% Stripe · 20% Vercel |

**The filter for every UI decision:** *"Would a CFO, finance manager, or
engineering leader trust this product with their revenue?"* Recurso isn't trying
to be the most visually striking SaaS — it's trying to be the most **trustworthy**
billing and accounting platform.

**Personality:** enterprise · professional · modern · trustworthy · minimal ·
fast · calm · premium. Never playful, never consumer, never "startup landing
page."

**Optimize for:** trust · clarity · auditability · data density · speed ·
accessibility.

**Prioritize:** readable tables · tabular numbers · accounting layouts ·
enterprise navigation · keyboard-first workflows · responsive dashboards · high
information density without clutter.

**Avoid:** flashy gradients · oversized hero sections · decorative animations ·
unnecessary whitespace · consumer aesthetics.

**Intended color:** almost-white ground with generous (not wasteful) whitespace,
**blue/indigo accent** (the Linear/Stripe lineage), green *only* for success,
red *only* for destructive, gradients only when they illustrate value.
**Cards:** rounded ~12px. **Tables:** virtualized, dense mode.

---

## §B. Colors (implemented)

**Accent = emerald**, not blue (`src/index.css:20` `--primary: 161 94% 30%`; the
file comment names it "Deep emerald"). Chart accent is emerald-500 `#10B981`.
Hover is `#059669` (`tailwind.config.js:59`). **Neutrals are warm stone**
(`--foreground: 24 10% 10%`, `--muted: 60 5% 96%`, hairline `--border: 20 6%
90%`) — "enterprise light, not cool gray" per the code comment. Body ground is
`bg-stone-50`.

**Semantic** (`src/components/ui/badge.jsx:12-17`, always paired with a text
label): success emerald, warning amber, destructive red. **Consequence:** green
currently does double duty as accent *and* success — see §C.

## §C. Typography

Inter → system (`tailwind.config.js:96-99`). Type scale is the Tremor scale
(label 12 / body 14 / title 18 / metric 30). **Money is tabular + monospace**:
`tabular-nums` on every `td` (`src/index.css:69-71`) and the `.money` mono stack
(`:74-79`); IDs shortened via `shortId()` (`src/lib/utils.js:94-96`). Money
values are exponent-aware (`currencyDecimals`, `utils.js:17-26`) — never `/100`.

## §D. Layout, elevation, shape

- **Spacing:** Tailwind's scale; layout uses flex/grid + `gap`, not per-element
  margins.
- **Radius:** base 8px (`--radius: 0.5rem`); buttons use `rounded-md` = 6px
  (`ui/button.jsx`); cards `rounded-lg` = 8px.
- **Elevation:** three soft shadows only — `tremor-input`/`-card`/`-dropdown`
  (`tailwind.config.js:115-119`). No heavy layered shadows.
- **Whitespace:** generous; content max-widths keep tables and text readable.

## §E. Components (house library)

- **Buttons** (`ui/button.jsx`): default `h-9 px-4 py-2`, sm `h-8`, lg `h-10`,
  icon `h-9 w-9`; `rounded-md`; focus ring `ring-2 ring-ring ring-offset-2`.
- **Cards** (`ui/card.jsx`): minimal border, soft shadow, `rounded-lg`, generous
  padding.
- **Inputs/forms** (`ui/input.jsx`, `patterns/FormField.jsx`): hairline border,
  emerald focus ring; per-field validation messages.
- **Tables** (`patterns/DataTable.jsx`): the accounting-first surface — sortable,
  built-in error/loading/empty/data states (`:91-104`), keyboard-operable rows
  (`:124-139`), optional pagination, right-aligned tabular money. Wrapped in a
  `Card overflow-hidden`.
- **Sheets** (`ui/sheet.jsx:35`): right-side `sm:max-w-md` for detail/create.
- **Dialogs** (`ui/confirm-dialog.jsx`): Radix, `busy` state, destructive styling
  for destructive actions.
- **Charts** (`charts/ChartTooltip.jsx`): shared premium tooltip; animation gated
  on visibility + `prefers-reduced-motion` (`:77-87`).
- **StatCard, PageHeader, EmptyState, ErrorState, LoadingSkeleton,
  CustomerSelect/Name** in `patterns/`.

## §F. Dashboard hierarchy

**Numbers first, charts second, actions third** — StatCards lead, then the
visualization, then controls. Mostly followed today; verify per page.

## §G. Do / Don't

**Do:** use emerald sparingly for accent + primary action; right-align tabular
money; give every list error/loading/empty/success; name the exact figure;
keyboard-operate everything. **Don't:** write `dark:` variants (light-only);
hardcode `/100`; use green for a non-success accent *and* success without
deciding (§C); make a destructive action the default button; render a blank page
on error; paginate a billing sweep. (Full list: `ANTI_PATTERNS.md`.)

## §H. Responsive

Works at 320/768/1024/1440; wide content scrolls inside `overflow-x-auto`; the
page body never scrolls sideways. **Violations:** native `<Table>` pages missing
the wrapper (Team `:122`, Security `:403`, DunningDashboard `:272`) and clipping
`overflow-hidden` (RevenueRecognition `:166`, FinanceReconciliation `:177`).

---

## §I. Direction vs. current — the deltas (design decisions to make)

| Aspect | Intent (§A) | Current (cited) | Decision |
|---|---|---|---|
| **Accent** | Blue | **Emerald** (`index.css:20`) | **Rebrand call** — touches `--primary`, the safelist, and the "light identity is the keeper" website decision. Not done. |
| Green usage | Success only | Accent **and** success | Follows the accent decision. |
| Card radius | 12px | 8px (`--radius`) | One-token change once confirmed. |
| Tables | Virtualized, dense | Not virtualized; native tables unpaginated | Real work — see `UX_RULES.md` audit. |

**The one needing your explicit go/no-go: the accent.** The app ships emerald
end-to-end; moving to blue is a rebrand, not a tweak. Until decided, this doc
documents emerald as reality and blue as the target.

## §J. Implementation audit (violations)

State-handling drift (BillingSettings has no error path), responsive table
wrappers, oversized components (SubscriptionDetail 1011 / Developers 948 /
PlanCharges 777), portal test hole. Full detail:
`docs/evidence/design-and-ux.md`; tracked in `../REMEDIATION.md`.

---

## Source of truth

- **Code:** `frontend/src/index.css`, `frontend/tailwind.config.js`,
  `frontend/src/lib/utils.js`, `frontend/src/components/{ui,patterns,charts}/`,
  `frontend/src/App.jsx`.
- **Evidence:** `docs/evidence/design-and-ux.md`.
- **Direction:** founder design input (§A), 2026-08-04.
- **Format:** structured after the `awesome-design-md` token-spec convention,
  populated with Recurso's real tokens.
- **Related:** `BRAND.md`, `UX_RULES.md`, `ANTI_PATTERNS.md`,
  `DOCUMENTATION_RULES.md`.
