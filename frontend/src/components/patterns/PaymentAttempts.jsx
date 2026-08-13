import { Money } from "@/components/ui/money";
import { Skeleton } from "@/components/patterns/LoadingSkeleton";
import { formatDateTime, cn } from "@/lib/utils";

/**
 * PaymentAttempts — an invoice's settlement/retry history: each attempt with
 * its lifecycle status, failure reason, gateway reference, and timestamps. The
 * "what happened when we tried to collect" layer of the invoice page.
 *
 * A single attempt carries ONE current status (cards jump to succeeded/failed;
 * ACH moves initiated → processing → succeeded, and can go → returned). Multiple
 * attempts are the retry history — the list is chronological.
 *
 * Props:
 *  - attempts: PaymentAttempt[] ({ id, status, method, gateway, failure_code,
 *    amount, gateway_payment_intent_id, created_at, settled_at })
 *  - currency: the invoice's currency
 *  - isLoading / error: request states
 */
const STATUS_TONE = {
  succeeded: "border-success/30 bg-success/5 text-success",
  processing: "border-warning/30 bg-warning/5 text-warning",
  initiated: "border-border bg-muted text-muted-foreground",
  failed: "border-destructive/30 bg-destructive/5 text-destructive",
  returned: "border-destructive/30 bg-destructive/5 text-destructive",
};

function Field({ label, children }) {
  return (
    <div className="min-w-0">
      <dt className="text-[10px] font-medium uppercase tracking-wide text-subtle">{label}</dt>
      <dd className="mt-0.5 truncate text-foreground">{children}</dd>
    </div>
  );
}

export function PaymentAttempts({ attempts, currency = "USD", isLoading, error }) {
  if (isLoading) {
    return (
      <div className="space-y-3" aria-busy="true">
        <Skeleton className="h-14 w-full" />
        <Skeleton className="h-14 w-full" />
      </div>
    );
  }
  if (error) {
    return <p className="text-sm text-muted-foreground" role="status">Couldn’t load payment attempts.</p>;
  }
  if (!attempts || attempts.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        No gateway payment attempts — this invoice was paid from credit or offline, or hasn’t been attempted yet.
      </p>
    );
  }

  return (
    <ol className="space-y-3">
      {attempts.map((a) => (
        <li key={a.id} className="rounded-lg border border-border p-3">
          <div className="flex items-center justify-between gap-3">
            <span
              className={cn(
                "rounded-full border px-2.5 py-0.5 font-mono text-[11px] font-medium",
                STATUS_TONE[a.status] || STATUS_TONE.initiated,
              )}
            >
              {a.status}
            </span>
            <span className="font-mono text-sm tabular-nums text-foreground">
              <Money amountMinor={a.amount} currency={currency} />
            </span>
          </div>
          <dl className="mt-2.5 grid grid-cols-2 gap-x-6 gap-y-2 text-xs sm:grid-cols-4">
            <Field label="Method">{a.method || "—"}</Field>
            <Field label="Gateway">{a.gateway || "—"}</Field>
            {a.failure_code ? (
              <Field label="Failure">
                <span className="text-destructive">{a.failure_code}</span>
              </Field>
            ) : null}
            <Field label="Started">{formatDateTime(a.created_at)}</Field>
            {a.settled_at ? <Field label="Settled">{formatDateTime(a.settled_at)}</Field> : null}
          </dl>
          {a.gateway_payment_intent_id ? (
            <p className="mt-2 truncate font-mono text-[11px] text-muted-foreground" title={a.gateway_payment_intent_id}>
              ref {a.gateway_payment_intent_id}
            </p>
          ) : null}
        </li>
      ))}
    </ol>
  );
}
