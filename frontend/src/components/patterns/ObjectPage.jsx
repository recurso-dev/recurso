import { Link } from "react-router";
import { ArrowLeft } from "lucide-react";

import { cn } from "@/lib/utils";
import { errorMessage } from "@/lib/httpError";
import { Card } from "@/components/ui/card";
import { Overline } from "@/components/ui/overline";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/patterns/LoadingSkeleton";
import { ErrorState } from "@/components/patterns/ErrorState";

/**
 * Object-page system (DASHBOARD_REDESIGN.md Phase 5).
 *
 * An object page is the canonical read view for one business object
 * (customer, subscription, invoice, …) under its addressable /x/:id route.
 * These primitives compose rather than prescribe — an object page picks the
 * sections it needs (attributes, related objects, financials, audit trail)
 * instead of being forced into one rigid layout.
 *
 *   <ObjectHeader backTo="/customers" backLabel="Customers"
 *                 kicker="Customer" title={name} badge={<StatusBadge …/>}
 *                 meta={<CopyableId …/>} actions={<Button …/>} />
 *   <ObjectPageLayout rail={<…metadata, audit…>}>
 *     <ObjectSection title="Overview"><AttributeList items={…}/></ObjectSection>
 *     <ObjectSection title="Subscriptions" action={<Link…>}>…</ObjectSection>
 *   </ObjectPageLayout>
 */

/**
 * ObjectHeader — identity header for an object page. Establishes the canonical
 * hierarchy an operator scans top-down: object identity → status → the ONE
 * primary financial fact → secondary metadata → actions. The `amount` slot is
 * the object hero: a single dominant amount (a `<Money size="hero">`), rendered
 * directly under the title so it reads authoritatively without a KPI-style card.
 * Objects with no meaningful primary amount (e.g. a customer) simply omit it.
 *
 * Props:
 *  - backTo / backLabel: list route this object belongs to
 *  - kicker: object type label ("Customer")
 *  - title:  the object's human identity (name, invoice number)
 *  - badge:  ReactNode, the object's ONE status (a <StatusBadge>)
 *  - amount: ReactNode, the hero amount (a <Money size="hero">) — optional
 *  - amountLabel: ReactNode, a small muted label/context beside the amount
 *    ("amount due", "MRR", the payment outcome) — optional
 *  - meta:   ReactNode under the amount (ids, dates — quiet, secondary)
 *  - actions: ReactNode, right-aligned (primary + contextual actions)
 */
export function ObjectHeader({
  backTo,
  backLabel,
  kicker,
  title,
  badge,
  amount,
  amountLabel,
  meta,
  actions,
  className,
}) {
  return (
    <div className={cn("mb-6", className)}>
      {backTo && (
        <Link
          to={backTo}
          className="mb-3 inline-flex items-center gap-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
        >
          <ArrowLeft className="h-3.5 w-3.5" aria-hidden="true" />
          {backLabel || "Back"}
        </Link>
      )}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0">
          {kicker && <Overline>{kicker}</Overline>}
          <div className="mt-0.5 flex flex-wrap items-center gap-x-3 gap-y-1">
            <h1 className="truncate text-2xl font-semibold tracking-tight text-foreground">
              {title}
            </h1>
            {badge}
          </div>
          {amount && (
            <div className="mt-2 flex flex-wrap items-baseline gap-x-2.5 gap-y-1">
              {amount}
              {amountLabel && (
                <span className="text-sm text-muted-foreground">{amountLabel}</span>
              )}
            </div>
          )}
          {meta && (
            <div className="mt-1.5 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-muted-foreground">
              {meta}
            </div>
          )}
        </div>
        {actions && <div className="flex shrink-0 items-center gap-2">{actions}</div>}
      </div>
    </div>
  );
}

/**
 * ObjectPageLayout — main column + metadata rail. The rail stacks below the
 * main column on small screens.
 */
export function ObjectPageLayout({ children, rail, className }) {
  return (
    <div className={cn("grid grid-cols-1 gap-6 lg:grid-cols-3", className)}>
      <div className="min-w-0 space-y-6 lg:col-span-2">{children}</div>
      {rail && <div className="min-w-0 space-y-6">{rail}</div>}
    </div>
  );
}

/**
 * ObjectSection — one titled section of an object page.
 *
 * Props:
 *  - title:  string (section heading)
 *  - action: ReactNode, right of the title (e.g. a "View all" link)
 *  - flush:  render children edge-to-edge (for tables); default padded
 */
export function ObjectSection({ title, action, flush = false, children, className }) {
  return (
    <Card className={cn("overflow-hidden", className)}>
      <div className="flex items-center justify-between gap-3 border-b border-border px-6 py-4">
        <h2 className="text-sm font-semibold text-foreground">{title}</h2>
        {action}
      </div>
      <div className={flush ? "" : "px-6 py-4"}>{children}</div>
    </Card>
  );
}

/**
 * AttributeList — the summary-attributes grid (label over value).
 *
 * Props:
 *  - items: [{ label, value }] — value is any ReactNode; null/undefined/""
 *    renders an em dash so absent data reads as absent, not broken.
 *  - columns: 1 | 2 | 3 (grid columns at ≥sm; always 1 below)
 */
export function AttributeList({ items, columns = 2, className }) {
  const colClass = { 1: "", 2: "sm:grid-cols-2", 3: "sm:grid-cols-3" }[columns];
  return (
    <dl className={cn("grid grid-cols-1 gap-x-8 gap-y-4", colClass, className)}>
      {items.map(({ label, value }) => (
        <div key={label} className="min-w-0">
          <Overline as="dt">{label}</Overline>
          <dd className="mt-1 break-words text-sm text-foreground">
            {value === null || value === undefined || value === "" ? (
              <span className="text-muted-foreground">—</span>
            ) : (
              value
            )}
          </dd>
        </div>
      ))}
    </dl>
  );
}

/**
 * RelatedRow — one row of a related-objects section (use inside an
 * ObjectSection with flush): the whole row is a real link to the related
 * object's route.
 */
export function RelatedRow({ to, children }) {
  return (
    <Link
      to={to}
      className="flex items-center justify-between gap-3 px-6 py-3 text-sm transition-colors hover:bg-muted/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring"
    >
      {children}
    </Link>
  );
}

/** RelatedEmpty — the quiet empty state for a related-objects section. */
export function RelatedEmpty({ children }) {
  return <p className="px-6 py-4 text-sm text-muted-foreground">{children}</p>;
}

/* ------------------------------------------------------------------------- *
 * Canonical object-page lifecycle states (Polish Batch B). Every object page
 * renders exactly these three, paired with useObjectQuery, so loading /
 * not-found / error speak one language across the app.
 * ------------------------------------------------------------------------- */

/**
 * ObjectPageSkeleton — the canonical loading state. Mirrors the object-page
 * geometry (header identity + hero, main column, optional rail) so data landing
 * doesn't jump the layout. Announced politely, once. `hasRail` matches whether
 * the page uses a metadata rail.
 */
export function ObjectPageSkeleton({ hasRail = true, className }) {
  return (
    <div role="status" aria-busy="true" className={className}>
      <span className="sr-only">Loading…</span>
      <div className="mb-6" aria-hidden="true">
        <Skeleton className="mb-2 h-4 w-24" />
        <Skeleton className="mb-2.5 h-8 w-64" />
        <Skeleton className="h-7 w-40" />
      </div>
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3" aria-hidden="true">
        <div className="space-y-6 lg:col-span-2">
          <Skeleton className="h-48 w-full" />
          <Skeleton className="h-32 w-full" />
        </div>
        {hasRail && <Skeleton className="h-56 w-full" />}
      </div>
    </div>
  );
}

// StateBackLink — the contextual back navigation shown in the not-found/error
// states (e.g. "Back to Invoices"). Renders nothing without a target.
function StateBackLink({ to, label }) {
  if (!to) return null;
  return (
    <Button asChild variant="ghost" size="sm">
      <Link to={to}>
        <ArrowLeft className="h-3.5 w-3.5" aria-hidden="true" />
        {label || "Back"}
      </Link>
    </Button>
  );
}

/**
 * ObjectNotFound — the canonical "this object doesn't exist / no access" state.
 * Retry is intentionally omitted (retrying a not-found can't help). Never leaks
 * tenant/security detail: an object in another workspace reads the same as a
 * deleted one. `objectLabel` is the lowercase noun ("invoice", "payment").
 */
export function ObjectNotFound({ objectLabel = "object", identifier, backTo, backLabel }) {
  const Label = objectLabel.charAt(0).toUpperCase() + objectLabel.slice(1);
  return (
    <ErrorState
      title={`${Label} not found`}
      message={`This ${objectLabel}${identifier ? ` (${identifier})` : ""} doesn’t exist, or you may not have access to it.`}
      action={<StateBackLink to={backTo} label={backLabel} />}
    />
  );
}

/**
 * ObjectPageError — the canonical retryable error state. Distinct from
 * not-found: the request genuinely failed. The message comes from the safe
 * errorMessage() helper (never a raw backend error / stack / gateway code).
 */
export function ObjectPageError({ objectLabel = "object", error, onRetry, backTo, backLabel }) {
  return (
    <ErrorState
      title={`Couldn’t load this ${objectLabel}`}
      message={errorMessage(error)}
      onRetry={onRetry}
      action={<StateBackLink to={backTo} label={backLabel} />}
    />
  );
}
