import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { toast } from "@/components/ui/sonner";
import {
  ClipboardCheck,
  CheckCircle2,
  AlertTriangle,
  Download,
  FileJson,
  Scale,
  ArrowRight,
} from "lucide-react";

import { endpoints } from "../lib/api";
import { PageHeader } from "@/components/patterns/PageHeader";
import { StatCard } from "@/components/patterns/StatCard";
import { ErrorState } from "@/components/patterns/ErrorState";
import { CardGridSkeleton } from "@/components/patterns/LoadingSkeleton";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
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

export default function MonthEndClose() {
  const now = new Date();
  const [month, setMonth] = useState(now.getMonth() + 1);
  const [year, setYear] = useState(now.getFullYear());
  const [pack, setPack] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [exporting, setExporting] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await endpoints.getClosePack(month, year);
      setPack(res.data?.data || null);
    } catch (err) {
      setError(
        err?.response?.data?.error?.message || "Failed to build the close pack",
      );
    } finally {
      setLoading(false);
    }
  }, [month, year]);

  useEffect(() => {
    load();
  }, [load]);

  const exportGL = async () => {
    setExporting(true);
    try {
      const res = await endpoints.exportGeneralLedger();
      const url = URL.createObjectURL(new Blob([res.data], { type: "text/csv" }));
      const a = document.createElement("a");
      a.href = url;
      a.download = `general-ledger-${year}-${String(month).padStart(2, "0")}.csv`;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
    } catch {
      toast.error("Failed to export the general ledger");
    } finally {
      setExporting(false);
    }
  };

  // The close pack itself is already in hand — serialize it client-side so the
  // full evidence bundle (trial balance, reconciliation, rollforward) is one
  // downloadable artifact. Print-to-PDF covers the signed-copy case.
  const downloadPack = () => {
    if (!pack) return;
    const url = URL.createObjectURL(
      new Blob([JSON.stringify(pack, null, 2)], { type: "application/json" }),
    );
    const a = document.createElement("a");
    a.href = url;
    a.download = `close-pack-${year}-${String(month).padStart(2, "0")}.json`;
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(url);
  };

  const years = [];
  for (let y = now.getFullYear() - 3; y <= now.getFullYear() + 1; y++) years.push(y);

  const tb = pack?.trial_balance;
  const recon = pack?.reconciliation;
  const rollforward = pack?.deferred_revenue?.rollforward;
  const recognition = pack?.deferred_revenue?.recognition;
  const ties = pack?.deferred_revenue?.ties;
  // Reporting currency (tenant base currency) for exponent-correct formatting.
  const cur = pack?.reporting_currency || "USD";
  const money = (minor) => formatCurrency(minor, cur);
  const blockers = pack?.blockers || [];

  return (
    <div>
      <PageHeader
        title="Month-End Close"
        description="One evidence pack per period: the books balance, billing ties to the ledger, and deferred revenue rolls forward."
        actions={
          <div className="flex items-center gap-2">
            <label className="sr-only" htmlFor="close-month">Month</label>
            <select
              id="close-month"
              className={selectClass}
              value={month}
              onChange={(e) => setMonth(Number(e.target.value))}
            >
              {MONTHS.map((m, i) => (
                <option key={m} value={i + 1}>{m}</option>
              ))}
            </select>
            <label className="sr-only" htmlFor="close-year">Year</label>
            <select
              id="close-year"
              className={selectClass}
              value={year}
              onChange={(e) => setYear(Number(e.target.value))}
            >
              {years.map((y) => (
                <option key={y} value={y}>{y}</option>
              ))}
            </select>
            <Button variant="outline" onClick={exportGL} disabled={exporting}>
              <Download className="h-4 w-4" />
              {exporting ? "Exporting…" : "GL (CSV)"}
            </Button>
            <Button variant="outline" onClick={downloadPack} disabled={!pack}>
              <FileJson className="h-4 w-4" />
              Pack (JSON)
            </Button>
          </div>
        }
      />

      {loading ? (
        <CardGridSkeleton count={4} />
      ) : error ? (
        <Card className="overflow-hidden">
          <ErrorState message={error} onRetry={load} />
        </Card>
      ) : (
        pack && (
          <div className="flex flex-col gap-6">
            {/* Ready-to-close verdict — the headline of the page. */}
            {pack.ready_to_close ? (
              <div className="flex items-start gap-3 rounded-lg border border-emerald-600/30 bg-emerald-600/10 px-4 py-3 text-emerald-700 dark:text-emerald-400">
                <CheckCircle2 className="mt-0.5 h-5 w-5 flex-shrink-0" />
                <div>
                  <p className="text-sm font-semibold">
                    {monthLabel(month, year)} is ready to close
                  </p>
                  <p className="text-sm">
                    The trial balance is in balance and reconciliation found no
                    discrepancies.
                  </p>
                </div>
              </div>
            ) : (
              <div className="flex items-start gap-3 rounded-lg border border-red-600/30 bg-red-600/10 px-4 py-3 text-red-700 dark:text-red-400">
                <AlertTriangle className="mt-0.5 h-5 w-5 flex-shrink-0" />
                <div>
                  <p className="text-sm font-semibold">
                    {monthLabel(month, year)} is not ready to close
                  </p>
                  <ul className="mt-1 list-inside list-disc text-sm">
                    {blockers.map((b) => (
                      <li key={b}>{b}</li>
                    ))}
                  </ul>
                </div>
              </div>
            )}

            {/* Summary cards */}
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
              <StatCard
                label="Trial balance"
                value={tb?.balanced ? "Balanced" : "Unbalanced"}
                hint={`Dr ${money(tb?.total_debits)} · Cr ${money(tb?.total_credits)}`}
              />
              <StatCard
                label="Reconciliation"
                value={(recon?.total_discrepancies || 0).toLocaleString()}
                hint={
                  (recon?.total_discrepancies || 0) === 0
                    ? `${(recon?.invoices_checked || 0).toLocaleString()} invoices agree`
                    : "discrepancies to resolve"
                }
              />
              <StatCard
                label="Deferred revenue (closing)"
                value={money(rollforward?.closing)}
                hint={`Opening ${money(rollforward?.opening)} + added ${money(rollforward?.added)} − released ${money(rollforward?.released)}`}
              />
              <Card className="p-5">
                <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                  Deferred tie-out
                </p>
                <div className="mt-3">
                  {recognition == null ? (
                    <Badge variant="secondary">Rev-rec not wired</Badge>
                  ) : ties ? (
                    <Badge variant="success">
                      <CheckCircle2 className="h-3.5 w-3.5" />
                      Ledger = schedule
                    </Badge>
                  ) : (
                    <Badge variant="warning">
                      <AlertTriangle className="h-3.5 w-3.5" />
                      Ledger ≠ schedule
                    </Badge>
                  )}
                </div>
                <p className="mt-2 text-xs text-muted-foreground">
                  {recognition == null
                    ? "Only the ledger rollforward is shown."
                    : `Schedule deferred balance ${money(recognition.deferred_balance)}`}
                </p>
              </Card>
            </div>

            {/* Deferred-revenue rollforward */}
            <Card className="overflow-hidden">
              <div className="border-b border-border px-6 py-4">
                <h2 className="text-base font-semibold text-foreground">
                  Deferred revenue rollforward
                </h2>
                <p className="text-sm text-muted-foreground">
                  Movement of the Deferred Revenue account across {monthLabel(month, year)}.
                </p>
              </div>
              <Table>
                <TableHeader>
                  <TableRow className="bg-muted/40 hover:bg-muted/40">
                    <TableHead>Opening</TableHead>
                    <TableHead className="text-right">Added</TableHead>
                    <TableHead className="text-right">Released</TableHead>
                    <TableHead className="text-right">Closing</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  <TableRow>
                    <TableCell className="font-mono text-sm tabular-nums">{money(rollforward?.opening)}</TableCell>
                    <TableCell className="text-right font-mono text-sm tabular-nums text-emerald-600 dark:text-emerald-400">
                      +{money(rollforward?.added)}
                    </TableCell>
                    <TableCell className="text-right font-mono text-sm tabular-nums text-amber-600 dark:text-amber-400">
                      −{money(rollforward?.released)}
                    </TableCell>
                    <TableCell className="text-right font-mono text-sm font-semibold tabular-nums">
                      {money(rollforward?.closing)}
                    </TableCell>
                  </TableRow>
                </TableBody>
              </Table>
            </Card>

            {/* Trial-balance detail */}
            <Card className="overflow-hidden">
              <div className="flex items-center justify-between border-b border-border px-6 py-4">
                <div>
                  <h2 className="text-base font-semibold text-foreground">Trial balance</h2>
                  <p className="text-sm text-muted-foreground">
                    Every account's posted totals; a wrong-sign balance is flagged.
                  </p>
                </div>
                <Link
                  to="/finance/reconciliation"
                  className="inline-flex items-center gap-1.5 text-sm font-medium text-primary hover:underline"
                >
                  Reconciliation detail <ArrowRight className="h-4 w-4" />
                </Link>
              </div>
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow className="bg-muted/40 hover:bg-muted/40">
                      <TableHead>Account</TableHead>
                      <TableHead className="text-right">Debits</TableHead>
                      <TableHead className="text-right">Credits</TableHead>
                      <TableHead className="text-right">Balance</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {(tb?.lines || []).map((l) => (
                      <TableRow key={l.account_id}>
                        <TableCell className="text-foreground">
                          <span className="font-mono text-xs text-muted-foreground">{l.code}</span>{" "}
                          {l.name}
                          {l.abnormal && (
                            <Badge variant="destructive" className="ml-2">abnormal</Badge>
                          )}
                        </TableCell>
                        <TableCell className="text-right font-mono text-sm tabular-nums">{money(l.debits)}</TableCell>
                        <TableCell className="text-right font-mono text-sm tabular-nums">{money(l.credits)}</TableCell>
                        <TableCell
                          className={`text-right font-mono text-sm tabular-nums ${l.abnormal ? "text-red-600 dark:text-red-400" : "text-foreground"}`}
                        >
                          {money(l.balance)}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
              {(tb?.lines || []).length === 0 && (
                <div className="flex items-center gap-2 px-6 py-8 text-sm text-muted-foreground">
                  <Scale className="h-4 w-4" />
                  No ledger activity yet for this tenant.
                </div>
              )}
            </Card>

            <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
              <ClipboardCheck className="h-3.5 w-3.5" />
              Generated {pack.generated_at ? new Date(pack.generated_at).toLocaleString() : "—"}.
              Nothing is persisted — closing the period stays your decision.
            </p>
          </div>
        )
      )}
    </div>
  );
}
