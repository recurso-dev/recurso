import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router";
import { BarChart } from "@tremor/react";
import { RotateCcw, RefreshCw, CheckCircle2, Percent, BarChart3, Settings2 } from "lucide-react";

import { useMemo } from "react";

import { endpoints } from "../lib/api";
import { makeChartTooltip, chartCategoryColors, chartDefaults } from "@/components/charts/ChartTooltip";
import { Button } from "@/components/ui/button";
import { formatNumber, fromMinorUnits, formatDateTime } from "@/lib/utils";
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
// Amounts arrive in minor units; convert with the currency's real exponent.
const formatMoney = (amount, currency) => {
  const major = fromMinorUnits(amount, currency);
  try {
    return new Intl.NumberFormat("en-US", {
      style: "currency",
      currency,
      maximumFractionDigits: 0,
    }).format(major);
  } catch {
    return `${currency} ${major.toFixed(0)}`;
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

// "USD:card_declined" → "USD · card declined" — the segment key, in words.
const segmentLabel = (k) => (k || "").replace(":", " · ").replace(/_/g, " ");

const fmtWhen = (x) =>
  formatDateTime(x);

const DunningDashboard = () => {
  const {
    data,
    isLoading: loading,
    error: queryError,
  } = useQuery({
    queryKey: ["dunning-dashboard"],
    queryFn: async () => {
      const [overviewRes, weightsRes, historyRes, recoveredRes, timingRes] = await Promise.all([
        endpoints.getDunningOverview(),
        endpoints.getDunningWeights(),
        endpoints.getDunningHistory({ limit: 50 }),
        endpoints.getDunningRecovered(),
        endpoints.getDunningTiming(),
      ]);
      return {
        overview: overviewRes.data?.data,
        weights: weightsRes.data?.data || [],
        history: historyRes.data?.data || [],
        recovered: recoveredRes.data?.data,
        timing: timingRes.data?.data,
      };
    },
  });
  const overview = data?.overview ?? null;
  const weights = data?.weights ?? [];
  const history = data?.history ?? [];
  const recovered = data?.recovered ?? null;
  const timing = data?.timing ?? null;

  // "Best time to retry" — success rate per UTC hour, with the best-performing
  // hour highlighted. Only meaningful once there's enough history.
  const hourRates = timing?.by_hour ?? [];
  const timingHasData = (timing?.sample_size ?? 0) > 0;
  const maxHourRate = hourRates.reduce((m, h) => Math.max(m, h.success_rate || 0), 0);
  const fmtHour = (h) => `${String(h).padStart(2, "0")}:00`;
  const DAY_NAMES = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
  const loadError = queryError
    ? queryError?.response?.data?.error?.message || queryError?.message || "Failed to load dunning data"
    : null;

  // Group weights by context key to find the winning arm per context.
  const contextGroups = {};
  weights.forEach((w) => {
    if (!contextGroups[w.context_key]) contextGroups[w.context_key] = [];
    contextGroups[w.context_key].push(w);
  });

  // Recovered revenue is normalized server-side into the tenant's reporting
  // currency (reporting_total / reporting_currency), so the headline is always
  // in the account's currency rather than whichever currency has the largest
  // raw minor-unit amount. The raw per-currency breakdown is shown as context.
  const recoveredTotals = recovered?.recovered_amount_total || {};
  const primaryCurrency = recovered?.reporting_currency || "USD";
  const recoveredValue = formatMoney(recovered?.reporting_total || 0, primaryCurrency);
  const sourceCurrencies = Object.keys(recoveredTotals).filter((c) => c !== primaryCurrency);
  const recoveredSubtitleParts = [`${recovered?.recovered_count || 0} invoices`];
  if (recovered?.recovered_count > 0) {
    recoveredSubtitleParts.push(`avg ${(recovered?.avg_attempts || 0).toFixed(1)} attempts`);
  }
  if (sourceCurrencies.length > 0) {
    recoveredSubtitleParts.push(
      `incl. ${sourceCurrencies.map((c) => formatMoney(recoveredTotals[c], c)).join(", ")}`
    );
  }

  // Monthly series is already normalized to the reporting currency server-side.
  const months = lastTwelveMonths();
  const monthlyByMonth = {};
  (recovered?.monthly || []).forEach((b) => {
    if (!monthlyByMonth[b.month]) monthlyByMonth[b.month] = { amount: 0, count: 0 };
    monthlyByMonth[b.month].amount += b.amount;
    monthlyByMonth[b.month].count += b.count;
  });
  const chartData = months.map((m) => ({
    // "2026-07" → "Jul" — bare month numbers on the axis read as data noise.
    month: new Date(`${m}-01T00:00:00Z`).toLocaleString("en-US", {
      month: "short",
      timeZone: "UTC",
    }),
    Recovered: fromMinorUnits(monthlyByMonth[m]?.amount || 0, primaryCurrency),
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
  // Rebuilt only when the currency changes so hover doesn't remount the tooltip.
  const chartTooltip = useMemo(
    () => makeChartTooltip(currencyFormatter),
    [primaryCurrency] // eslint-disable-line react-hooks/exhaustive-deps
  );

  return (
    <div>
      <PageHeader
        title="Dunning"
        description="Failed-payment retries that learn the best timing from your own outcomes."
        actions={
          <Button variant="outline" asChild>
            <Link to="/dunning/campaigns">
              <Settings2 className="h-4 w-4" />
              Manage campaigns
            </Link>
          </Button>
        }
      />

      {loadError && (
        <p className="mb-4 rounded-md bg-destructive/5 px-3 py-2 text-sm text-destructive">
          {loadError} — refresh to retry.
        </p>
      )}

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
            hint="retry attempts on failed payments"
          />
          <StatCard
            label="Successful Recoveries"
            value={formatNumber(overview?.total_successes || 0)}
            icon={CheckCircle2}
            hint={
              overview?.total_retries
                ? `of ${formatNumber(overview.total_retries)} retries`
                : "retries that recovered payment"
            }
          />
          <StatCard
            label="Success Rate"
            value={overview?.success_rate ? `${(overview.success_rate * 100).toFixed(1)}%` : "0%"}
            icon={Percent}
            hint={
              overview?.total_retries
                ? `${formatNumber(overview.total_successes || 0)} of ${formatNumber(overview.total_retries)} succeeded`
                : "no retries yet"
            }
            definition="Share of individual retry attempts that recovered a payment — successful recoveries ÷ total retries. Per-attempt, not per-invoice (one invoice may take several retries)."
          />
        </div>
      )}

      {/* Recovered Revenue by Month */}
      <Card className="mt-6">
        <CardHeader>
          <CardTitle className="text-base">Recovered Revenue by Month</CardTitle>
          <CardDescription>
            Revenue attributed to the retry/dunning engine over the last 12 months
            {` (in ${primaryCurrency})`}
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
              <div role="img" aria-label="Recovered revenue by month">
              <BarChart
                {...chartDefaults}
                className="h-72"
                data={chartData}
                index="month"
                categories={["Recovered"]}
                colors={chartCategoryColors}
                valueFormatter={currencyFormatter}
                customTooltip={chartTooltip}
                showLegend={false}
                showGridLines
                yAxisWidth={64}
              />
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Arm Performance by Context */}
      <Card className="mt-6">
        <CardHeader>
          <CardTitle className="text-base">Retry timing performance</CardTitle>
          <CardDescription>
            Each segment (currency &middot; failure code) independently learns which retry
            interval recovers most &mdash; a multi-armed bandit over your real outcomes.
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
                  <TableHead className="pl-6">Segment</TableHead>
                  <TableHead>Retry timing</TableHead>
                  <TableHead className="text-right">Success rate</TableHead>
                  <TableHead className="text-right">Attempts</TableHead>
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
                          className="pl-6 text-sm text-muted-foreground align-top"
                          rowSpan={arms.length}
                        >
                          {segmentLabel(contextKey)}
                        </TableCell>
                      )}
                      <TableCell className="font-mono text-sm text-foreground">
                        {arm.action_id}
                      </TableCell>
                      <TableCell className="text-right tabular-nums">
                        <span
                          className={
                            arm.average_reward > 0.5
                              ? "text-success"
                              : arm.average_reward > 0.2
                                ? "text-warning"
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
                  <TableHead>Segment</TableHead>
                  <TableHead>Retry timing</TableHead>
                  <TableHead className="pr-6">Outcome</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {history.map((h) => (
                  <TableRow key={h.id} className="hover:bg-transparent">
                    <TableCell className="pl-6 whitespace-nowrap text-sm text-muted-foreground">
                      {fmtWhen(h.created_at)}
                    </TableCell>
                    <TableCell className="font-mono text-sm" title={h.invoice_id}>
                      {h.invoice_id ? (
                        <Link
                          to={`/invoices/${h.invoice_id}`}
                          className="text-success hover:text-primary hover:underline"
                        >
                          {h.invoice_id.substring(0, 8)}…
                        </Link>
                      ) : (
                        <span className="text-muted-foreground">—</span>
                      )}
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {segmentLabel(h.context_key)}
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

      {/* Best time to retry (read-only insight from historical outcomes) */}
      <Card className="mt-6">
        <CardHeader>
          <CardTitle className="text-base">Best time to retry</CardTitle>
          <CardDescription>
            Historical retry success rate by hour of day (UTC). Insight only — the
            live bandit is unchanged.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {loading ? (
            <Skeleton className="h-40 w-full" />
          ) : !timingHasData ? (
            <EmptyState
              icon={BarChart3}
              title="Not enough history yet"
              description="Once retries accumulate, the best hours and days to retry appear here."
            />
          ) : (
            <div className="space-y-4">
              <div className="flex flex-wrap gap-2 text-sm">
                {timing?.best_hour != null && (
                  <Badge variant="success">Best hour: {fmtHour(timing.best_hour)} UTC</Badge>
                )}
                {timing?.best_day != null && (
                  <Badge variant="success">Best day: {DAY_NAMES[timing.best_day]}</Badge>
                )}
                <span className="text-muted-foreground">
                  {formatNumber(timing?.sample_size || 0)} retries analyzed
                </span>
              </div>
              {/* 24 hourly bars; height encodes success rate, best hour emphasized. */}
              <div className="flex h-32 items-end gap-1" role="img" aria-label="Retry success rate by hour of day">
                {hourRates.map((h) => (
                  <div key={h.bucket} className="flex flex-1 flex-col items-center gap-1" title={`${fmtHour(h.bucket)} — ${(h.success_rate * 100).toFixed(0)}% of ${h.total}`}>
                    <div
                      className={
                        "w-full rounded-t " +
                        (h.bucket === timing?.best_hour ? "bg-success/50" : "bg-muted-foreground/30")
                      }
                      style={{ height: `${maxHourRate > 0 ? Math.max(2, (h.success_rate / maxHourRate) * 100) : 2}%` }}
                    />
                    {h.bucket % 6 === 0 && (
                      <span className="text-[10px] text-muted-foreground">{h.bucket}</span>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
};

export default DunningDashboard;
