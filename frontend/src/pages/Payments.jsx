import { useQuery } from "@tanstack/react-query";
import { useNavigate, Link } from "react-router";
import { CreditCard } from "lucide-react";

import { endpoints as api } from "../lib/api";
import { useUrlState, useResetPageOnChange } from "@/lib/useUrlState";
import { formatDateTime, shortId } from "@/lib/utils";
import { humanizeFailure } from "@/lib/failureLabels";
import { Money } from "@/components/ui/money";
import { StatusBadge } from "@/components/ui/status-badge";
import { PageHeader } from "@/components/patterns/PageHeader";
import { DataTable } from "@/components/patterns/DataTable";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const PAGE_SIZE = 50;

// Payment-attempt lifecycle: cards jump to succeeded/failed; ACH moves
// initiated → processing → succeeded, and can go succeeded → returned days
// later when the bank claws it back. Rendered by the canonical StatusBadge
// (returned/initiated live in its registry) — no bespoke status pill.
//
// Exceptions first: the statuses an operator most often filters to lead.
const STATUS_FILTERS = [
  { value: "all", label: "All statuses" },
  { value: "failed", label: "Failed" },
  { value: "returned", label: "Returned" },
  { value: "processing", label: "Processing" },
  { value: "initiated", label: "Initiated" },
  { value: "succeeded", label: "Succeeded" },
];


// The tenant-wide payments log: every gateway settlement attempt across all
// invoices, newest first. The one place to answer "did this collection go
// through, and if not, why?" without opening each invoice.
const Payments = () => {
  const navigate = useNavigate();
  // List state in the URL so returning from an invoice restores the payments
  // filter + page (useUrlState).
  const [status, setStatus] = useUrlState("status", "all");
  const [page, setPage] = useUrlState("page", 1, { parse: Number }); // 1-based, matches the API

  const {
    data,
    isLoading,
    error: queryError,
    refetch,
  } = useQuery({
    queryKey: ["payment-attempts", { status, page }],
    queryFn: async () => {
      const params = { page, per_page: PAGE_SIZE };
      if (status !== "all") params.status = status;
      return (await api.getPaymentAttempts(params)).data;
    },
    placeholderData: (prev) => prev,
  });

  // Reset to page 1 when the status filter changes (URL-safe: separate tick,
  // skips mount so a page restored from the URL survives).
  useResetPageOnChange(setPage, [status]);

  const attempts = data?.data || [];
  const total = data?.pagination?.total ?? 0;
  const error = queryError
    ? queryError?.response?.data?.error?.message || "Failed to load payments"
    : null;

  const columns = [
    {
      key: "created",
      header: "When",
      className: "whitespace-nowrap",
      cell: (p) => (
        <span className="text-sm text-muted-foreground">{formatDateTime(p.created_at)}</span>
      ),
    },
    {
      key: "invoice",
      header: "Invoice",
      cell: (p) =>
        p.invoice_id ? (
          <Link
            to={`/invoices/${p.invoice_id}`}
            onClick={(e) => e.stopPropagation()}
            className="font-medium text-primary hover:underline"
          >
            {p.invoice_number || shortId(p.invoice_id)}
          </Link>
        ) : (
          <span className="text-muted-foreground">—</span>
        ),
    },
    {
      key: "amount",
      header: "Amount",
      align: "right",
      cell: (p) => (
        <Money amountMinor={p.amount} currency={p.currency || "USD"} className="font-medium" />
      ),
    },
    {
      key: "method",
      header: "Method",
      cell: (p) => (
        <div>
          <span className="text-sm capitalize text-foreground">
            {(p.method || "—").replace(/_/g, " ")}
          </span>
          {p.gateway ? (
            <p className="text-xs capitalize text-muted-foreground">{p.gateway}</p>
          ) : null}
        </div>
      ),
    },
    {
      key: "status",
      header: "Status",
      cell: (p) => <StatusBadge status={p.status} />,
    },
    {
      key: "failure",
      header: "Reason",
      // Lead with the human-readable reason; the raw gateway code stays as a
      // quiet technical detail in the title (mirrors PaymentAttempts).
      cell: (p) =>
        p.failure_code ? (
          <span className="text-sm text-destructive" title={`Gateway failure code: ${p.failure_code}`}>
            {humanizeFailure(p.failure_code)}
          </span>
        ) : (
          <span className="text-muted-foreground">—</span>
        ),
    },
  ];

  return (
    <div>
      <PageHeader
        title="Payments"
        description="Every gateway settlement attempt across your invoices — newest first. Filter to failed or returned to work the exceptions."
        actions={
          <Select
            value={status}
            onValueChange={setStatus}
          >
            <SelectTrigger className="w-[180px]" aria-label="Filter by status">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {STATUS_FILTERS.map((f) => (
                <SelectItem key={f.value} value={f.value}>
                  {f.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        }
      />

      <DataTable
        columns={columns}
        data={attempts}
        loading={isLoading}
        error={error}
        onRetry={refetch}
        onRowClick={(p) => p.id && navigate(`/payments/${p.id}`)}
        pagination={{
          page,
          pageSize: PAGE_SIZE,
          total,
          onPageChange: setPage,
        }}
        empty={{
          icon: CreditCard,
          title: status !== "all" ? `No ${status} payments` : "No payment attempts yet",
          description:
            "Gateway payment attempts appear here as invoices are collected — a card charge, an ACH debit, a retry.",
        }}
      />
    </div>
  );
};

export default Payments;
