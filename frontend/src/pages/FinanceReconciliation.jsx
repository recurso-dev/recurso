import { shortId, formatDate, formatDateTime } from "@/lib/utils";
import {
  DISCREPANCIES,
  formatMinorUnits,
  formatDifference,
} from "@/lib/reconciliationDiscrepancies";
import { useCallback, useEffect, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router";
import { AlertTriangle, CheckCircle2, Info, RefreshCw, ShieldCheck, History } from "lucide-react";

import { endpoints } from "../lib/api";
import { PageHeader } from "@/components/patterns/PageHeader";
import { StatCard } from "@/components/patterns/StatCard";
import { EmptyState } from "@/components/patterns/EmptyState";
import { ErrorState } from "@/components/patterns/ErrorState";
import { CardGridSkeleton } from "@/components/patterns/LoadingSkeleton";
import { MotionReveal } from "@/components/patterns/MotionReveal";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Overline } from "@/components/ui/overline";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

// Discrepancy label/reason wording + money formatters are shared with the
// recorded-run object page (lib/reconciliationDiscrepancies).

export default function FinanceReconciliation() {
  const [report, setReport] = useState(null);
  const [loading, setLoading] = useState(true);
  const [recording, setRecording] = useState(false);
  const [error, setError] = useState(null);
  const queryClient = useQueryClient();

  // The page auto-loads an ephemeral (GET) reconciliation so nothing is recorded
  // just by opening it; the "Run & record" button below writes to the audit trail.
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

  // Explicit run that also records a summary to the run history (the audit trail).
  const runAndRecord = useCallback(async () => {
    setRecording(true);
    setError(null);
    try {
      const res = await endpoints.recordReconciliation();
      setReport(res.data?.data || null);
      queryClient.invalidateQueries({ queryKey: ["reconciliation-runs"] });
    } catch (err) {
      setError(err?.response?.data?.error?.message || "Failed to run reconciliation");
    } finally {
      setRecording(false);
    }
  }, [queryClient]);

  useEffect(() => {
    runReconciliation();
  }, [runReconciliation]);

  // The recorded run history — best-effort; an empty list just hides the section.
  const { data: runs = [] } = useQuery({
    queryKey: ["reconciliation-runs"],
    queryFn: async () => (await endpoints.getReconciliationRuns({ limit: 20 })).data.data || [],
  });

  // Resolve run_by (a user id) to a human name — the audit trail should never
  // show a raw UUID. Shared ["team"] cache with the Team page; best-effort, so a
  // failure just falls back to a short id. Only fetched when there's history.
  const { data: teamMembers = [] } = useQuery({
    queryKey: ["team"],
    queryFn: async () => (await endpoints.getUsers()).data?.data || [],
    enabled: runs.length > 0,
  });
  const userNameById = Object.fromEntries(
    teamMembers.map((u) => [u.id, u.name || u.email]),
  );

  const discrepancies = report?.discrepancies || [];
  const totalDiscrepancies = report?.total_discrepancies || 0;
  const booksBalanced = report && totalDiscrepancies === 0;
  const reportCurrency = report?.reporting_currency || "USD";

  return (
    <div>
      <PageHeader
        title="Reconciliation"
        description="On-demand check that billing records, the Postgres ledger, and TigerBeetle agree."
        actions={
          <Button onClick={runAndRecord} disabled={loading || recording}>
            <RefreshCw className={recording ? "h-4 w-4 animate-spin" : "h-4 w-4"} />
            {recording ? "Recording…" : "Run & record"}
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
            {/* Verdict — the one thing a finance operator checks first. Keyed
                on the run so it settles in each time reconciliation completes:
                the resolution reads as an event, not a static state. */}
            <MotionReveal key={report.finished_at || totalDiscrepancies}>
              <div
                className={
                  booksBalanced
                    ? "flex items-center gap-3 rounded-lg border border-success/30 bg-success/5 px-5 py-4"
                    : "flex items-center gap-3 rounded-lg border border-destructive/30 bg-destructive/5 px-5 py-4"
                }
              >
                {booksBalanced ? (
                  <ShieldCheck className="h-5 w-5 shrink-0 text-success" aria-hidden="true" />
                ) : (
                  <AlertTriangle className="h-5 w-5 shrink-0 text-destructive" aria-hidden="true" />
                )}
                <div>
                  <p className={booksBalanced ? "text-sm font-semibold text-success" : "text-sm font-semibold text-destructive"}>
                    {booksBalanced ? "Reconciled" : `${totalDiscrepancies.toLocaleString()} discrepanc${totalDiscrepancies === 1 ? "y" : "ies"} to resolve`}
                  </p>
                  <p className="text-sm text-muted-foreground">
                    {booksBalanced
                      ? "Every invoice, payment, and credit ties to the ledger, and debits equal credits."
                      : "Billing records and the ledger disagree — each row below shows what, by how much, and why."}
                  </p>
                </div>
              </div>
            </MotionReveal>

            {/* Summary cards */}
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
              <StatCard
                label="Invoices Checked"
                value={report.invoices_checked || 0}
                hint={`${(report.paid_invoices_checked || 0).toLocaleString()} paid invoices`}
              />
              <StatCard
                label="Discrepancies"
                value={totalDiscrepancies}
                tone={booksBalanced ? undefined : "danger"}
                hint={
                  booksBalanced
                    ? "Nothing out of place"
                    : `${discrepancies.length.toLocaleString()} listed below`
                }
              />

              {/* TigerBeetle comparison status */}
              <Card className="p-5">
                <Overline as="p">
                  TigerBeetle
                </Overline>
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
                hint={report.finished_at ? formatDate(report.finished_at) : ""}
              />
            </div>

            {/* Truncation notice */}
            {report.truncated && (
              <div className="flex items-center gap-3 rounded-lg bg-warning/5 p-4 text-warning ring-1 ring-inset ring-warning/20">
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
                    Disagreements between billing records and the ledger, each with
                    what disagrees, by how much, and why.
                  </p>
                </div>
                <Table>
                  <TableHeader>
                    <TableRow className="bg-muted/40 hover:bg-muted/40">
                      <TableHead>What disagrees</TableHead>
                      <TableHead>Invoice</TableHead>
                      <TableHead>Transaction</TableHead>
                      <TableHead className="text-right">Expected</TableHead>
                      <TableHead className="text-right">Found</TableHead>
                      <TableHead className="text-right">Difference</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {discrepancies.map((d, i) => (
                      <TableRow
                        key={`${d.type}-${d.invoice_id || d.transaction_id || i}`}
                      >
                        <TableCell className="max-w-xs">
                          <Badge variant="destructive">
                            {DISCREPANCIES[d.type]?.label || d.type}
                          </Badge>
                          {DISCREPANCIES[d.type]?.reason && (
                            <p className="mt-1 text-xs leading-snug text-muted-foreground">
                              {DISCREPANCIES[d.type].reason}
                            </p>
                          )}
                        </TableCell>
                        <TableCell
                          className="font-mono text-xs text-muted-foreground"
                          title={d.invoice_id || undefined}
                        >
                          {d.invoice_id ? (
                            <Link
                              to={`/invoices/${d.invoice_id}`}
                              className="text-primary underline-offset-2 hover:underline"
                            >
                              {shortId(d.invoice_id)}
                            </Link>
                          ) : (
                            shortId(d.invoice_id)
                          )}
                        </TableCell>
                        <TableCell
                          className="font-mono text-xs text-muted-foreground"
                          title={d.transaction_id || d.reference_id || undefined}
                        >
                          {d.transaction_id ? (
                            <Link
                              to={`/ledger/transactions/${d.transaction_id}`}
                              className="text-primary underline-offset-2 hover:underline"
                            >
                              {shortId(d.transaction_id)}
                            </Link>
                          ) : (
                            shortId(d.transaction_id || d.reference_id)
                          )}
                        </TableCell>
                        <TableCell className="text-right font-mono text-sm text-foreground">
                          {formatMinorUnits(d.expected_amount, reportCurrency)}
                        </TableCell>
                        <TableCell className="text-right font-mono text-sm text-foreground">
                          {formatMinorUnits(d.found_amount, reportCurrency)}
                        </TableCell>
                        <TableCell className="text-right font-mono text-sm font-medium text-destructive">
                          {formatDifference(d, reportCurrency)}
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

      {runs.length > 0 && (
        <div className="mt-8">
          <div className="mb-3 flex items-center gap-2">
            <History className="h-4 w-4 text-muted-foreground" />
            <h2 className="text-base font-semibold text-foreground">Run history</h2>
            <span className="text-xs text-muted-foreground">
              — recorded reconciliations, newest first
            </span>
          </div>
          <Card>
            <Table>
              <TableHeader>
                <TableRow className="bg-muted/40 hover:bg-muted/40">
                  <TableHead>When</TableHead>
                  <TableHead>Result</TableHead>
                  <TableHead className="text-right">Invoices checked</TableHead>
                  <TableHead>TigerBeetle</TableHead>
                  <TableHead>Recorded by</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {runs.map((r) => {
                  const balanced = (r.total_discrepancies || 0) === 0;
                  // null run_by = a scheduled/system run (no user). A non-null id
                  // that isn't in the team map (removed teammate) falls back to a
                  // short id rather than showing nothing.
                  const recordedBy = r.run_by
                    ? userNameById[r.run_by] || shortId(r.run_by)
                    : "System";
                  return (
                    <TableRow key={r.id}>
                      <TableCell className="text-sm">
                        <Link
                          to={`/finance/reconciliation/runs/${r.id}`}
                          className="text-primary underline-offset-2 hover:underline"
                        >
                          {formatDateTime(r.run_at)}
                        </Link>
                      </TableCell>
                      <TableCell>
                        {balanced ? (
                          <Badge variant="success">Reconciled</Badge>
                        ) : (
                          <Badge variant="destructive">
                            {r.total_discrepancies.toLocaleString()} discrepanc
                            {r.total_discrepancies === 1 ? "y" : "ies"}
                          </Badge>
                        )}
                      </TableCell>
                      <TableCell className="text-right font-mono text-sm text-foreground">
                        {(r.invoices_checked ?? 0).toLocaleString()}
                      </TableCell>
                      <TableCell>
                        {r.tb_compared ? (
                          <span
                            className="text-sm text-muted-foreground"
                            title={`${(r.tb_accounts_checked ?? 0).toLocaleString()} accounts · ${(r.tb_transfers_checked ?? 0).toLocaleString()} transfers`}
                          >
                            Compared
                          </span>
                        ) : (
                          <span className="text-sm text-muted-foreground">—</span>
                        )}
                      </TableCell>
                      <TableCell
                        className="text-sm text-muted-foreground"
                        title={r.run_by || undefined}
                      >
                        {recordedBy}
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </Card>
        </div>
      )}
    </div>
  );
}
