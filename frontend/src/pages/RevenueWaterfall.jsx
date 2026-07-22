import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { TrendingUp } from "lucide-react";

import { endpoints } from "../lib/api";
import { PageHeader } from "@/components/patterns/PageHeader";
import { StatCard } from "@/components/patterns/StatCard";
import { EmptyState } from "@/components/patterns/EmptyState";
import { ErrorState } from "@/components/patterns/ErrorState";
import { CardGridSkeleton } from "@/components/patterns/LoadingSkeleton";
import { Card } from "@/components/ui/card";
import { formatCurrency } from "@/lib/utils";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

const MONTHS = [
  "January", "February", "March", "April", "May", "June",
  "July", "August", "September", "October", "November", "December",
];
const monthLabel = (m, y) => `${MONTHS[m - 1] || "—"} ${y}`;

const selectClass =
  "rounded-md border border-border bg-background px-2.5 py-1.5 text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring";

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
  const years = [];
  for (let y = now.getFullYear() - 3; y <= now.getFullYear() + 1; y++) years.push(y);

  return (
    <div>
      <PageHeader
        title="Revenue Waterfall"
        description="The recognition curve — revenue already recognized and revenue still scheduled — plus the deferred-revenue rollforward for a chosen month."
        actions={
          <div className="flex items-center gap-2">
            <label className="sr-only" htmlFor="wf-month">Month</label>
            <select
              id="wf-month"
              className={selectClass}
              value={month}
              onChange={(e) => setMonth(Number(e.target.value))}
            >
              {MONTHS.map((m, i) => (
                <option key={m} value={i + 1}>{m}</option>
              ))}
            </select>
            <label className="sr-only" htmlFor="wf-year">Year</label>
            <select
              id="wf-year"
              className={selectClass}
              value={year}
              onChange={(e) => setYear(Number(e.target.value))}
            >
              {years.map((y) => (
                <option key={y} value={y}>{y}</option>
              ))}
            </select>
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
                    <div className="text-xs uppercase tracking-wide text-muted-foreground">{label}</div>
                    <div className="mt-1 font-mono text-lg tabular-nums text-foreground">{money(val)}</div>
                  </div>
                ))}
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
