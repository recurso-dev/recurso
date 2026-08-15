# Dashboard Polish — Batch C Design
## Label / Overline / Section-role consolidation — INVESTIGATION (no code)

> **Read-only, code-cited, point-in-time (2026-08-15), against `main` @ `1ef831c3`.**
> No production code was written to produce this document. This is the Batch C
> **investigation deliverable** required before implementation. It ends with a
> STOP for approval.
>
> Scope guardrails (from the brief): typography-role consolidation only — **not**
> a visual redesign. Do **not** touch Money, ObjectHeader's amount signature,
> ObjectPage lifecycle, DataTable behaviour, tokens/colour system, motion,
> navigation, or the backend. Prefer the **smallest possible vocabulary**; avoid
> primitive proliferation (`<LabelSmall>`/`<LabelMedium>`/…).

---

## 1. The problem, precisely

One conceptual text role — the **uppercase micro-label** (the small, tracked,
uppercase eyebrow/label seen above object titles, on stat cards, as attribute
terms, and as table column headers) — is rendered with **many different raw
class strings** across the app, and even the shared primitives disagree with each
other. This is audit finding **#1** ("biggest single 'generated' tell") and the
core of Batch C.

**Verification against `main` (the audit estimated "~48 sites, 8 variants"):**

| Measure | Verified on `main` |
|---|---|
| Files containing `uppercase` | 39 |
| Total `uppercase` occurrences | 73 |
| `uppercase` + `tracking-*` (the micro-label signature) | **65** |
| …of those, inside shared primitives (to fix once) | **8** |
| …inline consumers (pages / slide-overs / layout) | **57**, across **28** files |
| Distinct micro-label class strings | **18** (the audit's "8" was a low estimate) |
| Non-label `uppercase` (badges / chips / mono codes / logic) — **exclude** | 8 |

So the drift is **worse** than the audit's estimate: **65 micro-label sites in 18
distinct class strings**, not 48/8. The good news is 8 of them are primitive
definitions — fixing those propagates automatically.

### 1a. The primitives themselves disagree

| Primitive | Site | Class string | Colour | Tracking |
|---|---|---|---|---|
| `ObjectHeader` kicker | `ObjectPage.jsx:74` | `text-xs font-medium uppercase tracking-wide **text-subtle**` | subtle (7.25:1) | wide |
| `AttributeList` term (`dt`) | `ObjectPage.jsx:151` | `text-xs font-medium uppercase tracking-wide **text-subtle**` | subtle | wide |
| `FinancialSummary` term (`dt`) | `FinancialSummary.jsx:51` | `text-xs font-medium uppercase tracking-wide **text-subtle**` | subtle | wide |
| `FinancialSummary` group header | `FinancialSummary.jsx:63` | `text-xs font-medium uppercase tracking-wide **text-muted-foreground**` | muted (5.27:1) | wide |
| `StatCard` label | `StatCard.jsx:69` | `text-xs font-medium uppercase tracking-wide **text-muted-foreground**` | muted | wide |
| `TableHead` (column labels) | `table.jsx:56` | `text-xs font-medium uppercase **tracking-wider** **text-muted-foreground**` | muted | **wider** |

The object-page family (kicker, AttributeList, FinancialSummary term) uses
**`text-subtle`**; the stat/table family uses **`text-muted-foreground`** — the
exact split audit #1 calls out. Column headers additionally use `tracking-wider`.

### 1b. The 18 inline variants (verified counts)

```
 12  text-[11px] font-medium uppercase tracking-wide text-muted-foreground
 11  text-xs uppercase tracking-wide text-muted-foreground
  9  text-xs font-medium uppercase tracking-wide text-subtle
  9  text-xs font-medium uppercase tracking-wide text-muted-foreground
  3  text-xs uppercase tracking-wide text-subtle
  3  text-xs font-semibold uppercase tracking-wider text-muted-foreground/70
  3  text-xs font-semibold uppercase tracking-wide text-muted-foreground
  3  text-xs font-medium uppercase tracking-wider text-muted-foreground
  3  text-left text-xs uppercase tracking-wide text-muted-foreground   (raw <tr> headers)
  3  text-[11px] font-semibold uppercase tracking-wider text-muted-foreground
  1  text-xs font-semibold uppercase tracking-wider text-success
  1  text-xs font-semibold uppercase tracking-wider text-subtle
  1  text-xs font-semibold uppercase tracking-wider text-muted-foreground
  1  text-xs font-medium uppercase tracking-wide text-primary
  1  text-sm font-semibold uppercase tracking-wide text-muted-foreground
  1  text-[11px] font-medium uppercase tracking-wider text-muted-foreground
  1  text-[10px] font-medium uppercase tracking-wide text-subtle
```

Drift dimensions: **size** (`text-[10px]`/`text-[11px]`/`text-xs`/`text-sm`),
**weight** (none/`medium`/`semibold`), **tracking** (`wide`/`wider`), **colour**
(`subtle`/`muted-foreground`/`muted-foreground/70`/`primary`/`success`).

---

## 2. Semantic roles (grouped by MEANING, not CSS)

The uppercase-micro style is used for four *different* semantic jobs. They look
alike today, and the design goal is that they **keep looking alike on purpose**,
via one atom — not by coincidence via 18 copies.

| Role | What it is | Example sites | Semantic element |
|---|---|---|---|
| **R1 — Overline / kicker** | The eyebrow above a title stating object/section type | `ObjectHeader` kicker ("INVOICE"); PlanDetail/CustomerDetail section eyebrows | `div`/`span` (not a heading) |
| **R2 — Metadata / attribute term** | The label half of a key→value pair | `AttributeList` `dt`, `FinancialSummary` `dt`, Details/rail rows | **`dt`** (paired with `dd`) |
| **R3 — Stat / KPI label** | The label above a metric numeral | `StatCard` label, `FinancialSummary` group headers | `p`/`span` labelling the metric |
| **R4 — Table column label** | The header of a data column | `TableHead` (all DataTables); 3 raw-`<tr>` headers | **`th`** (scope=col) |

Two adjacent roles are **already housed and must stay separate** (they are *not*
the uppercase micro-label and must not be folded in):

| Role | What it is | Home today | Style |
|---|---|---|---|
| **R5 — Field label** | The label of a form input | `ui/label.jsx` (Radix `Label`) | `text-sm font-medium`, **sentence case** |
| **R6 — Secondary / context text** | Help text, captions, descriptions, subtitles | de-facto canonical inline | `text-xs`/`text-sm text-muted-foreground`, sentence case |

---

## 3. Proposed canonical vocabulary (smallest that covers the roles)

**One new primitive. One reuse. One token alignment. One documented convention.**

### 3.1 `<Overline>` — ONE atom for R1 + R2 + R3 (NEW, `components/ui/overline.jsx`)

A single, prop-light primitive that renders the canonical uppercase micro-label.
It replaces every R1/R2/R3 raw class string.

```jsx
// One canonical style. No size/tone/weight variants (that is the anti-pattern
// the brief forbids). Polymorphic element only, to preserve semantics.
<Overline>Invoice</Overline>              // R1 overline  → <div>
<Overline as="dt">Amount due</Overline>   // R2 term      → <dt>
<Overline as="span">MRR</Overline>        // R3 stat label→ <span>
```

- **Canonical class:** `text-xs font-medium uppercase tracking-wide text-subtle`
  (12px / medium / +wide tracking / `foreground-subtle`).
- **`as` prop** (default `"div"`): the only knob. It sets the element so callers
  keep correct semantics (`dt` inside a `<dl>`, `span` inline) **without** any
  style variant. Merges a caller `className` for layout-only tweaks (margins),
  never for restyling the role.
- **No** `size`, `tone`, `variant`, or `muted` props. If a caller thinks it needs
  one, that is a signal to reconsider the role, not to add a prop.

### 3.2 `TableHead` (R4) — align tokens to the atom (IMPROVE existing)

Column labels are **already centralised** in `ui/table.jsx:56` — this is a
one-line token alignment, not a migration. Change its label tokens from
`tracking-wider text-muted-foreground` to the canonical
`tracking-wide text-subtle` so column headers match every other micro-label.
(The 3 raw-`<tr>` headers in `AccountPage`/`WalletPage`/`QuotePage` get the same
canonical class applied **as typography only** — their raw-`<table>` structure is
Batch D/E work and is *not* converted here.)

### 3.3 `ui/label.jsx` (R5) — reuse unchanged

Field labels already have a correct, distinct home (`text-sm font-medium`,
sentence case). **Do not** route uppercase labels through it; **do not** restyle
it. Documented as the field-label home; no change.

### 3.4 Secondary/context text (R6) — document, do not primitivise

`text-xs text-muted-foreground` appears **212×** and is already de-facto
consistent; wrapping it in a primitive would be **pure migration noise** for no
gain (and the brief warns against exactly that). **Decision: no new primitive.**
Instead, document the sanctioned classes — `text-xs text-muted-foreground`
(caption/help) and `text-sm text-muted-foreground` (description/subtitle) — and
only normalise the handful of genuine **off-scale** outliers
(`text-[13px]`, `text-[11px]`-as-body) onto `text-xs`. This keeps Batch C tight.

**Net new primitives: 1 (`<Overline>`).** Everything else is reuse, a one-line
token alignment, and documentation.

---

## 4. Canonical colour decision (needs your call)

Consolidating 18 variants into one **requires** choosing one colour token for the
role — that is the consolidation, and it is the only unavoidable visual change.
Both candidate tokens already exist and both pass WCAG AA:

| Option | Token | Contrast on white | Who uses it today | Effect of choosing it |
|---|---|---|---|---|
| **A (recommended)** | `text-subtle` (`--foreground-subtle`, L34%) | **7.25:1** | Object-page reference (kicker, AttributeList, FinancialSummary term) | Object pages unchanged; StatCard / FinancialSummary-group / TableHead / most inline labels get **slightly darker** (a11y ↑) |
| B | `text-muted-foreground` (L40%) | 5.27:1 | StatCard, TableHead, majority of inline | Reference object-page labels get **slightly lighter** (a11y ↓) |

**Recommendation: A (`text-subtle`).** It matches the Invoice **reference
standard**, raises contrast on small 12px text, and is the more defensible
"authored" choice. It is **not** a token/colour-system change (both tokens exist);
it is choosing which existing token the role uses. The trade-off — ~40 labels
currently on `muted-foreground` shift one step darker, uniformly — is the
consolidation working as intended, not a redesign. **If you prefer minimal pixel
change over the reference/a11y win, say so and I'll canonicalise on B instead.**

---

## 5. Exact migration map

### 5.1 Variant → canonical

| Legacy class string | Role | → Canonical |
|---|---|---|
| `text-xs font-medium uppercase tracking-wide text-subtle` | R1/R2 | `<Overline>` (no change to output) |
| `text-xs font-medium uppercase tracking-wide text-muted-foreground` | R3 | `<Overline>` (→ subtle) |
| `text-xs uppercase tracking-wide text-muted-foreground` | R1/R3 | `<Overline>` (+medium, →subtle) |
| `text-[11px] font-medium uppercase tracking-wide text-muted-foreground` | R2/R3 | `<Overline>` (→text-xs, →subtle) |
| `text-xs font-semibold uppercase tracking-wider text-muted-foreground[/70]` | R1/R3 | `<Overline>` (→medium, →wide, →subtle) |
| `text-xs font-medium uppercase tracking-wider text-muted-foreground` | R4-ish | `<Overline>` / `TableHead` (→wide, →subtle) |
| `text-[10px] font-medium uppercase tracking-wide text-subtle` | R2 | `<Overline>` (→text-xs) |
| `text-sm font-semibold uppercase tracking-wide text-muted-foreground` | R1 | `<Overline>` (→text-xs, →medium, →subtle) |
| `text-xs … uppercase … text-primary` / `text-success` | R1 accent | **case-by-case** — if the colour is meaningful (accent state) keep as a one-off and note it; if decorative, `<Overline>` |
| `text-left text-xs uppercase tracking-wide text-muted-foreground` (raw `<tr>`) | R4 | canonical class on the `<th>` (typography only; table structure untouched) |

### 5.2 Primitive consolidation points (fix once → propagates)

| File:line | Primitive | Action |
|---|---|---|
| `ObjectPage.jsx:74` | ObjectHeader kicker | render via `<Overline>` (output identical) |
| `ObjectPage.jsx:151` | AttributeList `dt` | render via `<Overline as="dt">` (identical) |
| `FinancialSummary.jsx:51` | term `dt` | `<Overline as="dt">` (identical) |
| `FinancialSummary.jsx:63` | group header | `<Overline>` (colour →subtle) |
| `StatCard.jsx:69` | KPI label | `<Overline>` (colour →subtle) |
| `table.jsx:56` | TableHead | token alignment: `tracking-wider text-muted-foreground` → `tracking-wide text-subtle` |
| `PaymentAttempts.jsx` (+1 more) | inline label | `<Overline>` |

### 5.3 Inline consumers — **57 sites across 28 files**

Pages (by descending count): `Ledger` (7), `Integrations` (6), `SubscriptionPage`
(5), `PaymentPage` (3), `Developers` (3), `Security`, `ExecutiveSummary`,
`AskAnalytics`, `WalletPage`, `UnitEconomics`, `MCPSettings`, `EUEInvoiceSettings`,
`BillingSettings`, `RevenueWaterfall`, `QuotePage`, `PortalDashboard`,
`Organizations`, `MonthEndClose`, `InvoicePage`, `FinanceReconciliation`, `Events`,
`Entities`, `AuditLog`, `AccountPage`. Slide-overs: `PlanDetail` (5),
`CustomerDetail` (5), `PlanCharges`. Layout/settings: `SettingsLayout`, `Sidebar`,
`BuyGiftModal`. Each site is replaced with `<Overline>` (with `as="dt"` where it
is an attribute term). No layout, spacing, or content changes beyond swapping the
label element/class.

---

## 6. Before / after examples

**R1 overline (object kicker) — primitive, output identical, colour already subtle:**
```diff
- <div className="text-xs font-medium uppercase tracking-wide text-subtle">{kicker}</div>
+ <Overline>{kicker}</Overline>
```

**R3 stat label — StatCard, colour normalises muted→subtle:**
```diff
- <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">{label}</p>
+ <Overline as="p">{label}</Overline>
```

**R2 attribute term — stays a `<dt>`:**
```diff
- <dt className="text-xs font-medium uppercase tracking-wide text-subtle">{label}</dt>
+ <Overline as="dt">{label}</Overline>
```

**R4 column label — one token edit, all tables inherit:**
```diff
- "h-10 px-4 … text-xs font-medium uppercase tracking-wider text-muted-foreground …"
+ "h-10 px-4 … text-xs font-medium uppercase tracking-wide text-subtle …"
```

**Inline drift (an `[11px]` semibold one-off) → the atom:**
```diff
- <span className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground/70">Method</span>
+ <Overline as="span">Method</Overline>
```

---

## 7. Accessibility implications

- **Case:** uppercasing stays a CSS `text-transform` (`uppercase` class), so
  screen readers read the underlying mixed-case text — unchanged. The atom keeps
  this; no literal-uppercase text is introduced.
- **Semantics preserved:** the `as` prop keeps every attribute term a `<dt>`
  (inside its `<dl>`), every column label a `<th scope="col">`, every overline a
  non-heading `<div>`/`<span>`. **No semantic heading becomes a span, and no span
  becomes a heading.** An overline is explicitly *not* an `<h*>`.
- **Contrast improves** under the recommended Option A (5.27:1 → 7.25:1 on the
  ~40 `muted-foreground` label sites); nothing regresses below AA.
- **No colour-only meaning** is introduced; labels remain text.
- **Focus / keyboard / reduced-motion:** the atom is static, non-interactive text
  — no focus, motion, or keyboard surface added or removed.
- **Tests:** the atom's only behavioural contract is "renders the given element
  and passes children/props through" — covered by a small component test
  (`as` renders `dt`/`span`/`div`; className merges; content passes). **No
  CSS-class snapshot tests** (the brief forbids them).

---

## 8. Cases that must NOT be migrated

1. **R5 form field labels** (`ui/label.jsx` and its ~all FormField consumers) —
   different role (sentence-case, `text-sm`). Left untouched.
2. **Badges / chips / codes that merely happen to be uppercase:**
   - `Entities.jsx:99` — `<Badge … className="text-[10px] uppercase">` (a tag on a Badge).
   - `ExecutiveSummary.jsx:268` — `bg-muted px-1.5 py-0.5 text-[10px] uppercase` (a chip).
   - `EUEInvoiceSettings.jsx:118` — `font-mono uppercase` (a scheme/country **code**).
   These are not the micro-label role; migrating them would be wrong.
3. **`status-badge.jsx` internal `uppercase`** — API-status normalisation logic /
   comment, not a label class.
4. **Section titles / headings** (`<h2 text-sm|text-base font-semibold>`) — these
   are the **section-title** role (audit #13), *not* overlines. See §10; **out of
   this batch's enumerated vocabulary** and deferred.
5. **Raw-`<table>` structure** of `AccountPage`/`WalletPage`/`QuotePage` — Batch C
   applies the canonical *typography* to their existing `<th>` cells only; it does
   **not** convert them to `DataTable` (that is Batch D/E).
6. **Accent-coloured labels with meaning** (`text-primary`/`text-success` on a
   label) — reviewed case-by-case; if the colour carries state it stays a
   documented one-off rather than being flattened.

---

## 9. Whether a new primitive is actually necessary

**Yes — exactly one (`<Overline>`), and no more.**

- The uppercase micro-label role is the single largest drift source (65 sites, 18
  strings) and has **no home**; the two flagship primitives already disagree on
  its colour. A shared atom is the only way to make R1–R4 consistent *by
  construction* instead of by 18 coincidences.
- It does **not** duplicate `ui/label.jsx` (that is R5 — sentence-case form
  labels; a genuinely different role). Overloading `Label` with an uppercase
  variant would be the "variant-proliferation" anti-pattern in disguise.
- Everything else is **reuse/alignment, not new primitives**: R4 is a one-line
  token edit to the existing `TableHead`; R5 is untouched; R6 is documented, not
  primitivised. StatCard/AttributeList/FinancialSummary are **improved by
  consuming the atom**, not replaced.

If, during migration, a site cannot be expressed as `<Overline>` + `as` without a
new prop, that is a **STOP condition** (§11) — I will document it rather than grow
the API.

---

## 10. Related but deferred (documented, not done in Batch C)

- **Section-title split (audit #13)** — `ObjectSection` uses `text-sm font-semibold`
  while ~11 report pages use `text-base font-semibold` (`RevenueWaterfall`,
  `MRRWaterfall`, `InvoiceAging`, `MonthEndClose`, `RevenueRecognition`,
  `TrialBalance`, `UnitEconomics`, …). This is the **section-title** role, which
  is **not** in Batch C's enumerated vocabulary (overlines / field / metadata /
  column / secondary). Consolidating it forces a visual size change on either the
  object pages (reference) or the report pages, so it is a **separate decision**
  best made explicitly. **Recommendation:** handle as a small follow-up (a single
  `<SectionTitle>` or routing report `<h2>` through the object canon) — not folded
  into Batch C. Flagging for your call; default is defer.
- **Shadow-ladder normalisation (audit #15)** — elevation, not typography; the
  brief bans touching tokens/colour/elevation. Out of scope.
- **Off-scale body sizes** (`text-[13px]`) beyond the label role — noted; only the
  label-role off-scale sizes are fixed here.

---

## 11. STOP conditions (will halt + document, not solve here)

- A site needs a **new `<Overline>` prop** (size/tone/weight) to render → stop,
  document; do not grow the API.
- Consolidation would require a **token/colour redesign** → stop (only choosing
  between existing `subtle`/`muted-foreground` is in scope).
- A page needs **bespoke label styling** to look right → stop, document.
- The work starts pulling in **section titles, tables, or elevation** as
  prerequisites → stop; those are other batches.

---

## 12. Proposed implementation shape (for when approved — NOT started)

1. Add `components/ui/overline.jsx` (`<Overline as>` + a small behavioural test).
2. Retrofit the 8 primitive sites (StatCard, ObjectHeader kicker, AttributeList,
   FinancialSummary ×2, TableHead token edit, PaymentAttempts) to the atom.
3. Migrate the 57 inline sites across 28 files, role-by-role, verifying `dt`/`th`
   semantics are preserved and no layout shifts.
4. Apply the canonical class to the 3 raw-`<tr>` headers (typography only).
5. Document R5 (reuse) and R6 (sanctioned classes) in the design docs; normalise
   label-role off-scale sizes.
6. Optional lint guard **only if** it can be added without migration noise or
   weakening CI (e.g., a targeted check that flags a *new* `uppercase tracking-*`
   raw string outside the atom). If it would be noisy, skip and document.
7. Run lint / build / full Vitest / CI; live visual QA on Home, Invoice,
   Subscription, Payment, Journal Entry, Reconciliation Run, Customer, and one
   dense DataTable page (hierarchy, density, alignment, contrast, wrapping, narrow
   viewport, light/dark, no heading/SR regressions).
8. Write `DASHBOARD_POLISH_BATCH_C_REPORT.md`; STOP.

---

## 13. Decisions — RESOLVED (2026-08-15)

1. **Canonical label colour → Option A, `text-subtle`.** The role canonicalises on
   `text-xs font-medium uppercase tracking-wide text-subtle`; the ~40 sites on
   `text-muted-foreground` shift one step darker (reference-aligned, a11y ↑).
2. **Section-title (#13) → DEFER.** Not in Batch C's enumerated vocabulary; kept a
   documented follow-up (§10). No `<SectionTitle>` in this batch.
3. **Lint guard → add only if non-noisy.** A targeted "no new raw uppercase-tracking
   outside `<Overline>`" check will be added **only** if it can be scoped to new
   violations without flagging legitimate non-migrated cases or weakening CI;
   otherwise skipped and documented (§12.6).

**Investigation complete and decisions locked. Ready to implement on the explicit
go-ahead — no production code written yet.**
