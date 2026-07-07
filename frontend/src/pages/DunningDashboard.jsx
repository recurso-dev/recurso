import { useEffect, useState } from "react";
import { BarChart } from "@tremor/react";
import { RotateCcw, RefreshCw, CheckCircle2, Percent, BarChart3 } from "lucide-react";

import { endpoints } from "../lib/api";
import { formatNumber } from "@/lib/utils";
import { PageHeader } from "@/components/patterns/PageHeader";
import { StatCard } from "@/components/patterns/StatCard";
import { EmptyState } from "@/components/patterns/EmptyState";
import { CardGridSkeleton, Skeleton } from "@/components/patterns/LoadingSkeleton";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

// Recovered-revenue money is shown with no fraction digits (headline currency).
const formatMoney = (amount, currency) => {
  try {
    return new Intl.NumberFormat("en-US", {
      style: "currency",
      currency,
      maximumFractionDigits: 0,
    }).format(amount / 100);
  } catch {
    return `${currency} ${(amount / 100).toFixed(0)}`;
  }
};

// Last 12 calendar months as "YYYY-MM", oldest first (matches the API window).
const lastTwelveMonths = () => {
  const months = [];
  const d = new Date();
  d.setDate(1);
  d.setMonth(d.getMonth() - 11);
  for (let i = 0; i < 12; i++) {
    months.push(`${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}`);
    d.setMonth(d.getMonth() + 1);
  }
  return months;
};

const DunningDashboard = () => {
  const [overview, setOverview] = useState(null);
  const [weights, setWeights] = useState([]);
  const [history, setHistory] = useState([]);
  const [recovered, setRecovered] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const [overviewRes, weightsRes, historyRes, recoveredRes] = await Promise.all([
          endpoints.getDunningOverview(),
          endpoints.getDunningWeights(),
          endpoints.getDunningHistory({ limit: 50 }),
          endpoints.getDunningRecovered(),
        ]);
        setOverview(overviewRes.data);
        setWeights(weightsRes.data?.data || []);
        setHistory(historyRes.data?.data || []);
        setRecovered(recoveredRes.data);
      } catch (err) {
        console.error("Failed to fetch dunning data:", err);
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);

  // Group weights by context key to find the winning arm per context.
  const contextGroups = {};
  weights.forEach((w) => {
    if (!contextGroups[w.context_key]) contextGroups[w.context_key] = [];
    contextGroups[w.context_key].push(w);
  });

  // Recovered revenue: pick the currency with the largest recovered total as
  // the headline; any other currencies are listed in the subtitle.
  const recoveredTotals = recovered?.recovered_amount_total || {};
  const currencies = Object.keys(recoveredTotals).sort(
    (a, b) => recoveredTotals[b] - recoveredTotals[a]
  );
  const primaryCurrency = currencies[0] || "USD";
  const recoveredValue =
    currencies.length > 0
      ? formatMoney(recoveredTotals[primaryCurrency], primaryCurrency)
      : formatMoney(0, "USD");
  const recoveredSubtitleParts = [`${recovered?.recovered_count || 0} invoices`];
  if (recovered?.recovered_count > 0) {
    recoveredSubtitleParts.push(`avg ${(recovered?.avg_attempts || 0).toFixed(1)} attempts`);
  }
  if (currencies.length > 1) {
    recoveredSubtitleParts.push(
      `+ ${currencies
        .slice(1)
        .map((c) => formatMoney(recoveredTotals[c], c))
        .join(", ")}`
    );
  }

  // Monthly recovered-revenue series (headline currency drives bar heights).
  const months = lastTwelveMonths();
  const monthlyByMonth = {};
  (recovered?.monthly || []).forEach((b) => {
    if (!monthlyByMonth[b.month]) monthlyByMonth[b.month] = { amount: 0, count: 0 };
    if (b.currency === primaryCurrency) monthlyByMonth[b.month].amount += b.amount;
    monthlyByMonth[b.month].count += b.count;
  });
  const chartData = months.map((m) => ({
    month: m.slice(5),
    Recovered: (monthlyByMonth[m]?.amount || 0) / 100,
  }));
  const hasRecovered = (recovered?.recovered_count || 0) > 0;

  const currencyFormatter = (v) => {
    try {
      return new Intl.NumberFormat("en-US", {
        style: "currency",
        currency: primaryCurrency,
        maximumFractionDigits: 0,
      }).format(v);
    } catch {
      return `${primaryCurrency} ${v}`;
    }
  };

  return (
    <div>
      <PageHeader
        title="Smart Dunning"
        description="RL-based payment retry optimization — epsilon-greedy multi-armed bandit."
      />

      {/* Overview KPIs */}
      {loading ? (
        <CardGridSkeleton count={4} />
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <StatCard
            label="Recovered Revenue"
            value={recoveredValue}
            icon={RotateCcw}
            hint={recoveredSubtitleParts.join(" · ")}
          />
          <StatCard
            label="Total Retries"
            value={formatNumber(overview?.total_retries || 0)}
            icon={RefreshCw}
          />
          <StatCard
            label="Successful Recoveries"
            value={formatNumber(overview?.total_successes || 0)}
            icon={CheckCircle2}
          />
          <StatCard
            label="Success Rate"
            value={overview?.success_rate ? `${(overview.success_rate * 100).toFixed(1)}%` : "0%"}
            icon={Percent}
          />
        </div>
      )}

      {/* Recovered Revenue by Month */}
      <Card className="mt-6">
        <CardHeader>
          <CardTitle className="text-base">Recovered Revenue by Month</CardTitle>
          <CardDescription>
            Revenue attributed to the retry/dunning engine over the last 12 months
            {currencies.length > 1 ? ` (${primaryCurrency} only)` : ""}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div data-testid="recovered-chart">
            {loading ? (
              <Skeleton className="h-72 w-full" />
            ) : !hasRecovered ? (
              <EmptyState
                icon={BarChart3}
                title="No recoveries yet"
                description="No recovered payments yet. Recoveries appear when a failed invoice is paid after retries."
              />
            ) : (
              <BarChart
                className="h-72"
                data={chartData}
                index="month"
                categories={["Recovered"]}
                colors={["emerald"]}
                valueFormatter={currencyFormatter}
                showLegend={false}
                showGridLines
                yAxisWidth={64}
              />
            )}
          </div>
        </CardContent>
      </Card>

      {/* Arm Performance by Context */}
      <Card className="mt-6">
        <CardHeader>
          <CardTitle className="text-base">Arm Performance by Context</CardTitle>
          <CardDescription>
            Each context (currency:error_code) learns independently which retry interval works best.
          </CardDescription>
        </CardHeader>
        <CardContent className="px-0 pb-0">
          {loading ? (
            <div className="space-y-3 px-6 pb-6">
              {Array.from({ length: 4 }).map((_, i) => (
                <Skeleton key={i} className="h-8 w-full" />
              ))}
            </div>
          ) : Object.keys(contextGroups).length === 0 ? (
            <EmptyState
              title="No data yet"
              description="Weights will appear after the first retry outcomes are recorded."
            />
          ) : (
            <Table>
              <TableHeader>
                <TableRow className="bg-muted/40 hover:bg-muted/40">
                  <TableHead className="pl-6">Context</TableHead>
                  <TableHead>Arm</TableHead>
                  <TableHead className="text-right">Avg Reward</TableHead>
                  <TableHead className="text-right">Samples</TableHead>
                  <TableHead className="pr-6">Status</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {Object.entries(contextGroups).map(([contextKey, arms]) => {
                  const bestArm = arms.reduce(
                    (best, arm) => (arm.average_reward > best.average_reward ? arm : best),
                    arms[0]
                  );
                  return arms.map((arm, idx) => (
                    <TableRow key={`${contextKey}-${arm.action_id}`} className="hover:bg-transparent">
                      {idx === 0 && (
                        <TableCell
                          className="pl-6 font-mono text-sm text-muted-foreground align-top"
                          rowSpan={arms.length}
                        >
                          {contextKey}
                        </TableCell>
                      )}
                      <TableCell className="font-mono text-sm text-foreground">
                        {arm.action_id}
                      </TableCell>
                      <TableCell className="text-right tabular-nums">
                        <span
                          className={
                            arm.average_reward > 0.5
                              ? "text-emerald-600"
                              : arm.average_reward > 0.2
                                ? "text-amber-600"
                                : "text-muted-foreground"
                          }
                        >
                          {(arm.average_reward * 100).toFixed(1)}%
                        </span>
                      </TableCell>
                      <TableCell className="text-right tabular-nums text-muted-foreground">
                        {arm.sample_count}
                      </TableCell>
                      <TableCell className="pr-6">
                        {arm.action_id === bestArm.action_id && arm.sample_count > 0 ? (
                          <Badge variant="success">Best</Badge>
                        ) : null}
                      </TableCell>
                    </TableRow>
                  ));
                })}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Recent Retry History */}
      <Card className="mt-6">
        <CardHeader>
          <CardTitle className="text-base">Recent Retry History</CardTitle>
        </CardHeader>
        <CardContent className="px-0 pb-0">
          {loading ? (
            <div className="space-y-3 px-6 pb-6">
              {Array.from({ length: 4 }).map((_, i) => (
                <Skeleton key={i} className="h-8 w-full" />
              ))}
            </div>
          ) : history.length === 0 ? (
            <EmptyState title="No retry history yet" />
          ) : (
            <Table>
              <TableHeader>
                <TableRow className="bg-muted/40 hover:bg-muted/40">
                  <TableHead className="pl-6">Time</TableHead>
                  <TableHead>Invoice</TableHead>
                  <TableHead>Context</TableHead>
                  <TableHead>Action</TableHead>
                  <TableHead className="pr-6">Outcome</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {history.map((h) => (
                  <TableRow key={h.id} className="hover:bg-transparent">
                    <TableCell className="pl-6 text-sm text-muted-foreground">
                      {new Date(h.created_at).toLocaleString()}
                    </TableCell>
                    <TableCell className="font-mono text-sm text-muted-foreground">
                      {h.invoice_id?.substring(0, 8)}...
                    </TableCell>
                    <TableCell className="font-mono text-sm text-muted-foreground">
                      {h.context_key}
                    </TableCell>
                    <TableCell className="font-mono text-sm text-foreground">
                      {h.action_id}
                    </TableCell>
                    <TableCell className="pr-6">
                      {h.outcome === "success" ? (
                        <Badge variant="success">Success</Badge>
                      ) : (
                        <Badge variant="destructive">Failed</Badge>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
};

export default DunningDashboard;
