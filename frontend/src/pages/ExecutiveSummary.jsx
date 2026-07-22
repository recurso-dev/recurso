import { useCallback, useEffect, useState } from "react";
import { AreaChart, BarChart } from "@tremor/react";

import { endpoints } from "../lib/api";
import { cn, formatCurrency, fromMinorUnits } from "@/lib/utils";
import { PageHeader } from "@/components/patterns/PageHeader";
import { StatCard } from "@/components/patterns/StatCard";
import { ErrorState } from "@/components/patterns/ErrorState";
import { CardGridSkeleton, Skeleton } from "@/components/patterns/LoadingSkeleton";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";

const iso = (d) => d.toISOString().slice(0, 10);
const PERIODS = [30, 60, 90];

// First day of the month `offset` months before now (UTC).
const monthStart = (offset) => {
  const d = new Date();
  return new Date(Date.UTC(d.getUTCFullYear(), d.getUTCMonth() - offset, 1));
};

// Board-grade overview: composes the shipped analytics endpoints into one view.
// Each endpoint is fetched independently (Promise.allSettled) so a single
// failure degrades one tile rather than the whole page.
export default function ExecutiveSummary() {
  const [m, setM] = useState(null);
  const [trend, setTrend] = useState(null); // [{ month, MRR }] in major units
  const [days, setDays] = useState(30);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const load = useCallback(async (windowDays) => {
    setLoading(true);
    setError(null);
    const now = new Date();
    const start = new Date(now);
    start.setDate(start.getDate() - windowDays);

    const [mrr, wf, ue, aging, rev] = await Promise.allSettled([
      endpoints.getMRR(),
      endpoints.getMRRWaterfall(iso(start), iso(now)),
      endpoints.getUnitEconomics(),
      endpoints.getInvoiceAging(),
      endpoints.getRevenueRecognition(now.getMonth() + 1, now.getFullYear()),
    ]);

    // All five now wrap the payload in { data } (contract standardization).
    const inner = (r) => (r.status === "fulfilled" ? r.value?.data?.data : null);

    const next = {
      mrr: inner(mrr),
      wf: inner(wf),
      ue: inner(ue),
      aging: inner(aging),
      rev: inner(rev),
    };
    if (!next.mrr && !next.wf && !next.ue && !next.aging && !next.rev) {
      setError("Could not load metrics. Is the server running?");
      setM(null);
    } else {
      setM(next);
    }
    setLoading(false);
  }, []);

  // Six-month MRR trend, built from monthly waterfall windows (ending_mrr per
  // month). Fetched once — independent of the movement window above.
  const loadTrend = useCallback(async () => {
    const windows = Array.from({ length: 6 }, (_, i) => 5 - i).map((offset) => ({
      start: monthStart(offset),
      end: monthStart(offset - 1),
    }));
    const results = await Promise.allSettled(
      windows.map((w) => endpoints.getMRRWaterfall(iso(w.start), iso(w.end)))
    );
    const points = results
      .map((r, i) => {
        const d = r.status === "fulfilled" ? r.value?.data?.data : null;
        if (d?.ending_mrr == null) return null;
        return {
          month: windows[i].start.toLocaleString("en", { month: "short", timeZone: "UTC" }),
          MRR: fromMinorUnits(d.ending_mrr, d.reporting_currency),
        };
      })
      .filter(Boolean);
    setTrend(points);
  }, []);

  useEffect(() => {
    load(days);
  }, [load, days]);

  useEffect(() => {
    loadTrend();
  }, [loadTrend]);

  const cur = m?.mrr?.reporting_currency || m?.ue?.reporting_currency || "USD";
  const money = (n) => (n == null ? "—" : formatCurrency(n, cur));
  const pct = (n) => (n == null ? "—" : `${Number(n).toFixed(1)}%`);
  const signed = (n) =>
    n == null ? "—" : `${n >= 0 ? "+" : "−"}${formatCurrency(Math.abs(n), cur)}`;
  const chartMoney = (v) =>
    new Intl.NumberFormat("en-US", { style: "currency", currency: cur, maximumFractionDigits: 0 }).format(v);

  const mrrVal = m?.mrr?.normalized_mrr ?? m?.mrr?.mrr ?? null;
  const netChange = m?.wf ? (m.wf.ending_mrr || 0) - (m.wf.starting_mrr || 0) : null;
  const ndr = m?.wf?.has_start_history ? m.wf.net_dollar_retention : null;
  const overdue =
    m?.aging?.buckets != null
      ? (m.aging.total_outstanding || 0) -
        (m.aging.buckets.find((b) => b.label === "current")?.amount || 0)
      : null;
  const overdueRatio =
    overdue != null && m?.aging?.total_outstanding ? overdue / m.aging.total_outstanding : 0;
  const churnExceedsGrowth =
    m?.wf != null && (m.wf.churned || 0) > (m.wf.new || 0) + (m.wf.expansion || 0);

  const movementData = m?.wf
    ? [
        { name: "New", Amount: fromMinorUnits(m.wf.new || 0, cur) },
        { name: "Expansion", Amount: fromMinorUnits(m.wf.expansion || 0, cur) },
        { name: "Contraction", Amount: -fromMinorUnits(m.wf.contraction || 0, cur) },
        { name: "Churned", Amount: -fromMinorUnits(m.wf.churned || 0, cur) },
      ]
    : [];

  return (
    <div>
      <PageHeader
        title="Executive Summary"
        description="Your revenue at a glance — recurring revenue, movement, unit economics, and receivables."
        actions={
          <div className="flex items-center gap-1 rounded-lg border border-border bg-white p-0.5">
            {PERIODS.map((p) => (
              <button
                key={p}
                onClick={() => setDays(p)}
                className={cn(
                  "rounded-md px-3 py-1 text-sm font-medium transition-colors",
                  days === p
                    ? "bg-emerald-50 text-emerald-700"
                    : "text-stone-500 hover:text-stone-900"
                )}
              >
                {p}d
              </button>
            ))}
          </div>
        }
      />

      {loading ? (
        <CardGridSkeleton count={4} />
      ) : error ? (
        <Card className="overflow-hidden">
          <ErrorState message={error} onRetry={() => load(days)} />
        </Card>
      ) : (
        m && (
          <div className="flex flex-col gap-8">
            <Section title="Revenue">
              <StatCard label="MRR" value={money(mrrVal)} hint="Monthly recurring revenue" to="/finance/mrr-waterfall" />
              <StatCard label="ARR" value={money(m?.mrr?.arr)} hint="Annual run-rate" to="/finance/revenue-by-plan" />
              <StatCard
                label={`Net change (${days}d)`}
                value={signed(netChange)}
                hint="Ending vs. starting MRR"
                tone={netChange != null && netChange < 0 ? "danger" : undefined}
                to="/finance/mrr-waterfall"
              />
              <StatCard
                label="Net dollar retention"
                value={pct(ndr)}
                hint={ndr == null ? "Needs MRR history" : `Trailing ${days} days`}
                tone={ndr != null && ndr < 100 ? "warning" : undefined}
                to="/finance/mrr-waterfall"
              />
            </Section>

            {/* Charts: six-month trend + movement in the selected window */}
            <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">MRR trend</CardTitle>
                  <CardDescription>End-of-month MRR, last 6 months</CardDescription>
                </CardHeader>
                <CardContent>
                  {trend == null ? (
                    <Skeleton className="h-56 w-full" />
                  ) : trend.length < 2 ? (
                    <p className="flex h-56 items-center justify-center text-sm text-muted-foreground">
                      Not enough history yet — the trend appears after two months of data.
                    </p>
                  ) : (
                    <AreaChart
                      className="h-56"
                      data={trend}
                      index="month"
                      categories={["MRR"]}
                      colors={["emerald"]}
                      valueFormatter={chartMoney}
                      showLegend={false}
                      yAxisWidth={72}
                    />
                  )}
                </CardContent>
              </Card>
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">MRR movement ({days}d)</CardTitle>
                  <CardDescription>What grew and what shrank in the window</CardDescription>
                </CardHeader>
                <CardContent>
                  <BarChart
                    className="h-56"
                    data={movementData}
                    index="name"
                    categories={["Amount"]}
                    colors={["emerald"]}
                    valueFormatter={chartMoney}
                    showLegend={false}
                    yAxisWidth={72}
                  />
                </CardContent>
              </Card>
            </div>

            <Section title={`MRR movement (${days} days)`}>
              <StatCard label="New" value={money(m?.wf?.new)} hint="From new subscriptions" to="/finance/mrr-waterfall" />
              <StatCard label="Expansion" value={money(m?.wf?.expansion)} hint="Upgrades" to="/finance/mrr-waterfall" />
              <StatCard
                label="Contraction"
                value={m?.wf ? `−${formatCurrency(m.wf.contraction || 0, cur)}` : "—"}
                hint="Downgrades"
                to="/finance/mrr-waterfall"
              />
              <StatCard
                label="Churned"
                value={m?.wf ? `−${formatCurrency(m.wf.churned || 0, cur)}` : "—"}
                hint="Cancellations"
                tone={churnExceedsGrowth ? "danger" : undefined}
                to="/churn"
              />
            </Section>

            <Section title="Unit economics">
              <StatCard label="ARPA" value={money(m?.ue?.arpa)} hint="Per account / month" to="/finance/unit-economics" />
              <StatCard label="ARPU" value={money(m?.ue?.arpu)} hint="Per subscription / month" to="/finance/unit-economics" />
              <StatCard
                label="LTV"
                value={m?.ue?.has_ltv ? money(m.ue.ltv) : "—"}
                hint={m?.ue?.has_ltv ? "At current churn" : "Needs history"}
                to="/finance/unit-economics"
              />
              <StatCard
                label="Active customers"
                value={m?.ue ? (m.ue.active_customers || 0).toLocaleString() : "—"}
                hint={m?.ue ? `${(m.ue.active_subscriptions || 0).toLocaleString()} subscriptions` : ""}
                to="/customers"
              />
            </Section>

            <Section title="Cash & recognition">
              <StatCard
                label="Outstanding"
                value={money(m?.aging?.total_outstanding)}
                hint={m?.aging ? `${m.aging.total_count || 0} open invoices` : ""}
                to="/finance/invoice-aging"
              />
              <StatCard
                label="Overdue"
                value={money(overdue)}
                hint={overdueRatio > 0.3 ? `${Math.round(overdueRatio * 100)}% of outstanding` : "Past due date"}
                tone={overdueRatio > 0.3 ? "danger" : overdueRatio > 0.1 ? "warning" : undefined}
                to="/finance/invoice-aging"
              />
              <StatCard
                label="Deferred revenue"
                value={money(m?.rev?.deferred_balance)}
                hint="Unearned, still to recognize"
                to="/finance/revenue-recognition"
              />
              <StatCard
                label="Recognized (MTD)"
                value={money(m?.rev?.recognized_amount)}
                hint="This month"
                to="/finance/revenue-recognition"
              />
            </Section>
          </div>
        )
      )}
    </div>
  );
}

function Section({ title, children }) {
  return (
    <div>
      <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-muted-foreground">{title}</h2>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">{children}</div>
    </div>
  );
}
