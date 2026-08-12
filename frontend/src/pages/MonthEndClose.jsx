import { useState } from "react";
import { useNavigate } from "react-router";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router";
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

export default function MonthEndClose() {
  const navigate = useNavigate();
  const now = new Date();
  const [month, setMonth] = useState(now.getMonth() + 1);
  const [year, setYear] = useState(now.getFullYear());
  const [exporting, setExporting] = useState(false);

  const {
    data: pack,
    isLoading: loading,
    error: queryError,
    refetch,
  } = useQuery({
    queryKey: ["close-pack", month, year],
    queryFn: async () => (await endpoints.getClosePack(month, year)).data?.data || null,
  });
  const error = queryError
    ? queryError?.response?.data?.error?.message || "Failed to build the close pack"
    : null;

  // Scope the export to the period being closed — the file is named by month,
  // so it must contain exactly that month's postings.
  const exportGL = async () => {
    setExporting(true);
    try {
      const res = await endpoints.exportGeneralLedger(undefined, { month, year });
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

  const tbRaw = pack?.trial_balance;
  // Same rollup as the Trial Balance page: per-customer AR sub-accounts share a
  // name+code, so collapse identical lines and keep the member count.
  const tb = tbRaw
    ? {
        ...tbRaw,
        lines: (() => {
          const byKey = new Map();
          for (const l of tbRaw.lines || []) {
            const key = `${l.code}|${l.name}`;
            const agg = byKey.get(key);
            if (!agg) byKey.set(key, { ...l, sub_count: 1 });
            else {
              agg.debits += l.debits;
              agg.credits += l.credits;
              agg.balance += l.balance;
              agg.abnormal = agg.abnormal || l.abnormal;
              agg.sub_count += 1;
            }
          }
          return [...byKey.values()].sort((a, b) => a.code - b.code);
        })(),
      }
    : tbRaw;
  const recon = pack?.reconciliation;
  const rollforward = pack?.deferred_revenue?.rollforward;
  const recognition = pack?.deferred_revenue?.recognition;
  const ties = pack?.deferred_revenue?.ties;
  const awaiting = pack?.deferred_revenue?.awaiting_payment || 0;
  const unexplained = pack?.deferred_revenue?.unexplained_delta || 0;
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
          <ErrorState message={error} onRetry={refetch} />
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
                      Deferred ties out
                    </Badge>
                  ) : (
                    <Badge variant="warning">
                      <AlertTriangle className="h-3.5 w-3.5" />
                      {money(Math.abs(unexplained))} to reconcile
                    </Badge>
                  )}
                </div>
                {recognition == null ? (
                  <p className="mt-2 text-xs text-muted-foreground">
                    Only the ledger rollforward is shown.
                  </p>
                ) : (
                  <div className="mt-3 space-y-1 text-xs text-muted-foreground">
                    <div className="flex justify-between">
                      <span>Recognizing on schedule</span>
                      <span className="tabular-nums text-foreground">
                        {money(recognition.deferred_balance)}
                      </span>
                    </div>
                    <div className="flex justify-between">
                      <span>Awaiting payment</span>
                      <span className="tabular-nums text-foreground">
                        {money(awaiting)}
                      </span>
                    </div>
                    {!ties && (
                      <div className="flex justify-between">
                        <span>To reconcile</span>
                        <span className="tabular-nums text-foreground">
                          {money(unexplained)}
                        </span>
                      </div>
                    )}
                    <p className="pt-1 leading-snug">
                      Revenue is deferred when an invoice is issued; its
                      recognition schedule is built when the invoice is paid — so
                      unpaid invoices sit in <em>awaiting payment</em> until they
                      settle.
                    </p>
                  </div>
                )}
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
                      <TableRow
                        key={l.account_id}
                        className={l.sub_count === 1 ? "cursor-pointer" : undefined}
                        title={l.sub_count === 1 ? "View this account's postings in the Ledger" : undefined}
                        onClick={
                          l.sub_count === 1
                            ? () => navigate(`/ledger?account_id=${l.account_id}`)
                            : undefined
                        }
                      >
                        <TableCell className="text-foreground">
                          <span className="font-mono text-xs text-muted-foreground">{l.code}</span>{" "}
                          {l.name}
                          {l.sub_count > 1 && (
                            <span className="ml-1.5 text-xs text-muted-foreground">· {l.sub_count} sub-accounts</span>
                          )}
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
              Generated {pack.generated_at ? new Date(pack.generated_at).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" }) : "—"}.
              Nothing is persisted — closing the period stays your decision.
            </p>
          </div>
        )
      )}
    </div>
  );
}
