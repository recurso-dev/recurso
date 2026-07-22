import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Info, TrendingUp } from "lucide-react";

import { endpoints } from "../lib/api";
import { formatCurrency } from "@/lib/utils";
import { PageHeader } from "@/components/patterns/PageHeader";
import { StatCard } from "@/components/patterns/StatCard";
import { EmptyState } from "@/components/patterns/EmptyState";
import { ErrorState } from "@/components/patterns/ErrorState";
import { CardGridSkeleton } from "@/components/patterns/LoadingSkeleton";
import { Card } from "@/components/ui/card";

const iso = (d) => d.toISOString().slice(0, 10);

const dateInputClass =
  "rounded-md border border-border bg-background px-2.5 py-1.5 text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring";

// buildSteps turns the API totals into positioned bars: totals span 0→value;
// movement steps float between the running total before and after them.
function buildSteps(wf) {
  const items = [
    { label: "Starting", value: wf.starting_mrr || 0, kind: "total" },
    { label: "New", value: wf.new || 0, kind: "up" },
    { label: "Expansion", value: wf.expansion || 0, kind: "up" },
    { label: "Reactivation", value: wf.reactivation || 0, kind: "up" },
    { label: "Contraction", value: -(wf.contraction || 0), kind: "down" },
    { label: "Churned", value: -(wf.churned || 0), kind: "down" },
    { label: "Ending", value: wf.ending_mrr || 0, kind: "total" },
  ];
  let running = 0;
  return items.map((it) => {
    if (it.kind === "total") {
      running = it.value;
      return { ...it, base: 0, top: it.value };
    }
    const prev = running;
    running = prev + it.value;
    return { ...it, base: Math.min(prev, running), top: Math.max(prev, running) };
  });
}

const barColor = { total: "bg-foreground/70", up: "bg-emerald-500", down: "bg-rose-500" };

export default function MRRWaterfall() {
  const now = new Date();
  const monthAgo = new Date(now);
  monthAgo.setMonth(monthAgo.getMonth() - 1);

  const [start, setStart] = useState(iso(monthAgo));
  const [end, setEnd] = useState(iso(now));

  const {
    data: wf,
    isLoading: loading,
    error: queryError,
    refetch,
  } = useQuery({
    queryKey: ["mrr-waterfall", start, end],
    queryFn: async () => (await endpoints.getMRRWaterfall(start, end)).data?.data || null,
  });
  const error = queryError
    ? queryError?.response?.data?.error?.message || "Failed to load the MRR waterfall"
    : null;

  const cur = wf?.reporting_currency || "USD";
  const money = (n) => formatCurrency(n || 0, cur);

  const steps = wf ? buildSteps(wf) : [];
  const maxVal = steps.length ? Math.max(1, ...steps.map((s) => s.top)) : 1;
  const netChange = wf ? (wf.ending_mrr || 0) - (wf.starting_mrr || 0) : 0;
  const hasData = wf && (wf.ending_mrr || wf.starting_mrr || steps.some((s) => s.kind !== "total" && s.value !== 0));

  return (
    <div>
      <PageHeader
        title="MRR Waterfall"
        description="How recurring revenue moved over the period — new, expansion, contraction, and churn."
        actions={
          <div className="flex items-center gap-2">
            <label className="sr-only" htmlFor="wf-start">Start date</label>
            <input id="wf-start" type="date" className={dateInputClass} value={start} max={end} onChange={(e) => setStart(e.target.value)} />
            <span className="text-sm text-muted-foreground">→</span>
            <label className="sr-only" htmlFor="wf-end">End date</label>
            <input id="wf-end" type="date" className={dateInputClass} value={end} min={start} onChange={(e) => setEnd(e.target.value)} />
          </div>
        }
      />

      {loading ? (
        <CardGridSkeleton count={3} />
      ) : error ? (
        <Card className="overflow-hidden">
          <ErrorState message={error} onRetry={refetch} />
        </Card>
      ) : (
        wf && (
          <div className="flex flex-col gap-6">
            {!wf.has_start_history && (
              <div className="flex items-start gap-3 rounded-lg bg-amber-50 p-4 text-amber-800 ring-1 ring-inset ring-amber-200">
                <Info className="mt-0.5 h-5 w-5 flex-shrink-0" />
                <p className="text-sm">
                  MRR history doesn&rsquo;t reach back to the start of this range yet — snapshots began more recently.
                  Everything present at the end is shown as <b>New</b>. Movement gets more accurate each day as history accrues.
                </p>
              </div>
            )}

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
              <StatCard label="Starting MRR" value={money(wf.starting_mrr)} hint={`as of ${wf.start_date ? wf.start_date.slice(0, 10) : start}`} />
              <StatCard
                label="Net change"
                value={`${netChange >= 0 ? "+" : "−"}${money(Math.abs(netChange))}`}
                hint={wf.starting_mrr ? `${((netChange / wf.starting_mrr) * 100).toFixed(1)}% vs. start` : "—"}
              />
              <StatCard label="Ending MRR" value={money(wf.ending_mrr)} hint={`as of ${wf.end_date ? wf.end_date.slice(0, 10) : end}`} />
            </div>

            {wf.starting_mrr > 0 && (
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <StatCard
                  label="Net Dollar Retention"
                  value={`${(wf.net_dollar_retention || 0).toFixed(1)}%`}
                  hint="Revenue kept from existing customers, expansion included"
                />
                <StatCard
                  label="Gross Dollar Retention"
                  value={`${(wf.gross_dollar_retention || 0).toFixed(1)}%`}
                  hint="Revenue kept before expansion — churn &amp; contraction only"
                />
              </div>
            )}

            {!hasData ? (
              <Card className="overflow-hidden">
                <EmptyState
                  icon={TrendingUp}
                  title="No MRR movement yet"
                  description="Once you have active subscriptions and a few days of captured history, the waterfall fills in here."
                />
              </Card>
            ) : (
              <>
                {/* Floating-bar waterfall */}
                <Card className="p-6">
                  <div className="flex items-end gap-2 sm:gap-4" style={{ height: 260 }} role="img" aria-label="MRR waterfall chart">
                    {steps.map((s) => (
                      <div key={s.label} className="flex h-full flex-1 flex-col items-center justify-end">
                        <div className="relative w-full flex-1">
                          <div
                            className={`absolute inset-x-0 rounded-sm ${barColor[s.kind]}`}
                            style={{
                              bottom: `${(s.base / maxVal) * 100}%`,
                              height: `${Math.max(((s.top - s.base) / maxVal) * 100, 0.6)}%`,
                            }}
                            title={`${s.label}: ${s.kind === "total" ? "" : s.value >= 0 ? "+" : "−"}${money(Math.abs(s.value))}`}
                          />
                          <span
                            className="absolute inset-x-0 -translate-y-full pb-1 text-center font-mono text-[11px] tabular-nums text-foreground"
                            style={{ bottom: `${(s.top / maxVal) * 100}%` }}
                          >
                            {s.kind === "total" ? "" : s.value >= 0 ? "+" : "−"}
                            {money(Math.abs(s.value))}
                          </span>
                        </div>
                        <span className="mt-2 text-center text-xs text-muted-foreground">{s.label}</span>
                      </div>
                    ))}
                  </div>
                </Card>

                {/* Exact breakdown */}
                <Card className="p-6">
                  <h2 className="mb-4 text-base font-semibold text-foreground">Breakdown</h2>
                  <dl className="flex flex-col gap-2">
                    <Row label="Starting MRR" value={money(wf.starting_mrr)} />
                    <Row label="New" value={`+${money(wf.new)}`} tone="up" />
                    <Row label="Expansion" value={`+${money(wf.expansion)}`} tone="up" />
                    <Row label="Reactivation" value={`+${money(wf.reactivation)}`} tone="up" />
                    <Row label="Contraction" value={`−${money(wf.contraction)}`} tone="down" />
                    <Row label="Churned" value={`−${money(wf.churned)}`} tone="down" />
                    <div className="mt-1 border-t border-border pt-2">
                      <Row label="Ending MRR" value={money(wf.ending_mrr)} strong />
                    </div>
                  </dl>
                </Card>
              </>
            )}
          </div>
        )
      )}
    </div>
  );
}

function Row({ label, value, tone, strong }) {
  const toneClass = tone === "up" ? "text-emerald-600" : tone === "down" ? "text-rose-600" : "text-foreground";
  return (
    <div className="flex items-baseline justify-between">
      <dt className={`text-sm ${strong ? "font-semibold text-foreground" : "text-muted-foreground"}`}>{label}</dt>
      <dd className={`font-mono text-sm tabular-nums ${strong ? "font-semibold text-foreground" : toneClass}`}>{value}</dd>
    </div>
  );
}
