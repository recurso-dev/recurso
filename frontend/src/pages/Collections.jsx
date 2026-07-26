import { useState } from "react";
import { useQuery, useMutation, useQueryClient, keepPreviousData } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import {
  Inbox, RefreshCw, AlertTriangle, Ban, CircleDollarSign, Clock, Percent, RotateCcw,
  MoreHorizontal, Play, Pause,
} from "lucide-react";

import { endpoints } from "../lib/api";
import { formatCurrency } from "@/lib/utils";
import { PageHeader } from "@/components/patterns/PageHeader";
import { StatCard } from "@/components/patterns/StatCard";
import { EmptyState } from "@/components/patterns/EmptyState";
import { Skeleton } from "@/components/patterns/LoadingSkeleton";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { toast } from "@/components/ui/sonner";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

const PER_PAGE = 25;

// Humanize the raw gateway / ACH failure codes stored on the last attempt so an
// operator reads "Insufficient funds" instead of "insufficient_funds". Unknown
// codes fall back to a title-cased version of the code itself.
const FAILURE_LABELS = {
  card_declined: "Card declined",
  insufficient_funds: "Insufficient funds",
  expired_card: "Expired card",
  incorrect_cvc: "Incorrect CVC",
  processing_error: "Processing error",
  do_not_honor: "Do not honor",
  lost_card: "Lost card",
  stolen_card: "Stolen card",
  authentication_required: "Authentication required",
  ach_return: "ACH return",
  R01: "ACH: insufficient funds",
  R02: "ACH: account closed",
  R03: "ACH: no account",
  R08: "ACH: payment stopped",
  R10: "ACH: unauthorized",
};
const humanizeFailure = (code) => {
  if (!code) return "—";
  if (FAILURE_LABELS[code]) return FAILURE_LABELS[code];
  return code
    .replace(/_/g, " ")
    .replace(/\b\w/g, (c) => c.toUpperCase());
};

const STATUS_TABS = [
  { value: "all", label: "All" },
  { value: "past_due", label: "Past due" },
  { value: "uncollectible", label: "Uncollectible" },
];

const relativeRetry = (iso) => {
  if (!iso) return "—";
  const then = new Date(iso).getTime();
  const now = Date.now();
  const diffMin = Math.round((then - now) / 60000);
  if (Math.abs(diffMin) < 60) return diffMin <= 0 ? "due now" : `in ${diffMin}m`;
  const diffHr = Math.round(diffMin / 60);
  if (Math.abs(diffHr) < 48) return diffHr <= 0 ? `${-diffHr}h ago` : `in ${diffHr}h`;
  return new Date(iso).toLocaleDateString();
};

// RowActions is the per-invoice manual-controls menu (Inc 3): retry now, pause /
// resume dunning, and mark uncollectible. Each mutation refreshes the queue +
// funnel on success and surfaces the server's precise refusal (e.g. mandate /
// in-flight → 409) as a toast.
const RowActions = ({ item }) => {
  const queryClient = useQueryClient();
  const [confirmWriteOff, setConfirmWriteOff] = useState(false);

  const refresh = () => {
    queryClient.invalidateQueries({ queryKey: ["collections-queue"] });
    queryClient.invalidateQueries({ queryKey: ["collections-analytics"] });
  };
  const onError = (err, fallback) =>
    toast.error(err?.response?.data?.error?.message || fallback);

  const retry = useMutation({
    mutationFn: () => endpoints.collectionsRetryNow(item.id),
    onSuccess: () => {
      toast.success(`Retry scheduled for ${item.invoice_number}`);
      refresh();
    },
    onError: (e) => onError(e, "Could not retry this invoice"),
  });
  const pause = useMutation({
    mutationFn: (paused) => endpoints.collectionsPauseDunning(item.id, paused),
    onSuccess: (_res, paused) => {
      toast.success(paused ? "Dunning paused" : "Dunning resumed");
      refresh();
    },
    onError: (e) => onError(e, "Could not update dunning"),
  });
  const writeOff = useMutation({
    mutationFn: () => endpoints.collectionsMarkUncollectible(item.id),
    onSuccess: () => {
      toast.success(`${item.invoice_number} written off`);
      setConfirmWriteOff(false);
      refresh();
    },
    onError: (e) => {
      onError(e, "Could not write off this invoice");
      setConfirmWriteOff(false);
    },
  });

  const isUncollectible = item.status === "uncollectible";
  const busy = retry.isPending || pause.isPending || writeOff.isPending;

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="sm" className="h-8 w-8 p-0" aria-label="Invoice actions" disabled={busy}>
            <MoreHorizontal className="h-4 w-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          {!isUncollectible && (
            <DropdownMenuItem onClick={() => retry.mutate()} disabled={item.dunning_paused}>
              <RefreshCw className="mr-2 h-4 w-4" /> Retry now
            </DropdownMenuItem>
          )}
          <DropdownMenuItem onClick={() => pause.mutate(!item.dunning_paused)}>
            {item.dunning_paused ? (
              <>
                <Play className="mr-2 h-4 w-4" /> Resume dunning
              </>
            ) : (
              <>
                <Pause className="mr-2 h-4 w-4" /> Pause dunning
              </>
            )}
          </DropdownMenuItem>
          {!isUncollectible && (
            <>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                className="text-red-600 focus:text-red-600"
                onClick={() => setConfirmWriteOff(true)}
              >
                <Ban className="mr-2 h-4 w-4" /> Mark uncollectible
              </DropdownMenuItem>
            </>
          )}
        </DropdownMenuContent>
      </DropdownMenu>

      <ConfirmDialog
        open={confirmWriteOff}
        onOpenChange={setConfirmWriteOff}
        title={`Write off ${item.invoice_number}?`}
        description="This marks the invoice uncollectible and stops all dunning. It won't be collected automatically anymore. You can still record a manual payment later."
        confirmLabel="Mark uncollectible"
        destructive
        busy={writeOff.isPending}
        onConfirm={() => writeOff.mutate()}
      />
    </>
  );
};

const Collections = () => {
  const [status, setStatus] = useState("all");
  const [page, setPage] = useState(1);

  const params = { page, per_page: PER_PAGE };
  if (status !== "all") params.status = status;

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ["collections-queue", status, page],
    queryFn: async () => {
      const res = await endpoints.getCollectionsQueue(params);
      return res.data;
    },
    placeholderData: keepPreviousData,
  });

  // Funnel + failure breakdown are tenant-wide and FX-normalized — independent of
  // the queue's status filter / page, so they get their own cached query.
  const { data: analytics } = useQuery({
    queryKey: ["collections-analytics"],
    queryFn: async () => {
      const [funnelRes, failuresRes] = await Promise.all([
        endpoints.getCollectionsFunnel(),
        endpoints.getCollectionsFailures(),
      ]);
      return {
        funnel: funnelRes.data?.data ?? null,
        failures: failuresRes.data?.data ?? [],
      };
    },
  });

  const items = data?.data ?? [];
  const total = data?.meta?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / PER_PAGE));
  const loadError = isError
    ? error?.response?.data?.error?.message || error?.message || "Failed to load the collections queue"
    : null;

  const funnel = analytics?.funnel ?? null;
  const failures = analytics?.failures ?? [];
  const reportingCurrency = funnel?.reporting_currency || "USD";
  const maxFailureAmount = failures.reduce((m, f) => Math.max(m, f.amount_at_risk || 0), 0);

  const onStatusChange = (v) => {
    setStatus(v);
    setPage(1);
  };

  return (
    <div>
      <PageHeader
        title="Collections"
        description="Every invoice currently in recovery — why it failed, who owns it, and what happens next."
        actions={
          <Button variant="outline" asChild>
            <Link to="/dunning">
              <RefreshCw className="h-4 w-4" />
              Smart Dunning
            </Link>
          </Button>
        }
      />

      {loadError && (
        <p className="mb-4 rounded-md bg-red-50 px-3 py-2 text-sm text-red-800" role="alert">
          {loadError} — refresh to retry.
        </p>
      )}

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          label="Revenue at risk"
          value={formatCurrency(funnel?.past_due?.amount || 0, reportingCurrency)}
          icon={CircleDollarSign}
          hint={`${(funnel?.past_due?.count || 0).toLocaleString()} invoices in dunning`}
        />
        <StatCard
          label="Recovery rate"
          value={funnel ? `${(funnel.recovery_rate * 100).toFixed(1)}%` : "—"}
          icon={Percent}
          hint="recovered vs written off"
        />
        <StatCard
          label="Recovered (all-time)"
          value={formatCurrency(funnel?.recovered?.amount || 0, reportingCurrency)}
          icon={RotateCcw}
          hint={`${(funnel?.recovered?.count || 0).toLocaleString()} invoices`}
        />
        <StatCard
          label="Written off"
          value={formatCurrency(funnel?.uncollectible?.amount || 0, reportingCurrency)}
          icon={Ban}
          hint={`${(funnel?.uncollectible?.count || 0).toLocaleString()} uncollectible`}
        />
      </div>

      {/* Failure reasons ranked by money at risk */}
      {failures.length > 0 && (
        <Card className="mt-6">
          <CardContent className="p-6">
            <div className="mb-4 flex items-center justify-between">
              <h3 className="text-sm font-semibold text-foreground">Top failure reasons</h3>
              <span className="text-xs text-muted-foreground">by revenue at risk ({reportingCurrency})</span>
            </div>
            <ul className="space-y-3">
              {failures.slice(0, 6).map((f) => (
                <li key={f.error_code}>
                  <div className="mb-1 flex items-center justify-between text-sm">
                    <span className="text-foreground">{humanizeFailure(f.error_code)}</span>
                    <span className="tabular-nums text-muted-foreground">
                      {formatCurrency(f.amount_at_risk, reportingCurrency)}
                      <span className="ml-2 text-xs">({f.count})</span>
                    </span>
                  </div>
                  <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
                    <div
                      className="h-full rounded-full bg-amber-500"
                      style={{
                        width: `${maxFailureAmount > 0 ? Math.max(4, (f.amount_at_risk / maxFailureAmount) * 100) : 0}%`,
                      }}
                    />
                  </div>
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      )}

      <div className="mt-6 flex items-center justify-between gap-4">
        <Tabs value={status} onValueChange={onStatusChange}>
          <TabsList>
            {STATUS_TABS.map((t) => (
              <TabsTrigger key={t.value} value={t.value}>
                {t.label}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
      </div>

      <Card className="mt-4">
        <CardContent className="px-0 py-0">
          {isLoading ? (
            <div className="space-y-3 p-6">
              {Array.from({ length: 6 }).map((_, i) => (
                <Skeleton key={i} className="h-10 w-full" />
              ))}
            </div>
          ) : items.length === 0 ? (
            <EmptyState
              icon={Inbox}
              title="Nothing in collections"
              description={
                status === "all"
                  ? "No invoices are currently failing. Recovered and paid invoices don't appear here."
                  : `No ${status.replace("_", " ")} invoices right now.`
              }
            />
          ) : (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow className="bg-muted/40 hover:bg-muted/40">
                    <TableHead className="pl-6">Customer</TableHead>
                    <TableHead>Invoice</TableHead>
                    <TableHead className="text-right">Amount</TableHead>
                    <TableHead className="text-right">Overdue</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Last failure</TableHead>
                    <TableHead>Next retry</TableHead>
                    <TableHead>Owner</TableHead>
                    <TableHead className="pr-6 text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {items.map((it) => (
                    <TableRow key={it.id} className="hover:bg-muted/20">
                      <TableCell className="pl-6">
                        <Link
                          to={`/customers/${it.customer_id}`}
                          className="font-medium text-foreground hover:underline"
                        >
                          {it.customer_name || "Unnamed customer"}
                        </Link>
                        <div className="text-xs text-muted-foreground">{it.customer_email}</div>
                      </TableCell>
                      <TableCell className="font-mono text-sm text-muted-foreground">
                        {it.invoice_number}
                      </TableCell>
                      <TableCell className="text-right tabular-nums font-medium">
                        {formatCurrency(it.amount_remaining, it.currency)}
                      </TableCell>
                      <TableCell className="text-right tabular-nums">
                        <span
                          className={
                            it.days_overdue >= 30
                              ? "text-red-600"
                              : it.days_overdue >= 7
                                ? "text-amber-600"
                                : "text-muted-foreground"
                          }
                        >
                          {it.days_overdue}d
                        </span>
                      </TableCell>
                      <TableCell>
                        {it.status === "uncollectible" ? (
                          <Badge variant="destructive" className="gap-1">
                            <Ban className="h-3 w-3" /> Uncollectible
                          </Badge>
                        ) : (
                          <Badge variant="warning" className="gap-1">
                            <AlertTriangle className="h-3 w-3" /> Past due
                          </Badge>
                        )}
                      </TableCell>
                      <TableCell className="text-sm">
                        <span className="text-foreground">{humanizeFailure(it.last_payment_error)}</span>
                        {it.attempt_status === "returned" && (
                          <Badge variant="outline" className="ml-2 text-xs">
                            returned
                          </Badge>
                        )}
                        {it.attempt_status === "processing" && (
                          <Badge variant="outline" className="ml-2 text-xs">
                            settling
                          </Badge>
                        )}
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {it.dunning_paused ? (
                          <Badge variant="outline" className="gap-1 text-xs">
                            <Pause className="h-3 w-3" /> paused
                          </Badge>
                        ) : (
                          <span className="inline-flex items-center gap-1">
                            {it.status !== "uncollectible" && <Clock className="h-3 w-3" />}
                            {it.status === "uncollectible" ? "—" : relativeRetry(it.next_retry_at)}
                          </span>
                        )}
                      </TableCell>
                      <TableCell className="text-xs capitalize text-muted-foreground">
                        {it.managed_by}
                      </TableCell>
                      <TableCell className="pr-6 text-right">
                        <RowActions item={it} />
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>

      {total > PER_PAGE && (
        <div className="mt-4 flex items-center justify-between text-sm text-muted-foreground">
          <span>
            Page {page} of {totalPages} · {total.toLocaleString()} total
          </span>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={page <= 1}
              onClick={() => setPage((p) => Math.max(1, p - 1))}
            >
              Previous
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={page >= totalPages}
              onClick={() => setPage((p) => p + 1)}
            >
              Next
            </Button>
          </div>
        </div>
      )}
    </div>
  );
};

export default Collections;
