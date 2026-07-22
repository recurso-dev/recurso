import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { FileText, Plus, Send, ArrowRight, MoreHorizontal } from "lucide-react";

import { endpoints } from "../lib/api";
import QuoteDetail from "../components/slide-overs/QuoteDetail";
import { formatCurrency, formatDate } from "@/lib/utils";
import { PageHeader } from "@/components/patterns/PageHeader";
import { DataTable } from "@/components/patterns/DataTable";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const quoteStatusVariant = (status) =>
  ({
    draft: "neutral",
    sent: "info",
    accepted: "success",
    declined: "destructive",
    expired: "warning",
  })[status] || "neutral";

const Quotes = () => {
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState("");
  const [selectedQuote, setSelectedQuote] = useState(null);
  const [isDetailOpen, setIsDetailOpen] = useState(false);
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
  const error = queryError ? "Failed to load quotes" : null;

  const invalidateQuotes = () => queryClient.invalidateQueries({ queryKey: ["quotes"] });

  const sendMutation = useMutation({
    mutationFn: (id) => endpoints.sendQuote(id),
    onSuccess: invalidateQuotes,
    onError: (err) => console.error("Failed to send quote:", err),
  });
  const convertMutation = useMutation({
    mutationFn: (id) => endpoints.convertQuoteToInvoice(id),
    onSuccess: () => {
      invalidateQuotes();
      // A converted quote becomes an invoice — refresh the invoices list too.
      queryClient.invalidateQueries({ queryKey: ["invoices"] });
    },
    onError: (err) => console.error("Failed to convert quote:", err),
  });

  const handleSend = (id, e) => {
    e?.stopPropagation();
    sendMutation.mutate(id);
  };

  const handleConvert = (id, e) => {
    e?.stopPropagation();
    convertMutation.mutate(id);
  };

  const handleRowClick = (quote) => {
    setSelectedQuote(quote);
    setIsDetailOpen(true);
  };

  const closeDetail = () => {
    setIsDetailOpen(false);
    setTimeout(() => setSelectedQuote(null), 300);
  };

  const columns = [
    {
      key: "quote_number",
      header: "Quote",
      cell: (q) => (
        <button
          type="button"
          className="font-medium text-primary hover:underline"
          onClick={(e) => {
            e.stopPropagation();
            handleRowClick(q);
          }}
        >
          {q.quote_number}
        </button>
      ),
    },
    {
      key: "customer",
      header: "Customer",
      cell: (q) => (
        <span className="font-mono text-xs text-muted-foreground">
          {q.customer_id ? `${q.customer_id.substring(0, 8)}…` : "—"}
        </span>
      ),
    },
    {
      key: "amount",
      header: "Amount",
      align: "right",
      cell: (q) => (
        <span className="tabular-nums">{formatCurrency(q.total, q.currency)}</span>
      ),
    },
    {
      key: "status",
      header: "Status",
      cell: (q) => <Badge variant={quoteStatusVariant(q.status)}>{q.status}</Badge>,
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
              className="rounded-md p-1.5 text-blue-600 transition-colors hover:bg-blue-50"
              title="Send quote"
            >
              <Send className="h-4 w-4" />
            </button>
          )}
          {q.status === "accepted" && !q.invoice_id && (
            <button
              onClick={(e) => handleConvert(q.id, e)}
              className="rounded-md p-1.5 text-emerald-600 transition-colors hover:bg-emerald-50"
              title="Convert to invoice"
            >
              <ArrowRight className="h-4 w-4" />
            </button>
          )}
          <button
            onClick={(e) => {
              e.stopPropagation();
              handleRowClick(q);
            }}
            className="rounded-md p-1.5 text-stone-400 transition-colors hover:bg-stone-100 hover:text-stone-900"
            title="View details"
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
        onRowClick={handleRowClick}
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
