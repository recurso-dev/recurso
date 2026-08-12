import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate, useParams } from "react-router";
import { FileText, Plus, Send, ArrowRight, MoreHorizontal } from "lucide-react";

import { endpoints } from "../lib/api";
import { CustomerName } from "@/components/patterns/CustomerSelect";
import { useCustomers } from "@/lib/useCustomers";
import QuoteDetail from "../components/slide-overs/QuoteDetail";
import { formatDate } from "@/lib/utils";
import { PageHeader } from "@/components/patterns/PageHeader";
import { DataTable } from "@/components/patterns/DataTable";
import { Money } from "@/components/ui/money";
import { toast } from "@/components/ui/sonner";
import { StatusBadge } from "@/components/ui/status-badge";
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
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState("");
  // URL-driven detail (/quotes/:id) — shareable, refresh/back-safe.
  const { id: routeId } = useParams();
  const { data: routedObject } = useQuery({
    queryKey: ["quote", routeId],
    queryFn: async () => (await endpoints.getQuote(routeId)).data.data,
    enabled: Boolean(routeId),
  });
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  // Server-driven by status + search: each combination is its own cache entry;
  // placeholderData keeps the current rows while the next query loads.
  const {
    data: quotes = [],
    isLoading: loading,
    error: queryError,
    refetch,
  } = useQuery({
    queryKey: ["quotes", { status: statusFilter, search: searchQuery }],
    queryFn: async () => {
      const params = {};
      if (statusFilter) params.status = statusFilter;
      if (searchQuery) params.search = searchQuery;
      return (await endpoints.getQuotes(params)).data.data || [];
    },
    placeholderData: (prev) => prev,
  });
  const error = queryError
    ? queryError?.response?.data?.error?.message || queryError?.message || "Failed to load quotes"
    : null;

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

  const handleSend = (id, e) => {
    e?.stopPropagation();
    sendMutation.mutate(id);
  };

  const handleConvert = (id, e) => {
    e?.stopPropagation();
    convertMutation.mutate(id);
  };


  const closeDetail = () => navigate("/quotes");
  const isDetailOpen = Boolean(routeId);
  const selectedQuote = routedObject || null;

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
            <button
              onClick={(e) => handleSend(q.id, e)}
              className="rounded-md p-1.5 text-info transition-colors hover:bg-info/10"
              title="Send quote"
            >
              <Send className="h-4 w-4" />
            </button>
          )}
          {q.status === "accepted" && !q.invoice_id && (
            <button
              onClick={(e) => handleConvert(q.id, e)}
              className="rounded-md p-1.5 text-success transition-colors hover:bg-primary/10"
              title="Convert to invoice"
            >
              <ArrowRight className="h-4 w-4" />
            </button>
          )}
          <button
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
      />

      <QuoteDetail quote={selectedQuote} isOpen={isDetailOpen} onClose={closeDetail} />
    </div>
  );
};

export default Quotes;
