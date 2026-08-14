# Recurso Dashboard — Motion System

Motion in the dashboard exists to communicate **state, change, causality,
progress, hierarchy, and feedback** — never decoration. If an animation doesn't
say one of those things, it doesn't ship. The feel is Stripe / Linear / Vercel:
precise, fast, restrained. This is financial infrastructure; it should read as
confident, not entertaining.

## Audit (what already exists)

- **`tailwindcss-animate`** drives every Radix primitive (Dialog, Sheet,
  Dropdown, Tooltip, Select) via `data-[state=open]:animate-in` /
  `fade-in` / `zoom-in` / `slide-in-from-*`. **No `framer-motion` — and we are
  not adding one.** Motion is CSS + a little rAF, matching the website's system.
- **Buttons** already carry `transition-[…,transform] active:translate-y-px` +
  hover surface shifts. **Badges** have `transition-colors`. **StatCard** has
  `hover:shadow-md`. **Skeletons** use `animate-pulse` (not spinners).
- **Reduced motion** is respected globally: `src/index.css` neutralizes CSS
  `animation`/`transition` durations under `prefers-reduced-motion: reduce`.
- **Gaps this system fills:** no semantic duration/easing tokens (durations were
  ad-hoc — Sheet used 300/500ms); no JS-side reduced-motion hook (needed for
  rAF-driven number interpolation); no reusable motion primitives.

## Tokens

Defined once as CSS custom properties in `src/index.css` and mirrored into the
Tailwind theme (`transitionDuration` / `transitionTimingFunction`). Never
scatter raw millisecond values.

| Token | Value | Use |
|---|---|---|
| `--motion-fast` / `duration-fast` | `140ms` | micro-interactions: hover, press, focus, icon, tooltip |
| `--motion-normal` / `duration-normal` | `200ms` | component motion: cards, rows, panels, reveals |
| `--motion-slow` / `duration-slow` | `340ms` | drawers, signature moments |
| `--ease-standard` / `ease-standard` | `cubic-bezier(0.2,0,0,1)` | default — quick out, gentle settle |
| `--ease-out` / `ease-out-soft` | `cubic-bezier(0.16,1,0.3,1)` | entrances / reveals (decelerate) |

Prefer `transform` and `opacity`. Never animate `width/height/top/left/margin`
without a strong reason. No continuously-running animation.

## Levels

- **L1 micro** — hover / press / focus / icon / tooltip / status color. Very subtle.
- **L2 component** — stat cards, table rows, filters, panels, reveals. Noticeable, restrained.
- **L3 signature** — only a few: a financial value settling, a status advancing,
  reconciliation resolving to `₹0`, a journal entry balancing. Memorable, never distracting.

## Primitives (`src/components/patterns/` + `src/lib/`)

| Primitive | Purpose |
|---|---|
| `useReducedMotion()` | rAF/JS-driven motion must gate on this; SSR-safe, live-updating |
| `<MotionNumber>` | interpolates a financial value to its new figure (tabular-nums, no bounce); snaps to the final value under reduced motion; only animates on real change |
| `<MotionReveal>` | mount reveal (fade + 4px rise) with optional delay |
| `<MotionStagger>` | applies incremental reveal delays to children (header → primary → secondary) |
| `<MotionState>` | briefly flashes a highlight when a keyed value/status actually changes ("something happened") |

Deferred until the phase that needs them (avoid speculative abstraction):
`MotionPresence` (list add/remove) and `MotionDrawer` (Sheet already animates).

## Accessibility

Under `prefers-reduced-motion: reduce`: no stagger, no rise, no number
interpolation (snap to final), no flash — but **every state change and all
information is preserved**. Nothing important depends on animation. CSS honors
it globally; JS primitives honor it via `useReducedMotion()`.

## Rollout (phases)

1. **Tokens + primitives** ← this doc / PR
2. Shell — sidebar, page transitions, page header
3. Core feedback — buttons, dropdowns, dialogs, toasts, forms
4. Data — stat cards, numbers, tables, filters
5. Financial state — invoice / payment / subscription lifecycle
6. Signature — reconciliation resolving, ledger posting balancing
7. Polish — activity feeds, charts, empty states

Each phase ships as its own green-CI PR (`npm run lint && npm run build && npx vitest run`).
