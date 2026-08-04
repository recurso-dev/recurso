# Recurso — Design Language

> How the product looks and why. Describes philosophy and the real design
> tokens, not pixels. When this doc and the code disagree, fix one deliberately.

## Personality

Enterprise · professional · modern · trustworthy · minimal · fast · calm ·
premium.

Never playful. Never consumer. Never "startup landing page." A finance operator
should feel the same confidence they feel in Mercury or Ramp — that the tool is
precise and will not surprise them.

## Inspiration

Stripe · Linear · Mercury · Vercel · Ramp · Notion · Raycast · Anthropic. The
common thread: restraint, whitespace, typography doing the work, and color used
sparingly to mean something.

## Color

The accent is **emerald**, not blue. This is the shipped brand identity
(`--primary` resolves to emerald; `primary-hover` is emerald-600 `#059669`), and
the marketing site's light identity is the keeper. Do not silently switch the
accent; changing it is a brand decision, not a styling tweak.

- **Ground:** near-white, generous whitespace. Dark theme is a first-class
  peer, not an afterthought.
- **Accent (emerald):** brand, primary actions, active nav — used sparingly.
- **Green = success only** (a state, distinct from the emerald accent).
- **Red = destructive/critical only** (delete, failed, past-due).
- **Amber = warning/attention** (needs review, degraded).
- **Neutrals** carry a slight cool bias — chosen, not defaulted.
- **No gradients** unless they illustrate value (a chart fill, a data viz).
  Never a decorative gradient hero.

Semantic status color (good/warning/critical) is separate from the accent and
never doubles as it.

## Typography

- **Body:** Inter / the system sans stack, 14–16px. Consistent type scale.
- **Numbers:** `font-variant-numeric: tabular-nums` everywhere digits align —
  money columns, tables, metrics. Money never jitters between rows.
- **IDs / codes:** monospace (`ui-monospace`). An invoice number or a posting
  code is mono; a customer name is not.
- Readable tables and readable numbers are the priority over display flair.

## Spacing & layout

- One spacing scale (Tailwind's). No off-scale pixel values.
- Layout does the spacing (flex/grid + `gap`), not per-element margins.
- Content max-widths keep running text ~65ch; tables get their own horizontal
  scroll container so the page body never scrolls sideways.

## Cards

Minimal borders · large padding · soft shadows · rounded (~12px, the shadcn
default). A card is a quiet container, not a decorated box.

## Tables (accounting-first)

Tables are the core surface — they must be excellent:

- Sortable, filterable, with a clear empty state.
- Sticky headers on long lists; **virtualized** for high-cardinality data.
- **Exportable** (CSV) — a finance user will always want the data out.
- Tabular numbers; right-aligned money; currency-exponent-correct formatting
  (JPY has no decimals, KWD has three — never hardcode `/100`).
- Server-side pagination for anything that grows with the customer base.
- Dense mode is welcome; finance users scan a lot of rows.

## Dashboard hierarchy

**Numbers first. Charts second. Actions third.** A finance operator opens a page
to read a figure, not to admire a hero. Lead with the summary metrics, then the
visualization, then the controls.

## Components & system

Built on **shadcn/Radix** (`src/components/ui/`) + **Tremor** for charts +
house **patterns** (`src/components/patterns/`: DataTable, PageHeader, StatCard,
EmptyState). Detail views and create/edit forms are **right-side Sheets**
(`sm:max-w-md`). Confirmations use `ConfirmDialog`. Reuse these; don't reinvent.

## Charts

Give charts the same care as type: an area fill, a faint grid, an emphasized
endpoint, the shared `ChartTooltip`. Animate only when the document is visible
and the user hasn't asked for reduced motion (a hidden-tab chart must render at
full height, not frozen at frame zero).

## Accessibility is part of design, not a checklist bolt-on

Keyboard operable, visible focus, labelled controls, contrast ≥ 4.5:1 for text,
and status never conveyed by color alone (pair it with text or an icon). See
`UX_RULES.md`.

## Related

- `BRAND.md` — voice and words
- `UX_RULES.md` — required states and interaction rules
- `ANTI_PATTERNS.md` — what never to build
