import { Check } from "lucide-react";

import { Money } from "@/components/ui/money";
import { Skeleton } from "@/components/patterns/LoadingSkeleton";
import { MotionReveal, MotionStagger } from "@/components/patterns/MotionReveal";
import { LedgerAccountLink } from "@/components/patterns/LedgerAccountLink";
import { formatDateTime } from "@/lib/utils";

/**
 * JournalEntries — the finance-accounting drill for an object: the ledger
 * postings that reference it, each as a balanced transfer (one debit, one
 * credit). The counterpart to the customer-facing document. Shared across the
 * Invoice, Payment, and Ledger object pages.
 *
 * Each posting is the product's real transfer-based entry — Code 1 issuance,
 * Code 6 tax reclass, Code 3 payment, credit/refund/write-off legs. In a
 * transfer model every row is inherently balanced, so the set ties to zero.
 *
 * Props:
 *  - entries: GeneralLedgerRow[] ({ code, description, debit_account_code/name,
 *    credit_account_code/name, amount, timestamp })
 *  - currency: the object's currency (postings are in it)
 *  - isLoading / error: request states
 *  - emptyMessage: what to say when there are no postings. Defaults to the
 *    invoice wording; other object pages (credit notes, …) pass their own so
 *    the empty state never talks about "a draft invoice" on the wrong object.
 */
export function JournalEntries({
  entries,
  currency = "USD",
  isLoading,
  error,
  emptyMessage = "No postings yet — a draft invoice hasn’t hit the ledger. Finalizing it posts the first entry.",
}) {
  if (isLoading) {
    return (
      <div className="space-y-3" aria-busy="true">
        <Skeleton className="h-4 w-1/2" />
        <Skeleton className="h-4 w-2/3" />
        <Skeleton className="h-4 w-1/3" />
      </div>
    );
  }
  if (error) {
    return <p className="text-sm text-muted-foreground" role="status">Couldn’t load the journal entries.</p>;
  }
  if (!entries || entries.length === 0) {
    return <p className="text-sm text-muted-foreground">{emptyMessage}</p>;
  }

  const total = entries.reduce((sum, e) => sum + (e.amount || 0), 0);
  // Postings reveal in sequence, then the balance line settles just after the
  // last one — you watch the entry post and tie to zero. Capped so a long set
  // never turns into a slow crawl.
  const step = 55;
  const footerDelay = Math.min(entries.length, 6) * step + 40;

  return (
    <div>
      <ol className="space-y-4">
        <MotionStagger step={step}>
        {entries.map((e, i) => (
          <li key={e.transaction_id || i}>
            <div className="mb-1 flex items-baseline justify-between gap-3">
              <span className="text-xs font-medium text-foreground">
                <span className="font-mono text-muted-foreground">Code {e.code}</span>
                {e.description ? ` · ${e.description}` : ""}
              </span>
              {e.timestamp && (
                <span className="shrink-0 text-xs text-muted-foreground">{formatDateTime(e.timestamp)}</span>
              )}
            </div>
            <div className="grid grid-cols-[1.5rem_1fr_auto] items-center gap-x-3 gap-y-1 font-mono text-[13px] tabular-nums">
              <span className="text-muted-foreground">DR</span>
              <LedgerAccountLink
                id={e.debit_account_id}
                className="min-w-0 truncate text-foreground"
                label={
                  <>
                    <span className="text-muted-foreground">{e.debit_account_code}</span> {e.debit_account_name}
                  </>
                }
              />
              <span className="text-foreground">
                <Money amountMinor={e.amount} currency={currency} />
              </span>
              <span className="text-muted-foreground">CR</span>
              <LedgerAccountLink
                id={e.credit_account_id}
                className="min-w-0 truncate text-foreground"
                label={
                  <>
                    <span className="text-muted-foreground">{e.credit_account_code}</span> {e.credit_account_name}
                  </>
                }
              />
              <span className="text-foreground">
                <Money amountMinor={e.amount} currency={currency} />
              </span>
            </div>
          </li>
        ))}
        </MotionStagger>
      </ol>
      <MotionReveal
        as="div"
        delay={footerDelay}
        className="mt-4 flex items-baseline justify-between border-t border-border pt-3 text-sm font-medium"
      >
        <span className="flex items-center gap-1.5 text-success">
          <Check className="h-3.5 w-3.5" aria-hidden="true" />
          Debits = Credits
        </span>
        <span className="tabular-nums text-foreground">
          <Money amountMinor={total} currency={currency} />
        </span>
      </MotionReveal>
    </div>
  );
}
