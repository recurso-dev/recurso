import { useMemo, useState } from "react";
import { Link } from "react-router";
import { useQuery } from "@tanstack/react-query";
import { BarChart } from "@tremor/react";
import { TrendingUp } from "lucide-react";

import { makeChartTooltip, chartCategoryColors, chartDefaults } from "@/components/charts/ChartTooltip";

import { endpoints } from "../lib/api";
import { PageHeader } from "@/components/patterns/PageHeader";
import { StatCard } from "@/components/patterns/StatCard";
import { EmptyState } from "@/components/patterns/EmptyState";
import { ErrorState } from "@/components/patterns/ErrorState";
import { CardGridSkeleton } from "@/components/patterns/LoadingSkeleton";
import { Card } from "@/components/ui/card";
import { formatCurrency, fromMinorUnits } from "@/lib/utils";
import { Overline } from "@/components/ui/overline";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const MONTHS = [
  "January", "February", "March", "April", "May", "June",
  "July", "August", "September", "October", "November", "December",
];
const monthLabel = (m, y) => `${MONTHS[m - 1] || "—"} ${y}`;

export default function RevenueWaterfall() {
  const now = new Date();
  const [month, setMonth] = useState(now.getMonth() + 1);
  const [year, setYear] = useState(now.getFullYear());

  // The recognition curve is period-independent; the deferred rollforward is
  // keyed by the selected month/year (its own cache entry, refetched on change).
  const waterfallQuery = useQuery({
    queryKey: ["revenue-waterfall"],
    queryFn: async () => (await endpoints.getRevenueWaterfall()).data?.data || null,
  });
  const rollforwardQuery = useQuery({
    queryKey: ["deferred-rollforward", month, year],
    queryFn: async () => (await endpoints.getDeferredRollforward(month, year)).data?.data || null,
  });
  const waterfall = waterfallQuery.data;
  const rollforward = rollforwardQuery.data;
  const loading = waterfallQuery.isLoading || rollforwardQuery.isLoading;
  const queryError = waterfallQuery.error || rollforwardQuery.error;
  const error = queryError
    ? queryError?.response?.data?.error?.message || "Failed to load the revenue waterfall"
    : null;
  const load = () => {
    waterfallQuery.refetch();
    rollforwardQuery.refetch();
  };

  // Reporting currency (tenant base currency) for exponent-correct formatting.
  const cur = waterfall?.reporting_currency || "USD";
  const money = (minor) => formatCurrency(minor, cur);

  const buckets = waterfall?.buckets || [];
  // The chart is the waterfall — recognized stacked against scheduled, month by
  // month, in major units for the axis.
  const chartData = buckets.map((b) => ({
    month: `${(MONTHS[b.month - 1] || "").slice(0, 3)} ${String(b.year).slice(2)}`,
    Recognized: fromMinorUnits(b.recognized || 0, cur),
    Scheduled: fromMinorUnits(b.scheduled || 0, cur),
  }));
  const axisFormatter = (v) => {
    try {
      return new Intl.NumberFormat("en-US", {
        style: "currency",
        currency: cur,
        maximumFractionDigits: 0,
        notation: "compact",
      }).format(v);
    } catch {
      return `${cur} ${v}`;
    }
  };
  // eslint-disable-next-line react-hooks/exhaustive-deps
  const chartTooltip = useMemo(() => makeChartTooltip(axisFormatter), [cur]);
  const years = [];
  for (let y = now.getFullYear() - 3; y <= now.getFullYear() + 1; y++) years.push(y);

  return (
    <div>
      <PageHeader
        title="Revenue Waterfall"
        description={
          <>
            The recognition curve — revenue already recognized and revenue still scheduled — plus
            the deferred-revenue rollforward for a chosen month. For one month&apos;s close detail
            and its ledger postings, see{" "}
            <Link
              to="/finance/revenue-recognition"
              className="text-primary underline-offset-2 hover:underline"
            >
              Revenue Recognition
            </Link>
            .
          </>
        }
        actions={
          <div className="flex items-center gap-2">
            <Select value={String(month)} onValueChange={(v) => setMonth(Number(v))}>
              <SelectTrigger className="w-[130px]" aria-label="Month">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {MONTHS.map((m, i) => (
                  <SelectItem key={m} value={String(i + 1)}>{m}</SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Select value={String(year)} onValueChange={(v) => setYear(Number(v))}>
              <SelectTrigger className="w-[92px]" aria-label="Year">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {years.map((y) => (
                  <SelectItem key={y} value={String(y)}>{y}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        }
      />

      {loading ? (
        <CardGridSkeleton count={3} />
      ) : error ? (
        <Card className="overflow-hidden">
          <ErrorState message={error} onRetry={load} />
        </Card>
      ) : (
        <div className="flex flex-col gap-6">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            <StatCard
              label="Total recognized"
              value={money(waterfall?.total_recognized || 0)}
              hint="Booked as earned across all periods"
            />
            <StatCard
              label="Total scheduled"
              value={money(waterfall?.total_scheduled || 0)}
              hint="Still to recognize"
            />
            <StatCard
              label="Months on the curve"
              value={(buckets.length || 0).toLocaleString()}
              hint="Distinct recognition months"
            />
          </div>

          {/* Deferred-revenue rollforward for the selected month. */}
          {rollforward && (
            <Card className="overflow-hidden">
              <div className="border-b border-border px-6 py-4">
                <h2 className="text-base font-semibold text-foreground">
                  Deferred rollforward — {monthLabel(month, year)}
                </h2>
                <p className="text-sm text-muted-foreground">
                  How the Deferred Revenue balance moved: opening + added − released = closing.
                </p>
              </div>
              <div className="grid grid-cols-2 gap-px bg-border sm:grid-cols-4">
                {[
                  ["Opening", rollforward.opening],
                  ["+ Added", rollforward.added],
                  ["− Released", rollforward.released],
                  ["Closing", rollforward.closing],
                ].map(([label, val]) => (
                  <div key={label} className="bg-card px-6 py-4">
                    <Overline>{label}</Overline>
                    <div className="mt-1 font-mono text-lg tabular-nums text-foreground">{money(val)}</div>
                  </div>
                ))}
              </div>
            </Card>
          )}

          {/* The waterfall itself — recognized vs scheduled, stacked by month. */}
          {buckets.length > 0 && (
            <Card>
              <div className="px-6 pt-6">
                <h2 className="text-base font-semibold text-foreground">Waterfall</h2>
                <p className="text-sm text-muted-foreground">
                  Recognized revenue (historical) stacked with scheduled releases (future).
                </p>
              </div>
              <div className="p-6" data-testid="waterfall-chart">
                <div role="img" aria-label="Revenue waterfall by component">
                <BarChart
                  {...chartDefaults}
                  className="h-72"
                  data={chartData}
                  index="month"
                  categories={["Recognized", "Scheduled"]}
                  colors={chartCategoryColors}
                  valueFormatter={axisFormatter}
                  customTooltip={chartTooltip}
                  stack
                  showLegend
                  showGridLines
                  yAxisWidth={64}
                />
                </div>
              </div>
            </Card>
          )}

          {/* The month-by-month recognition curve. */}
          <Card className="overflow-hidden">
            <div className="border-b border-border px-6 py-4">
              <h2 className="text-base font-semibold text-foreground">Recognition curve</h2>
              <p className="text-sm text-muted-foreground">
                Recognized (historical) and scheduled (future) revenue, by month.
              </p>
            </div>
            {buckets.length === 0 ? (
              <EmptyState
                icon={TrendingUp}
                title="No recognition schedule yet"
                description="Invoice for a period — say an annual plan — and its revenue schedule appears here as a month-by-month curve."
              />
            ) : (
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow className="bg-muted/40 hover:bg-muted/40">
                      <TableHead>Month</TableHead>
                      <TableHead className="text-right">Recognized</TableHead>
                      <TableHead className="text-right">Scheduled</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {buckets.map((b) => (
                      <TableRow key={`${b.year}-${b.month}`}>
                        <TableCell className="text-foreground">{monthLabel(b.month, b.year)}</TableCell>
                        <TableCell className="text-right font-mono text-sm tabular-nums text-foreground">
                          {money(b.recognized)}
                        </TableCell>
                        <TableCell className="text-right font-mono text-sm tabular-nums text-muted-foreground">
                          {money(b.scheduled)}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
          </Card>
        </div>
      )}
    </div>
  );
}
