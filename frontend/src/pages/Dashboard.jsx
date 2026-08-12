import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import OnboardingChecklist from "../components/onboarding/OnboardingChecklist";
import { Link, useNavigate } from "react-router";
import { AreaChart, DonutChart } from "@tremor/react";
import {
  DollarSign,
  Users,
  TrendingDown,
  RotateCcw,
  BarChart3,
  Plus,
  AlertTriangle,
  FileQuestion,
  CheckCircle2,
} from "lucide-react";

import { endpoints } from "../lib/api";
import { makeChartTooltip, chartDefaults } from "@/components/charts/ChartTooltip";
import {
  cn,
  formatCurrency,
  formatCurrencyHeadline,
  formatDate,
  fromMinorUnits,
} from "@/lib/utils";
import { Money } from "@/components/ui/money";
import { PageHeader } from "@/components/patterns/PageHeader";
import { ErrorState } from "@/components/patterns/ErrorState";
import { StatCard } from "@/components/patterns/StatCard";
import { CardGridSkeleton, Skeleton } from "@/components/patterns/LoadingSkeleton";
import { EmptyState } from "@/components/patterns/EmptyState";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { StatusBadge } from "@/components/ui/status-badge";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

// Revenue axis/tooltip share one formatter so both read identically. Values are
// already in major units, so this is a plain thousands-grouped integer.
const revenueFormatter = (v) =>
  new Intl.NumberFormat("en-US", { maximumFractionDigits: 0 }).format(v);
const revenueTooltip = makeChartTooltip(revenueFormatter);
// Donut hover shows a subscription count, not money.
const subMixTooltip = makeChartTooltip((v) => `${v.toLocaleString()} subscriptions`);

// Map an invoice status to a Badge variant.
// Subscription-mix donut: current (non-canceled) statuses, healthiest first.
// Tremor color names must stay in the Tailwind safelist; the hex twins drive
// the custom legend swatches so donut and legend agree exactly.
const SUB_STATUS = [
  { key: "active", label: "Active", color: "emerald", hex: "#10b981" },
  { key: "trialing", label: "Trialing", color: "blue", hex: "#3b82f6" },
  { key: "past_due", label: "Past due", color: "amber", hex: "#f59e0b" },
  { key: "paused", label: "Paused", color: "zinc", hex: "#a1a1aa" },
];

// AR aging bands, oldest-money-is-reddest. Labels/colors for the receivables
// widget; hex is inline (not a Tailwind class) so arbitrary bucket hues are ok.
const AGING_BANDS = [
  { key: "current", label: "Current", hex: "#10b981" },
  { key: "1-30", label: "1–30 days", hex: "#f59e0b" },
  { key: "31-60", label: "31–60 days", hex: "#f97316" },
  { key: "61-90", label: "61–90 days", hex: "#ef4444" },
  { key: "90+", label: "90+ days", hex: "#dc2626" },
];

export default function Dashboard() {
  const navigate = useNavigate();

  // One aggregate query for the whole overview. Each endpoint catches to null so
  // Promise.all never rejects — a single failed tile degrades rather than
  // blanking the page. But if the core reads ALL fail (a real outage, not a
  // genuinely empty tenant), we surface a page-level error instead of a
  // dashboard of zeros that reads as a healthy, empty business — `null` means
  // the fetch failed, `[]` means it succeeded with no rows.
  const { data, isLoading: loading, refetch } = useQuery({
    queryKey: ["dashboard-overview"],
    queryFn: async () => {
      const [subsRes, invRes, custRes, mrrRes, recRes, dispRes, churnRes, agingRes] =
        await Promise.all([
          endpoints.getSubscriptions({ limit: 1000 }).catch(() => null),
          endpoints.getInvoices({ limit: 1000 }).catch(() => null),
          endpoints.getCustomers({ limit: 1000 }).catch(() => null),
          endpoints.getMRR().catch(() => null),
          endpoints.getDunningRecovered().catch(() => null),
          endpoints.getDisputes("open").catch(() => null),
          endpoints.getChurnAlerts().catch(() => null),
          endpoints.getInvoiceAging().catch(() => null),
        ]);
      const names = {};
      (custRes?.data?.data || []).forEach((c) => {
        names[c.id] = c.name;
      });
      // Recovered revenue, normalized server-side into the tenant's reporting
      // currency (reporting_total / reporting_currency). Summing the raw
      // per-currency map would add ₹ and $ minor units together.
      const rec = recRes?.data?.data ?? recRes?.data;
      return {
        // A systemic failure: every core read failed (not one empty tenant).
        loadFailed: [subsRes, invRes, custRes, mrrRes].every((r) => r === null),
        subscriptions: subsRes?.data?.data || [],
        invoices: invRes?.data?.data || [],
        customerNames: names,
        // MRR endpoint may return { mrr } or { data: { mrr } }; null => unavailable.
        mrr: (mrrRes?.data?.mrr ?? mrrRes?.data?.data?.mrr) ?? null,
        // ...alongside the reporting currency the figure is denominated in, so
        // the hero metric isn't mislabeled for a non-USD tenant.
        mrrCurrency:
          mrrRes?.data?.reporting_currency ?? mrrRes?.data?.data?.reporting_currency ?? "USD",
        recovered: rec?.reporting_total ?? null,
        recoveredCurrency: rec?.reporting_currency || "USD",
        openDisputes: (dispRes?.data?.data || []).length,
        churnAlerts: (churnRes?.data?.data || []).length,
        // Receivables aging, already normalized server-side into the reporting
        // currency (so buckets are summable, unlike raw per-invoice amounts).
        aging: agingRes?.data?.data ?? agingRes?.data ?? null,
      };
    },
  });
  // Stable references (only change when the query result does) so the derived
  // useMemos below don't recompute every render.
  const subscriptions = useMemo(() => data?.subscriptions ?? [], [data]);
  const invoices = useMemo(() => data?.invoices ?? [], [data]);
  const customerNames = data?.customerNames ?? {};
  const mrr = data?.mrr ?? null;
  const mrrCurrency = data?.mrrCurrency ?? "USD";
  const recovered = data?.recovered ?? null;
  const recoveredCurrency = data?.recoveredCurrency ?? "USD";
  const openDisputes = data?.openDisputes ?? 0;
  const churnAlerts = data?.churnAlerts ?? 0;
  const loadFailed = data?.loadFailed ?? false;

  const activeSubs = useMemo(
    () => subscriptions.filter((s) => s.status === "active").length,
    [subscriptions]
  );

  // Churn rate = canceled / (active + canceled). Derived from real data only.
  const churnRate = useMemo(() => {
    const canceled = subscriptions.filter((s) => s.status === "canceled").length;
    const denom = activeSubs + canceled;
    if (denom === 0) return null;
    return (canceled / denom) * 100;
  }, [subscriptions, activeSubs]);

  // Overdue receivables, per currency, from the already-fetched invoices.
  const overdueByCur = useMemo(() => {
    const sums = {};
    invoices
      .filter((inv) => inv.status === "past_due")
      .forEach((inv) => {
        const cur = (inv.currency || "USD").toUpperCase();
        sums[cur] = (sums[cur] || 0) + (inv.amount_due ?? inv.total ?? 0);
      });
    return sums;
  }, [invoices]);
  const overdueCount = useMemo(
    () => invoices.filter((inv) => inv.status === "past_due").length,
    [invoices]
  );
  const attentionCount = overdueCount + openDisputes + churnAlerts;

  // Revenue-over-time, one series per currency: different currencies cannot be
  // summed into one line without FX, so each gets its own (₹ and $ don't add).
  // Windowed to the trailing 90 days — a year of daily bars is unreadable.
  // Currencies are ordered by window total so [0] is the dominant one.
  const { revenueSeries, revenueCurrencies } = useMemo(() => {
    const cutoff = new Date();
    cutoff.setDate(cutoff.getDate() - 90);
    const byDay = {};
    const totals = {};
    invoices.forEach((inv) => {
      if (!inv.created_at || new Date(inv.created_at) < cutoff) return;
      const key = new Date(inv.created_at).toISOString().slice(0, 10);
      const cur = (inv.currency || "USD").toUpperCase();
      totals[cur] = (totals[cur] || 0) + (inv.total || 0);
      byDay[key] = byDay[key] || {};
      byDay[key][cur] = (byDay[key][cur] || 0) + (inv.total || 0);
    });
    const curs = Object.keys(totals).sort((a, b) => totals[b] - totals[a]);
    const series = Object.keys(byDay)
      .sort()
      .map((day) => {
        const row = { date: formatDate(day, { month: "short", day: "numeric" }) };
        curs.forEach((c) => {
          row[c] = fromMinorUnits(byDay[day][c] || 0, c);
        });
        return row;
      });
    return { revenueSeries: series, revenueCurrencies: curs };
  }, [invoices]);

  // One currency at a time: axes can't honestly hold ₹ and $ together, and a
  // large series flattens the others into noise. Default to the dominant one.
  const [revenueCur, setRevenueCur] = useState(null);
  useEffect(() => {
    if (revenueCurrencies.length > 0 && !revenueCurrencies.includes(revenueCur)) {
      setRevenueCur(revenueCurrencies[0]);
    }
  }, [revenueCurrencies, revenueCur]);

  const recentInvoices = useMemo(
    () =>
      [...invoices]
        .sort((a, b) => new Date(b.created_at) - new Date(a.created_at))
        .slice(0, 6),
    [invoices]
  );

  // Subscription mix — current (non-canceled) subs by status. Count-based, so
  // it needs no FX and is honest across currencies.
  const subMix = useMemo(() => {
    const counts = {};
    subscriptions.forEach((s) => {
      if (s.status === "canceled") return;
      counts[s.status] = (counts[s.status] || 0) + 1;
    });
    return SUB_STATUS.filter((s) => counts[s.key]).map((s) => ({
      name: s.label,
      value: counts[s.key],
      color: s.color,
      hex: s.hex,
    }));
  }, [subscriptions]);
  const totalCurrentSubs = useMemo(
    () => subMix.reduce((sum, s) => sum + s.value, 0),
    [subMix]
  );

  // Receivables aging, in the reporting currency. Buckets are pre-normalized.
  const aging = data?.aging ?? null;
  const agingCur = aging?.reporting_currency || "USD";
  const agingTotal = aging?.total_outstanding ?? 0;
  const agingRows = useMemo(() => {
    const byLabel = {};
    (aging?.buckets || []).forEach((b) => {
      byLabel[b.label] = b;
    });
    return AGING_BANDS.map((band) => ({
      ...band,
      amount: byLabel[band.key]?.amount ?? 0,
      count: byLabel[band.key]?.count ?? 0,
    }));
  }, [aging]);

  // Total outage: header stays (so the page isn't blank) but the tiles are
  // replaced by a retryable error — never a page of zeros that looks like a
  // real, empty business.
  if (loadFailed) {
    return (
      <div>
        <PageHeader title="Home" description="A snapshot of your billing performance." />
        <ErrorState
          title="Couldn't load your dashboard"
          message="We couldn't reach your billing data. Check your connection and try again."
          onRetry={() => refetch()}
        />
      </div>
    );
  }

  return (
    <div>
      <OnboardingChecklist />
      <PageHeader
        title="Home"
        description="A snapshot of your billing performance."
        actions={
          <div className="flex gap-2">
            <Button variant="outline" size="sm" asChild>
              <Link to="/customers/new">
                <Plus className="h-4 w-4" />
                Customer
              </Link>
            </Button>
            <Button variant="outline" size="sm" asChild>
              <Link to="/subscriptions/new">
                <Plus className="h-4 w-4" />
                Subscription
              </Link>
            </Button>
            <Button size="sm" asChild>
              <Link to="/plans/new">
                <Plus className="h-4 w-4" />
                Plan
              </Link>
            </Button>
          </div>
        }
      />

      {/* Needs attention: the "what should I fix today" strip */}
      {!loading &&
        (attentionCount > 0 ? (
          <div className="mb-6 grid grid-cols-1 gap-3 sm:grid-cols-3">
            {overdueCount > 0 && (
              <Link
                to="/finance/invoice-aging"
                className="flex items-center gap-3 rounded-lg border border-destructive/20 bg-destructive/5 px-4 py-3 transition-colors hover:bg-destructive/15"
              >
                <AlertTriangle className="h-5 w-5 shrink-0 text-destructive" />
                <div className="min-w-0">
                  <p className="text-sm font-semibold text-destructive">
                    {overdueCount} overdue invoice{overdueCount === 1 ? "" : "s"}
                  </p>
                  <p className="truncate text-xs text-destructive">
                    {Object.entries(overdueByCur)
                      .map(([c, v]) => formatCurrency(v, c))
                      .join(" + ")}{" "}
                    past due
                  </p>
                </div>
              </Link>
            )}
            {openDisputes > 0 && (
              <Link
                to="/disputes"
                className="flex items-center gap-3 rounded-lg border border-warning/20 bg-warning/5 px-4 py-3 transition-colors hover:bg-warning/15"
              >
                <FileQuestion className="h-5 w-5 shrink-0 text-warning" />
                <div>
                  <p className="text-sm font-semibold text-warning">
                    {openDisputes} open dispute{openDisputes === 1 ? "" : "s"}
                  </p>
                  <p className="text-xs text-warning">Customers are waiting on you</p>
                </div>
              </Link>
            )}
            {churnAlerts > 0 && (
              <Link
                to="/churn"
                className="flex items-center gap-3 rounded-lg border border-warning/20 bg-warning/5 px-4 py-3 transition-colors hover:bg-warning/15"
              >
                <TrendingDown className="h-5 w-5 shrink-0 text-warning" />
                <div>
                  <p className="text-sm font-semibold text-warning">
                    {churnAlerts} churn alert{churnAlerts === 1 ? "" : "s"}
                  </p>
                  <p className="text-xs text-warning">Risk scores spiked — review them</p>
                </div>
              </Link>
            )}
          </div>
        ) : (
          <div className="mb-6 flex items-center gap-2 rounded-lg border border-border bg-muted/30 px-4 py-2.5 text-sm text-muted-foreground">
            <CheckCircle2 className="h-4 w-4 text-success" />
            All clear — no overdue invoices, open disputes, or churn alerts.
          </div>
        ))}

      {/* KPI row */}
      {loading ? (
        <CardGridSkeleton count={4} />
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <StatCard
            label="MRR"
            value={mrr != null ? formatCurrencyHeadline(mrr, mrrCurrency) : "—"}
            icon={DollarSign}
            hint="Monthly recurring revenue"
            to="/overview"
            definition="Monthly recurring revenue across active subscriptions, normalized to your reporting currency."
          />
          <StatCard
            label="Active Subscriptions"
            value={activeSubs.toLocaleString()}
            icon={Users}
            hint="Currently active"
            to="/subscriptions"
          />
          <StatCard
            label="Churn"
            value={churnRate != null ? `${churnRate.toFixed(1)}%` : "—"}
            icon={TrendingDown}
            hint="Canceled vs. total"
            to="/churn"
            definition="Canceled subscriptions as a share of active + canceled. Count-based, not revenue-weighted."
          />
          <StatCard
            label="Recovered Revenue"
            value={recovered != null ? formatCurrencyHeadline(recovered, recoveredCurrency) : "—"}
            icon={RotateCcw}
            hint="Via smart dunning"
            to="/dunning"
            definition="Revenue collected by automatic dunning retries after an initial payment failure."
          />
        </div>
      )}

      {/* Charts row: revenue trend (wide) + live subscription mix */}
      <div className="mt-6 grid grid-cols-1 gap-4 lg:grid-cols-12">
        <Card className="lg:col-span-8">
          <CardHeader className="flex flex-row items-center justify-between space-y-0">
            <CardTitle className="text-base">Revenue over time</CardTitle>
            {revenueCurrencies.length > 1 && (
              <div className="flex items-center gap-1 rounded-lg bg-muted p-0.5" role="group" aria-label="Chart currency">
                {revenueCurrencies.map((c) => (
                  <button
                    key={c}
                    type="button"
                    onClick={() => setRevenueCur(c)}
                    aria-pressed={revenueCur === c}
                    className={cn(
                      "rounded-md px-2.5 py-1 text-xs font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
                      revenueCur === c
                        ? "bg-white text-foreground shadow-sm"
                        : "text-muted-foreground hover:text-foreground"
                    )}
                  >
                    {c}
                  </button>
                ))}
              </div>
            )}
          </CardHeader>
          <CardContent>
            {loading ? (
              <Skeleton className="h-80 w-full" />
            ) : revenueSeries.length > 0 && revenueCur ? (
              <AreaChart
                {...chartDefaults}
                className="h-80"
                data={revenueSeries}
                index="date"
                categories={[revenueCur]}
                colors={["emerald"]}
                valueFormatter={revenueFormatter}
                customTooltip={revenueTooltip}
                showLegend={false}
                showGradient
                startEndOnly
                curveType="monotone"
                yAxisWidth={64}
              />
            ) : (
              <EmptyState
                icon={BarChart3}
                title="No revenue yet"
                description="Revenue will appear here once you start issuing invoices."
              />
            )}
          </CardContent>
        </Card>

        {/* Subscription mix — a live donut of who's paying you right now */}
        <Card className="lg:col-span-4">
          <CardHeader className="flex flex-row items-center justify-between space-y-0">
            <CardTitle className="text-base">Subscription mix</CardTitle>
            <Link
              to="/subscriptions"
              className="text-sm font-medium text-success hover:underline"
            >
              View all
            </Link>
          </CardHeader>
          <CardContent>
            {loading ? (
              <Skeleton className="h-80 w-full" />
            ) : subMix.length === 0 ? (
              <EmptyState
                icon={Users}
                title="No active subscriptions"
                description="Active, trialing, and past-due subscriptions appear here."
              />
            ) : (
              <div>
                <DonutChart
                  {...chartDefaults}
                  className="mx-auto h-44"
                  data={subMix}
                  category="value"
                  index="name"
                  colors={subMix.map((s) => s.color)}
                  valueFormatter={(v) => v.toLocaleString()}
                  customTooltip={subMixTooltip}
                  showLabel
                  label={`${totalCurrentSubs.toLocaleString()}`}
                />
                <dl className="mt-6 space-y-2.5">
                  {subMix.map((s) => (
                    <div key={s.name} className="flex items-center justify-between text-sm">
                      <dt className="flex items-center gap-2 text-muted-foreground">
                        <span
                          className="h-2.5 w-2.5 rounded-full"
                          style={{ backgroundColor: s.hex }}
                        />
                        {s.name}
                      </dt>
                      <dd className="font-medium tabular-nums text-foreground">
                        {s.value.toLocaleString()}
                        <span className="ml-1.5 text-xs text-muted-foreground">
                          {Math.round((s.value / totalCurrentSubs) * 100)}%
                        </span>
                      </dd>
                    </div>
                  ))}
                </dl>
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Operational row: latest activity + where the money is stuck */}
      <div className="mt-4 grid grid-cols-1 gap-4 lg:grid-cols-12">
        <Card className="lg:col-span-8">
          <CardHeader className="flex flex-row items-center justify-between space-y-0">
            <CardTitle className="text-base">Recent invoices</CardTitle>
            <Link to="/invoices" className="text-sm font-medium text-success hover:underline">
              View all
            </Link>
          </CardHeader>
          <CardContent className="px-0 pb-0">
            {loading ? (
              <div className="space-y-3 px-6 pb-6">
                {Array.from({ length: 5 }).map((_, i) => (
                  <Skeleton key={i} className="h-8 w-full" />
                ))}
              </div>
            ) : recentInvoices.length === 0 ? (
              <EmptyState title="No invoices yet" />
            ) : (
              <Table>
                <TableHeader>
                  <TableRow className="hover:bg-transparent">
                    <TableHead className="pl-6">Customer</TableHead>
                    <TableHead className="text-right">Amount</TableHead>
                    <TableHead className="pr-6 text-right">Status</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {recentInvoices.map((inv) => (
                    <TableRow
                      key={inv.id}
                      role="button"
                      tabIndex={0}
                      onClick={() => navigate(`/invoices/${inv.id}`)}
                      onKeyDown={(e) => {
                        if (e.key === "Enter" || e.key === " ") {
                          e.preventDefault();
                          navigate(`/invoices/${inv.id}`);
                        }
                      }}
                      className="cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring"
                    >
                      <TableCell className="pl-6">
                        <div className="truncate text-sm font-medium text-foreground">
                          {customerNames[inv.customer_id] || "Customer"}
                        </div>
                        <div className="text-xs text-muted-foreground">
                          {formatDate(inv.created_at)}
                        </div>
                      </TableCell>
                      <TableCell className="text-right tabular-nums">
                        <Money amountMinor={inv.total} currency={inv.currency} />
                      </TableCell>
                      <TableCell className="pr-6 text-right">
                        <StatusBadge status={inv.status || "unknown"} />
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>

        {/* Receivables aging — how much is owed and how stale it's getting */}
        <Card className="lg:col-span-4">
          <CardHeader className="flex flex-row items-center justify-between space-y-0">
            <CardTitle className="text-base">Receivables</CardTitle>
            <Link
              to="/finance/invoice-aging"
              className="text-sm font-medium text-success hover:underline"
            >
              Aging
            </Link>
          </CardHeader>
          <CardContent>
            {loading ? (
              <Skeleton className="h-56 w-full" />
            ) : agingTotal === 0 ? (
              <EmptyState
                icon={CheckCircle2}
                title="Nothing outstanding"
                description="Open invoice balances appear here as they age."
              />
            ) : (
              <div>
                <p className="text-3xl font-semibold tracking-tight tabular-nums text-foreground">
                  {formatCurrencyHeadline(agingTotal, agingCur)}
                </p>
                <p className="mt-1 text-xs text-muted-foreground">
                  Outstanding across {aging?.total_count ?? 0} open invoice
                  {(aging?.total_count ?? 0) === 1 ? "" : "s"}
                </p>
                <div className="mt-5 space-y-3">
                  {agingRows.map((band) => {
                    const pct = agingTotal ? (band.amount / agingTotal) * 100 : 0;
                    return (
                      <div key={band.key}>
                        <div className="flex items-baseline justify-between text-sm">
                          <span className="text-muted-foreground">{band.label}</span>
                          <span className="font-medium tabular-nums text-foreground">
                            {formatCurrencyHeadline(band.amount, agingCur)}
                          </span>
                        </div>
                        <div className="mt-1.5 h-1.5 overflow-hidden rounded-full bg-muted">
                          <div
                            className="h-full rounded-full"
                            style={{ width: `${pct}%`, backgroundColor: band.hex }}
                          />
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
