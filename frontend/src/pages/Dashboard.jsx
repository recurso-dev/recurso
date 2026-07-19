import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { AreaChart } from "@tremor/react";
import { DollarSign, Users, TrendingDown, RotateCcw, BarChart3 } from "lucide-react";

import { endpoints } from "../lib/api";
import { cn, formatDate } from "@/lib/utils";
import { Money } from "@/components/ui/money";
import { PageHeader } from "@/components/patterns/PageHeader";
import { StatCard } from "@/components/patterns/StatCard";
import { CardGridSkeleton, Skeleton } from "@/components/patterns/LoadingSkeleton";
import { EmptyState } from "@/components/patterns/EmptyState";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
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
  const [loading, setLoading] = useState(true);
  const [subscriptions, setSubscriptions] = useState([]);
  const [invoices, setInvoices] = useState([]);
  const [customerNames, setCustomerNames] = useState({});
  const [mrr, setMrr] = useState(null);
  const [recovered, setRecovered] = useState(null);
  const [recoveredCurrency, setRecoveredCurrency] = useState("USD");

  useEffect(() => {
    let active = true;
    (async () => {
      const [subsRes, invRes, custRes, mrrRes, recRes] = await Promise.all([
        endpoints.getSubscriptions({ limit: 1000 }).catch(() => null),
        endpoints.getInvoices({ limit: 1000 }).catch(() => null),
        endpoints.getCustomers({ limit: 1000 }).catch(() => null),
        endpoints.getMRR().catch(() => null),
        endpoints.getDunningRecovered().catch(() => null),
      ]);
      if (!active) return;

      setSubscriptions(subsRes?.data?.data || []);
      setInvoices(invRes?.data?.data || []);

      const names = {};
      (custRes?.data?.data || []).forEach((c) => {
        names[c.id] = c.name;
      });
      setCustomerNames(names);

      // MRR endpoint may return { mrr } or { data: { mrr } }; null => unavailable.
      const mrrVal = mrrRes?.data?.mrr ?? mrrRes?.data?.data?.mrr;
      setMrr(mrrVal ?? null);

      // Recovered revenue, normalized server-side into the tenant's reporting
      // currency (reporting_total / reporting_currency). Summing the raw
      // per-currency map would add ₹ and $ minor units together.
      const rec = recRes?.data?.data ?? recRes?.data;
      setRecovered(rec?.reporting_total ?? null);
      setRecoveredCurrency(rec?.reporting_currency || "USD");

      setLoading(false);
    })();
    return () => {
      active = false;
    };
  }, []);

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

  // Revenue-over-time, one series per currency: different currencies cannot be
  // summed into one line without FX, so each gets its own (₹ and $ don't add).
  const { revenueSeries, revenueCurrencies } = useMemo(() => {
    const byDay = {};
    const currencies = new Set();
    invoices.forEach((inv) => {
      if (!inv.created_at) return;
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
          row[c] = (byDay[day][c] || 0) / 100;
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
      />

      {/* KPI row */}
      {loading ? (
        <CardGridSkeleton count={4} />
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <StatCard
            label="MRR"
            value={mrr != null ? <Money amountMinor={mrr} /> : "—"}
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
            value={recovered != null ? <Money amountMinor={recovered} currency={recoveredCurrency} /> : "—"}
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
                    <TableRow key={inv.id} className="hover:bg-transparent">
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
