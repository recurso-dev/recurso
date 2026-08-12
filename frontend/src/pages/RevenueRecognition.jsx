import { useState } from "react";
import { Link } from "react-router";
import { useQuery } from "@tanstack/react-query";
import { CalendarClock, Coins } from "lucide-react";

import { endpoints } from "../lib/api";
import { formatCurrency } from "@/lib/utils";
import { PageHeader } from "@/components/patterns/PageHeader";
import { StatCard } from "@/components/patterns/StatCard";
import { EmptyState } from "@/components/patterns/EmptyState";
import { ErrorState } from "@/components/patterns/ErrorState";
import { CardGridSkeleton } from "@/components/patterns/LoadingSkeleton";
import { Card } from "@/components/ui/card";
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

export default function RevenueRecognition() {
  const now = new Date();
  const [month, setMonth] = useState(now.getMonth() + 1);
  const [year, setYear] = useState(now.getFullYear());

  const {
    data: report,
    isLoading: loading,
    error: queryError,
    refetch,
  } = useQuery({
    queryKey: ["revenue-recognition", month, year],
    queryFn: async () => (await endpoints.getRevenueRecognition(month, year)).data?.data || null,
  });
  const error = queryError
    ? queryError?.response?.data?.error?.message || "Failed to load the revenue-recognition report"
    : null;

  const byCurrency = report?.by_currency || [];
  const upcoming = report?.upcoming || [];
  const multiCurrency = byCurrency.length > 1;
  const primaryCurrency = byCurrency[0]?.currency || "USD";

  // When more than one currency is deferred, the cross-currency sums (deferred
  // balance, recognized total, schedule buckets) can't honestly carry a single
  // symbol — show them as grouped minor units and let the by-currency card
  // carry the real, per-currency numbers.
  const fmt = (minor) =>
    multiCurrency
      ? `${(Number(minor) || 0).toLocaleString()}`
      : formatCurrency(minor, primaryCurrency);

  const hasData =
    report &&
    ((report.deferred_balance || 0) > 0 ||
      (report.recognized_amount || 0) > 0 ||
      upcoming.length > 0);

  const years = [];
  for (let y = now.getFullYear() - 3; y <= now.getFullYear() + 1; y++) years.push(y);

  return (
    <div>
      <PageHeader
        title="Revenue Recognition"
        description={
          <>
            Deferred revenue still on the books, and the schedule for when it recognizes. For the
            recognized-vs-scheduled curve across all months, see the{" "}
            <Link
              to="/finance/revenue-waterfall"
              className="text-primary underline-offset-2 hover:underline"
            >
              Revenue Waterfall
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
          <ErrorState message={error} onRetry={refetch} />
        </Card>
      ) : (
        report && (
          <div className="flex flex-col gap-6">
            {multiCurrency && (
              <p className="text-xs text-muted-foreground">
                Deferred across {byCurrency.length} currencies — combined totals
                are shown in minor units; see the per-currency breakdown below.
              </p>
            )}

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
              <Link
                to="/ledger?account_code=4100&code=2"
                className="block"
                title="View the recognition postings in the Ledger"
              >
                <StatCard
                  label={`Recognized in ${monthLabel(month, year)}`}
                  value={fmt(report.recognized_amount || 0)}
                  hint="Earned revenue booked this period — click for the postings"
                />
              </Link>
              <Link
                to="/ledger?account_code=2100"
                className="block"
                title="View the Deferred Revenue account in the Ledger"
              >
                <StatCard
                  label="Deferred balance"
                  value={fmt(report.deferred_balance || 0)}
                  hint="Unearned revenue still to recognize — click for the account"
                />
              </Link>
              <StatCard
                label="Currencies deferred"
                value={(byCurrency.length || 0).toLocaleString()}
                hint={byCurrency.map((c) => c.currency).join(" · ") || "None"}
              />
            </div>

            {!hasData ? (
              <Card className="overflow-hidden">
                <EmptyState
                  icon={CalendarClock}
                  title="No recognition schedules yet"
                  description="When you invoice for a period — say an annual plan — Recurso schedules that revenue over its term, and the release schedule appears here."
                />
              </Card>
            ) : (
              <div className="grid gap-6 lg:grid-cols-2">
                {/* Release schedule */}
                <Card className="overflow-hidden">
                  <div className="border-b border-border px-6 py-4">
                    <h2 className="text-base font-semibold text-foreground">
                      Release schedule
                    </h2>
                    <p className="text-sm text-muted-foreground">
                      When the deferred balance is scheduled to recognize.
                    </p>
                  </div>
                  {upcoming.length === 0 ? (
                    <EmptyState
                      icon={CalendarClock}
                      title="Nothing scheduled"
                      description="The deferred balance has no future recognition events."
                    />
                  ) : (
                    <Table>
                      <TableHeader>
                        <TableRow className="bg-muted/40 hover:bg-muted/40">
                          <TableHead>Month</TableHead>
                          <TableHead className="text-right">To recognize</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {(() => {
                          const max = Math.max(...upcoming.map((b) => b.amount || 0), 1);
                          return upcoming.map((b) => (
                            <TableRow key={`${b.year}-${b.month}`}>
                              <TableCell className="text-foreground">
                                <div className="flex items-center gap-3">
                                  <span className="w-32 shrink-0">{monthLabel(b.month, b.year)}</span>
                                  <div className="hidden h-1.5 flex-1 overflow-hidden rounded-full bg-muted sm:block">
                                    <div
                                      className="h-full rounded-full bg-success/50/70"
                                      style={{ width: `${Math.max(2, ((b.amount || 0) / max) * 100)}%` }}
                                    />
                                  </div>
                                </div>
                              </TableCell>
                              <TableCell className="text-right font-mono text-sm tabular-nums text-foreground">
                                {fmt(b.amount || 0)}
                              </TableCell>
                            </TableRow>
                          ));
                        })()}
                      </TableBody>
                    </Table>
                  )}
                </Card>

                {/* By currency */}
                <Card className="overflow-hidden">
                  <div className="border-b border-border px-6 py-4">
                    <h2 className="text-base font-semibold text-foreground">
                      Deferred by currency
                    </h2>
                    <p className="text-sm text-muted-foreground">
                      The still-deferred balance in each schedule's own currency.
                    </p>
                  </div>
                  {byCurrency.length === 0 ? (
                    <EmptyState
                      icon={Coins}
                      title="No deferred balance"
                      description="There is no unearned revenue on the books right now."
                    />
                  ) : (
                    <Table>
                      <TableHeader>
                        <TableRow className="bg-muted/40 hover:bg-muted/40">
                          <TableHead>Currency</TableHead>
                          <TableHead className="text-right">Deferred</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {byCurrency.map((c) => (
                          <TableRow key={c.currency}>
                            <TableCell className="font-mono text-sm text-foreground">
                              {c.currency}
                            </TableCell>
                            <TableCell className="text-right font-mono text-sm tabular-nums text-foreground">
                              {formatCurrency(c.deferred || 0, c.currency)}
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  )}
                </Card>
              </div>
            )}
          </div>
        )
      )}
    </div>
  );
}
