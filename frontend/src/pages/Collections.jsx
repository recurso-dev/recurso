import { useState } from "react";
import { useQuery, keepPreviousData } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Inbox, RefreshCw, AlertTriangle, Ban, CircleDollarSign, Clock } from "lucide-react";

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

  const items = data?.data ?? [];
  const total = data?.meta?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / PER_PAGE));
  const loadError = isError
    ? error?.response?.data?.error?.message || error?.message || "Failed to load the collections queue"
    : null;

  // At-risk amount on this page (a page-scoped hint, not the tenant-wide total —
  // Inc 2 adds the FX-normalized revenue-at-risk figure).
  const pageAtRisk = items.reduce((sum, it) => sum + (it.amount_remaining || 0), 0);
  const pageCurrency = items[0]?.currency || "USD";
  const mixedCurrency = items.some((it) => it.currency !== pageCurrency);

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

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <StatCard label="Invoices in collections" value={total.toLocaleString()} icon={Inbox} />
        <StatCard
          label="At risk (this page)"
          value={formatCurrency(pageAtRisk, pageCurrency)}
          icon={CircleDollarSign}
          hint={mixedCurrency ? "mixed currencies — page subtotal" : `${items.length} shown`}
        />
        <StatCard
          label="Uncollectible"
          value={items.filter((it) => it.status === "uncollectible").length.toLocaleString()}
          icon={Ban}
          hint="on this page"
        />
      </div>

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
                    <TableHead className="pr-6">Owner</TableHead>
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
                        <span className="inline-flex items-center gap-1">
                          {it.status !== "uncollectible" && <Clock className="h-3 w-3" />}
                          {it.status === "uncollectible" ? "—" : relativeRetry(it.next_retry_at)}
                        </span>
                      </TableCell>
                      <TableCell className="pr-6 text-xs capitalize text-muted-foreground">
                        {it.managed_by}
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
