import { useQuery } from "@tanstack/react-query";
import { FileClock } from "lucide-react";

import { endpoints } from "../lib/api";
import { formatCurrency } from "@/lib/utils";
import { PageHeader } from "@/components/patterns/PageHeader";
import { StatCard } from "@/components/patterns/StatCard";
import { EmptyState } from "@/components/patterns/EmptyState";
import { ErrorState } from "@/components/patterns/ErrorState";
import { CardGridSkeleton } from "@/components/patterns/LoadingSkeleton";
import { Card } from "@/components/ui/card";

const BUCKET_LABELS = {
  current: "Current",
  "1-30": "1–30 days",
  "31-60": "31–60 days",
  "61-90": "61–90 days",
  "90+": "90+ days",
};

// Severity ramp: current is healthy, older is worse.
const BUCKET_COLOR = {
  current: "bg-emerald-500",
  "1-30": "bg-amber-400",
  "31-60": "bg-orange-500",
  "61-90": "bg-red-500",
  "90+": "bg-rose-700",
};

export default function InvoiceAging() {
  const {
    data: report,
    isLoading: loading,
    error: queryError,
    refetch,
  } = useQuery({
    queryKey: ["invoice-aging"],
    queryFn: async () => (await endpoints.getInvoiceAging()).data?.data || null,
  });
  const error = queryError
    ? queryError?.response?.data?.error?.message || "Failed to load invoice aging"
    : null;

  const cur = report?.reporting_currency || "USD";
  const money = (n) => formatCurrency(n || 0, cur);
  const buckets = report?.buckets || [];
  const total = report?.total_outstanding || 0;
  const count = report?.total_count || 0;
  const current = buckets.find((b) => b.label === "current")?.amount || 0;
  const overdue = total - current;
  const maxAmt = Math.max(1, ...buckets.map((b) => b.amount || 0));
  const hasData = total > 0 || count > 0;

  return (
    <div>
      <PageHeader
        title="Invoice Aging"
        description="Outstanding receivables by how far past due each open invoice is."
      />

      {loading ? (
        <CardGridSkeleton count={3} />
      ) : error ? (
        <Card className="overflow-hidden">
          <ErrorState message={error} onRetry={refetch} />
        </Card>
      ) : (
        report && (
          <div className="flex flex-col gap-6">
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
              <StatCard label="Total outstanding" value={money(total)} hint={`${count} open invoice${count === 1 ? "" : "s"}`} />
              <StatCard label="Overdue" value={money(overdue)} hint="Past their due date" />
              <StatCard label="Current" value={money(current)} hint="Not yet due" />
            </div>

            {!hasData ? (
              <Card className="overflow-hidden">
                <EmptyState
                  icon={FileClock}
                  title="Nothing outstanding"
                  description="No open invoices — everything issued is paid or not yet billed."
                />
              </Card>
            ) : (
              <Card className="p-6">
                <h2 className="mb-4 text-base font-semibold text-foreground">By age</h2>
                <div className="flex flex-col gap-3">
                  {buckets.map((b) => (
                    <div key={b.label} className="flex items-center gap-3">
                      <span className="w-24 shrink-0 text-sm text-muted-foreground">
                        {BUCKET_LABELS[b.label] || b.label}
                      </span>
                      <div className="relative h-6 flex-1 overflow-hidden rounded bg-muted/40">
                        <div
                          className={`h-6 rounded ${BUCKET_COLOR[b.label] || "bg-foreground/70"}`}
                          style={{ width: `${((b.amount || 0) / maxAmt) * 100}%` }}
                          title={`${BUCKET_LABELS[b.label] || b.label}: ${money(b.amount)}`}
                        />
                      </div>
                      <span className="w-28 shrink-0 text-right font-mono text-sm tabular-nums text-foreground">
                        {money(b.amount)}
                      </span>
                      <span className="w-16 shrink-0 text-right text-xs text-muted-foreground">
                        {b.count} inv
                      </span>
                    </div>
                  ))}
                </div>
              </Card>
            )}
          </div>
        )
      )}
    </div>
  );
}
