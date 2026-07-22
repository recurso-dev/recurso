import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link, useNavigate } from "react-router-dom";
import { AreaChart } from "@tremor/react";
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
import { cn, formatCurrency, formatDate, fromMinorUnits } from "@/lib/utils";
import { Money } from "@/components/ui/money";
import { PageHeader } from "@/components/patterns/PageHeader";
import { StatCard } from "@/components/patterns/StatCard";
import { CardGridSkeleton, Skeleton } from "@/components/patterns/LoadingSkeleton";
import { EmptyState } from "@/components/patterns/EmptyState";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

// Map an invoice status to a Badge variant.
const invoiceStatusVariant = (status) =>
  ({
    paid: "success",
    open: "info",
    past_due: "destructive",
    void: "neutral",
    draft: "neutral",
  })[status] || "neutral";

export default function Dashboard() {
  const navigate = useNavigate();

  // One aggregate query for the whole overview. Each endpoint catches to null so
  // Promise.all never rejects — a failed tile degrades rather than blanking the
  // page (there's no error state, matching the original).
  const { data, isLoading: loading } = useQuery({
    queryKey: ["dashboard-overview"],
    queryFn: async () => {
      const [subsRes, invRes, custRes, mrrRes, recRes, dispRes, churnRes] = await Promise.all([
        endpoints.getSubscriptions({ limit: 1000 }).catch(() => null),
        endpoints.getInvoices({ limit: 1000 }).catch(() => null),
        endpoints.getCustomers({ limit: 1000 }).catch(() => null),
        endpoints.getMRR().catch(() => null),
        endpoints.getDunningRecovered().catch(() => null),
        endpoints.getDisputes("open").catch(() => null),
        endpoints.getChurnAlerts().catch(() => null),
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
        subscriptions: subsRes?.data?.data || [],
        invoices: invRes?.data?.data || [],
        customerNames: names,
        // MRR endpoint may return { mrr } or { data: { mrr } }; null => unavailable.
        mrr: (mrrRes?.data?.mrr ?? mrrRes?.data?.data?.mrr) ?? null,
        recovered: rec?.reporting_total ?? null,
        recoveredCurrency: rec?.reporting_currency || "USD",
        openDisputes: (dispRes?.data?.data || []).length,
        churnAlerts: (churnRes?.data?.data || []).length,
      };
    },
  });
  // Stable references (only change when the query result does) so the derived
  // useMemos below don't recompute every render.
  const subscriptions = useMemo(() => data?.subscriptions ?? [], [data]);
  const invoices = useMemo(() => data?.invoices ?? [], [data]);
  const customerNames = data?.customerNames ?? {};
  const mrr = data?.mrr ?? null;
  const recovered = data?.recovered ?? null;
  const recoveredCurrency = data?.recoveredCurrency ?? "USD";
  const openDisputes = data?.openDisputes ?? 0;
  const churnAlerts = data?.churnAlerts ?? 0;

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
  const { revenueSeries, revenueCurrencies } = useMemo(() => {
    const cutoff = new Date();
    cutoff.setDate(cutoff.getDate() - 90);
    const byDay = {};
    const currencies = new Set();
    invoices.forEach((inv) => {
      if (!inv.created_at || new Date(inv.created_at) < cutoff) return;
      const key = new Date(inv.created_at).toISOString().slice(0, 10);
      const cur = (inv.currency || "USD").toUpperCase();
      currencies.add(cur);
      byDay[key] = byDay[key] || {};
      byDay[key][cur] = (byDay[key][cur] || 0) + (inv.total || 0);
    });
    const curs = [...currencies].sort();
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

  const recentInvoices = useMemo(
    () =>
      [...invoices]
        .sort((a, b) => new Date(b.created_at) - new Date(a.created_at))
        .slice(0, 8),
    [invoices]
  );

  return (
    <div>
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
                className="flex items-center gap-3 rounded-lg border border-red-200 bg-red-50 px-4 py-3 transition-colors hover:bg-red-100"
              >
                <AlertTriangle className="h-5 w-5 shrink-0 text-red-600" />
                <div className="min-w-0">
                  <p className="text-sm font-semibold text-red-800">
                    {overdueCount} overdue invoice{overdueCount === 1 ? "" : "s"}
                  </p>
                  <p className="truncate text-xs text-red-700">
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
                className="flex items-center gap-3 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 transition-colors hover:bg-amber-100"
              >
                <FileQuestion className="h-5 w-5 shrink-0 text-amber-600" />
                <div>
                  <p className="text-sm font-semibold text-amber-800">
                    {openDisputes} open dispute{openDisputes === 1 ? "" : "s"}
                  </p>
                  <p className="text-xs text-amber-700">Customers are waiting on you</p>
                </div>
              </Link>
            )}
            {churnAlerts > 0 && (
              <Link
                to="/churn"
                className="flex items-center gap-3 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 transition-colors hover:bg-amber-100"
              >
                <TrendingDown className="h-5 w-5 shrink-0 text-amber-600" />
                <div>
                  <p className="text-sm font-semibold text-amber-800">
                    {churnAlerts} churn alert{churnAlerts === 1 ? "" : "s"}
                  </p>
                  <p className="text-xs text-amber-700">Risk scores spiked — review them</p>
                </div>
              </Link>
            )}
          </div>
        ) : (
          <div className="mb-6 flex items-center gap-2 rounded-lg border border-border bg-muted/30 px-4 py-2.5 text-sm text-muted-foreground">
            <CheckCircle2 className="h-4 w-4 text-emerald-600" />
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
            value={mrr != null ? formatCurrency(mrr, "USD") : "—"}
            icon={DollarSign}
            hint="Monthly recurring revenue"
            to="/overview"
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
          />
          <StatCard
            label="Recovered Revenue"
            value={recovered != null ? formatCurrency(recovered, recoveredCurrency) : "—"}
            icon={RotateCcw}
            hint="Via smart dunning"
            to="/dunning"
          />
        </div>
      )}

      {/* Chart + recent invoices */}
      <div className="mt-6 grid grid-cols-1 gap-4 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle className="text-base">Revenue over time</CardTitle>
          </CardHeader>
          <CardContent>
            {loading ? (
              <Skeleton className="h-72 w-full" />
            ) : revenueSeries.length > 0 ? (
              <AreaChart
                className="h-72"
                data={revenueSeries}
                index="date"
                categories={revenueCurrencies}
                colors={["emerald", "blue", "amber", "violet"]}
                valueFormatter={(v) =>
                  new Intl.NumberFormat("en-US", { maximumFractionDigits: 0 }).format(v)
                }
                showLegend={revenueCurrencies.length > 1}
                showGridLines
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

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0">
            <CardTitle className="text-base">Recent invoices</CardTitle>
            <Link to="/invoices" className="text-sm font-medium text-emerald-700 hover:underline">
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
                      onClick={() => navigate("/invoices", { state: { openInvoiceId: inv.id } })}
                      onKeyDown={(e) => {
                        if (e.key === "Enter" || e.key === " ") {
                          e.preventDefault();
                          navigate("/invoices", { state: { openInvoiceId: inv.id } });
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
                        <Badge
                          variant={invoiceStatusVariant(inv.status)}
                          className={cn("capitalize")}
                        >
                          {(inv.status || "unknown").replace("_", " ")}
                        </Badge>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
