import { useCallback, useEffect, useState } from "react";
import { AlertTriangle, CheckCircle2, Info, RefreshCw, ShieldCheck } from "lucide-react";

import { endpoints } from "../lib/api";
import { PageHeader } from "@/components/patterns/PageHeader";
import { StatCard } from "@/components/patterns/StatCard";
import { EmptyState } from "@/components/patterns/EmptyState";
import { ErrorState } from "@/components/patterns/ErrorState";
import { CardGridSkeleton } from "@/components/patterns/LoadingSkeleton";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

// Human labels for the backend's discrepancy type constants
// (internal/service/reconciliation.go).
const DISCREPANCY_LABELS = {
  missing_invoice_transaction: "Missing invoice transaction",
  invoice_amount_mismatch: "Invoice amount mismatch",
  missing_payment_transaction: "Missing payment transaction",
  payment_amount_mismatch: "Payment amount mismatch",
  orphaned_transaction: "Orphaned transaction",
  missing_in_tigerbeetle: "Missing in TigerBeetle",
  missing_in_postgres: "Missing in Postgres",
  tb_amount_mismatch: "TigerBeetle amount mismatch",
};

const shortId = (id) => (id ? `${id.substring(0, 8)}…` : "—");

// Discrepancy amounts are minor units (cents/paise); the report carries no
// currency, so render them as plain integers rather than guessing a symbol.
const formatMinorUnits = (n) => (typeof n === "number" ? n.toLocaleString() : "—");

export default function FinanceReconciliation() {
  const [report, setReport] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const runReconciliation = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await endpoints.runReconciliation();
      setReport(res.data?.data || null);
    } catch (err) {
      setError(err?.response?.data?.error?.message || "Failed to run reconciliation");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    runReconciliation();
  }, [runReconciliation]);

  const discrepancies = report?.discrepancies || [];
  const totalDiscrepancies = report?.total_discrepancies || 0;
  const booksBalanced = report && totalDiscrepancies === 0;

  return (
    <div>
      <PageHeader
        title="Reconciliation"
        description="On-demand check that billing records, the Postgres ledger, and TigerBeetle agree."
        actions={
          <Button onClick={runReconciliation} disabled={loading}>
            <RefreshCw className={loading ? "h-4 w-4 animate-spin" : "h-4 w-4"} />
            {loading ? "Running…" : "Run again"}
          </Button>
        }
      />

      {loading ? (
        <CardGridSkeleton count={4} />
      ) : error ? (
        <Card className="overflow-hidden">
          <ErrorState message={error} onRetry={runReconciliation} />
        </Card>
      ) : (
        report && (
          <div className="flex flex-col gap-6">
            {/* Summary cards */}
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
              <StatCard
                label="Invoices Checked"
                value={(report.invoices_checked || 0).toLocaleString()}
                hint={`${(report.paid_invoices_checked || 0).toLocaleString()} paid invoices`}
              />
              <StatCard
                label="Discrepancies"
                value={totalDiscrepancies.toLocaleString()}
                hint={
                  booksBalanced
                    ? "Nothing out of place"
                    : `${discrepancies.length.toLocaleString()} listed below`
                }
              />

              {/* TigerBeetle comparison status */}
              <Card className="p-5">
                <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                  TigerBeetle
                </p>
                <div className="mt-3">
                  {report.tb_compared ? (
                    <Badge variant="success">
                      <CheckCircle2 className="h-3.5 w-3.5" />
                      Compared
                    </Badge>
                  ) : (
                    <Badge
                      variant="warning"
                      title={report.tb_skip_reason || "Comparison skipped"}
                      data-testid="tb-skipped-badge"
                      className="cursor-help"
                    >
                      <Info className="h-3.5 w-3.5" />
                      Skipped
                    </Badge>
                  )}
                </div>
                <p className="mt-2 text-xs text-muted-foreground">
                  {report.tb_compared
                    ? `${(report.tb_accounts_checked || 0).toLocaleString()} accounts · ${(report.tb_transfers_checked || 0).toLocaleString()} transfers`
                    : report.tb_skip_reason || "Comparison skipped"}
                </p>
              </Card>

              <StatCard
                label="Last Run"
                value={
                  report.finished_at
                    ? new Date(report.finished_at).toLocaleTimeString()
                    : "—"
                }
                hint={
                  report.finished_at
                    ? new Date(report.finished_at).toLocaleDateString()
                    : ""
                }
              />
            </div>

            {/* Truncation notice */}
            {report.truncated && (
              <div className="flex items-center gap-3 rounded-lg bg-amber-50 p-4 text-amber-800 ring-1 ring-inset ring-amber-200">
                <AlertTriangle className="h-5 w-5 flex-shrink-0" />
                <p className="text-sm font-medium">
                  Showing the first {discrepancies.length.toLocaleString()} of{" "}
                  {totalDiscrepancies.toLocaleString()} discrepancies. Resolve these
                  and run again to see the rest.
                </p>
              </div>
            )}

            {/* Discrepancies */}
            {booksBalanced ? (
              <Card className="overflow-hidden">
                <EmptyState
                  icon={ShieldCheck}
                  title="Books balanced"
                  description="Every invoice and payment agrees with the ledger. Nothing to fix here."
                />
              </Card>
            ) : (
              <Card className="overflow-hidden">
                <div className="border-b border-border px-6 py-4">
                  <h2 className="text-base font-semibold text-foreground">
                    Discrepancies
                  </h2>
                  <p className="text-sm text-muted-foreground">
                    Disagreements between billing records and the ledger. Amounts are
                    in minor units.
                  </p>
                </div>
                <Table>
                  <TableHeader>
                    <TableRow className="bg-muted/40 hover:bg-muted/40">
                      <TableHead>Type</TableHead>
                      <TableHead>Invoice</TableHead>
                      <TableHead>Transaction</TableHead>
                      <TableHead>Reference</TableHead>
                      <TableHead className="text-right">Expected</TableHead>
                      <TableHead className="text-right">Found</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {discrepancies.map((d, i) => (
                      <TableRow
                        key={`${d.type}-${d.invoice_id || d.transaction_id || i}`}
                      >
                        <TableCell>
                          <Badge variant="destructive">
                            {DISCREPANCY_LABELS[d.type] || d.type}
                          </Badge>
                        </TableCell>
                        <TableCell
                          className="font-mono text-xs text-muted-foreground"
                          title={d.invoice_id || undefined}
                        >
                          {shortId(d.invoice_id)}
                        </TableCell>
                        <TableCell
                          className="font-mono text-xs text-muted-foreground"
                          title={d.transaction_id || undefined}
                        >
                          {shortId(d.transaction_id)}
                        </TableCell>
                        <TableCell
                          className="font-mono text-xs text-muted-foreground"
                          title={d.reference_id || undefined}
                        >
                          {shortId(d.reference_id)}
                        </TableCell>
                        <TableCell className="text-right font-mono text-sm text-foreground">
                          {formatMinorUnits(d.expected_amount)}
                        </TableCell>
                        <TableCell className="text-right font-mono text-sm text-foreground">
                          {formatMinorUnits(d.found_amount)}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </Card>
            )}
          </div>
        )
      )}
    </div>
  );
}
