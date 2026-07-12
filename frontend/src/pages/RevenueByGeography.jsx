import { useCallback, useEffect, useState } from "react";
import { Globe } from "lucide-react";

import { endpoints } from "../lib/api";
import { formatCurrency } from "@/lib/utils";
import { PageHeader } from "@/components/patterns/PageHeader";
import { StatCard } from "@/components/patterns/StatCard";
import { EmptyState } from "@/components/patterns/EmptyState";
import { ErrorState } from "@/components/patterns/ErrorState";
import { CardGridSkeleton } from "@/components/patterns/LoadingSkeleton";
import { Card } from "@/components/ui/card";

const BAR_COLORS = ["bg-emerald-500", "bg-emerald-600", "bg-teal-500", "bg-cyan-600", "bg-sky-600", "bg-indigo-500"];

export default function RevenueByGeography() {
  const [report, setReport] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await endpoints.getRevenueByGeography();
      setReport(res.data?.data || null);
    } catch (err) {
      setError(err?.response?.data?.error?.message || "Failed to load revenue by geography");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const cur = report?.reporting_currency || "USD";
  const money = (n) => formatCurrency(n || 0, cur);
  const segments = report?.segments || [];
  const total = report?.total_mrr || 0;

  return (
    <div>
      <PageHeader
        title="Revenue by Geography"
        description="Where your recurring revenue comes from, by customer country."
      />

      {loading ? (
        <CardGridSkeleton count={3} />
      ) : error ? (
        <Card className="overflow-hidden">
          <ErrorState message={error} onRetry={load} />
        </Card>
      ) : (
        report && (
          <div className="flex flex-col gap-6">
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
              <StatCard label="Total MRR" value={money(total)} hint="Across all regions" />
              <StatCard label="Countries" value={segments.length.toLocaleString()} hint="With revenue" />
              <StatCard
                label="Top region"
                value={segments[0]?.label || "—"}
                hint={segments[0] ? `${segments[0].share_pct.toFixed(0)}% of MRR` : ""}
              />
            </div>

            {segments.length === 0 ? (
              <Card className="overflow-hidden">
                <EmptyState
                  icon={Globe}
                  title="No revenue yet"
                  description="Once you have active subscriptions, the revenue mix by country appears here."
                />
              </Card>
            ) : (
              <Card className="p-6">
                <div className="flex flex-col gap-4">
                  {segments.map((s, i) => (
                    <div key={s.key || s.label} className="flex items-center gap-3">
                      <span className="w-28 shrink-0 truncate text-sm font-medium text-foreground" title={s.label}>
                        {s.label}
                      </span>
                      <div className="relative h-6 flex-1 overflow-hidden rounded bg-muted/40">
                        <div
                          className={`h-6 rounded ${BAR_COLORS[i % BAR_COLORS.length]}`}
                          style={{ width: `${Math.max(s.share_pct, 1.5)}%` }}
                          title={`${s.label}: ${money(s.mrr)} (${s.share_pct.toFixed(1)}%)`}
                        />
                      </div>
                      <span className="w-14 shrink-0 text-right text-xs tabular-nums text-muted-foreground">
                        {s.share_pct.toFixed(1)}%
                      </span>
                      <span className="w-28 shrink-0 text-right font-mono text-sm tabular-nums text-foreground">
                        {money(s.mrr)}
                      </span>
                      <span className="w-16 shrink-0 text-right text-xs text-muted-foreground">
                        {s.subscriptions} sub{s.subscriptions === 1 ? "" : "s"}
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
