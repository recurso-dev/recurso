import { useCallback, useEffect, useState } from "react";
import { toast } from "sonner";
import { Scale, Download, AlertTriangle, CheckCircle2 } from "lucide-react";

import { endpoints } from "../lib/api";
import { PageHeader } from "@/components/patterns/PageHeader";
import { StatCard } from "@/components/patterns/StatCard";
import { EmptyState } from "@/components/patterns/EmptyState";
import { ErrorState } from "@/components/patterns/ErrorState";
import { CardGridSkeleton } from "@/components/patterns/LoadingSkeleton";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

// Trial-balance amounts are summed across a tenant's accounts (which may span
// currencies), so we show major units without asserting a single symbol.
const money = (minor) =>
  (Number(minor || 0) / 100).toLocaleString(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });

const typeLabel = (t) =>
  ({ 1: "Asset", 2: "Liability", 3: "Equity", 4: "Revenue", 5: "Expense" }[t] || "—");

export default function TrialBalance() {
  const [tb, setTb] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [exporting, setExporting] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await endpoints.getTrialBalance();
      setTb(res.data?.data || null);
    } catch (err) {
      setError(
        err?.response?.data?.error?.message || "Failed to load the trial balance",
      );
    } finally {
      setLoading(false);
    }
  }, []);

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
      a.download = "general-ledger.csv";
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

  const lines = tb?.lines || [];
  const abnormal = lines.filter((l) => l.abnormal);

  return (
    <div>
      <PageHeader
        title="Trial Balance"
        description="Every account's posted totals, with the double-entry invariant — debits must equal credits."
        actions={
          <Button variant="outline" onClick={exportGL} disabled={exporting}>
            <Download className="h-4 w-4" />
            {exporting ? "Exporting…" : "Export GL (CSV)"}
          </Button>
        }
      />

      {loading ? (
        <CardGridSkeleton count={3} />
      ) : error ? (
        <Card className="overflow-hidden">
          <ErrorState message={error} onRetry={load} />
        </Card>
      ) : (
        tb && (
          <div className="flex flex-col gap-6">
            {/* Integrity banner — the standing double-entry assertion. */}
            {tb.balanced && abnormal.length === 0 ? (
              <div className="flex items-center gap-2 rounded-lg border border-emerald-600/30 bg-emerald-600/10 px-4 py-3 text-sm text-emerald-700 dark:text-emerald-400">
                <CheckCircle2 className="h-4 w-4" />
                Books balance — total debits equal total credits, and every account
                carries its expected sign.
              </div>
            ) : (
              <div className="flex items-center gap-2 rounded-lg border border-red-600/30 bg-red-600/10 px-4 py-3 text-sm text-red-700 dark:text-red-400">
                <AlertTriangle className="h-4 w-4" />
                {!tb.balanced
                  ? "Out of balance — total debits do not equal total credits."
                  : `${abnormal.length} account${abnormal.length > 1 ? "s carry" : " carries"} a wrong-sign balance.`}
              </div>
            )}

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
              <StatCard label="Total debits" value={money(tb.total_debits)} hint="Sum across all accounts" />
              <StatCard label="Total credits" value={money(tb.total_credits)} hint="Sum across all accounts" />
              <StatCard
                label="Status"
                value={tb.balanced ? "Balanced" : "Unbalanced"}
                hint={tb.balanced ? "Debits = credits" : "Investigate immediately"}
              />
            </div>

            <Card className="overflow-hidden">
              <div className="border-b border-border px-6 py-4">
                <h2 className="text-base font-semibold text-foreground">Accounts</h2>
                <p className="text-sm text-muted-foreground">
                  Balance is shown on each account's normal side; a wrong sign is flagged.
                </p>
              </div>
              {lines.length === 0 ? (
                <EmptyState
                  icon={Scale}
                  title="No ledger activity yet"
                  description="Once invoices and payments post to the ledger, the trial balance appears here."
                />
              ) : (
                <div className="overflow-x-auto">
                  <Table>
                    <TableHeader>
                      <TableRow className="bg-muted/40 hover:bg-muted/40">
                        <TableHead>Account</TableHead>
                        <TableHead>Type</TableHead>
                        <TableHead className="text-right">Debits</TableHead>
                        <TableHead className="text-right">Credits</TableHead>
                        <TableHead className="text-right">Balance</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {lines.map((l) => (
                        <TableRow key={l.account_id}>
                          <TableCell className="text-foreground">
                            <span className="font-mono text-xs text-muted-foreground">{l.code}</span>{" "}
                            {l.name}
                            {l.abnormal && (
                              <Badge variant="destructive" className="ml-2">
                                abnormal
                              </Badge>
                            )}
                          </TableCell>
                          <TableCell className="text-muted-foreground">{typeLabel(l.type)}</TableCell>
                          <TableCell className="text-right font-mono text-sm tabular-nums text-foreground">
                            {money(l.debits)}
                          </TableCell>
                          <TableCell className="text-right font-mono text-sm tabular-nums text-foreground">
                            {money(l.credits)}
                          </TableCell>
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
              )}
            </Card>
          </div>
        )
      )}
    </div>
  );
}
