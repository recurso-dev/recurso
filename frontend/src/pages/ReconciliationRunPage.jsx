import { Link, useParams } from "react-router";
import { useQuery } from "@tanstack/react-query";

import { endpoints } from "../lib/api";
import { useObjectQuery } from "@/lib/useObjectQuery";
import { shortId, formatDateTime } from "@/lib/utils";
import {
  DISCREPANCIES,
  formatMinorUnits,
  formatDifference,
} from "@/lib/reconciliationDiscrepancies";
import {
  ObjectHeader,
  ObjectPageLayout,
  ObjectSection,
  AttributeList,
  ObjectPageSkeleton,
  ObjectNotFound,
  ObjectPageError,
} from "@/components/patterns/ObjectPage";
import { Badge } from "@/components/ui/badge";
import { CopyableId } from "@/components/ui/copyable-id";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

// Recorded reconciliation runs don't persist a reporting currency, and the
// ledger is single functional currency (ADR-010) — so discrepancy amounts are
// formatted in the tenant's functional currency (USD default), exponent-aware.
const RUN_CURRENCY = "USD";

export default function ReconciliationRunPage() {
  const { id } = useParams();

  const {
    object: run,
    loading,
    notFound,
    isError,
    error,
    refetch,
  } = useObjectQuery(
    ["reconciliation-run", id],
    async () => (await endpoints.getReconciliationRun(id)).data.data,
    { enabled: Boolean(id) }
  );

  // Resolve run_by → a human name (never a raw UUID); best-effort, shared cache.
  const { data: teamMembers = [] } = useQuery({
    queryKey: ["team"],
    queryFn: async () => (await endpoints.getUsers()).data?.data || [],
    enabled: Boolean(run?.run_by),
  });
  const userNameById = Object.fromEntries(teamMembers.map((u) => [u.id, u.name || u.email]));

  if (loading) return <ObjectPageSkeleton />;
  if (notFound) {
    return (
      <ObjectNotFound
        objectLabel="reconciliation run"
        identifier={id ? String(id).slice(0, 8) : undefined}
        backTo="/finance/reconciliation"
        backLabel="Reconciliation"
      />
    );
  }
  if (isError) {
    return (
      <ObjectPageError
        objectLabel="reconciliation run"
        error={error}
        onRetry={refetch}
        backTo="/finance/reconciliation"
        backLabel="Reconciliation"
      />
    );
  }

  const balanced = run.total_discrepancies === 0;
  const discrepancies = run.discrepancies || [];
  const recordedBy = run.run_by ? userNameById[run.run_by] || shortId(run.run_by) : "System";

  return (
    <div>
      <ObjectHeader
        backTo="/finance/reconciliation"
        backLabel="Reconciliation"
        kicker="Reconciliation run"
        title={formatDateTime(run.run_at)}
        badge={
          balanced ? (
            <Badge variant="success">Reconciled</Badge>
          ) : (
            <Badge variant="destructive">
              {run.total_discrepancies.toLocaleString()} discrepanc
              {run.total_discrepancies === 1 ? "y" : "ies"}
            </Badge>
          )
        }
        meta={
          <span className="flex flex-wrap items-center gap-x-3 gap-y-1">
            <CopyableId value={run.id} />
            <span className="text-muted-foreground">by {recordedBy}</span>
          </span>
        }
      />

      <ObjectPageLayout
        rail={
          <ObjectSection title="Scope">
            <AttributeList
              columns={1}
              items={[
                { label: "Run at", value: formatDateTime(run.run_at) },
                { label: "Recorded by", value: recordedBy },
                { label: "Invoices checked", value: (run.invoices_checked ?? 0).toLocaleString() },
                {
                  label: "Paid invoices checked",
                  value: (run.paid_invoices_checked ?? 0).toLocaleString(),
                },
                {
                  label: "TigerBeetle",
                  value: run.tb_compared
                    ? `Compared · ${(run.tb_accounts_checked ?? 0).toLocaleString()} accounts, ${(run.tb_transfers_checked ?? 0).toLocaleString()} transfers`
                    : "Not compared",
                },
                { label: "Run ID", value: <CopyableId value={run.id} /> },
              ]}
            />
          </ObjectSection>
        }
      >
        {/* Verdict — the one thing an auditor needs first. */}
        <ObjectSection title="Result">
          {balanced ? (
            <p className="text-sm text-success">
              The books tied out — billing records, the Postgres ledger
              {run.tb_compared ? ", and TigerBeetle" : ""} all agreed. No discrepancies.
            </p>
          ) : (
            <p className="text-sm text-destructive">
              {run.total_discrepancies.toLocaleString()} discrepanc
              {run.total_discrepancies === 1 ? "y" : "ies"} found — billing records and the ledger
              disagreed. Each row below shows what disagreed, by how much, and why.
            </p>
          )}
        </ObjectSection>

        {/* The persisted discrepancy rows — what disagreed, by how much, and why. */}
        {discrepancies.length > 0 && (
          <ObjectSection title="Discrepancies" flush>
            <div className="overflow-x-auto">
              <Table className="min-w-[720px]">
                <TableHeader>
                  <TableRow>
                    <TableHead>What disagrees</TableHead>
                    <TableHead>Invoice</TableHead>
                    <TableHead>Transaction</TableHead>
                    <TableHead className="text-right">Expected</TableHead>
                    <TableHead className="text-right">Found</TableHead>
                    <TableHead className="text-right">Difference</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {discrepancies.map((d, i) => {
                    const meta = DISCREPANCIES[d.type];
                    return (
                      <TableRow key={`${d.type}-${i}`}>
                        <TableCell>
                          <span className="font-medium text-foreground">
                            {meta?.label || d.type}
                          </span>
                          {meta?.reason ? (
                            <span className="mt-0.5 block max-w-md text-xs text-muted-foreground">
                              {meta.reason}
                            </span>
                          ) : null}
                        </TableCell>
                        <TableCell className="font-mono text-xs text-muted-foreground">
                          {d.invoice_id ? (
                            <Link
                              to={`/invoices/${d.invoice_id}`}
                              className="text-primary underline-offset-2 hover:underline"
                            >
                              {shortId(d.invoice_id)}
                            </Link>
                          ) : (
                            "—"
                          )}
                        </TableCell>
                        <TableCell className="font-mono text-xs text-muted-foreground">
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
                          {formatMinorUnits(d.expected_amount, RUN_CURRENCY)}
                        </TableCell>
                        <TableCell className="text-right font-mono text-sm text-foreground">
                          {formatMinorUnits(d.found_amount, RUN_CURRENCY)}
                        </TableCell>
                        <TableCell className="text-right font-mono text-sm text-foreground">
                          {formatDifference(d, RUN_CURRENCY)}
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </div>
            {run.discrepancies_truncated ? (
              <p className="border-t border-border px-6 py-2.5 text-xs text-muted-foreground">
                Showing {discrepancies.length.toLocaleString()} of{" "}
                {run.total_discrepancies.toLocaleString()} — this run’s detail was capped when
                recorded (or predates per-run detail capture). The count is exact; the stored rows
                are a subset.
              </p>
            ) : null}
          </ObjectSection>
        )}

        {/* Counted discrepancies but stored no rows — an older run recorded
            before per-run detail capture. Honest, not an empty "all clear". */}
        {discrepancies.length === 0 && !balanced && (
          <ObjectSection title="Discrepancies">
            <p className="text-sm text-muted-foreground">
              This run counted {run.total_discrepancies.toLocaleString()} discrepanc
              {run.total_discrepancies === 1 ? "y" : "ies"}, but their detail wasn’t captured —
              it predates per-run discrepancy storage. Re-run reconciliation to see the current
              detail.
            </p>
          </ObjectSection>
        )}
      </ObjectPageLayout>
    </div>
  );
}
