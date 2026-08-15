import { Money } from "@/components/ui/money";
import { Overline } from "@/components/ui/overline";
import { Skeleton } from "@/components/patterns/LoadingSkeleton";
import { MotionStagger } from "@/components/patterns/MotionReveal";
import { MotionState } from "@/components/patterns/MotionState";
import { humanizeFailure } from "@/lib/failureLabels";
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
      <Overline as="dt">{label}</Overline>
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
      <MotionStagger step={50}>
      {attempts.map((a) => (
        <li key={a.id} className="rounded-lg border border-border p-3">
          <div className="flex items-center justify-between gap-3">
            {/* The attempt's status can advance while you watch (initiated →
                processing → succeeded / failed). Flash the pill on the
                transition so a settlement reads as an event. */}
            <MotionState motionKey={a.status}>
              <span
                className={cn(
                  "rounded-full border px-2.5 py-0.5 font-mono text-[11px] font-medium",
                  STATUS_TONE[a.status] || STATUS_TONE.initiated,
                )}
              >
                {a.status}
              </span>
            </MotionState>
            <span className="font-mono text-sm tabular-nums text-foreground">
              <Money amountMinor={a.amount} currency={currency} />
            </span>
          </div>
          <dl className="mt-2.5 grid grid-cols-2 gap-x-6 gap-y-2 text-xs sm:grid-cols-4">
            <Field label="Method">{a.method || "—"}</Field>
            <Field label="Gateway">{a.gateway || "—"}</Field>
            {a.failure_code ? (
              <Field label="Failure">
                {/* Lead with the human-readable reason; the raw gateway code is
                    technical detail, shown quietly beneath (never the primary
                    operator-facing explanation). Reveals in rather than blinking. */}
                <span className="inline-block animate-motion-reveal text-destructive">
                  {humanizeFailure(a.failure_code)}
                </span>
                <span
                  className="mt-0.5 block font-mono text-[10px] normal-case text-subtle"
                  title={`Gateway failure code: ${a.failure_code}`}
                >
                  {a.failure_code}
                </span>
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
      </MotionStagger>
    </ol>
  );
}
