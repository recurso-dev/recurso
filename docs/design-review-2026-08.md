# Recurso — Product Design Review (2026-08)

> **Review-first artifact.** A code-grounded design audit of every dashboard,
> portal, auth, and settings screen (~60) plus the shared component layer, scored
> against Recurso's own design language (`DESIGN.md` / `UX_RULES.md` /
> `ANTI_PATTERNS.md`, priority order **Correctness > Trust > Readability > Density
> > Beauty**) **and** a world-class external bar (Stripe, Linear, Vercel, GitHub,
> Mercury, Clerk, Resend). Every finding cites the file it came from. Fixes are
> sequenced into waves below; this document is the plan of record and is updated
> as waves land.

## Verdict

**Overall product: ~74/100 — "trustworthy but uneven."** The architecture is
genuinely strong and often *ahead* of peers on the axis that matters most for a
finance product — correctness and honest states. The gap to a Stripe/Linear bar is
**not** in the framework; it is concentrated in a small number of **repeated,
mostly shared-component** issues at the leaf level. Fix the component foundation
and a handful of cross-cutting patterns and the whole surface moves into the mid-80s
at once.

**What's already world-class** (keep, don't touch): MonthEndClose's deferred
tie-out explanation (85), TrialBalance's integrity banner (84), ExecutiveSummary's
semantic-tone KPIs (84), ImportData's Compare gate (86), Security's MFA/SSO flows
(85), AskAnalytics "show the SQL", RevenueRecognition's ledger deep-links,
DataTable/StatCard/Money/Button in the component layer, and the honest-state habit
throughout (FX-exclusion banners, "nothing persisted", `has_start_history`
caveats). This is serious finance UX.

**What holds it back**: state-handling consistency, universal money rendering,
missing shared primitives, form accessibility, and the "last hop" drill-to-source —
none of them deep, all of them repeated.

---

## Scores by screen

Calibrated to the external bar (Stripe/Linear/etc.), not to "does it work."

### Commerce lists — avg ~77
| Screen | Score | Screen | Score | Screen | Score |
|---|---|---|---|---|---|
| Customers | 85 | Mandates | 83 | Invoices | 80 |
| Plans | 80 | Subscriptions | 76 | Credit Notes | 76 |
| Gifts | 74 | Coupons | 73 | Quotes | 68 |

### Create / edit forms — avg ~72
| Screen | Score | Screen | Score | Screen | Score |
|---|---|---|---|---|---|
| CreateQuote | 82 | CreatePlan | 79 | CreateCustomer | 78 |
| CreateSubscription | 68 | CreateCoupon | 66 | CreateCreditNote | 58 |

### Finance / accounting — avg ~75
| Screen | Score | Screen | Score | Screen | Score |
|---|---|---|---|---|---|
| MonthEndClose | 85 | TrialBalance | 84 | Collections | 82 |
| DunningDashboard | 80 | DunningCampaigns | 78 | Ledger | 77 |
| FinanceReconciliation | 71 | OfflinePayments | 70 | Disputes | 67 |
| GSTReturns | 55 | | | | |

### Dashboard / analytics — avg ~73
| Screen | Score | Screen | Score | Screen | Score |
|---|---|---|---|---|---|
| ExecutiveSummary | 84 | MRRWaterfall | 80 | AskAnalytics | 78 |
| RevenueRecognition | 76 | RevenueWaterfall | 74 | UnitEconomics | 74 |
| InvoiceAging | 73 | Dashboard (Home) | 72 | RevenueByPlan | 66 |
| RevenueByGeography | 64 | Churn | 58 | | |

### Auth / portal / checkout — avg ~76
| Screen | Score | Screen | Score | Screen | Score |
|---|---|---|---|---|---|
| PortalPaymentMethod | 80 | ForgotPassword | 80 | ResetPassword | 80 |
| Login | 78 | PortalLogin | 78 | PortalVerify | 78 |
| AcceptInvite | 77 | PortalRedeem | 75 | Checkout | 74 |
| Register | 72 | PortalDashboard | 70 | CustomerPortal (legacy) | 66 |

### Ops / admin / settings — avg ~74
| Screen | Score | Screen | Score | Screen | Score |
|---|---|---|---|---|---|
| ImportData | 86 | Security | 85 | AuditLog | 84 |
| GST Settings | 84 | Entities | 83 | InvoiceBranding | 83 |
| Settings hub | 82 | Legal Entities | 82 | CancelFlows | 80 |
| Tax Nexus | 80 | MCP Settings | 80 | Metering | 76 |
| Events | 74 | Team | 74 | USTax | 72 |
| IRP | 72 | EU E-Invoice | 72 | Referrals | 70 |
| Developers | 68 | Integrations | 66 | Notifications | 62 |
| Organizations | 61 | Profile | 58 | BillingSettings | 55 |

**Component layer: 80/100** — strong core (DataTable, StatCard, Money, Button),
held back by three systemic gaps (below).

---

## Cross-cutting themes (ranked by leverage × trust-impact)

Each appears in 3–6 of the six cluster reviews. Fixing a theme once pays off across
many screens — this is where the score moves.

### T1 · The state contract (loading / error / empty) is the systemic weak spot
UX_RULES makes skeleton-loading, error+retry, and a real empty state *mandatory*.
Reality: `ErrorState.jsx:16` **defaults its title to "Something went wrong"** — the
exact phrase ANTI_PATTERNS/BRAND forbid. ~15 screens are missing or masking an error
path: **BillingSettings (none at all — worst), Dashboard (none), Collections
(analytics fails to silent zeros), Developers (events/keys/webhooks show "empty" on
error), Referrals & Gifts & Quotes (console-only), Metering (empty catch), Integrations
& Entities (banner, no retry), Notifications (error string as the empty title), portal
(no retry), InvoiceBranding/USTax/IRP/EUEInvoice (silent defaults).** Bare "Loading…"
spinners appear where a `Skeleton` is required (Team, Developers, Organizations,
TaxNexus, BillingSettings, Notifications). Analytics skeletons don't match content
shape (MRRWaterfall/RevenueWaterfall/RevenueRecognition use `CardGridSkeleton`
before charts+tables → layout jump). **Fix:** change the `ErrorState` default; route
every screen through `ErrorState`/`DataTable`/`Skeleton`; add a `ReportSkeleton` that
matches analytics shape. Silent-zero on a money screen is worse than an honest error.

### T2 · Money is not rendered universally
The `Money` component (the only thing that renders the `.money` tabular-mono stack)
is used on a *minority* of screens; most hand-roll `formatCurrency()` in a
`tabular-nums` span, and several **bypass exponent-aware formatting entirely**:
Dashboard hardcodes `"USD"` on the MRR hero (`Dashboard.jsx:346`); AskAnalytics
renders all values with bare `toLocaleString` (`:171`); RevenueRecognition shows raw
**minor units** in multi-currency (`:64`); Checkout mixes `formatCurrency` and
`"USD 49.00"` in the same receipt (`:350,361`); FinanceReconciliation prints raw
integer minor units (`:223`). Money is **not right-aligned** on ~7 screens
(Subscriptions, Plans, Coupons, Mandates, OfflinePayments) despite DESIGN §6.
**Fix:** standardize on `<Money align="right">` in every money cell/headline; delete
the three ad-hoc money formatters; drive every headline off the real reporting
currency. This is the product's signature rule and the place a CFO can be shown a
materially wrong figure.

### T3 · Missing shared primitives (build once, adopt everywhere)
The same UI is reinvented across the app: **toggle switch** hand-built ≥4× (CreateCustomer,
CreateCoupon, IRP, EUEInvoice) and raw `accent-emerald` checkboxes in ~6 more (MCP,
GST-LUT, Security-SSO, CancelFlows, Register, Developers) → no `ui/Switch` / `ui/Checkbox`;
**three filter-control idioms** (segmented / rounded-pills / Select) and **three
date/segment controls** → no `SegmentedControl` / `DateRangePicker`; **CopyableSecret/
CopyField reinvented 5×** (Security, AuditLog, Events, Integrations, Developers) with
divergent behavior; **no `CodeSample`** block anywhere (the defining developer gap);
**no metric-definition affordance** (every KPI leans on a one-line hint); **Textarea**
class duplicated across GST/branding/EU; **`Button` has no `loading` state** so every
caller hand-rolls `<Loader2 animate-spin>` + disabled. **Fix:** a component-foundation
wave that ships these primitives and a `StatCard` `definition` prop; dozens of screens
inherit consistency for free.

### T4 · Form accessibility & validation are split-brain
`FormField` renders its error as a plain `<p>` with no `id`, no `role="alert"`, and is
**not linked to the input** (`aria-describedby`/`aria-invalid`) — the single highest-value
a11y fix. `Input` has no error visual state. No form focuses the first invalid field on
submit or announces errors. Validation is inconsistent: `errors`-object + FormField
(Customer/Plan/Quote/CreditNote) vs `toast.warning` (Subscription) vs native `required`
only (Coupon). **Fix:** wire FormField/Input a11y; add a tiny `useFormErrors` helper
(live region + focus-first-error); standardize every form on FormField.

### T5 · Drill-to-source is only half-wired
Reports drill *into* the ledger well, but the last hop out to the source document is
text, not a link: Ledger `reference_id` (`:341`), FinanceReconciliation invoice/txn IDs
(`:205`), Collections invoice number (`:379`), Disputes invoice (`:113`), InvoiceAging
buckets (dead-end — the Home widget links *to* Aging which then goes nowhere),
UnitEconomics / RevenueByPlan / RevenueByGeography / Churn metrics, Entities AR cell.
UX_RULES: "every figure traces to its postings/source." **Fix:** make every
`invoice_id`/`reference_id`/`transaction_id`/bucket/metric a link. This is the single
change that most raises auditor-trust.

### T6 · Raw UUIDs / raw IDs leak on high-trust screens
Against ANTI_PATTERNS: CreateCreditNote's "Linked invoice" is a **free-text UUID box**
(`:169` — worst single instance), Disputes shows customer+invoice UUIDs (`:113,118`),
PortalDashboard shows `invoice.id.substring(0,8)` instead of the invoice number (`:438`),
Organizations add-tenant is a raw-UUID paste + full-UUID list, OfflinePayments `recorded_by`
(`:215`), AuditLog actor slice, Metering sub fallback, FinanceReconciliation un-scaled
integers. **Fix:** pickers on input; resolve IDs to names on display.

### T7 · Global polish & motion accessibility
`index.css` has **no `prefers-reduced-motion` block** — every spinner/`animate-pulse`/
Radix enter-exit runs unconditionally (charts respect it; the chrome doesn't). No
autofocus on any auth field; no password show/hide. Stray `dark:` variants (TrialBalance,
MonthEndClose) violate light-only. Token drift toward raw Tailwind palette
(`text-red-600`/`bg-emerald-50`/`text-stone-*`) is pervasive — **Checkout is almost
entirely raw** — where semantic tokens (`text-destructive`, `bg-muted`) are required.
Three different brand marks across auth/checkout/portal. Off-scale radius (`rounded-xl`
on the Login logo). `sheet.jsx` open/close (300/500ms) exceeds the 200ms target and
isn't reduced-motion gated. **Fix:** a polish sweep — reduced-motion block, autofocus +
password toggle, token discipline, one brand mark, strip `dark:`.

### T8 · ADR-005 react-query drift (~12–15 pages)
Hand-rolled `useEffect`+fetch (no caching, no shared retry/error semantics) in Churn,
FinanceReconciliation, OfflinePayments, Integrations, Organizations, AuditLog, Events,
Metering, CancelFlows, TaxNexus, BillingSettings, and portal/*. Converting them delivers
most of T1's error/loading consistency **for free**. **Fix:** migrate to `useQuery`/
`useMutation` per ADR-005.

### T9 · Correctness-of-trust defects (fix regardless of polish)
- **Coupons "Redemptions" column is fabricated** — hardcoded `0` (`Coupons.jsx:45`). A fixed fake number in a finance product; wire it or remove the column.
- **Profile load-error renders *inside* the "Edit profile" button** (`Profile.jsx:58` — a real visible defect).
- Currency assumed-USD on money forms (CreateCoupon amount-off, CreateCreditNote) → 100×/10× wrong for JPY/KWD.
- Dashboard MRR hero hardcoded USD; Subscriptions "Amount" shows *list price* not billed amount (ANTI_PATTERNS ambiguous-label).
- Client-side filtering layered over server pagination (Subscriptions, Plans, Events) → silently filters one page while counts reflect the unfiltered set.
- Pagination/truncation: Invoices fetches 250 with no control (CSV then exports a truncated set), CreditNotes/portal unpaginated.
- MonthEndClose `exportGL()` names the file by period but passes no period/scope.
- GSTReturns renders a statutory filing as a raw `<pre>` JSON dump with no states.

### T10 · Redundant surfaces to consolidate
**Four event surfaces** (Events, Developers "Event logs", Notifications, +) render `GET
/events` three+ ways; RevenueWaterfall ≈ RevenueRecognition (both deferred/recognized/
schedule); RevenueByPlan ≈ RevenueByGeography (literal near-duplicate — extract one
`ShareBreakdown`); legacy `CustomerPortal.jsx` duplicates PortalDashboard; ResetPassword
≈ AcceptInvite; `ConsentCheckbox` injects a whole off-brand cool-gray/`#10b981`
stylesheet. **Fix:** one `EventInspector`, merge the revenue pages, extract shared
components, retire the legacy portal, rebuild ConsentCheckbox on tokens.

### T11 · Destructive / consequential actions are inconsistently guarded
Well-guarded: key/webhook/entity/org/metric/session/SSO deletes, disconnect, write-off,
coupon deactivate. **Un-guarded and should be:** **enabling MCP money-path agent tools**
(highest-stakes toggle in the app — no confirm), Team role elevation / owner transfer,
TaxNexus "Save states" clearing all declared nexus, Metering usage-alert delete,
CreateCreditNote (money-out, single click), Quotes "Convert to invoice" (no confirm, no
in-flight disable → double-click double-creates), business-country change (re-derives tax
regime). **Fix:** `ConfirmDialog` on each; the pattern already exists.

### T12 · Settings has no persistent sub-navigation
Every `settings/*` page is a standalone route reachable only via the hub; lateral movement
is hub-and-back (>3 clicks). `PageHeader` already supports `breadcrumbs` — none use it.
**Fix:** a settings left-rail (Stripe/Vercel/Linear all keep one), or breadcrumbs at minimum.

### T13 · Developer experience: no code samples
Developers.jsx / the API surface show a key but **no `curl`, no auth header, no
webhook-signature-verification snippet**. Stripe/Resend/Plaid put copy-pasteable code
exactly here — it's what makes a developer *enjoy* the surface (DESIGN §1). **Fix:** a
shared `CodeSample` block (one `curl` on the keys tab, one signature-verify snippet on
webhooks).

---

## Sequenced implementation plan

Each wave is a set of green-CI PRs. Order is chosen so the highest-leverage, lowest-risk
foundation lands first and later waves inherit it. **Review-first for each screen-level
redesign: land the shared primitive, then adopt it, then tune.**

- **Wave 0 — Component foundation (inherited by everything).** Fix `ErrorState` default
  title; `prefers-reduced-motion` block + faster gated sheet; `Button` `loading` prop +
  active press; `FormField`/`Input` `aria-invalid`+`aria-describedby` + error visual;
  new primitives `ui/Switch`, `ui/Checkbox`, `SegmentedControl`, `CopyableSecret`,
  `CodeSample`, `Textarea`; `StatCard` `definition` tooltip; `ReportSkeleton`;
  `table.jsx` `scope="col"` + optional sticky header.
- **Wave 1 — Money & trust correctness (T2, T9).** `<Money align="right">` everywhere;
  kill the currency bypasses (Dashboard/AskAnalytics/RevenueRecognition/Checkout/
  Reconciliation); Coupons fabricated column; currency selectors on money forms; Profile
  bug; Subscriptions "List price"; GSTReturns structured summary.
- **Wave 2 — State contract + drill-down (T1, T5, T6).** Route all errors → ErrorState;
  add missing error/empty/loading; link every id/bucket/metric to source; raw UUID → pickers/names.
- **Wave 3 — Forms & controls (T3, T4, T11).** FormField/Input a11y + focus-first-error;
  standardize validation on FormField; adopt Switch/Checkbox/SegmentedControl/Select;
  CustomerSelect adoption; ConsentCheckbox rebuild; ConfirmDialog on money/consequential actions.
- **Wave 4 — Migration & consolidation (T8, T10, T12).** react-query the holdouts; merge
  redundant pages; extract `ShareBreakdown`/`EventInspector`; retire legacy portal; settings sub-nav.
- **Wave 5 — Auth / portal / developer polish (T7, T13).** Autofocus + password toggle;
  one brand mark; token-discipline sweep; portal states; `CodeSample` on Developers.

## Immediate one-liners (ship in Wave 0/1 first)
1. `ErrorState` default title → a specific, honest string (removes a forbidden phrase).
2. `Profile.jsx` — move the load-error block out of the `<Button>`.
3. Coupons — remove or wire the fabricated Redemptions column.
4. MCP settings — `ConfirmDialog` before enabling money-path agent tools.
5. `index.css` — add the `prefers-reduced-motion` block.

---

*Source: six parallel code-grounded cluster reviews (commerce lists, forms, finance,
analytics, auth/portal/components, ops/settings), 2026-08. Exhaustive per-screen issue
lists live in the review transcripts; this document is the actionable synthesis and the
score of record.*
