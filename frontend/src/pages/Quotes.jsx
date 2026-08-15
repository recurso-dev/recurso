import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router";
import { FileText, Plus, Send, ArrowRight, MoreHorizontal } from "lucide-react";

import { endpoints } from "../lib/api";
import { CustomerName } from "@/components/patterns/CustomerSelect";
import { useCustomers } from "@/lib/useCustomers";
import { useDebounce } from "../hooks/useDebounce";
import { useUrlState, useResetPageOnChange } from "@/lib/useUrlState";
import { LIST_PAGE_SIZE, pageOffset } from "@/lib/pagination";
import { formatDate } from "@/lib/utils";
import { PageHeader } from "@/components/patterns/PageHeader";
import { DataTable } from "@/components/patterns/DataTable";
import { Money } from "@/components/ui/money";
import { toast } from "@/components/ui/sonner";
import { StatusBadge } from "@/components/ui/status-badge";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const Quotes = () => {
  const { names: customerNames } = useCustomers();
  // List state in the URL so returning from a quote restores search / status /
  // page (useUrlState).
  const [searchQuery, setSearchQuery] = useUrlState("q", "");
  const [statusFilter, setStatusFilter] = useUrlState("status", "");
  const [page, setPage] = useUrlState("page", 1, { parse: Number });
  const debouncedSearch = useDebounce(searchQuery, 400);
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  // Server-driven by status + search + page: /quotes filters and paginates
  // server-side, so each (status, search, page) is its own cache entry;
  // placeholderData keeps the current rows while the next query loads.
  const {
    data: quotes = [],
    isLoading: loading,
    error: queryError,
    refetch,
  } = useQuery({
    queryKey: ["quotes", { status: statusFilter, search: debouncedSearch, page }],
    queryFn: async () => {
      const params = { limit: LIST_PAGE_SIZE, offset: pageOffset(page) };
      if (statusFilter) params.status = statusFilter;
      if (debouncedSearch) params.search = debouncedSearch;
      return (await endpoints.getQuotes(params)).data.data || [];
    },
    placeholderData: (prev) => prev,
  });
  const error = queryError
    ? queryError?.response?.data?.error?.message || queryError?.message || "Failed to load quotes"
    : null;

  // A new status / search starts back at page 1 (URL-safe: separate tick, skips
  // mount). Keyed on the debounced search so it aligns with the query refetch.
  useResetPageOnChange(setPage, [statusFilter, debouncedSearch]);

  const invalidateQuotes = () => queryClient.invalidateQueries({ queryKey: ["quotes"] });

  const sendMutation = useMutation({
    mutationFn: (id) => endpoints.sendQuote(id),
    onSuccess: () => {
      invalidateQuotes();
      toast.success("Quote sent.");
    },
    onError: (err) =>
      toast.error(err?.response?.data?.error?.message || err?.message || "Failed to send quote"),
  });
  const convertMutation = useMutation({
    mutationFn: (id) => endpoints.convertQuoteToInvoice(id),
    onSuccess: () => {
      invalidateQuotes();
      // A converted quote becomes an invoice — refresh the invoices list too.
      queryClient.invalidateQueries({ queryKey: ["invoices"] });
      toast.success("Quote converted to an invoice.");
    },
    onError: (err) =>
      toast.error(
        err?.response?.data?.error?.message || err?.message || "Failed to convert quote"
      ),
  });

  // One-click money ops get a confirm step (audit §7: quote send/convert
  // were fire-on-click icon buttons). { action: "send" | "convert", id }.
  const [confirmOp, setConfirmOp] = useState(null);

  const handleSend = (id, e) => {
    e?.stopPropagation();
    setConfirmOp({ action: "send", id });
  };

  const handleConvert = (id, e) => {
    e?.stopPropagation();
    setConfirmOp({ action: "convert", id });
  };

  const runConfirmedOp = () => {
    if (!confirmOp) return;
    const m = confirmOp.action === "send" ? sendMutation : convertMutation;
    m.mutate(confirmOp.id, { onSettled: () => setConfirmOp(null) });
  };

  const columns = [
    {
      key: "quote_number",
      header: "Quote",
      // rowHref wraps this first cell in the row's <Link>, so the cell stays
      // non-interactive — nesting a button inside that link is invalid HTML.
      cell: (q) => (
        <span className="font-medium text-primary hover:underline">
          {q.quote_number}
        </span>
      ),
    },
    {
      key: "customer",
      header: "Customer",
      cell: (q) => <CustomerName id={q.customer_id} names={customerNames} />,
    },
    {
      key: "amount",
      header: "Amount",
      align: "right",
      cell: (q) => <Money amountMinor={q.total} currency={q.currency} />,
    },
    {
      key: "status",
      header: "Status",
      cell: (q) => <StatusBadge status={q.status} />,
    },
    {
      key: "created",
      header: "Created",
      cell: (q) => (
        <span className="text-muted-foreground">{formatDate(q.created_at)}</span>
      ),
    },
    {
      key: "actions",
      header: "",
      align: "right",
      cell: (q) => (
        <div
          className="flex items-center justify-end gap-1"
          onClick={(e) => e.stopPropagation()}
        >
          {q.status === "draft" && (
            <button type="button"
              onClick={(e) => handleSend(q.id, e)}
              className="rounded-md p-1.5 text-info transition-colors hover:bg-info/10"
              title="Send quote"
              aria-label="Send quote"
            >
              <Send className="h-4 w-4" />
            </button>
          )}
          {q.status === "accepted" && !q.invoice_id && (
            <button type="button"
              onClick={(e) => handleConvert(q.id, e)}
              className="rounded-md p-1.5 text-success transition-colors hover:bg-primary/10"
              title="Convert to invoice"
              aria-label="Convert to invoice"
            >
              <ArrowRight className="h-4 w-4" />
            </button>
          )}
          <button type="button"
            onClick={(e) => {
              e.stopPropagation();
              navigate(`/quotes/${q.id}`);
            }}
            className="rounded-md p-1.5 text-subtle transition-colors hover:bg-muted hover:text-foreground"
            title="View details"
            aria-label="View quote details"
          >
            <MoreHorizontal className="h-4 w-4" />
          </button>
        </div>
      ),
    },
  ];

  return (
    <div>
      <PageHeader
        title="Quotes"
        description="Create and manage price quotes for customers."
        actions={
          <Button onClick={() => navigate("/quotes/new")}>
            <Plus className="h-4 w-4" />
            New quote
          </Button>
        }
      />

      <DataTable
        columns={columns}
        data={quotes}
        loading={loading}
        error={error}
        onRetry={refetch}
        rowHref={(row) => `/quotes/${row.id}`}
        search={{
          value: searchQuery,
          onChange: setSearchQuery,
          placeholder: "Search quotes...",
        }}
        toolbar={
          <Select
            value={statusFilter || "all"}
            onValueChange={(v) => setStatusFilter(v === "all" ? "" : v)}
          >
            <SelectTrigger className="w-[150px]">
              <SelectValue placeholder="All status" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All status</SelectItem>
              <SelectItem value="draft">Draft</SelectItem>
              <SelectItem value="sent">Sent</SelectItem>
              <SelectItem value="accepted">Accepted</SelectItem>
              <SelectItem value="declined">Declined</SelectItem>
              <SelectItem value="expired">Expired</SelectItem>
            </SelectContent>
          </Select>
        }
        empty={{
          icon: FileText,
          title:
            searchQuery || statusFilter ? "No matching quotes" : "No quotes yet",
          description:
            searchQuery || statusFilter
              ? "Try adjusting your search or filters."
              : "Create your first quote to send to customers.",
          action:
            !searchQuery && !statusFilter ? (
              <Button onClick={() => navigate("/quotes/new")}>
                <Plus className="h-4 w-4" />
                Create quote
              </Button>
            ) : null,
        }}
        pagination={{
          page,
          onPrev: () => setPage((p) => Math.max(1, p - 1)),
          onNext: () => setPage((p) => p + 1),
          hasNext: quotes.length >= LIST_PAGE_SIZE,
        }}
      />

      <ConfirmDialog
        open={Boolean(confirmOp)}
        onOpenChange={(o) => !o && setConfirmOp(null)}
        title={
          confirmOp?.action === "convert" ? "Convert this quote to an invoice?" : "Send this quote?"
        }
        description={
          confirmOp?.action === "convert"
            ? "An invoice is created for the quoted amount and the quote is locked. This can't be undone."
            : "The quote is emailed to the customer and marked as sent."
        }
        confirmLabel={confirmOp?.action === "convert" ? "Convert to invoice" : "Send quote"}
        busy={sendMutation.isPending || convertMutation.isPending}
        onConfirm={runConfirmedOp}
      />
    </div>
  );
};

export default Quotes;
