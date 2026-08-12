import { Link } from "react-router";
import { ArrowLeft } from "lucide-react";

import { cn } from "@/lib/utils";
import { Card } from "@/components/ui/card";

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
 * ObjectHeader — identity header for an object page.
 *
 * Props:
 *  - backTo / backLabel: list route this object belongs to
 *  - kicker: object type label ("Customer")
 *  - title:  the object's human identity (name, invoice number)
 *  - badge:  ReactNode, the object's ONE status (a <StatusBadge>)
 *  - meta:   ReactNode under the title (ids, dates — quiet, small)
 *  - actions: ReactNode, right-aligned (primary + contextual actions)
 */
export function ObjectHeader({
  backTo,
  backLabel,
  kicker,
  title,
  badge,
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
          {kicker && (
            <div className="text-xs font-medium uppercase tracking-wide text-subtle">
              {kicker}
            </div>
          )}
          <div className="mt-0.5 flex flex-wrap items-center gap-x-3 gap-y-1">
            <h1 className="truncate text-2xl font-semibold tracking-tight text-foreground">
              {title}
            </h1>
            {badge}
          </div>
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
          <dt className="text-xs font-medium uppercase tracking-wide text-subtle">
            {label}
          </dt>
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
