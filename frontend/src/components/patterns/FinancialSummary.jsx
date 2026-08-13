import { Money } from "@/components/ui/money";
import { Skeleton } from "@/components/patterns/LoadingSkeleton";
import { cn } from "@/lib/utils";

/**
 * FinancialSummary — an object's invoice-derived financial position, the number
 * strip an object page leads with (DASHBOARD_OPERATIONAL_DEPTH.md, layer 6).
 *
 * Money is never summed across currencies, so this renders one metric block per
 * currency. Outstanding carries a danger tone when non-zero; past-due carries a
 * warning tone with its invoice count. Not a card grid — a quiet, dense strip.
 *
 * Props:
 *  - currencies: [{ currency, outstanding, past_due, past_due_count, billed, paid }]
 *  - isLoading / error: request states
 */
export function FinancialSummary({ currencies, isLoading, error }) {
  if (isLoading) {
    return (
      <div className="grid grid-cols-2 gap-x-8 gap-y-4 sm:grid-cols-4" aria-busy="true">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i}>
            <Skeleton className="h-3 w-16" />
            <Skeleton className="mt-2 h-5 w-20" />
          </div>
        ))}
      </div>
    );
  }
  if (error) {
    return <p className="text-sm text-muted-foreground" role="status">Couldn’t load the financial summary.</p>;
  }
  if (!currencies || currencies.length === 0) {
    return <p className="text-sm text-muted-foreground">No invoices yet — nothing billed.</p>;
  }

  return (
    <div className="space-y-6">
      {currencies.map((c) => (
        <CurrencyBlock key={c.currency} c={c} showCurrency={currencies.length > 1} />
      ))}
    </div>
  );
}

function Metric({ label, children, tone }) {
  const toneClass =
    tone === "danger" ? "text-destructive" : tone === "warning" ? "text-warning" : "text-foreground";
  return (
    <div className="min-w-0">
      <dt className="text-xs font-medium uppercase tracking-wide text-subtle">{label}</dt>
      <dd className={cn("mt-1 text-lg font-semibold tabular-nums", toneClass)}>{children}</dd>
    </div>
  );
}

function CurrencyBlock({ c, showCurrency }) {
  const owes = (c.outstanding ?? 0) > 0;
  const pastDue = (c.past_due ?? 0) > 0;
  return (
    <div>
      {showCurrency && (
        <div className="mb-3 text-xs font-medium uppercase tracking-wide text-muted-foreground">
          {c.currency}
        </div>
      )}
      <dl className="grid grid-cols-2 gap-x-8 gap-y-4 sm:grid-cols-4">
        <Metric label="Outstanding" tone={owes ? "danger" : undefined}>
          <Money amountMinor={c.outstanding ?? 0} currency={c.currency} />
        </Metric>
        <Metric label="Past due" tone={pastDue ? "warning" : undefined}>
          <span className="inline-flex items-baseline gap-1.5">
            <Money amountMinor={c.past_due ?? 0} currency={c.currency} />
            {c.past_due_count > 0 && (
              <span className="text-xs font-normal text-muted-foreground">
                {c.past_due_count} {c.past_due_count === 1 ? "invoice" : "invoices"}
              </span>
            )}
          </span>
        </Metric>
        <Metric label="Billed lifetime">
          <Money amountMinor={c.billed ?? 0} currency={c.currency} />
        </Metric>
        <Metric label="Paid lifetime">
          <Money amountMinor={c.paid ?? 0} currency={c.currency} />
        </Metric>
      </dl>
    </div>
  );
}
